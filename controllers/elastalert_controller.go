/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	esv1alpha1 "elastalert/api/v1alpha1"
	"elastalert/controllers/event"
	ob "elastalert/controllers/observer"
	"elastalert/controllers/podspec"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const name = "elastialert-controller"

var log = ctrl.Log.WithName(name)

// ElastalertReconciler reconciles a Elastalert object
type ElastalertReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Observer ob.Manager
}

//+kubebuilder:rbac:groups=es.noah.domain,resources=elastalerts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=es.noah.domain,resources=elastalerts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=es.noah.domain,resources=elastalerts/finalizers,verbs=update
func (r *ElastalertReconciler) Reconcile(ctx context.Context, req reconcile.Request) (ctrl.Result, error) {
	elastalert := &esv1alpha1.Elastalert{}
	err := r.Get(ctx, req.NamespacedName, elastalert)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			r.Observer.StopObserving(req.NamespacedName)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Elastalert from server")
		return ctrl.Result{}, err
	}
	cond := r.findSuccessCondition(elastalert)
	if cond == nil || cond.ObservedGeneration != elastalert.Generation {
		if statusError := ob.UpdateElastalertStatus(r.Client, ctx, elastalert, esv1alpha1.ResourcesCreating); statusError != nil {
			return ctrl.Result{}, statusError
		}
		if err = applySecret(r.Client, r.Scheme, ctx, elastalert); err != nil {
			r.emitK8sEvent(elastalert, corev1.EventTypeWarning, event.EventReasonError, "Failed to apply Secret.")
			if statusError := ob.UpdateElastalertStatus(r.Client, ctx, elastalert, esv1alpha1.ActionFailed); statusError != nil {
				return ctrl.Result{}, statusError
			}
			return ctrl.Result{}, err
		}
		r.emitK8sEvent(elastalert, corev1.EventTypeNormal, event.EventReasonCreated, "Apply cert secret successfully.")
		if err = applyConfigMaps(r.Client, r.Scheme, ctx, elastalert); err != nil {
			r.emitK8sEvent(elastalert, corev1.EventTypeWarning, event.EventReasonError, "Failed to apply configmaps")
			if statusError := ob.UpdateElastalertStatus(r.Client, ctx, elastalert, esv1alpha1.ActionFailed); statusError != nil {
				return ctrl.Result{}, statusError
			}
			return ctrl.Result{}, err
		}
		r.emitK8sEvent(elastalert, corev1.EventTypeNormal, event.EventReasonCreated, "Apply configmaps successfully.")
		if _, err = applyDeployment(r.Client, r.Scheme, ctx, elastalert); err != nil {
			r.emitK8sEvent(elastalert, corev1.EventTypeWarning, event.EventReasonError, "Failed to apply deployment.")
			if statusError := ob.UpdateElastalertStatus(r.Client, ctx, elastalert, esv1alpha1.ActionFailed); statusError != nil {
				return ctrl.Result{}, statusError
			}
			return ctrl.Result{}, err
		}
		r.emitK8sEvent(elastalert, corev1.EventTypeNormal, event.EventReasonSuccess, "Apply deployment done, reconcile Elastalert resources successfully.")
	}
	r.startObservingHealth(elastalert)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ElastalertReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&esv1alpha1.Elastalert{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 5}).
		Complete(r)
}

func (r *ElastalertReconciler) startObservingHealth(e *esv1alpha1.Elastalert) {
	r.Observer.Observe(e, r.Client)
}

func (r *ElastalertReconciler) findSuccessCondition(e *esv1alpha1.Elastalert) *metav1.Condition {
	return meta.FindStatusCondition(e.Status.Condictions, esv1alpha1.ElastAlertAvailableType)
}

func (r *ElastalertReconciler) emitK8sEvent(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	r.Recorder.Eventf(object, eventtype, reason, messageFmt, args)
}

