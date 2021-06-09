package controllers

import (
	"context"
	"elastalert/api/v1alpha1"
	ob "elastalert/controllers/observer"
	"elastalert/controllers/podspec"
	"errors"
	"fmt"
	"github.com/bouk/monkey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
	"time"
)

var TerminationGracePeriodSeconds int64 = 10
var Replicas int32 = 1
var varTrue = true

func TestApplyConfigMaps(t *testing.T) {
	testCases := []struct {
		desc       string
		elastalert v1alpha1.Elastalert
		c          client.Client
	}{
		{
			desc: "test create configmap",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Spec: v1alpha1.ElastalertSpec{
					ConfigSetting: v1alpha1.NewFreeForm(map[string]interface{}{
						"config": "test",
					}),
				},
			},
			c: fake.NewClientBuilder().Build(),
		},
		{
			desc: "test update configmap",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Spec: v1alpha1.ElastalertSpec{
					ConfigSetting: v1alpha1.NewFreeForm(map[string]interface{}{
						"config": "test",
					}),
				},
			},
			c: fake.NewClientBuilder().WithLists(&corev1.ConfigMapList{
				Items: []corev1.ConfigMap{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "esa1",
							Name:      "my-esa-config",
						},
						Data: map[string]string{
							"config.yaml": "test: Updatingconfigmaps",
						},
					},
				},
			}).Build(),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			s := scheme.Scheme
			var log = ctrl.Log.WithName("test").WithName("Elastalert")

			r := &ElastalertReconciler{
				Client: tc.c,
				Log:    log,
				Scheme: s,
			}
			cms := corev1.ConfigMapList{}
			r.Scheme.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
			r.Scheme.AddKnownTypes(corev1.SchemeGroupVersion, &cms)
			err := applyConfigMaps(r.Client, r.Scheme, context.Background(), &tc.elastalert)
			assert.NoError(t, err)
			err = r.Client.List(context.Background(), &cms)
			require.NoError(t, err)
			assert.Len(t, cms.Items, 2)
		})
	}
}

func TestApplySecret(t *testing.T) {
	testCases := []struct {
		desc       string
		elastalert v1alpha1.Elastalert
		c          client.Client
	}{
		{
			desc: "test to create secret",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Spec: v1alpha1.ElastalertSpec{
					Cert: "abc",
				},
			},
			c: fake.NewClientBuilder().Build(),
		},
		{
			desc: "test to update secret",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Spec: v1alpha1.ElastalertSpec{
					Cert: "abc",
				},
			},
			c: fake.NewClientBuilder().WithRuntimeObjects(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa" + podspec.DefaultCertSuffix,
				},
				Data: map[string][]byte{
					"elasticCA.crt": []byte("1"),
				},
			}).Build(),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// prepare
			s := scheme.Scheme
			var log = ctrl.Log.WithName("test").WithName("Elastalert")
			r := &ElastalertReconciler{
				Client: tc.c,
				Log:    log,
				Scheme: s,
			}
			se := corev1.Secret{}
			r.Scheme.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
			r.Scheme.AddKnownTypes(corev1.SchemeGroupVersion, &se)
			err := applySecret(r.Client, r.Scheme, context.Background(), &tc.elastalert)
			assert.NoError(t, err)
			err = r.Client.Get(context.Background(), types.NamespacedName{
				Namespace: tc.elastalert.Namespace,
				Name:      "my-esa" + podspec.DefaultCertSuffix,
			}, &se)
			require.NoError(t, err)
			assert.Equal(t, se.Data, map[string][]byte{
				"elasticCA.crt": []byte("abc"),
			})
		})
	}
}

