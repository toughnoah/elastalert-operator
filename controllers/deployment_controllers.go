package controllers

import (
	"context"
	esv1alpha1 "elastalert/api/v1alpha1"
	"elastalert/controllers/podspec"
	"github.com/go-logr/logr"
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
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *DeploymentReconciler) Reconcile(ctx context.Context, req reconcile.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("deployment", req.NamespacedName)
	elastalert := &esv1alpha1.Elastalert{}
	err := r.Get(ctx, req.NamespacedName, elastalert)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.V(1).Info("Elastalert resource not found in this namespace. Ignoring since deployment should not be created.", "Deployment.Namespace", req.Namespace)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get deployment from server")
		return ctrl.Result{}, err
	}
	newDeploy, err := recreateDeployment(r.Client, r.Scheme, ctx, elastalert)
	if err != nil {
		log.Error(err, "Failed to recreate Deployment by steps", "Deployment.Namespace", req.Namespace)
		if err := UpdateElastalertStatus(r.Client, ctx, elastalert, esv1alpha1.ActionFailed); err != nil {
			log.Error(err, "Failed to update elastalert status")
			return ctrl.Result{}, err
		}
	}
	if newDeploy != nil {
		log.V(1).Info("Recreating deployment, stabilizing", "Deployment.Namespace", req.Namespace)
		if err := podspec.WaitForStability(r.Client, ctx, *newDeploy); err != nil {
			log.Error(err, "Failed to stabilized Deployment.", "Deployment.Namespace", req.Namespace)
			if err := UpdateElastalertStatus(r.Client, ctx, elastalert, esv1alpha1.ActionFailed); err != nil {
				log.Error(err, "Failed to update elastalert status")
				return ctrl.Result{}, err
			}
		}

		log.V(1).Info("Deployment reconcile success.", "Deployment.Namespace", req.Namespace)
	}
	return ctrl.Result{}, nil
}

func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 5}).
		//Watches(
		//	&source.Kind{Type: &esv1alpha1.Elastalert{}},
		//	handler.EnqueueRequestsFromMapFunc(r.syncOnElastAlertChanges)).
		Complete(r)
}

//func (r *DeploymenttReconciler) syncOnElastAlertChanges(client client.Object) []reconcile.Request {
//	additional logic
//	reconciliations := []reconcile.Request{}
//	client.GetObjectKind()
//	return reconciliations
//}

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
			return newDeploy, nil
		}
		return nil, err
	}
	// if err if nil, means that event is about about other deployment in same namespace. so just return nil
	return nil, nil
}