func applyConfigMaps(c client.Client, Scheme *runtime.Scheme, ctx context.Context, e *esv1alpha1.Elastalert) error {
	stringCert := e.Spec.Cert
	err := podspec.PatchConfigSettings(e, stringCert)
	if err != nil {
		log.Error(
			err,
			"Failed to patch config.yaml configmaps",
			"Elastalert.Namespace", e.Namespace,
			"Configmaps.Namespace", e.Namespace,
		)
		return err
	}
	err = podspec.PatchAlertSettings(e)
	if err != nil {
		log.Error(
			err,
			"Failed to patch alert for rules configmaps",
			"Elastalert.Namespace", e.Namespace,
			"Configmaps.Namespace", e.Namespace,
		)
		return err
	}
	list := &corev1.ConfigMapList{}
	opts := client.InNamespace(e.Namespace)
	if err = c.List(ctx, list, opts); err != nil {
		return err
	}
	config, err := podspec.GenerateNewConfigmap(Scheme, e, esv1alpha1.ConfigSuffx)
	if err != nil {
		return err
	}
	rule, err := podspec.GenerateNewConfigmap(Scheme, e, esv1alpha1.RuleSuffx)
	if err != nil {
		return err
	}
	mexist := podspec.ConfigMapsToMap(list.Items)
	var mupdate []corev1.ConfigMap
	mupdate = append(mupdate, *rule, *config)
	if len(list.Items) != 0 {
		for _, cm := range mupdate {
			if _, ok := mexist[cm.Name]; ok {
				if err = c.Update(ctx, &cm); err != nil {
					log.Error(
						err,
						"Failed to update configmaps",
						"Elastalert.Namespace", e.Namespace,
						"Configmaps.Namespace", e.Namespace,
					)
					return err
				}
			} else {
				if err = c.Create(ctx, &cm); err != nil {
					log.Error(
						err,
						"Failed to create configmaps",
						"Elastalert.Namespace", e.Namespace,
						"Configmaps.Namespace", e.Namespace,
					)
					return err
				}
			}
		}
		return nil
	} else {
		for _, cm := range mupdate {
			if err = c.Create(ctx, &cm); err != nil {
				log.Error(
					err,
					"Failed to create configmaps",
					"Elastalert.Namespace", e.Namespace,
					"Configmaps.Namespace", e.Namespace,
				)
				return err
			}
		}

	}
	log.V(1).Info(
		"Apply configmaps successfully",
		"Elastalert.Namespace", e.Namespace,
		"Configmaps.Namespace", e.Namespace,
	)
	return nil
}

func applySecret(c client.Client, Scheme *runtime.Scheme, ctx context.Context, e *esv1alpha1.Elastalert) error {
	secret := &corev1.Secret{}
	newSecret, err := podspec.GenerateCertSecret(Scheme, e)
	if err != nil {
		return err
	}
	if err = c.Get(ctx, types.NamespacedName{
		Namespace: e.Namespace,
		Name:      e.Name + podspec.DefaultCertSuffix,
	},
		secret); err != nil {
		if k8serrors.IsNotFound(err) {
			if err = c.Create(ctx, newSecret); err != nil {
				log.Error(
					err,
					"Failed to create Secret",
					"Elastalert.Namespace", e.Namespace,
					"Secret.Name", secret.Name,
				)
				return err
			}
		}
	} else {
		if err = c.Update(ctx, newSecret); err != nil {
			log.Error(
				err,
				"Failed to update Secret",
				"Elastalert.Namespace", e.Namespace,
			)
			return err
		}
	}
	log.V(1).Info(
		"Apply cert secret successfully",
		"Elastalert.Namespace", e.Namespace,
		"Secret.Name", secret.Name,
	)
	return nil
}

func applyDeployment(c client.Client, Scheme *runtime.Scheme, ctx context.Context, e *esv1alpha1.Elastalert) (*appsv1.Deployment, error) {
	deploy := &appsv1.Deployment{}
	err := c.Get(ctx,
		types.NamespacedName{
			Namespace: e.Namespace,
			Name:      e.Name,
		}, deploy)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			deploy, err = podspec.GenerateNewDeployment(Scheme, e)
			if err != nil {
				return nil, err
			}
			err = c.Create(ctx, deploy)
			if err != nil {
				log.Error(
					err,
					"Failed to create Deployment",
					"Elastalert.Name", e.Name,
					"Deployment.Name", e.Name,
				)
				return nil, err
			}
			return deploy, nil
		}
		return nil, err
	} else {
		deploy, err = podspec.GenerateNewDeployment(Scheme, e)
		if err != nil {
			return nil, err
		}
		err = c.Update(ctx, deploy)
		if err != nil {
			log.Error(
				err,
				"Failed to update Deployment",
				"Elastalert.Name", e.Name,
				"Deployment.Name", e.Name,
			)
			return nil, err
		}
		log.V(1).Info(
			"Apply deployment successfully",
			"Elastalert.Name", e.Name,
			"Deployment.Name", e.Name,
		)
		return deploy, nil
	}
}