func TestApplyDeployment(t *testing.T) {
	testCases := []struct {
		desc       string
		elastalert v1alpha1.Elastalert
		c          client.Client
	}{
		{
			desc: "test create deployment",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					//Namespace: "esa1",
					Name: "my-esa",
				},
				Spec: v1alpha1.ElastalertSpec{
					Cert: "abc",
				},
			},
			c: fake.NewClientBuilder().Build(),
		},
		{
			desc: "test update deployment",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					//Namespace: "esa1",
					Name: "my-esa",
				},
				Spec: v1alpha1.ElastalertSpec{
					Cert: "abc",
				},
			},
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-esa",
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
												corev1.ResourceMemory: resource.MustParse("4Gi"),
												corev1.ResourceCPU:    resource.MustParse("2"),
											},
											Limits: map[corev1.ResourceName]resource.Quantity{
												corev1.ResourceMemory: resource.MustParse("1Gi"),
												corev1.ResourceCPU:    resource.MustParse("1"),
											},
										},
										Ports: []corev1.ContainerPort{
											{Name: "http", ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
										},
									},
								},
								Volumes: []corev1.Volume{
									// have to keep sequence
									{
										Name: "elasticsearch-cert",
										VolumeSource: corev1.VolumeSource{
											Secret: &corev1.SecretVolumeSource{
												SecretName: "elasticsearch-cert",
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
									PodAntiAffinity: &corev1.PodAntiAffinity{
										PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{},
									},
								},
							},
						},
					},
				},
			).Build(),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			s := scheme.Scheme
			var log = ctrl.Log.WithName("test").WithName("Elastalert")
			r := &ElastalertReconciler{
				Client: tc.c,
				Log:    log,
				Scheme: s,
			}

			monkey.Patch(podspec.GetUtcTimeString, func() string {
				return "2021-05-17T01:38:44+08:00"
			})

			dep := appsv1.Deployment{}
			r.Scheme.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
			r.Scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &dep)
			_, err := applyDeployment(r.Client, r.Scheme, context.Background(), &tc.elastalert)
			assert.NoError(t, err)
			err = r.Client.Get(context.Background(), types.NamespacedName{
				Namespace: tc.elastalert.Namespace,
				Name:      tc.elastalert.Name,
			}, &dep)
			require.NoError(t, err)
		})
	}

}

