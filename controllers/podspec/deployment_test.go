package podspec

import (
	"elastalert/api/v1alpha1"
	mock_podspec "elastalert/controllers/podspec/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"testing"
)

var TerminationGracePeriodSeconds int64 = 30
var Replicas int32 = 1

func TestBuildPodTemplateSpec(t *testing.T) {
	testCases := []struct {
		name       string
		elastalert v1alpha1.Elastalert
		want       v1.PodTemplateSpec
	}{
		{
			name: "test default elastalert resources",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-elastalert",
				},
				Spec: v1alpha1.ElastalertSpec{
					PodTemplateSpec: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "elastalert",
								},
							},
						},
					},
				},
			},
			want: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "elastalert",
					},
					Annotations: map[string]string{
						"sidecar.istio.io/inject":           "false",
						"kubectl.kubernetes.io/restartedAt": "2021-05-17T01:38:44+08:00",
					},
				},

				Spec: v1.PodSpec{
					AutomountServiceAccountToken:  &varFalse,
					TerminationGracePeriodSeconds: &TerminationGracePeriodSeconds,
					Containers: []v1.Container{
						{
							Name:  "elastalert",
							Image: "toughnoah/elastalert:v1.0",
							VolumeMounts: []v1.VolumeMount{
								// have to keep sequence
								{
									Name:      "elasticsearch-cert",
									MountPath: "/ssl",
								},
								{
									Name:      "test-elastalert-config",
									MountPath: "/etc/elastalert",
								},
								{
									Name:      "test-elastalert-rule",
									MountPath: "/etc/elastalert/rules",
								},
							},
							Command: []string{"elastalert", "--config", "/etc/elastalert/config.yaml", "--verbose"},
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									v1.ResourceMemory: DefaultMemoryLimits,
								},
								Limits: map[v1.ResourceName]resource.Quantity{
									v1.ResourceMemory: DefaultMemoryLimits,
								},
							},
							Ports: []v1.ContainerPort{
								{Name: "http", ContainerPort: 8080, Protocol: v1.ProtocolTCP},
							},
						},
					},
					Volumes: []v1.Volume{
						// have to keep sequence
						{
							Name: "elasticsearch-cert",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "elasticsearch-cert",
								},
							},
						},
						{
							Name: "test-elastalert-config",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "test-elastalert-config",
									},
								},
							},
						},
						{
							Name: "test-elastalert-rule",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "test-elastalert-rule",
									},
								},
							},
						},
					},
					Affinity: &v1.Affinity{
						PodAntiAffinity: &v1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{},
						},
					},
				},
			},
		},
		{
			name: "test change elastalert Image, labels and annotations,",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-elastalert",
				},
				Spec: v1alpha1.ElastalertSpec{
					Image: "toughnoah/elastalert-test-image:v1.0",
					PodTemplateSpec: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"test": "elastalert",
							},
							Annotations: map[string]string{
								"test": "elastalert",
							},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "elastalert",
								},
							},
						},
					},
				},
			},
			want: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":  "elastalert",
						"test": "elastalert",
					},
					Annotations: map[string]string{
						"sidecar.istio.io/inject":           "false",
						"kubectl.kubernetes.io/restartedAt": "2021-05-17T01:38:44+08:00",
						"test":                              "elastalert",
					},
				},

				Spec: v1.PodSpec{
					AutomountServiceAccountToken:  &varFalse,
					TerminationGracePeriodSeconds: &TerminationGracePeriodSeconds,
					Containers: []v1.Container{
						{
							Name:  "elastalert",
							Image: "toughnoah/elastalert-test-image:v1.0",
							VolumeMounts: []v1.VolumeMount{
								// have to keep sequence
								{
									Name:      "elasticsearch-cert",
									MountPath: "/ssl",
								},
								{
									Name:      "test-elastalert-config",
									MountPath: "/etc/elastalert",
								},
								{
									Name:      "test-elastalert-rule",
									MountPath: "/etc/elastalert/rules",
								},
							},
							Command: []string{"elastalert", "--config", "/etc/elastalert/config.yaml", "--verbose"},
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									v1.ResourceMemory: DefaultMemoryLimits,
								},
								Limits: map[v1.ResourceName]resource.Quantity{
									v1.ResourceMemory: DefaultMemoryLimits,
								},
							},
							Ports: []v1.ContainerPort{
								{Name: "http", ContainerPort: 8080, Protocol: v1.ProtocolTCP},
							},
						},
					},
					Volumes: []v1.Volume{
						// have to keep sequence
						{
							Name: "elasticsearch-cert",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "elasticsearch-cert",
								},
							},
						},
						{
							Name: "test-elastalert-config",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "test-elastalert-config",
									},
								},
							},
						},
						{
							Name: "test-elastalert-rule",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "test-elastalert-rule",
									},
								},
							},
						},
					},
					Affinity: &v1.Affinity{
						PodAntiAffinity: &v1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{},
						},
					},
				},
			},
		}, {
			name: "test change resources",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-elastalert",
				},
				Spec: v1alpha1.ElastalertSpec{
					PodTemplateSpec: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "elastalert",
									Resources: v1.ResourceRequirements{
										Requests: map[v1.ResourceName]resource.Quantity{
											v1.ResourceMemory: resource.MustParse("4Gi"),
											v1.ResourceCPU:    resource.MustParse("2"),
										},
										Limits: map[v1.ResourceName]resource.Quantity{
											v1.ResourceMemory: resource.MustParse("1Gi"),
											v1.ResourceCPU:    resource.MustParse("1"),
										},
									},
								},
							},
						},
					},
				},
			},
			want: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "elastalert",
					},
					Annotations: map[string]string{
						"sidecar.istio.io/inject":           "false",
						"kubectl.kubernetes.io/restartedAt": "2021-05-17T01:38:44+08:00",
					},
				},

				Spec: v1.PodSpec{
					AutomountServiceAccountToken:  &varFalse,
					TerminationGracePeriodSeconds: &TerminationGracePeriodSeconds,
					Containers: []v1.Container{
						{
							Name:  "elastalert",
							Image: "toughnoah/elastalert:v1.0",
							VolumeMounts: []v1.VolumeMount{
								// have to keep sequence
								{
									Name:      "elasticsearch-cert",
									MountPath: "/ssl",
								},
								{
									Name:      "test-elastalert-config",
									MountPath: "/etc/elastalert",
								},
								{
									Name:      "test-elastalert-rule",
									MountPath: "/etc/elastalert/rules",
								},
							},
							Command: []string{"elastalert", "--config", "/etc/elastalert/config.yaml", "--verbose"},
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									v1.ResourceMemory: resource.MustParse("4Gi"),
									v1.ResourceCPU:    resource.MustParse("2"),
								},
								Limits: map[v1.ResourceName]resource.Quantity{
									v1.ResourceMemory: resource.MustParse("1Gi"),
									v1.ResourceCPU:    resource.MustParse("1"),
								},
							},
							Ports: []v1.ContainerPort{
								{Name: "http", ContainerPort: 8080, Protocol: v1.ProtocolTCP},
							},
						},
					},
					Volumes: []v1.Volume{
						// have to keep sequence
						{
							Name: "elasticsearch-cert",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "elasticsearch-cert",
								},
							},
						},
						{
							Name: "test-elastalert-config",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "test-elastalert-config",
									},
								},
							},
						},
						{
							Name: "test-elastalert-rule",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "test-elastalert-rule",
									},
								},
							},
						},
					},
					Affinity: &v1.Affinity{
						PodAntiAffinity: &v1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{},
						},
					},
				},
			},
		},
		{
			name: "test another container",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-elastalert",
				},
				Spec: v1alpha1.ElastalertSpec{
					PodTemplateSpec: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							InitContainers: []v1.Container{
								{
									Name:  "test-init-container",
									Image: "test/init-elastalert:latest",
								},
							},
							Containers: []v1.Container{
								{
									Name:  "test-another-container",
									Image: "test/elastalert:latest",
								},
							},
						},
					},
				},
			},
			want: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "elastalert",
					},
					Annotations: map[string]string{
						"sidecar.istio.io/inject":           "false",
						"kubectl.kubernetes.io/restartedAt": "2021-05-17T01:38:44+08:00",
					},
				},
				Spec: v1.PodSpec{
					AutomountServiceAccountToken:  &varFalse,
					TerminationGracePeriodSeconds: &TerminationGracePeriodSeconds,
					InitContainers: []v1.Container{
						{
							Name:  "test-init-container",
							Image: "test/init-elastalert:latest",
							VolumeMounts: []v1.VolumeMount{
								// have to keep sequence
								{
									Name:      "elasticsearch-cert",
									MountPath: "/ssl",
								},
								{
									Name:      "test-elastalert-config",
									MountPath: "/etc/elastalert",
								},
								{
									Name:      "test-elastalert-rule",
									MountPath: "/etc/elastalert/rules",
								},
							},
						},
					},
					Containers: []v1.Container{
						{
							Name:  "test-another-container",
							Image: "test/elastalert:latest",
						},
						{
							Name:  "elastalert",
							Image: "toughnoah/elastalert:v1.0",
							VolumeMounts: []v1.VolumeMount{
								// have to keep sequence
								{
									Name:      "elasticsearch-cert",
									MountPath: "/ssl",
								},
								{
									Name:      "test-elastalert-config",
									MountPath: "/etc/elastalert",
								},
								{
									Name:      "test-elastalert-rule",
									MountPath: "/etc/elastalert/rules",
								},
							},
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									v1.ResourceMemory: DefaultMemoryLimits,
								},
								Limits: map[v1.ResourceName]resource.Quantity{
									v1.ResourceMemory: DefaultMemoryLimits,
								},
							},
							Command: []string{"elastalert", "--config", "/etc/elastalert/config.yaml", "--verbose"},
							Ports: []v1.ContainerPort{
								{Name: "http", ContainerPort: 8080, Protocol: v1.ProtocolTCP},
							},
						},
					},
					Volumes: []v1.Volume{
						// have to keep sequence
						{
							Name: "elasticsearch-cert",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "elasticsearch-cert",
								},
							},
						},
						{
							Name: "test-elastalert-config",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "test-elastalert-config",
									},
								},
							},
						},
						{
							Name: "test-elastalert-rule",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "test-elastalert-rule",
									},
								},
							},
						},
					},
					Affinity: &v1.Affinity{
						PodAntiAffinity: &v1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{},
						},
					},
				},
			},
		},
		{
			name: "test change command",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-elastalert",
				},
				Spec: v1alpha1.ElastalertSpec{
					PodTemplateSpec: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:    "elastalert",
									Command: []string{"elastalert", "--debug"},
								},
							},
						},
					},
				},
			},
			want: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "elastalert",
					},
					Annotations: map[string]string{
						"sidecar.istio.io/inject":           "false",
						"kubectl.kubernetes.io/restartedAt": "2021-05-17T01:38:44+08:00",
					},
				},

				Spec: v1.PodSpec{
					AutomountServiceAccountToken:  &varFalse,
					TerminationGracePeriodSeconds: &TerminationGracePeriodSeconds,
					Containers: []v1.Container{
						{
							Name:  "elastalert",
							Image: "toughnoah/elastalert:v1.0",
							VolumeMounts: []v1.VolumeMount{
								// have to keep sequence
								{
									Name:      "elasticsearch-cert",
									MountPath: "/ssl",
								},
								{
									Name:      "test-elastalert-config",
									MountPath: "/etc/elastalert",
								},
								{
									Name:      "test-elastalert-rule",
									MountPath: "/etc/elastalert/rules",
								},
							},
							Command: []string{"elastalert", "--debug"},
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									v1.ResourceMemory: DefaultMemoryLimits,
								},
								Limits: map[v1.ResourceName]resource.Quantity{
									v1.ResourceMemory: DefaultMemoryLimits,
								},
							},
							Ports: []v1.ContainerPort{
								{Name: "http", ContainerPort: 8080, Protocol: v1.ProtocolTCP},
							},
						},
					},
					Volumes: []v1.Volume{
						// have to keep sequence
						{
							Name: "elasticsearch-cert",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "elasticsearch-cert",
								},
							},
						},
						{
							Name: "test-elastalert-config",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "test-elastalert-config",
									},
								},
							},
						},
						{
							Name: "test-elastalert-rule",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "test-elastalert-rule",
									},
								},
							},
						},
					},
					Affinity: &v1.Affinity{
						PodAntiAffinity: &v1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			mock_t := mock_podspec.NewMockUtil(ctl)
			mock_t.EXPECT().GetUtcTimeString().Return("2021-05-17T01:38:44+08:00")
			have, err := BuildPodTemplateSpec(tc.elastalert, mock_t)
			require.NoError(t, err)
			require.Equal(t, tc.want, have)
		})
	}

}

