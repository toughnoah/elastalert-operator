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
	"elastalert/controllers/podspec"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

// ElastalertReconciler reconciles a Elastalert object
type ElastalertReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=es.noah.domain,resources=elastalerts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=es.noah.domain,resources=elastalerts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=es.noah.domain,resources=elastalerts/finalizers,verbs=update
func (r *ElastalertReconciler) Reconcile(ctx context.Context, req reconcile.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("elastalert", req.NamespacedName)
	elastalert := &esv1alpha1.Elastalert{}
	err := r.Get(ctx, req.NamespacedName, elastalert)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("Elastalert deleted", "Elastalert.Namespace/Name", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Elastalert from server")
		return ctrl.Result{}, err
	}
	condition := meta.FindStatusCondition(elastalert.Status.Condictions, esv1alpha1.ElastAlertAvailableType)
	if condition == nil || condition.ObservedGeneration != elastalert.Generation {
		if err := applySecret(r.Client, r.Scheme, ctx, elastalert); err != nil {
			log.Error(err, "Failed to apply Secret", "Secret.Namespace", req.Namespace)
			if err := UpdateElastalertStatus(r.Client, ctx, elastalert, esv1alpha1.ActionFailed); err != nil {
				log.Error(err, "Failed to update elastalert status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}
		log.Info("Apply secret success", "Configmaps.Namespace", req.Namespace)

		if err := applyConfigMaps(r.Client, r.Scheme, ctx, elastalert); err != nil {
			log.Error(err, "Failed to apply configmaps", "Configmaps.Namespace", req.Namespace)
			if err := UpdateElastalertStatus(r.Client, ctx, elastalert, esv1alpha1.ActionFailed); err != nil {
				log.Error(err, "Failed to update elastalert status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}
		log.Info("Apply configmaps success", "Secret.Namespace", req.Namespace)
		deploy, err := applyDeployment(r.Client, r.Scheme, ctx, elastalert)
		if err != nil {
			log.Error(err, "Failed to apply Deployment", "Deployment.Namespace", req.Namespace)
			if err := UpdateElastalertStatus(r.Client, ctx, elastalert, esv1alpha1.ActionFailed); err != nil {
				log.Error(err, "Failed to update elastalert status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}
		if err := podspec.WaitForStability(r.Client, ctx, *deploy); err != nil {
			log.Error(err, "Deployment stabiliz failed ", "Deployment.Namespace", req.Namespace)
			if err := UpdateElastalertStatus(r.Client, ctx, elastalert, esv1alpha1.ActionFailed); err != nil {
				log.Error(err, "Failed to update elastalert status")
				return ctrl.Result{}, err
			}
		}
		log.Info("Deployment has stabilized", "Deployment.Namespace", req.Namespace)
		if err := UpdateElastalertStatus(r.Client, ctx, elastalert, esv1alpha1.ActionSuccess); err != nil {
			log.Error(err, "Failed to update elastalert status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil

	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ElastalertReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&esv1alpha1.Elastalert{}).
		Complete(r)
}

func UpdateElastalertStatus(c client.Client, ctx context.Context, e *esv1alpha1.Elastalert, flag string) error {
	switch flag {
	case esv1alpha1.ActionSuccess:
		e.Status.Phase = esv1alpha1.ElastAlertPhraseSucceeded
		meta.SetStatusCondition(&e.Status.Condictions, metav1.Condition{
			Type:               esv1alpha1.ElastAlertAvailableType,
			Status:             esv1alpha1.ElastAlertAvailableStatus,
			ObservedGeneration: e.Generation,
			LastTransitionTime: metav1.NewTime(time.Now().UTC()),
			Reason:             esv1alpha1.ElastAlertAvailableReason,
			Message:            "ElastAlert " + e.Name + " has successfully progressed.",
		})
		meta.RemoveStatusCondition(&e.Status.Condictions, esv1alpha1.ElastAlertPhraseFailed)
	case esv1alpha1.ActionFailed:
		e.Status.Phase = esv1alpha1.ElastAlertPhraseFailed
		meta.SetStatusCondition(&e.Status.Condictions, metav1.Condition{
			Type:               esv1alpha1.ElastAlertUnAvailableReason,
			Status:             esv1alpha1.ElastAlertUnAvailableStatus,
			ObservedGeneration: e.Generation,
			LastTransitionTime: metav1.NewTime(time.Now().UTC()),
			Reason:             esv1alpha1.ElastAlertUnAvailableReason,
			Message:            "Failed to apply ElastAlert " + e.Name + " resources.",
		})
		meta.RemoveStatusCondition(&e.Status.Condictions, esv1alpha1.ElastAlertPhraseSucceeded)
	}
	e.Status.Version = esv1alpha1.ElastAlertVersion
	if err := c.Status().Update(ctx, e); err != nil {
		return err
	}
	return nil
}

func applyConfigMaps(c client.Client, Scheme *runtime.Scheme, ctx context.Context, e *esv1alpha1.Elastalert) error {
	stringCert := e.Spec.Cert[podspec.DefaultElasticCertName]
	err := podspec.PatchConfigSettings(e, stringCert)
	if err != nil {
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
				err = c.Update(ctx, &cm)
				if err != nil {
					return err
				}
			} else {
				err = c.Create(ctx, &cm)
				if err != nil {
					return err
				}
			}
		}
		return nil
	} else {
		for _, cm := range mupdate {
			err = c.Create(ctx, &cm)
			if err != nil {
				return err
			}
		}

	}
	return nil
}

func applySecret(c client.Client, Scheme *runtime.Scheme, ctx context.Context, e *esv1alpha1.Elastalert) error {
	secret := &corev1.Secret{}
	newsecret := podspec.GenerateCertSecret(e)
	if err := c.Get(ctx, types.NamespacedName{
		Namespace: e.Namespace,
		Name:      podspec.DefaultCertName,
	},
		secret); err != nil {
		if k8serrors.IsNotFound(err) {
			if err = c.Create(ctx, newsecret); err != nil {
				return err
			}
		}
	} else {
		if err = c.Update(ctx, newsecret); err != nil {
			return err
		}
	}
	_ = ctrl.SetControllerReference(e, newsecret, Scheme)
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
			return nil, err
		}
		return deploy, nil
	}
}