func TestReconcile(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
	testCases := []struct {
		desc       string
		elastalert v1alpha1.Elastalert
		c          client.Client
		testFunc   func(c client.Client, Scheme *runtime.Scheme, ctx context.Context, e *v1alpha1.Elastalert) error
		result     bool
	}{
		{
			desc: "test elastalert reconcile delete elastalert",
			c:    fake.NewClientBuilder().Build(),
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Spec: v1alpha1.ElastalertSpec{
					Cert: "abc",
				},
			},
			result: true,
		},
		{
			desc: "test elastalert reconcile  with same generation",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  "esa1",
						Name:       "my-esa",
						Generation: int64(1),
					},
					Spec: v1alpha1.ElastalertSpec{
						Cert: "abc",
					},
					Status: v1alpha1.ElastalertStatus{
						Version: "v1.0",
						Phase:   "RUNNING",
						Condictions: []metav1.Condition{
							{
								Type:               "Progressing",
								Status:             "True",
								ObservedGeneration: int64(1),
								LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057012345600)),
								Reason:             "NewElastAlertAvailable",
								Message:            "ElastAlert my-esa has successfully progressed.",
							},
						},
					},
				},
			).Build(),
			result: true,
		},
		{
			desc: "test elastalert reconcile success",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  "esa1",
						Name:       "my-esa",
						Generation: int64(2),
					},
					Spec: v1alpha1.ElastalertSpec{
						Cert: "abc",
						ConfigSetting: v1alpha1.NewFreeForm(map[string]interface{}{
							"config": "test",
						}),
						Rule: []v1alpha1.FreeForm{
							v1alpha1.NewFreeForm(map[string]interface{}{
								"name": "test-elastalert", "type": "any",
							}),
						},
					},
					Status: v1alpha1.ElastalertStatus{
						Version: "v1.0",
						Phase:   "RUNNING",
						Condictions: []metav1.Condition{
							{
								Type:               "Progressing",
								Status:             "True",
								ObservedGeneration: int64(1),
								LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057012345600)),
								Reason:             "NewElastAlertAvailable",
								Message:            "ElastAlert my-esa has successfully progressed.",
							},
						},
					},
				},
			).Build(),
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Spec: v1alpha1.ElastalertSpec{
					Cert: "abdec",
					ConfigSetting: v1alpha1.NewFreeForm(map[string]interface{}{
						"config": "test",
					}),
					Rule: []v1alpha1.FreeForm{
						v1alpha1.NewFreeForm(map[string]interface{}{
							"name": "test-elastalert", "type": "any",
						}),
					},
				},
			},
			result: true,
		},
		{
			desc: "test elastalert reconcile failed",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  "esa1",
						Name:       "my-esa",
						Generation: int64(2),
					},
					Spec: v1alpha1.ElastalertSpec{
						Cert: "abc",
						ConfigSetting: v1alpha1.NewFreeForm(map[string]interface{}{
							"config": "test",
						}),
						Rule: []v1alpha1.FreeForm{
							v1alpha1.NewFreeForm(map[string]interface{}{
								"name": "test-elastalert", "type": "any",
							}),
						},
					},
					Status: v1alpha1.ElastalertStatus{
						Version: "v1.0",
						Phase:   "RUNNING",
						Condictions: []metav1.Condition{
							{
								Type:               "Progressing",
								Status:             "True",
								ObservedGeneration: int64(1),
								LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057012345600)),
								Reason:             "NewElastAlertAvailable",
								Message:            "ElastAlert my-esa has successfully progressed.",
							},
						},
					},
				},
			).Build(),
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Spec: v1alpha1.ElastalertSpec{
					Cert: "abdec",
					ConfigSetting: v1alpha1.NewFreeForm(map[string]interface{}{
						"config": "test",
					}),
					Rule: []v1alpha1.FreeForm{
						v1alpha1.NewFreeForm(map[string]interface{}{
							"name": "test-elastalert", "type": "any",
						}),
					},
				},
			},
			result: true,
		},
		{
			desc: "test elastalert apply secret failed",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  "esa1",
						Name:       "my-esa",
						Generation: int64(2),
					},
					Spec: v1alpha1.ElastalertSpec{
						Cert: "abc",
						ConfigSetting: v1alpha1.NewFreeForm(map[string]interface{}{
							"config": "test",
						}),
						Rule: []v1alpha1.FreeForm{
							v1alpha1.NewFreeForm(map[string]interface{}{
								"name": "test-elastalert", "type": "any",
							}),
						},
					},
				},
			).Build(),
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Spec: v1alpha1.ElastalertSpec{
					Cert: "abdec",
					ConfigSetting: v1alpha1.NewFreeForm(map[string]interface{}{
						"config": "test",
					}),
					Rule: []v1alpha1.FreeForm{
						v1alpha1.NewFreeForm(map[string]interface{}{
							"name": "test-elastalert", "type": "any",
						}),
					},
				},
			},
			testFunc: applySecret,
			result:   false,
		},
		{
			desc: "test elastalert apply configmap failed",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  "esa1",
						Name:       "my-esa",
						Generation: int64(2),
					},
					Spec: v1alpha1.ElastalertSpec{
						Cert: "abc",
						ConfigSetting: v1alpha1.NewFreeForm(map[string]interface{}{
							"config": "test",
						}),
						Rule: []v1alpha1.FreeForm{
							v1alpha1.NewFreeForm(map[string]interface{}{
								"name": "test-elastalert", "type": "any",
							}),
						},
					},
				},
			).Build(),
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Spec: v1alpha1.ElastalertSpec{
					Cert: "abdec",
					ConfigSetting: v1alpha1.NewFreeForm(map[string]interface{}{
						"config": "test",
					}),
					Rule: []v1alpha1.FreeForm{
						v1alpha1.NewFreeForm(map[string]interface{}{
							"name": "test-elastalert", "type": "any",
						}),
					},
				},
			},
			testFunc: applyConfigMaps,
			result:   false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			defer monkey.Unpatch(tc.testFunc)
			defer monkey.Unpatch(ob.UpdateElastalertStatus)
			defer monkey.Unpatch(podspec.WaitForStability)
			log := ctrl.Log.WithName("test").WithName("Elastalert")
			r := &ElastalertReconciler{
				Client:   tc.c,
				Log:      log,
				Scheme:   s,
				Recorder: record.NewBroadcaster().NewRecorder(s, corev1.EventSource{}),
				Observer: *ob.NewManager(),
			}
			nsn := types.NamespacedName{Name: "my-esa", Namespace: "esa1"}
			req := reconcile.Request{NamespacedName: nsn}
			if tc.result {
				monkey.Patch(podspec.WaitForStability, func(c client.Client, ctx context.Context, dep appsv1.Deployment) error {
					return nil
				})
				_, err := r.Reconcile(context.Background(), req)
				assert.NoError(t, err)
			} else {
				monkey.Patch(tc.testFunc, func(c client.Client, Scheme *runtime.Scheme, ctx context.Context, e *v1alpha1.Elastalert) error {
					return errors.New("test apply func failed")
				})
				monkey.Patch(ob.UpdateElastalertStatus, func(c client.Client, ctx context.Context, e *v1alpha1.Elastalert, flag string) error {
					return errors.New("test update failed")
				})
				monkey.Patch(podspec.WaitForStability, func(c client.Client, ctx context.Context, dep appsv1.Deployment) error {
					return errors.New("test WaitForStability failed")
				})
				_, err := r.Reconcile(context.Background(), req)
				assert.Error(t, err)
			}
		})
	}
}

