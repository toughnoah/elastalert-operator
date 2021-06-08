package podspec

import (
	"context"
	"elastalert/api/v1alpha1"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = ctrl.Log.WithName("Deployment")

type PodTemplateBuilder struct {
	PodTemplate        corev1.PodTemplateSpec
	containerName      string
	containerDefaulter Defaulter
}

func GenerateNewDeployment(Scheme *runtime.Scheme, e *v1alpha1.Elastalert) (*appsv1.Deployment, error) {
	deploy := BuildDeployment(*e)
	if err := ctrl.SetControllerReference(e, deploy, Scheme); err != nil {
		return nil, err
	}
	return deploy, nil
}

func BuildDeployment(elastalert v1alpha1.Elastalert) *appsv1.Deployment {
	var replicas = new(int32)
	*replicas = 1
	podTemplate := BuildPodTemplateSpec(elastalert)
	varTrue := true
	//deliberate action to enable
	podTemplate.Spec.AutomountServiceAccountToken = &varTrue

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      elastalert.Name,
			Namespace: elastalert.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "elastalert"},
			},
			Template: podTemplate,
		},
	}
	return deploy
}

func WaitForStability(c client.Client, ctx context.Context, dep appsv1.Deployment) error {
	// the images, subsequent runs should take only a few seconds
	seen := false
	res := wait.Poll(v1alpha1.ElastAlertPollInterval, v1alpha1.ElastAlertPollTimeout, func() (done bool, err error) {
		d := &appsv1.Deployment{}
		if err := c.Get(ctx, types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, d); err != nil {
			if k8serrors.IsNotFound(err) {
				if seen {
					// we have seen this object before, but it doesn't exist anymore!
					// we don't have anything else to do here, break the poll
					//"Deployment has been removed."
					log.V(1).Info("Have seen this deployment before, but it doesn't exist anymore!", "deployment", dep.Name)
					return true, err
				}
				// the object might have not been created yet
				//"Deployment doesn't exist yet."
				log.V(1).Info("Deployment doesn't exist yet.", "deployment", dep.Name)
				return false, nil
			}
			return false, err
		}
		seen = true
		if d.Status.AvailableReplicas != *d.Spec.Replicas {
			//"Deployment has not stabilized yet"
			log.V(1).Info(fmt.Sprintf("Deployment has not stabilized yet, expected %d, got %d.", *d.Spec.Replicas, d.Status.AvailableReplicas), "deployment", dep.Name)
			return false, nil
		}
		//"Deployment has stabilized"
		return true, nil
	})
	return res
}
