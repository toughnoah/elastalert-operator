package podspec

import (
	"elastalert/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

type PodTemplateBuilder struct {
	PodTemplate        corev1.PodTemplateSpec
	containerName      string
	containerDefaulter Defaulter
}

func BuildDeployment(elastalert v1alpha1.Elastalert) (*appsv1.Deployment, error) {
	var replicas = new(int32)
	*replicas = 1
	podTemplate, err := BuildPodTemplateSpec(elastalert)
	if err != nil {
		return nil, err
	}
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

func buildLabels() map[string]string {
	return map[string]string{"app": "elastalert"}
}

func BuildPodTemplateSpec(elastalert v1alpha1.Elastalert) (corev1.PodTemplateSpec, error) {
	var DefaultAnnotations = map[string]string{
		"kubectl.kubernetes.io/restartedAt": getUtcTime(),
		"sidecar.istio.io/inject":           "false",
	}
	//var DefaultCommand = []string{"sh", "-c", "tail -f /dev/null"}
	var DefaultCommand = []string{"elastalert", "--config", "/etc/elastalert/config.yaml", "--verbose"}
	volumes, volumeMounts := buildVolumes(elastalert.Name)
	labelselector := buildLabels()
	builder := NewPodTemplateBuilder(elastalert.Spec.PodTemplateSpec, DefaultElastAlertName)
	builder = builder.
		WithLabels(labelselector).
		WithAnnotations(DefaultAnnotations).
		WithDockerImage(elastalert.Spec.Image, DefautlImage).
		WithResources(DefaultResources).
		WithTerminationGracePeriod(DefaultTerminationGracePeriodSeconds).
		WithPorts(getDefaultContainerPorts()).
		WithAffinity(DefaultAffinity(elastalert.Name)).
		WithEnv().
		WithCommand(DefaultCommand).
		WithInitContainers().
		WithVolumes(volumes...).
		WithVolumeMounts(volumeMounts...).
		WithInitContainerDefaults().
		WithPreStopHook(*NewPreStopHook())
	return builder.PodTemplate, nil
}

func buildVolumes(cmName string) ([]corev1.Volume, []corev1.VolumeMount) {
	var elastAlertVolumes []corev1.Volume
	var elastAlertVolumesMounts []corev1.VolumeMount

	var volumesTypeMap = map[string]string{
		v1alpha1.RuleSuffx:   v1alpha1.RuleMountPath,
		v1alpha1.ConfigSuffx: v1alpha1.ConfigMountPath,
	}
	for typeSuffix, Path := range volumesTypeMap {
		elastalertVolume := corev1.Volume{
			Name: cmName + typeSuffix,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cmName + typeSuffix,
					},
				},
			},
		}
		elastalertVolumeMount := corev1.VolumeMount{
			Name:      cmName + typeSuffix,
			MountPath: Path,
		}
		elastAlertVolumes = append(elastAlertVolumes, elastalertVolume)
		elastAlertVolumesMounts = append(elastAlertVolumesMounts, elastalertVolumeMount)
	}

	certVolume := &corev1.Volume{
		Name: DefaultCertName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: DefaultCertName,
			},
		},
	}
	certVolumeMount := &corev1.VolumeMount{
		Name:      DefaultCertName,
		MountPath: DefaultCertMountPath,
	}

	elastAlertVolumes = append(elastAlertVolumes, *certVolume)
	elastAlertVolumesMounts = append(elastAlertVolumesMounts, *certVolumeMount)
	return elastAlertVolumes, elastAlertVolumesMounts
}

// PodTemplateBuilder helps with building a pod template inheriting values
// from a user-provided pod template. It focuses on building a pod with
// one main Container.

func NewPodTemplateBuilder(base corev1.PodTemplateSpec, containerName string) *PodTemplateBuilder {
	builder := &PodTemplateBuilder{
		PodTemplate:   *base.DeepCopy(),
		containerName: containerName,
	}
	return builder.setDefaults()
}

func (b *PodTemplateBuilder) setDefaults() *PodTemplateBuilder {
	// retrieve the existing Container from the pod template
	getContainer := func() *corev1.Container {
		for i, c := range b.PodTemplate.Spec.Containers {
			if c.Name == b.containerName {
				return &b.PodTemplate.Spec.Containers[i]
			}
		}
		return nil
	}
	userContainer := getContainer()
	if userContainer == nil {
		// create the default Container if not provided by the user
		b.PodTemplate.Spec.Containers = append(b.PodTemplate.Spec.Containers, corev1.Container{Name: b.containerName})
		b.containerDefaulter = NewDefaulter(getContainer())
	} else {
		b.containerDefaulter = NewDefaulter(userContainer)
	}

	// disable service account token auto mount, unless explicitly enabled by the user
	//varFalse := false
	//if b.PodTemplate.Spec.AutomountServiceAccountToken == nil {
	//	b.PodTemplate.Spec.AutomountServiceAccountToken = &varFalse
	//}
	return b
}

func getDefaultContainerPorts() []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{Name: "http", ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
	}
}

func getUtcTime() string {
	return time.Now().UTC().Format("\"2006-01-02T15:04:05+08:00\"")
}