func TestReconcileWaitForStabilityFailed(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
	testCases := []struct {
		desc       string
		elastalert v1alpha1.Elastalert
		c          client.Client
	}{
		{
			desc: "test elastalert reconcile WaitForStability Failed",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  "esa1",
						Name:       "my-esa",
						Generation: int64(2),
					},
					Spec: v1alpha1.ElastalertSpec{
						Cert: "abc",
						ConfigSetting: v1alpha1.NewFreeForm(map[string]interface{}{
							"config": "test",
						}),
						Rule: []v1alpha1.FreeForm{
							v1alpha1.NewFreeForm(map[string]interface{}{
								"name": "test-elastalert", "type": "any",
							}),
						},
					},
					Status: v1alpha1.ElastalertStatus{
						Version: "v1.0",
						Phase:   "RUNNING",
						Condictions: []metav1.Condition{
							{
								Type:               "Progressing",
								Status:             "True",
								ObservedGeneration: int64(1),
								LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057012345600)),
								Reason:             "NewElastAlertAvailable",
								Message:            "ElastAlert my-esa has successfully progressed.",
							},
						},
					},
				},
			).Build(),
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Spec: v1alpha1.ElastalertSpec{
					Cert: "abdec",
					ConfigSetting: v1alpha1.NewFreeForm(map[string]interface{}{
						"config": "test",
					}),
					Rule: []v1alpha1.FreeForm{
						v1alpha1.NewFreeForm(map[string]interface{}{
							"name": "test-elastalert", "type": "any",
						}),
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			defer monkey.Unpatch(wait.Poll)
			defer monkey.Unpatch(ob.UpdateElastalertStatus)
			log := ctrl.Log.WithName("test").WithName("Elastalert")
			r := &ElastalertReconciler{
				Client:   tc.c,
				Log:      log,
				Scheme:   s,
				Recorder: record.NewBroadcaster().NewRecorder(s, corev1.EventSource{}),
				Observer: *ob.NewManager(),
			}
			nsn := types.NamespacedName{Name: "my-esa", Namespace: "esa1"}
			req := reconcile.Request{NamespacedName: nsn}

			monkey.Patch(wait.Poll, func(interval, timeout time.Duration, condition wait.ConditionFunc) error {
				return errors.New("test WaitForStability failed")
			})

			_, err := r.Reconcile(context.Background(), req)
			assert.Error(t, err)
		})
	}
}

