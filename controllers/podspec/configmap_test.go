package podspec

import (
	"github.com/stretchr/testify/require"
	esv1alpha1 "github.com/toughnoah/elastalert-operator/api/v1alpha1"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"reflect"
	"testing"
)

var (
	wantUseSSLAndCertUndefined = `
use_ssl: True
rules_folder: /etc/elastalert/rules/..data/
verify_certs: False
`

	wantNotUseSSL = `
use_ssl: False
rules_folder: /etc/elastalert/rules/..data/
`

	wantUseSSLAndCertDndefined = `
use_ssl: True
rules_folder: /etc/elastalert/rules/..data/
verify_certs: True
ca_certs: /ssl/elasticCA.crt

`

	wantRulesFolder = `
rules_folder: /etc/elastalert/rules/..data/
`
)

func TestPatchConfigSettings(t *testing.T) {
	testCases := []struct {
		name       string
		certString string
		elastalert *esv1alpha1.Elastalert
		want       string
	}{
		{
			name:       "test use ssl and cert undefined",
			certString: "",
			elastalert: &esv1alpha1.Elastalert{
				Spec: esv1alpha1.ElastalertSpec{
					ConfigSetting: esv1alpha1.NewFreeForm(map[string]interface{}{
						"use_ssl": true,
					}),
				},
			},
			want: wantUseSSLAndCertUndefined,
		},
		{
			name:       "test not use ssl",
			certString: "",
			elastalert: &esv1alpha1.Elastalert{
				Spec: esv1alpha1.ElastalertSpec{
					ConfigSetting: esv1alpha1.NewFreeForm(map[string]interface{}{
						"use_ssl": false,
					}),
				},
			},
			want: wantNotUseSSL,
		},
		{
			name:       "test use ssl and cert defined",
			certString: "abc",
			elastalert: &esv1alpha1.Elastalert{
				Spec: esv1alpha1.ElastalertSpec{
					ConfigSetting: esv1alpha1.NewFreeForm(map[string]interface{}{
						"use_ssl":      true,
						"verify_certs": true,
					}),
				},
			},
			want: wantUseSSLAndCertDndefined,
		},
		{
			name:       "test add rules folder",
			certString: "abc",
			elastalert: &esv1alpha1.Elastalert{
				Spec: esv1alpha1.ElastalertSpec{
					ConfigSetting: esv1alpha1.NewFreeForm(map[string]interface{}{}),
				},
			},
			want: wantRulesFolder,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var want map[string]interface{}
			err := PatchConfigSettings(tc.elastalert, tc.certString)
			require.NoError(t, err)
			err = yaml.Unmarshal([]byte(tc.want), &want)
			require.NoError(t, err)
			have, err := tc.elastalert.Spec.ConfigSetting.GetMap()
			require.Equal(t, want, have)
		})
	}
}

func TestConfigMapsToMap(t *testing.T) {
	testCases := []struct {
		name      string
		confimaps []corev1.ConfigMap
		want      map[string]corev1.ConfigMap
	}{
		{
			name: "test configmaps to map",
			confimaps: []corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "myconfigmap1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "myconfigmap2",
					},
				},
			},
			want: map[string]corev1.ConfigMap{
				"myconfigmap1": {
					ObjectMeta: metav1.ObjectMeta{
						Name: "myconfigmap1",
					},
				},
				"myconfigmap2": {
					ObjectMeta: metav1.ObjectMeta{
						Name: "myconfigmap2",
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			have := ConfigMapsToMap(tc.confimaps)
			if !reflect.DeepEqual(have, tc.want) {
				t.Errorf("podspec.ConfigMapsToMap() = %v, want %v", have, tc.want)
			}
		})
	}
}

func TestGenerateNewConfigmap(t *testing.T) {
	testCases := []struct {
		name       string
		elastalert esv1alpha1.Elastalert
		suffx      string
		want       corev1.ConfigMap
	}{
		{
			name:  "test generate default config",
			suffx: "-config",
			elastalert: esv1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-elastalert",
				},
				Spec: esv1alpha1.ElastalertSpec{
					ConfigSetting: esv1alpha1.NewFreeForm(map[string]interface{}{
						"config": "test",
					}),
				},
			},
			want: corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-elastalert-config",
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
				Data: map[string]string{
					"config.yaml": "config: test\n",
				},
			},
		},
		{
			name:  "test generate default rule",
			suffx: "-rule",
			elastalert: esv1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-elastalert",
				},
				Spec: esv1alpha1.ElastalertSpec{
					ConfigSetting: esv1alpha1.NewFreeForm(map[string]interface{}{
						"config": "test",
					}),
					Rule: []esv1alpha1.FreeForm{
						esv1alpha1.NewFreeForm(map[string]interface{}{
							"name": "test-elastalert", "type": "any",
						}),
					},
				},
			},
			want: corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-elastalert-rule",
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
				Data: map[string]string{
					"test-elastalert.yaml": "name: test-elastalert\ntype: any\n",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := scheme.Scheme
			s.AddKnownTypes(corev1.SchemeGroupVersion, &esv1alpha1.Elastalert{})
			have, err := GenerateNewConfigmap(s, &tc.elastalert, tc.suffx)
			require.NoError(t, err)
			require.Equal(t, tc.want, *have)
		})
	}
}