func TestBuildDeployment(t *testing.T) {
	testCases := []struct {
		name       string
		elastalert v1alpha1.Elastalert
		want       appsv1.Deployment
	}{
		{
			name: "test build elastalert deployment",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-elastalert",
				},
				Spec: v1alpha1.ElastalertSpec{
					PodTemplateSpec: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "elastalert",
								},
							},
						},
					},
				},
			},
			want: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-elastalert",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &Replicas,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "elastalert"},
					},
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": "elastalert",
							},
							Annotations: map[string]string{
								"sidecar.istio.io/inject":           "false",
								"kubectl.kubernetes.io/restartedAt": "2021-05-17T01:38:44+08:00",
							},
						},

						Spec: v1.PodSpec{
							AutomountServiceAccountToken:  &varTrue,
							TerminationGracePeriodSeconds: &TerminationGracePeriodSeconds,
							Containers: []v1.Container{
								{
									Name:  "elastalert",
									Image: "toughnoah/elastalert:v1.0",
									VolumeMounts: []v1.VolumeMount{
										// have to keep sequence
										{
											Name:      "elasticsearch-cert",
											MountPath: "/ssl",
										},
										{
											Name:      "test-elastalert-config",
											MountPath: "/etc/elastalert",
										},
										{
											Name:      "test-elastalert-rule",
											MountPath: "/etc/elastalert/rules",
										},
									},
									Command: []string{"elastalert", "--config", "/etc/elastalert/config.yaml", "--verbose"},
									Resources: v1.ResourceRequirements{
										Requests: map[v1.ResourceName]resource.Quantity{
											v1.ResourceMemory: DefaultMemoryLimits,
										},
										Limits: map[v1.ResourceName]resource.Quantity{
											v1.ResourceMemory: DefaultMemoryLimits,
										},
									},
									Ports: []v1.ContainerPort{
										{Name: "http", ContainerPort: 8080, Protocol: v1.ProtocolTCP},
									},
								},
							},
							Volumes: []v1.Volume{
								// have to keep sequence
								{
									Name: "elasticsearch-cert",
									VolumeSource: v1.VolumeSource{
										Secret: &v1.SecretVolumeSource{
											SecretName: "elasticsearch-cert",
										},
									},
								},
								{
									Name: "test-elastalert-config",
									VolumeSource: v1.VolumeSource{
										ConfigMap: &v1.ConfigMapVolumeSource{
											LocalObjectReference: v1.LocalObjectReference{
												Name: "test-elastalert-config",
											},
										},
									},
								},
								{
									Name: "test-elastalert-rule",
									VolumeSource: v1.VolumeSource{
										ConfigMap: &v1.ConfigMapVolumeSource{
											LocalObjectReference: v1.LocalObjectReference{
												Name: "test-elastalert-rule",
											},
										},
									},
								},
							},
							Affinity: &v1.Affinity{
								PodAntiAffinity: &v1.PodAntiAffinity{
									PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			mock_t := mock_podspec.NewMockUtil(ctl)
			mock_t.EXPECT().GetUtcTimeString().Return("2021-05-17T01:38:44+08:00")
			have, err := BuildDeployment(tc.elastalert, mock_t)
			require.NoError(t, err)

			have.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = "2021-05-17T01:38:44+08:00"
			require.Equal(t, tc.want, *have)
		})
	}
}