func TestReconcileApplyDeploymentFailed(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
	testCases := []struct {
		desc       string
		elastalert v1alpha1.Elastalert
		c          client.Client
		testFunc   func(c client.Client, Scheme *runtime.Scheme, ctx context.Context, e *v1alpha1.Elastalert) (*appsv1.Deployment, error)
		result     bool
	}{

		{
			desc: "test elastalert apply deployment failed",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  "esa1",
						Name:       "my-esa",
						Generation: int64(2),
					},
					Spec: v1alpha1.ElastalertSpec{
						Cert: "abc",
						ConfigSetting: v1alpha1.NewFreeForm(map[string]interface{}{
							"config": "test",
						}),
						Rule: []v1alpha1.FreeForm{
							v1alpha1.NewFreeForm(map[string]interface{}{
								"name": "test-elastalert", "type": "any",
							}),
						},
					},
				},
			).Build(),
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Spec: v1alpha1.ElastalertSpec{
					Cert: "abdec",
					ConfigSetting: v1alpha1.NewFreeForm(map[string]interface{}{
						"config": "test",
					}),
					Rule: []v1alpha1.FreeForm{
						v1alpha1.NewFreeForm(map[string]interface{}{
							"name": "test-elastalert", "type": "any",
						}),
					},
				},
			},
			testFunc: applyDeployment,
			result:   false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			defer monkey.Unpatch(tc.testFunc)
			defer monkey.Unpatch(ob.UpdateElastalertStatus)
			log := ctrl.Log.WithName("test").WithName("Elastalert")
			r := &ElastalertReconciler{
				Client:   tc.c,
				Log:      log,
				Scheme:   s,
				Recorder: record.NewBroadcaster().NewRecorder(s, corev1.EventSource{}),
			}
			nsn := types.NamespacedName{Name: "my-esa", Namespace: "esa1"}
			req := reconcile.Request{NamespacedName: nsn}
			if tc.result {
				_, err := r.Reconcile(context.Background(), req)
				assert.NoError(t, err)

				monkey.Patch(podspec.GenerateNewDeployment, func(Scheme *runtime.Scheme, e *v1alpha1.Elastalert) (*appsv1.Deployment, error) {
					return nil, errors.New("test update failed")
				})
				_, err = r.Reconcile(context.Background(), req)
				fmt.Println(err)
				assert.Error(t, err)
			} else {
				monkey.Patch(tc.testFunc, func(c client.Client, Scheme *runtime.Scheme, ctx context.Context, e *v1alpha1.Elastalert) (*appsv1.Deployment, error) {
					return nil, errors.New("test")
				})

				monkey.Patch(ob.UpdateElastalertStatus, func(c client.Client, ctx context.Context, e *v1alpha1.Elastalert, flag string) error {
					return errors.New("test update failed")
				})
				_, err := r.Reconcile(context.Background(), req)
				assert.Error(t, err)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
	testCases := []struct {
		desc string
		c    client.Client
		cond metav1.Condition
		want v1alpha1.Elastalert
	}{
		{
			desc: "test to update elasalert success status",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "esa1",
						Name:      "my-esa",
					},
				}).Build(),
			cond: metav1.Condition{
				Type:               v1alpha1.ElastAlertAvailableType,
				Status:             v1alpha1.ElastAlertAvailableStatus,
				LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057012345600)),
				ObservedGeneration: 1,
				Reason:             v1alpha1.ElastAlertAvailableReason,
				Message:            "ElastAlert my-esa has successfully progressed.",
			},
			want: v1alpha1.Elastalert{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Elastalert",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Status: v1alpha1.ElastalertStatus{
					Version: "v1.0",
					Phase:   "RUNNING",
					Condictions: []metav1.Condition{
						{
							Type:               "Progressing",
							Status:             "True",
							ObservedGeneration: int64(1),
							LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057012345600)),
							Reason:             "NewElastAlertAvailable",
							Message:            "ElastAlert my-esa has successfully progressed.",
						},
					},
				},
			},
		},
		{
			desc: "test to update elasalert failed status",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "esa1",
						Name:      "my-esa",
					},
				}).Build(),
			cond: metav1.Condition{
				Type:               v1alpha1.ElastAlertUnAvailableType,
				Status:             v1alpha1.ElastAlertUnAvailableStatus,
				LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057012345600)),
				ObservedGeneration: 1,
				Reason:             v1alpha1.ElastAlertUnAvailableReason,
				Message:            "Failed to apply ElastAlert my-esa resources.",
			},
			want: v1alpha1.Elastalert{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Elastalert",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Status: v1alpha1.ElastalertStatus{
					Version: "v1.0",
					Phase:   "FAILED",
					Condictions: []metav1.Condition{
						{
							Type:               "Stopped",
							Status:             "False",
							ObservedGeneration: int64(1),
							LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057012345600)),
							Reason:             "ElastAlertUnAvailable",
							Message:            "Failed to apply ElastAlert my-esa resources.",
						},
					},
				},
			},
		},
		{
			desc: "test to remove elasalert success status",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "esa1",
						Name:      "my-esa",
					},
					Status: v1alpha1.ElastalertStatus{
						Condictions: []metav1.Condition{
							{
								Type:               "Progressing",
								Status:             "True",
								ObservedGeneration: int64(1),
								LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057012345600)),
								Reason:             "NewElastAlertAvailable",
								Message:            "ElastAlert my-esa has successfully progressed.",
							},
						},
					},
				}).Build(),
			cond: metav1.Condition{
				Type:               v1alpha1.ElastAlertUnAvailableType,
				Status:             v1alpha1.ElastAlertUnAvailableStatus,
				LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057012345600)),
				ObservedGeneration: 1,
				Reason:             v1alpha1.ElastAlertUnAvailableReason,
				Message:            "Failed to apply ElastAlert my-esa resources.",
			},
			want: v1alpha1.Elastalert{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Elastalert",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Status: v1alpha1.ElastalertStatus{
					Version: "v1.0",
					Phase:   "FAILED",
					Condictions: []metav1.Condition{
						{
							Type:               "Stopped",
							Status:             "False",
							ObservedGeneration: int64(1),
							LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057012345600)),
							Reason:             "ElastAlertUnAvailable",
							Message:            "Failed to apply ElastAlert my-esa resources.",
						},
					},
				},
			},
		},
		{
			desc: "test to update initializing status",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "esa1",
						Name:      "my-esa",
					},
					Status: v1alpha1.ElastalertStatus{
						Condictions: []metav1.Condition{
							{
								Type:               "Progressing",
								Status:             "True",
								ObservedGeneration: int64(1),
								LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057000000000)),
								Reason:             "NewElastAlertAvailable",
								Message:            "ElastAlert my-esa has successfully progressed.",
							},
						},
					},
				}).Build(),
			cond: metav1.Condition{
				Type: v1alpha1.ElastAlertResourcesCreating,
			},
			want: v1alpha1.Elastalert{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Elastalert",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Status: v1alpha1.ElastalertStatus{
					Version: "v1.0",
					Phase:   "INITIALIZING",
					Condictions: []metav1.Condition{
						{
							Type:               "Progressing",
							Status:             "True",
							ObservedGeneration: int64(1),
							LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057000000000)),
							Reason:             "NewElastAlertAvailable",
							Message:            "ElastAlert my-esa has successfully progressed.",
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// prepare
			var log = ctrl.Log.WithName("test").WithName("Elastalert")
			r := &ElastalertReconciler{
				Client: tc.c,
				Log:    log,
				Scheme: s,
			}
			esa := v1alpha1.Elastalert{}
			err := r.Client.Get(context.Background(), types.NamespacedName{
				Namespace: "esa1",
				Name:      "my-esa",
			}, &esa)

			err = ob.UpdateStatus(r.Client, context.Background(), &esa, tc.cond)
			require.NoError(t, err)
			assert.Equal(t, tc.want.Status, esa.Status)
		})
	}
}

