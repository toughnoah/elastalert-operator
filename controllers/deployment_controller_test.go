package controllers

import (
	"context"
	"elastalert/api/v1alpha1"
	"elastalert/controllers/podspec"
	"errors"
	"github.com/bouk/monkey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

func TestReCreateDeployment(t *testing.T) {
	testCases := []struct {
		desc       string
		elastalert v1alpha1.Elastalert
	}{
		{
			desc: "test recreate deployment",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Spec: v1alpha1.ElastalertSpec{
					Cert: "abc",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			s := scheme.Scheme
			var log = ctrl.Log.WithName("test").WithName("Elastalert")
			cl := fake.NewClientBuilder().Build()
			r := &ElastalertReconciler{
				Client: cl,
				Log:    log,
				Scheme: s,
			}
			monkey.Patch(podspec.GetUtcTimeString, func() string {
				return "2021-05-17T01:38:44+08:00"
			})
			dep := appsv1.Deployment{}
			r.Scheme.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
			r.Scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &dep)
			_, err := recreateDeployment(cl, r.Scheme, context.Background(), &tc.elastalert)
			assert.NoError(t, err)
			err = cl.Get(context.Background(), types.NamespacedName{
				Namespace: tc.elastalert.Namespace,
				Name:      tc.elastalert.Name,
			}, &dep)
			require.NoError(t, err)
		})
	}

}

func TestDeploymentReconcile(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
	testCases := []struct {
		desc         string
		c            client.Client
		testNotfound bool
		want         appsv1.Deployment
	}{
		{
			desc: "test deployment reconcile",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test",
						Name:      "test-elastalert",
					},
					Spec: v1alpha1.ElastalertSpec{
						PodTemplateSpec: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name: "elastalert",
									},
								},
							},
						},
					},
				}).Build(),
			want: appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
					Namespace:       "test",
					Name:            "test-elastalert",
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
								PodAntiAffinity: &corev1.PodAntiAffinity{},
							},
						},
					},
				},
			},
		},
		{
			desc:         "test deployment reconcile 1",
			c:            fake.NewClientBuilder().Build(),
			testNotfound: true,
		},
	}
	for _, tc := range testCases {

		t.Run(tc.desc, func(t *testing.T) {
			log := ctrl.Log.WithName("test").WithName("Elastalert")
			r := &DeploymentReconciler{
				Client: tc.c,
				Log:    log,
				Scheme: s,
			}
			ctx := context.Background()
			nsn := types.NamespacedName{Name: "test-elastalert", Namespace: "test"}
			req := reconcile.Request{NamespacedName: nsn}
			_, err := r.Reconcile(ctx, req)
			assert.NoError(t, err)
			if !tc.testNotfound {
				dep := appsv1.Deployment{}
				err = r.Client.Get(ctx, req.NamespacedName, &dep)
				assert.NoError(t, err)
				dep.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = "2021-05-17T01:38:44+08:00"
				assert.Equal(t, tc.want, dep)
			}
		})
	}
}

func TestDeploymentReconcileFailed(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
	testCases := []struct {
		desc     string
		c        client.Client
		isToWait bool
	}{
		{
			desc: "test deployment reconcile failed",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test",
						Name:      "test-elastalert",
					},
					Spec: v1alpha1.ElastalertSpec{
						PodTemplateSpec: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name: "elastalert",
									},
								},
							},
						},
					},
				}).Build(),
			isToWait: false,
		},
		{
			desc: "test deployment reconcile failed",
			c: fake.NewClientBuilder().WithRuntimeObjects(
				&v1alpha1.Elastalert{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test",
						Name:      "test-elastalert",
					},
					Spec: v1alpha1.ElastalertSpec{
						PodTemplateSpec: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name: "elastalert",
									},
								},
							},
						},
					},
				}).Build(),
			isToWait: true,
		},
	}
	for _, tc := range testCases {
		defer monkey.Unpatch(recreateDeployment)
		defer monkey.Unpatch(UpdateElastalertStatus)
		defer monkey.Unpatch(podspec.WaitForStability)
		t.Run(tc.desc, func(t *testing.T) {
			log := ctrl.Log.WithName("test").WithName("Elastalert")
			r := &DeploymentReconciler{
				Client: tc.c,
				Log:    log,
				Scheme: s,
			}
			if !tc.isToWait {
				monkey.Patch(recreateDeployment, func(c client.Client, Scheme *runtime.Scheme, ctx context.Context, e *v1alpha1.Elastalert) (*appsv1.Deployment, error) {
					return nil, errors.New("test")
				})
				monkey.Patch(UpdateElastalertStatus, func(c client.Client, ctx context.Context, e *v1alpha1.Elastalert, flag string) error {
					return errors.New("test update failed")
				})
			} else {
				monkey.Patch(recreateDeployment, func(c client.Client, Scheme *runtime.Scheme, ctx context.Context, e *v1alpha1.Elastalert) (*appsv1.Deployment, error) {
					return nil, errors.New("test")
				})
				monkey.Patch(podspec.WaitForStability, func(c client.Client, ctx context.Context, dep appsv1.Deployment) error {
					return errors.New("test")
				})
				monkey.Patch(UpdateElastalertStatus, func(c client.Client, ctx context.Context, e *v1alpha1.Elastalert, flag string) error {
					return errors.New("test update failed")
				})
			}
			ctx := context.Background()
			nsn := types.NamespacedName{Name: "test-elastalert", Namespace: "test"}
			req := reconcile.Request{NamespacedName: nsn}
			_, err := r.Reconcile(ctx, req)
			assert.Error(t, err)
		})
	}
}