func TestGenerateNewDeployment(t *testing.T) {
	testCases := []struct {
		name       string
		elastalert v1alpha1.Elastalert
		want       appsv1.Deployment
	}{
		{
			name: "test set owner reference",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-elastalert",
				},
				Spec: v1alpha1.ElastalertSpec{
					PodTemplateSpec: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "elastalert",
								},
							},
						},
					},
				},
			},
			want: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-elastalert",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "v1",
							Kind:               "Elastalert",
							Name:               "test-elastalert",
							UID:                "",
							Controller:         &varTrue,
							BlockOwnerDeletion: &varTrue,
						},
					},
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &Replicas,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "elastalert"},
					},
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": "elastalert",
							},
							Annotations: map[string]string{
								"sidecar.istio.io/inject":           "false",
								"kubectl.kubernetes.io/restartedAt": "2021-05-17T01:38:44+08:00",
							},
						},

						Spec: v1.PodSpec{
							AutomountServiceAccountToken:  &varTrue,
							TerminationGracePeriodSeconds: &TerminationGracePeriodSeconds,
							Containers: []v1.Container{
								{
									Name:  "elastalert",
									Image: "toughnoah/elastalert:v1.0",
									VolumeMounts: []v1.VolumeMount{
										// have to keep sequence
										{
											Name:      "elasticsearch-cert",
											MountPath: "/ssl",
										},
										{
											Name:      "test-elastalert-config",
											MountPath: "/etc/elastalert",
										},
										{
											Name:      "test-elastalert-rule",
											MountPath: "/etc/elastalert/rules",
										},
									},
									Command: []string{"elastalert", "--config", "/etc/elastalert/config.yaml", "--verbose"},
									Resources: v1.ResourceRequirements{
										Requests: map[v1.ResourceName]resource.Quantity{
											v1.ResourceMemory: DefaultMemoryLimits,
										},
										Limits: map[v1.ResourceName]resource.Quantity{
											v1.ResourceMemory: DefaultMemoryLimits,
										},
									},
									Ports: []v1.ContainerPort{
										{Name: "http", ContainerPort: 8080, Protocol: v1.ProtocolTCP},
									},
								},
							},
							Volumes: []v1.Volume{
								// have to keep sequence
								{
									Name: "elasticsearch-cert",
									VolumeSource: v1.VolumeSource{
										Secret: &v1.SecretVolumeSource{
											SecretName: "elasticsearch-cert",
										},
									},
								},
								{
									Name: "test-elastalert-config",
									VolumeSource: v1.VolumeSource{
										ConfigMap: &v1.ConfigMapVolumeSource{
											LocalObjectReference: v1.LocalObjectReference{
												Name: "test-elastalert-config",
											},
										},
									},
								},
								{
									Name: "test-elastalert-rule",
									VolumeSource: v1.VolumeSource{
										ConfigMap: &v1.ConfigMapVolumeSource{
											LocalObjectReference: v1.LocalObjectReference{
												Name: "test-elastalert-rule",
											},
										},
									},
								},
							},
							Affinity: &v1.Affinity{
								PodAntiAffinity: &v1.PodAntiAffinity{
									PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			mock_t := mock_podspec.NewMockUtil(ctl)
			mock_t.EXPECT().GetUtcTimeString().Return("2021-05-17T01:38:44+08:00")
			s := scheme.Scheme
			s.AddKnownTypes(v1.SchemeGroupVersion, &v1alpha1.Elastalert{})
			have, err := GenerateNewDeployment(s, &tc.elastalert, mock_t)
			require.NoError(t, err)
			require.Equal(t, tc.want, *have)
		})
	}
}