func TestNewCondition(t *testing.T) {
	testCases := []struct {
		name       string
		flag       string
		elastalert v1alpha1.Elastalert
		want       metav1.Condition
	}{
		{
			name: "test failed condition",
			flag: "failed",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "esa1",
					Name:       "my-esa",
					Generation: int64(1),
				},
			},
			want: metav1.Condition{
				Type:               "Stopped",
				Status:             "False",
				ObservedGeneration: int64(1),
				LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057012345600)),
				Reason:             "ElastAlertUnAvailable",
				Message:            "Failed to apply ElastAlert my-esa resources.",
			},
		},
		{
			name: "test success condition",
			flag: "success",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "esa1",
					Name:       "my-esa",
					Generation: int64(1),
				},
			},
			want: metav1.Condition{
				Type:               "Progressing",
				Status:             "True",
				ObservedGeneration: int64(1),
				LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057012345600)),
				Reason:             "NewElastAlertAvailable",
				Message:            "ElastAlert my-esa has successfully progressed.",
			},
		},
		{
			name: "test starting condition",
			flag: "starting",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "esa1",
					Name:       "my-esa",
					Generation: int64(1),
				},
			},
			want: metav1.Condition{
				Type: "starting",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			monkey.Patch(podspec.GetUtcTime, func() time.Time {
				return time.Unix(0, 1233810057012345600)
			})
			have := ob.NewCondition(&tc.elastalert, tc.flag)
			require.Equal(t, tc.want, *have)
		})
	}
}

func TestUpdateElastalertStatus(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
	testCases := []struct {
		desc string
		flag string
		c    client.Client
		want v1alpha1.Elastalert
	}{
		{
			desc: "test to update elasalert success status",
			flag: "success",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  "esa1",
						Name:       "my-esa",
						Generation: int64(1),
					},
				}).Build(),
			want: v1alpha1.Elastalert{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Elastalert",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Status: v1alpha1.ElastalertStatus{
					Version: "v1.0",
					Phase:   "RUNNING",
					Condictions: []metav1.Condition{
						{
							Type:               "Progressing",
							Status:             "True",
							ObservedGeneration: int64(1),
							LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057012345600)),
							Reason:             "NewElastAlertAvailable",
							Message:            "ElastAlert my-esa has successfully progressed.",
						},
					},
				},
			},
		},
		{
			desc: "test to update elasalert failed status",
			flag: "failed",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  "esa1",
						Name:       "my-esa",
						Generation: int64(1),
					},
				}).Build(),
			want: v1alpha1.Elastalert{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Elastalert",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Status: v1alpha1.ElastalertStatus{
					Version: "v1.0",
					Phase:   "FAILED",
					Condictions: []metav1.Condition{
						{
							Type:               "Stopped",
							Status:             "False",
							ObservedGeneration: int64(1),
							LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057012345600)),
							Reason:             "ElastAlertUnAvailable",
							Message:            "Failed to apply ElastAlert my-esa resources.",
						},
					},
				},
			},
		},
		{
			desc: "test to update elasalert initializing status",
			flag: "starting",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  "esa1",
						Name:       "my-esa",
						Generation: int64(1),
					},
				}).Build(),
			want: v1alpha1.Elastalert{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Elastalert",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Status: v1alpha1.ElastalertStatus{
					Version: "v1.0",
					Phase:   "INITIALIZING",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// prepare
			var log = ctrl.Log.WithName("test").WithName("Elastalert")
			r := &ElastalertReconciler{
				Client: tc.c,
				Log:    log,
				Scheme: s,
			}
			esa := v1alpha1.Elastalert{}
			err := r.Client.Get(context.Background(), types.NamespacedName{
				Namespace: "esa1",
				Name:      "my-esa",
			}, &esa)

			monkey.Patch(podspec.GetUtcTime, func() time.Time {
				return time.Unix(0, 1233810057012345600)
			})
			err = ob.UpdateElastalertStatus(r.Client, context.Background(), &esa, tc.flag)
			require.NoError(t, err)
			assert.Equal(t, tc.want.Status, esa.Status)
		})
	}
}