func TestGenerateYamlMap(t *testing.T) {
	testCases := []struct {
		name     string
		maparray []esv1alpha1.FreeForm
		want     map[string]string
	}{
		{
			name: "test generate yaml map",
			maparray: []esv1alpha1.FreeForm{
				esv1alpha1.NewFreeForm(map[string]interface{}{
					"name": "test-elastalert", "type": "any",
				}),
				esv1alpha1.NewFreeForm(map[string]interface{}{
					"name": "test-elastalert2", "type": "aggs",
				}),
			},
			want: map[string]string{
				"test-elastalert.yaml":  "name: test-elastalert\ntype: any\n",
				"test-elastalert2.yaml": "name: test-elastalert2\ntype: aggs\n",
			},
		},
	}
	for _, tc := range testCases {
		have, err := GenerateYamlMap(tc.maparray)
		require.NoError(t, err)
		require.Equal(t, tc.want, have)
	}
}

func TestPatchAlertSettings(t *testing.T) {
	testCases := []struct {
		name       string
		certString string
		elastalert *esv1alpha1.Elastalert
		want       esv1alpha1.Elastalert
	}{
		{
			name:       "test use global alert",
			certString: "",
			elastalert: &esv1alpha1.Elastalert{
				Spec: esv1alpha1.ElastalertSpec{
					ConfigSetting: esv1alpha1.NewFreeForm(map[string]interface{}{
						"use_ssl": true,
					}),
					Rule: []esv1alpha1.FreeForm{
						esv1alpha1.NewFreeForm(map[string]interface{}{
							"name": "test-elastalert1", "type": "any",
						}),
						esv1alpha1.NewFreeForm(map[string]interface{}{
							"name": "test-elastalert2", "type": "any",
						}),
					},
					Alert: esv1alpha1.NewFreeForm(map[string]interface{}{
						"alert": []string{"post"}, "http_post_url": "https://test.com",
					}),
				},
			},
			want: esv1alpha1.Elastalert{
				Spec: esv1alpha1.ElastalertSpec{
					ConfigSetting: esv1alpha1.NewFreeForm(map[string]interface{}{
						"use_ssl": true,
					}),
					Rule: []esv1alpha1.FreeForm{
						esv1alpha1.NewFreeForm(map[string]interface{}{
							"name": "test-elastalert1", "type": "any", "alert": []string{"post"}, "http_post_url": "https://test.com",
						}),
						esv1alpha1.NewFreeForm(map[string]interface{}{
							"name": "test-elastalert2", "type": "any", "alert": []string{"post"}, "http_post_url": "https://test.com",
						}),
					},
					Alert: esv1alpha1.NewFreeForm(map[string]interface{}{
						"alert": []string{"post"}, "http_post_url": "https://test.com",
					}),
				},
			},
		},
		{
			name:       "test not global alert",
			certString: "",
			elastalert: &esv1alpha1.Elastalert{
				Spec: esv1alpha1.ElastalertSpec{
					ConfigSetting: esv1alpha1.NewFreeForm(map[string]interface{}{
						"use_ssl": true,
					}),
					Rule: []esv1alpha1.FreeForm{
						esv1alpha1.NewFreeForm(map[string]interface{}{
							"name": "test-elastalert1", "type": "any", "alert": []string{"get"}, "http_post_url": "https://elatalert.com",
						}),
						esv1alpha1.NewFreeForm(map[string]interface{}{
							"name": "test-elastalert2", "type": "any",
						}),
					},
					Alert: esv1alpha1.NewFreeForm(map[string]interface{}{
						"alert": []string{"post"}, "http_post_url": "https://test.com",
					}),
				},
			},
			want: esv1alpha1.Elastalert{
				Spec: esv1alpha1.ElastalertSpec{
					ConfigSetting: esv1alpha1.NewFreeForm(map[string]interface{}{
						"use_ssl": true,
					}),
					Rule: []esv1alpha1.FreeForm{
						esv1alpha1.NewFreeForm(map[string]interface{}{
							"name": "test-elastalert1", "type": "any", "alert": []string{"get"}, "http_post_url": "https://elatalert.com",
						}),
						esv1alpha1.NewFreeForm(map[string]interface{}{
							"name": "test-elastalert2", "type": "any", "alert": []string{"post"}, "http_post_url": "https://test.com",
						}),
					},
					Alert: esv1alpha1.NewFreeForm(map[string]interface{}{
						"alert": []string{"post"}, "http_post_url": "https://test.com",
					}),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := PatchAlertSettings(tc.elastalert)
			require.NoError(t, err)
			require.Equal(t, tc.want, *tc.elastalert)
		})
	}
}
