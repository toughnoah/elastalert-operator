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
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sync"
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
			log.Info("Elastalert deleted", "Elastalert.Namespace", req.Namespace)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Elastalert from server")
		return ctrl.Result{}, err
	}
	condiction := meta.FindStatusCondition(elastalert.Status.Condictions, esv1alpha1.ElastAlertAvailableType)
	if condiction == nil || condiction.ObservedGeneration != elastalert.Generation {
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

		if err := applyDeployment(r.Client, log, r.Scheme, ctx, elastalert); err != nil {
			log.Error(err, "Failed to apply Deployment", "Deployment.Namespace", req.Namespace)
			if err := UpdateElastalertStatus(r.Client, ctx, elastalert, esv1alpha1.ActionFailed); err != nil {
				log.Error(err, "Failed to update elastalert status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}
		log.Info("Apply deployment success", "Deployment.Namespace", req.Namespace)
		if err := UpdateElastalertStatus(r.Client, ctx, elastalert, esv1alpha1.ActionSuccess); err != nil {
			log.Error(err, "Failed to update elastalert status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil

	}
	log.Info("Generation not chaneged, skiping reconcile.", "Elastalert.Namespace", req.NamespacedName)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ElastalertReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&esv1alpha1.Elastalert{}).
		Complete(r)
}

func GenerateNewDeployment(Scheme *runtime.Scheme, e *esv1alpha1.Elastalert) (*appsv1.Deployment, error) {
	deploy, err := podspec.BuildDeployment(*e)
	if err != nil {
		return nil, err
	}
	err = ctrl.SetControllerReference(e, deploy, Scheme)
	if err != nil {
		return nil, err
	}
	return deploy, nil
}

func GenerateNewConfigmap(Scheme *runtime.Scheme, e *esv1alpha1.Elastalert, suffix string) (*corev1.ConfigMap, error) {
	var data map[string]string
	switch suffix {
	case esv1alpha1.RuleSuffx:
		data = e.Spec.Rule
	case esv1alpha1.ConfigSuffx:
		data = e.Spec.ConfigSetting
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Name + suffix,
			Namespace: e.Namespace,
		},
		Data: data,
	}
	err := ctrl.SetControllerReference(e, cm, Scheme)
	if err != nil {
		return nil, err
	}
	return cm, nil
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

func configsMap(deps []corev1.ConfigMap) map[string]corev1.ConfigMap {
	m := map[string]corev1.ConfigMap{}
	for _, d := range deps {
		m[d.Name] = d
	}
	return m
}

func applyConfigMaps(c client.Client, Scheme *runtime.Scheme, ctx context.Context, e *esv1alpha1.Elastalert) error {
	stringCert := e.Spec.Cert[podspec.DefaultElasticCertName]
	err := patchConfigSettings(e, stringCert)
	if err != nil {
		return err
	}
	list := &corev1.ConfigMapList{}
	opts := client.InNamespace(e.Namespace)
	if err = c.List(ctx, list, opts); err != nil {
		return err
	}
	config, err := GenerateNewConfigmap(Scheme, e, esv1alpha1.ConfigSuffx)
	if err != nil {
		return err
	}
	rule, err := GenerateNewConfigmap(Scheme, e, esv1alpha1.RuleSuffx)
	if err != nil {
		return err
	}
	mexist := configsMap(list.Items)
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
func patchConfigSettings(e *esv1alpha1.Elastalert, stringCert string) error {
	var config = map[string]interface{}{}
	var bytesConfig []byte
	var err error
	if err = yaml.Unmarshal([]byte(e.Spec.ConfigSetting["config.yaml"]), &config); err != nil {
		return err
	}
	config["rules_folder"] = podspec.DefaultRulesFolder
	if config["use_ssl"] != nil && config["use_ssl"].(bool) == true && stringCert == "" {
		config["verify_certs"] = false
	}
	if config["use_ssl"] != nil && config["use_ssl"].(bool) == true && stringCert != "" {
		config["verify_certs"] = true
		config["ca_certs"] = podspec.DefaultElasticCertPath
	}
	if config["verify_certs"].(bool) == false && stringCert != "" {
		delete(config, "ca_certs")
	}
	if config["use_ssl"] == nil {
		delete(config, "verify_certs")
		delete(config, "ca_certs")
	}
	if bytesConfig, err = yaml.Marshal(config); err != nil {
		return err
	}
	e.Spec.ConfigSetting["config.yaml"] = string(bytesConfig)
	return nil
}

func GenerateCertSecret(e *esv1alpha1.Elastalert) *corev1.Secret {
	var data = map[string][]byte{}
	stringCert := e.Spec.Cert[podspec.DefaultElasticCertName]
	data[podspec.DefaultElasticCertName] = []byte(stringCert)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podspec.DefaultCertName,
			Namespace: e.Namespace,
		},
		Data: data,
	}

	return secret

}

func applySecret(c client.Client, Scheme *runtime.Scheme, ctx context.Context, e *esv1alpha1.Elastalert) error {
	secret := &corev1.Secret{}
	newsecret := GenerateCertSecret(e)
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

func applyDeployment(c client.Client, log logr.Logger, Scheme *runtime.Scheme, ctx context.Context, e *esv1alpha1.Elastalert) error {
	deploy := &appsv1.Deployment{}
	err := c.Get(ctx,
		types.NamespacedName{
			Namespace: e.Namespace,
			Name:      e.Name,
		}, deploy)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			deploy, err = GenerateNewDeployment(Scheme, e)
			if err != nil {
				return err
			}
			err = c.Create(ctx, deploy)
			if err != nil {
				return err
			}
			if err = waitForStability(c, log, ctx, *deploy); err != nil {
				return err
			}
			return nil
		}
		return err
	} else {
		deploy, err = GenerateNewDeployment(Scheme, e)
		if err != nil {
			return err
		}
		err = c.Update(ctx, deploy)
		if err != nil {
			return err
		}
		if err = waitForStability(c, log, ctx, *deploy); err != nil {
			return err
		}
		return nil
	}
}

func waitForStability(c client.Client, log logr.Logger, ctx context.Context, dep appsv1.Deployment) error {

	// TODO: decide what's a good timeout... the first cold run might take a while to download
	// the images, subsequent runs should take only a few seconds
	seen := false
	once := &sync.Once{}
	return wait.PollImmediate(time.Second, 5*time.Minute, func() (done bool, err error) {
		d := &appsv1.Deployment{}
		if err := c.Get(ctx, types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, d); err != nil {
			if k8serrors.IsNotFound(err) {
				if seen {
					// we have seen this object before, but it doesn't exist anymore!
					// we don't have anything else to do here, break the poll
					log.Info("Deployment has been removed.")
					return true, err
				}

				// the object might have not been created yet
				log.Info("Deployment doesn't exist yet.")
				return false, nil
			}
			return false, err
		}

		seen = true
		if d.Status.ReadyReplicas != d.Status.Replicas {
			once.Do(func() {
				log.Info("Waiting for deployment to stabilize")
			})
			return false, nil
		}

		log.Info("Deployment has stabilized")
		return true, nil
	})
}