func TestElastalertReconciler_SetupWithManager(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
	var log = ctrl.Log.WithName("test").WithName("Elastalert")
	r := &ElastalertReconciler{
		Client: fake.NewClientBuilder().WithRuntimeObjects(
			&v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "esa1",
					Name:       "my-esa",
					Generation: int64(1),
				},
			}).Build(),
		Log:    log,
		Scheme: s,
	}
	assert.Error(t, r.SetupWithManager(nil))
}

func TestReconcileApplyDeploymentFailedWithUpdateStatus(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
	testCases := []struct {
		desc string
		c    client.Client
		want v1alpha1.Elastalert
	}{

		{
			desc: "test elastalert apply deployment failed with update deployment status",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "esa1",
						Name:      "my-esa",
					},
				},
			).Build(),
			want: v1alpha1.Elastalert{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Elastalert",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       "esa1",
					Name:            "my-esa",
					ResourceVersion: "1001",
				},
				Spec: v1alpha1.ElastalertSpec{
					ConfigSetting: v1alpha1.NewFreeForm(map[string]interface{}{
						"rules_folder": "/etc/elastalert/rules/..data/",
					}),
					Alert: v1alpha1.NewFreeForm(map[string]interface{}{}),
				},
				Status: v1alpha1.ElastalertStatus{
					Version: "v1.0",
					Phase:   "FAILED",
					Condictions: []metav1.Condition{
						{
							Type:               "Stopped",
							Status:             "False",
							ObservedGeneration: int64(0),
							LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057000000000)),
							Reason:             "ElastAlertUnAvailable",
							Message:            "Failed to apply ElastAlert my-esa resources.",
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		defer monkey.Unpatch(podspec.GenerateNewDeployment)
		t.Run(tc.desc, func(t *testing.T) {
			log := ctrl.Log.WithName("test").WithName("Elastalert")
			r := &ElastalertReconciler{
				Client:   tc.c,
				Log:      log,
				Scheme:   s,
				Recorder: record.NewBroadcaster().NewRecorder(s, corev1.EventSource{}),
			}
			nsn := types.NamespacedName{Name: "my-esa", Namespace: "esa1"}
			req := reconcile.Request{NamespacedName: nsn}

			monkey.Patch(podspec.GenerateNewDeployment, func(Scheme *runtime.Scheme, e *v1alpha1.Elastalert) (*appsv1.Deployment, error) {
				return nil, errors.New("test update failed")
			})
			monkey.Patch(podspec.GetUtcTime, func() time.Time {
				return time.Unix(0, 1233810057012345600)
			})
			_, err := r.Reconcile(context.Background(), req)
			assert.Error(t, err)
			ea := v1alpha1.Elastalert{}
			err = r.Client.Get(context.Background(), nsn, &ea)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, ea)

		})
	}
}

func TestReconcileApplySecretFailedWithUpdateStatus(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
	testCases := []struct {
		desc string
		c    client.Client
		want v1alpha1.Elastalert
	}{

		{
			desc: "test elastalert apply deployment failed with update secret status",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "esa1",
						Name:      "my-esa",
					},
				},
			).Build(),
			want: v1alpha1.Elastalert{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Elastalert",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       "esa1",
					Name:            "my-esa",
					ResourceVersion: "1001",
				},
				Spec: v1alpha1.ElastalertSpec{
					ConfigSetting: v1alpha1.NewFreeForm(map[string]interface{}{}),
					Alert:         v1alpha1.NewFreeForm(map[string]interface{}{}),
				},
				Status: v1alpha1.ElastalertStatus{
					Version: "v1.0",
					Phase:   "FAILED",
					Condictions: []metav1.Condition{
						{
							Type:               "Stopped",
							Status:             "False",
							ObservedGeneration: int64(0),
							LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057000000000)),
							Reason:             "ElastAlertUnAvailable",
							Message:            "Failed to apply ElastAlert my-esa resources.",
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		defer monkey.Unpatch(podspec.GenerateCertSecret)
		t.Run(tc.desc, func(t *testing.T) {
			log := ctrl.Log.WithName("test").WithName("Elastalert")
			r := &ElastalertReconciler{
				Client:   tc.c,
				Log:      log,
				Scheme:   s,
				Recorder: record.NewBroadcaster().NewRecorder(s, corev1.EventSource{}),
			}
			nsn := types.NamespacedName{Name: "my-esa", Namespace: "esa1"}
			req := reconcile.Request{NamespacedName: nsn}

			monkey.Patch(podspec.GenerateCertSecret, func(Scheme *runtime.Scheme, e *v1alpha1.Elastalert) (*corev1.Secret, error) {
				return nil, errors.New("test update failed")
			})
			monkey.Patch(podspec.GetUtcTime, func() time.Time {
				return time.Unix(0, 1233810057012345600)
			})
			_, err := r.Reconcile(context.Background(), req)
			assert.Error(t, err)
			ea := v1alpha1.Elastalert{}
			err = r.Client.Get(context.Background(), nsn, &ea)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, ea)
		})
	}
}

