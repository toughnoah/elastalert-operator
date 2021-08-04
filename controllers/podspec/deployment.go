package podspec

import (
	"github.com/toughnoah/elastalert-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

var log = ctrl.Log.WithName("deployment")

type PodTemplateBuilder struct {
	PodTemplate        corev1.PodTemplateSpec
	containerName      string
	containerDefaulter Defaulter
}

func GenerateNewDeployment(Scheme *runtime.Scheme, e *v1alpha1.Elastalert) (*appsv1.Deployment, error) {
	deploy := BuildDeployment(*e)
	if err := ctrl.SetControllerReference(e, deploy, Scheme); err != nil {
		log.Error(err, "Failed to generate Deployment", "Elastalert.Name", e.Name, "Deployment.Name", e.Name)
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
