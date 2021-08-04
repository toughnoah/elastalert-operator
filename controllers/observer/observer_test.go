package observer

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/toughnoah/elastalert-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

var (
	varTrue                             = true
	Replicas                      int32 = 1
	TerminationGracePeriodSeconds int64 = 10
)

func TestObserver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Observer Suite")
}

var _ = Describe("Test Observer", func() {
	s := scheme.Scheme
	s.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
	recoder := record.NewBroadcaster().NewRecorder(s, corev1.EventSource{})
	ea := types.NamespacedName{
		Name:      "elastalert",
		Namespace: "ns",
	}
	testCases := []struct {
		client  client.Client
		eaPhase string
	}{
		{
			client: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "elastalert",
						Namespace: "ns",
					},
					Status: v1alpha1.ElastalertStatus{Phase: v1alpha1.ElastAlertPhraseSucceeded},
				},
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						ResourceVersion: "1",
						Namespace:       "ns",
						Name:            "elastalert",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "v1",
								Kind:               "Elastalert",
								Name:               "elastalert",
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
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app": "elastalert",
								},
								Annotations: map[string]string{
									"kubectl.kubernetes.io/restartedAt": "2021-05-17T01:38:44+08:00",
								},
							},

							Spec: corev1.PodSpec{
								AutomountServiceAccountToken:  &varTrue,
								TerminationGracePeriodSeconds: &TerminationGracePeriodSeconds,
								Containers: []corev1.Container{
									{
										Name:  "elastalert",
										Image: "toughnoah/elastalert:v1.0",
										VolumeMounts: []corev1.VolumeMount{
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
										Resources: corev1.ResourceRequirements{
											Requests: map[corev1.ResourceName]resource.Quantity{
												corev1.ResourceMemory: resource.MustParse("2Gi"),
											},
											Limits: map[corev1.ResourceName]resource.Quantity{
												corev1.ResourceMemory: resource.MustParse("2Gi"),
											},
										},
										Ports: []corev1.ContainerPort{
											{Name: "http", ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
										},
										ReadinessProbe: &corev1.Probe{
											Handler: corev1.Handler{
												Exec: &corev1.ExecAction{
													Command: []string{
														"cat",
														"/etc/elastalert/config.yaml",
													},
												},
											},
											InitialDelaySeconds: 20,
											TimeoutSeconds:      3,
											PeriodSeconds:       2,
											SuccessThreshold:    5,
											FailureThreshold:    3,
										},
										LivenessProbe: &corev1.Probe{
											Handler: corev1.Handler{
												Exec: &corev1.ExecAction{
													Command: []string{
														"sh",
														"-c",
														"ps -ef|grep -v grep|grep elastalert",
													},
												},
											},
											InitialDelaySeconds: 50,
											TimeoutSeconds:      3,
											PeriodSeconds:       2,
											SuccessThreshold:    1,
											FailureThreshold:    3,
										},
									},
								},
								Volumes: []corev1.Volume{
									// have to keep sequence
									{
										Name: "elasticsearch-cert",
										VolumeSource: corev1.VolumeSource{
											Secret: &corev1.SecretVolumeSource{
												SecretName: "test-elastalert-es-cert",
											},
										},
									},
									{
										Name: "test-elastalert-config",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-elastalert-config",
												},
											},
										},
									},
									{
										Name: "test-elastalert-rule",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-elastalert-rule",
												},
											},
										},
									},
								},
								Affinity: &corev1.Affinity{
									PodAntiAffinity: &corev1.PodAntiAffinity{},
								},
							},
						},
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: int32(0),
					},
				},
			).Build(),
			eaPhase: v1alpha1.ElastAlertPhraseFailed,
		},
		{
			client: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "elastalert",
						Namespace: "ns",
					},
					Status: v1alpha1.ElastalertStatus{Phase: v1alpha1.ElastAlertPhraseSucceeded},
				},
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						ResourceVersion: "1",
						Namespace:       "ns",
						Name:            "elastalert",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "v1",
								Kind:               "Elastalert",
								Name:               "elastalert",
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
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app": "elastalert",
								},
								Annotations: map[string]string{
									"kubectl.kubernetes.io/restartedAt": "2021-05-17T01:38:44+08:00",
								},
							},

							Spec: corev1.PodSpec{
								AutomountServiceAccountToken:  &varTrue,
								TerminationGracePeriodSeconds: &TerminationGracePeriodSeconds,
								Containers: []corev1.Container{
									{
										Name:  "elastalert",
										Image: "toughnoah/elastalert:v1.0",
										VolumeMounts: []corev1.VolumeMount{
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
										Resources: corev1.ResourceRequirements{
											Requests: map[corev1.ResourceName]resource.Quantity{
												corev1.ResourceMemory: resource.MustParse("2Gi"),
											},
											Limits: map[corev1.ResourceName]resource.Quantity{
												corev1.ResourceMemory: resource.MustParse("2Gi"),
											},
										},
										Ports: []corev1.ContainerPort{
											{Name: "http", ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
										},
										ReadinessProbe: &corev1.Probe{
											Handler: corev1.Handler{
												Exec: &corev1.ExecAction{
													Command: []string{
														"cat",
														"/etc/elastalert/config.yaml",
													},
												},
											},
											InitialDelaySeconds: 20,
											TimeoutSeconds:      3,
											PeriodSeconds:       2,
											SuccessThreshold:    5,
											FailureThreshold:    3,
										},
										LivenessProbe: &corev1.Probe{
											Handler: corev1.Handler{
												Exec: &corev1.ExecAction{
													Command: []string{
														"sh",
														"-c",
														"ps -ef|grep -v grep|grep elastalert",
													},
												},
											},
											InitialDelaySeconds: 50,
											TimeoutSeconds:      3,
											PeriodSeconds:       2,
											SuccessThreshold:    1,
											FailureThreshold:    3,
										},
									},
								},
								Volumes: []corev1.Volume{
									// have to keep sequence
									{
										Name: "elasticsearch-cert",
										VolumeSource: corev1.VolumeSource{
											Secret: &corev1.SecretVolumeSource{
												SecretName: "test-elastalert-es-cert",
											},
										},
									},
									{
										Name: "test-elastalert-config",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-elastalert-config",
												},
											},
										},
									},
									{
										Name: "test-elastalert-rule",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-elastalert-rule",
												},
											},
										},
									},
								},
								Affinity: &corev1.Affinity{
									PodAntiAffinity: &corev1.PodAntiAffinity{},
								},
							},
						},
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: int32(1),
					},
				},
			).Build(),
			eaPhase: v1alpha1.ElastAlertPhraseSucceeded,
		},
		{
			client: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "elastalert",
						Namespace: "ns",
					},
					Status: v1alpha1.ElastalertStatus{Phase: v1alpha1.ElastAlertPhraseFailed},
				},
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						ResourceVersion: "1",
						Namespace:       "ns",
						Name:            "elastalert",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "v1",
								Kind:               "Elastalert",
								Name:               "elastalert",
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
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app": "elastalert",
								},
								Annotations: map[string]string{
									"kubectl.kubernetes.io/restartedAt": "2021-05-17T01:38:44+08:00",
								},
							},

							Spec: corev1.PodSpec{
								AutomountServiceAccountToken:  &varTrue,
								TerminationGracePeriodSeconds: &TerminationGracePeriodSeconds,
								Containers: []corev1.Container{
									{
										Name:  "elastalert",
										Image: "toughnoah/elastalert:v1.0",
										VolumeMounts: []corev1.VolumeMount{
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
										Resources: corev1.ResourceRequirements{
											Requests: map[corev1.ResourceName]resource.Quantity{
												corev1.ResourceMemory: resource.MustParse("2Gi"),
											},
											Limits: map[corev1.ResourceName]resource.Quantity{
												corev1.ResourceMemory: resource.MustParse("2Gi"),
											},
										},
										Ports: []corev1.ContainerPort{
											{Name: "http", ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
										},
										ReadinessProbe: &corev1.Probe{
											Handler: corev1.Handler{
												Exec: &corev1.ExecAction{
													Command: []string{
														"cat",
														"/etc/elastalert/config.yaml",
													},
												},
											},
											InitialDelaySeconds: 20,
											TimeoutSeconds:      3,
											PeriodSeconds:       2,
											SuccessThreshold:    5,
											FailureThreshold:    3,
										},
										LivenessProbe: &corev1.Probe{
											Handler: corev1.Handler{
												Exec: &corev1.ExecAction{
													Command: []string{
														"sh",
														"-c",
														"ps -ef|grep -v grep|grep elastalert",
													},
												},
											},
											InitialDelaySeconds: 50,
											TimeoutSeconds:      3,
											PeriodSeconds:       2,
											SuccessThreshold:    1,
											FailureThreshold:    3,
										},
									},
								},
								Volumes: []corev1.Volume{
									// have to keep sequence
									{
										Name: "elasticsearch-cert",
										VolumeSource: corev1.VolumeSource{
											Secret: &corev1.SecretVolumeSource{
												SecretName: "test-elastalert-es-cert",
											},
										},
									},
									{
										Name: "test-elastalert-config",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-elastalert-config",
												},
											},
										},
									},
									{
										Name: "test-elastalert-rule",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-elastalert-rule",
												},
											},
										},
									},
								},
								Affinity: &corev1.Affinity{
									PodAntiAffinity: &corev1.PodAntiAffinity{},
								},
							},
						},
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: int32(1),
					},
				},
			).Build(),
			eaPhase: v1alpha1.ElastAlertPhraseSucceeded,
		},
		{
			client: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "elastalert",
						Namespace: "ns",
					},
					Status: v1alpha1.ElastalertStatus{Phase: v1alpha1.ElastAlertPhraseSucceeded},
				},
			).Build(),
			eaPhase: v1alpha1.ElastAlertPhraseFailed,
		},
	}

	Context("test observing", func() {
		It("test checkDeploymentHeath", func() {
			for _, tc := range testCases {

				ob := NewObserver(tc.client, ea, time.Second*2, recoder)
				ob.Start()
				defer ob.Stop()
				Eventually(func() bool {
					elastalert := &v1alpha1.Elastalert{}
					_ = tc.client.Get(context.Background(), ea, elastalert)
					return elastalert.Status.Phase == tc.eaPhase
				}, 2*time.Minute, time.Second).Should(Equal(true))
			}
		})
	})
	Context("test manager", func() {
		It("test manager observes", func() {
			elastalert := &v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "elastalert",
					Namespace: "ns",
				},
				Status: v1alpha1.ElastalertStatus{Phase: v1alpha1.ElastAlertPhraseSucceeded},
			}
			client := fake.NewClientBuilder().Build()
			manager := NewManager()
			manager.Observe(elastalert, client, recoder)
			defer manager.StopObserving(ea)
			Eventually(func() bool {
				_, ok := manager.getObserver(ea)
				return ok
			}, time.Second*10, time.Second).Should(Equal(true))
		})
	})
})
