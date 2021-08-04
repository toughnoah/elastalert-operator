package controllers

import (
	"context"
	esv1alpha1 "github.com/toughnoah/elastalert-operator/api/v1alpha1"
	ob "github.com/toughnoah/elastalert-operator/controllers/observer"
	"github.com/toughnoah/elastalert-operator/controllers/podspec"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type DeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DeploymentReconciler) Reconcile(ctx context.Context, req reconcile.Request) (ctrl.Result, error) {
	elastalert := &esv1alpha1.Elastalert{}
	err := r.Get(ctx, req.NamespacedName, elastalert)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get deployment from server")
		return ctrl.Result{}, err
	}
	if _, err = recreateDeployment(r.Client, r.Scheme, ctx, elastalert); err != nil {
		if statusError := ob.UpdateElastalertStatus(r.Client, ctx, elastalert, esv1alpha1.ActionFailed); statusError != nil {
			return ctrl.Result{}, statusError
		}
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 5}).
		Complete(r)
}

func recreateDeployment(c client.Client, Scheme *runtime.Scheme, ctx context.Context, e *esv1alpha1.Elastalert) (*appsv1.Deployment, error) {
	deploy := &appsv1.Deployment{}
	err := c.Get(ctx,
		types.NamespacedName{
			Namespace: e.Namespace,
			Name:      e.Name,
		},
		deploy)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			newDeploy, err := podspec.GenerateNewDeployment(Scheme, e)
			if err != nil {
				return nil, err
			}
			if err = applySecret(c, Scheme, ctx, e); err != nil {
				return nil, err
			}
			if err = applyConfigMaps(c, Scheme, ctx, e); err != nil {
				return nil, err
			}
			if err = c.Create(ctx, newDeploy); err != nil {
				return nil, err
			}
			log.V(1).Info(
				"Deployment reconcile success.",
				"Elastalert.Name", e.Name,
				"Deployment.Name", e.Name,
			)
			return newDeploy, nil
		}
		log.Error(err, "Failed to get deployment from server", "Elastalert.Name", e.Name, "Deployment.Name", e.Name)
		return nil, err
	}
	// if err is nil, means that event is about about other deployment in same namespace. so just return nil
	return nil, nil
}
