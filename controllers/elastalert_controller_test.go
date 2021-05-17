package controllers

import (
	"context"
	"elastalert/api/v1alpha1"
	"elastalert/controllers/podspec"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestApplyConfigMaps(t *testing.T) {
	testCases := []struct {
		desc       string
		elastalert v1alpha1.Elastalert
	}{
		{
			desc: "test apply configmap",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Spec: v1alpha1.ElastalertSpec{
					ConfigSetting: map[string]string{
						"config.yaml":    "test: configmaps",
						"test-rule.yaml": "test: configmaps",
					},
				},
			},
		},
		{
			desc: "test apply configmap2",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Spec: v1alpha1.ElastalertSpec{
					ConfigSetting: map[string]string{
						"config.yaml":     "test: configmaps",
						"test-rule1.yaml": "test: configmaps",
						"test-rule2.yaml": "test: configmaps",
						"test-rule3.yaml": "test: configmaps",
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			s := scheme.Scheme
			var log logr.Logger
			cl := fake.NewClientBuilder().Build()
			r := &ElastalertReconciler{
				Client: cl,
				Log:    log,
				Scheme: s,
			}
			cms := corev1.ConfigMapList{}
			r.Scheme.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
			r.Scheme.AddKnownTypes(corev1.SchemeGroupVersion, &cms)
			err := applyConfigMaps(cl, r.Scheme, context.Background(), &tc.elastalert)
			assert.NoError(t, err)
			err = cl.List(context.Background(), &cms)
			require.NoError(t, err)
			assert.Len(t, cms.Items, 2)
		})
	}
}

func TestApplySecret(t *testing.T) {
	testCases := []struct {
		desc       string
		elastalert v1alpha1.Elastalert
	}{
		{
			desc: "test apply secret",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "esa1",
					Name:      "my-esa",
				},
				Spec: v1alpha1.ElastalertSpec{
					Cert: map[string]string{
						"elasticCA.crt": "abc",
					},
				},
			},
		},
		{
			desc: "test apply secret2",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					//Namespace: "esa1",
					Name: "my-esa",
				},
				Spec: v1alpha1.ElastalertSpec{
					Cert: map[string]string{
						"elasticCA.crt": "abc",
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// prepare
			s := scheme.Scheme
			var log logr.Logger
			cl := fake.NewClientBuilder().Build()
			r := &ElastalertReconciler{
				Client: cl,
				Log:    log,
				Scheme: s,
			}
			se := corev1.Secret{}
			r.Scheme.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
			r.Scheme.AddKnownTypes(corev1.SchemeGroupVersion, &se)
			err := applySecret(cl, r.Scheme, context.Background(), &tc.elastalert)
			assert.NoError(t, err)
			err = cl.Get(context.Background(), types.NamespacedName{
				Namespace: tc.elastalert.Namespace,
				Name:      podspec.DefaultCertName,
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
	}{
		{
			desc: "test apply deployment",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					//Namespace: "esa1",
					Name: "my-esa",
				},
				Spec: v1alpha1.ElastalertSpec{
					Cert: map[string]string{
						"elasticCA.crt": "abc",
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			s := scheme.Scheme
			var log logr.Logger
			cl := fake.NewClientBuilder().Build()
			r := &ElastalertReconciler{
				Client: cl,
				Log:    log,
				Scheme: s,
			}
			dep := appsv1.Deployment{}
			r.Scheme.AddKnownTypes(corev1.SchemeGroupVersion, &v1alpha1.Elastalert{})
			r.Scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &dep)
			_, err := applyDeployment(cl, r.Scheme, context.Background(), &tc.elastalert)
			assert.NoError(t, err)
			err = cl.Get(context.Background(), types.NamespacedName{
				Namespace: tc.elastalert.Namespace,
				Name:      tc.elastalert.Name,
			}, &dep)
			require.NoError(t, err)
		})
	}

}