func TestReconcileApplyConfigMapsFailedWithUpdateStatus(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
	testCases := []struct {
		desc string
		c    client.Client
		want v1alpha1.Elastalert
	}{

		{
			desc: "test elastalert apply deployment failed with update configmaps status",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "esa1",
						Name:      "my-esa",
					},
				},
			).Build(),
			want: v1alpha1.Elastalert{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Elastalert",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       "esa1",
					Name:            "my-esa",
					ResourceVersion: "1001",
				},
				Spec: v1alpha1.ElastalertSpec{
					ConfigSetting: v1alpha1.NewFreeForm(map[string]interface{}{
						"rules_folder": "/etc/elastalert/rules/..data/",
					}),
					Alert: v1alpha1.NewFreeForm(map[string]interface{}{}),
				},
				Status: v1alpha1.ElastalertStatus{
					Version: "v1.0",
					Phase:   "FAILED",
					Condictions: []metav1.Condition{
						{
							Type:               "Stopped",
							Status:             "False",
							ObservedGeneration: int64(0),
							LastTransitionTime: metav1.NewTime(time.Unix(0, 1233810057000000000)),
							Reason:             "ElastAlertUnAvailable",
							Message:            "Failed to apply ElastAlert my-esa resources.",
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		defer monkey.Unpatch(podspec.PatchAlertSettings)
		t.Run(tc.desc, func(t *testing.T) {
			log := ctrl.Log.WithName("test").WithName("Elastalert")
			r := &ElastalertReconciler{
				Client:   tc.c,
				Log:      log,
				Scheme:   s,
				Recorder: record.NewBroadcaster().NewRecorder(s, corev1.EventSource{}),
			}
			nsn := types.NamespacedName{Name: "my-esa", Namespace: "esa1"}
			req := reconcile.Request{NamespacedName: nsn}

			monkey.Patch(podspec.PatchAlertSettings, func(e *v1alpha1.Elastalert) error {
				return errors.New("test update failed")
			})
			monkey.Patch(podspec.GetUtcTime, func() time.Time {
				return time.Unix(0, 1233810057012345600)
			})
			_, err := r.Reconcile(context.Background(), req)
			assert.Error(t, err)
			ea := v1alpha1.Elastalert{}
			err = r.Client.Get(context.Background(), nsn, &ea)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, ea)
		})
	}
}

func TestReconcileRuntimedWithUpdateStatusFailed(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
	testCases := []struct {
		desc string
		c    client.Client
		want v1alpha1.Elastalert
	}{

		{
			desc: "test elastalert runtime with updating status failed",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "esa1",
						Name:      "my-esa",
					},
				},
			).Build(),
			want: v1alpha1.Elastalert{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Elastalert",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       "esa1",
					Name:            "my-esa",
					ResourceVersion: "999",
				},
				Spec: v1alpha1.ElastalertSpec{
					ConfigSetting: v1alpha1.NewFreeForm(map[string]interface{}{}),
					Alert:         v1alpha1.NewFreeForm(map[string]interface{}{}),
				},
			},
		},
	}
	for _, tc := range testCases {
		defer monkey.Unpatch(ob.UpdateStatus)
		t.Run(tc.desc, func(t *testing.T) {
			log := ctrl.Log.WithName("test").WithName("Elastalert")
			r := &ElastalertReconciler{
				Client:   tc.c,
				Log:      log,
				Scheme:   s,
				Recorder: record.NewBroadcaster().NewRecorder(s, corev1.EventSource{}),
			}
			nsn := types.NamespacedName{Name: "my-esa", Namespace: "esa1"}
			req := reconcile.Request{NamespacedName: nsn}

			monkey.Patch(ob.UpdateStatus, func(c client.Client, ctx context.Context, e *v1alpha1.Elastalert, condition metav1.Condition) error {
				return errors.New("test update failed")
			})
			_, err := r.Reconcile(context.Background(), req)
			assert.Error(t, err)
			ea := v1alpha1.Elastalert{}
			err = r.Client.Get(context.Background(), nsn, &ea)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, ea)
		})
	}
}
