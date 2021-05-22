package podspec

import (
	"context"
	"elastalert/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type PodTemplateBuilder struct {
	PodTemplate        corev1.PodTemplateSpec
	containerName      string
	containerDefaulter Defaulter
}

func GenerateNewDeployment(Scheme *runtime.Scheme, e *v1alpha1.Elastalert) (*appsv1.Deployment, error) {
	deploy, err := BuildDeployment(*e)
	if err != nil {
		return nil, err
	}
	if err := ctrl.SetControllerReference(e, deploy, Scheme); err != nil {
		return nil, err
	}
	return deploy, nil
}

func BuildDeployment(elastalert v1alpha1.Elastalert) (*appsv1.Deployment, error) {
	var replicas = new(int32)
	*replicas = 1
	podTemplate, err := BuildPodTemplateSpec(elastalert)
	if err != nil {
		return nil, err
	}
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
	return deploy, nil
}

func WaitForStability(c client.Client, ctx context.Context, dep appsv1.Deployment) error {
	// the images, subsequent runs should take only a few seconds
	seen := false
	return wait.PollImmediate(time.Second, 3*time.Minute, func() (done bool, err error) {
		d := &appsv1.Deployment{}
		if err := c.Get(ctx, types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, d); err != nil {
			if k8serrors.IsNotFound(err) {
				if seen {
					// we have seen this object before, but it doesn't exist anymore!
					// we don't have anything else to do here, break the poll
					//"Deployment has been removed."
					return true, err
				}

				// the object might have not been created yet
				//"Deployment doesn't exist yet."
				return false, nil
			}
			return false, err
		}
		seen = true
		if d.Status.ReadyReplicas != *d.Spec.Replicas {
			//"Deployment has not stabilized yet"
			return false, nil
		}

		//"Deployment has stabilized"
		return true, nil
	})
}
