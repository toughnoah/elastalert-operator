package podspec

import (
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
)

var (
	testUseSSLAndCertUndefined = `
use_ssl: True`

	wantUseSSLAndCertUndefined = `
use_ssl: True
rules_folder: /etc/elastalert/rules/..data/
verify_certs: False

`
	testNotUseSSL = `
use_ssl: False
`
	wantNotUseSSL = `
use_ssl: False
rules_folder: /etc/elastalert/rules/..data/
`

	testUseSSLAndCertDefined = `
use_ssl: True`

	wantUseSSLAndCertDndefined = `
use_ssl: True
rules_folder: /etc/elastalert/rules/..data/
verify_certs: True
ca_certs: /ssl/elasticCA.crt

`
	testRulesFolder = `
`

	wantRulesFolder = `
rules_folder: /etc/elastalert/rules/..data/
`
)

//func TestPatchConfigSettings(t *testing.T) {
//	testCases := []struct {
//		name       string
//		yamlString string
//		certString string
//		elastalert *esv1alpha1.Elastalert
//		want       string
//	}{
//		{
//			name:       "test use ssl and cert undefined",
//			certString: "",
//			yamlString: testUseSSLAndCertUndefined,
//			elastalert: &esv1alpha1.Elastalert{
//				Spec: esv1alpha1.ElastalertSpec{
//					ConfigSetting: map[string]string{
//						"config.yaml": testUseSSLAndCertUndefined,
//					},
//				},
//			},
//			want: wantUseSSLAndCertUndefined,
//		},
//		{
//			name:       "test not use ssl",
//			certString: "",
//			yamlString: testNotUseSSL,
//			elastalert: &esv1alpha1.Elastalert{
//				Spec: esv1alpha1.ElastalertSpec{
//					ConfigSetting: map[string]string{
//						"config.yaml": testNotUseSSL,
//					},
//				},
//			},
//			want: wantNotUseSSL,
//		},
//		{
//			name:       "test use ssl and cert defined",
//			certString: "abc",
//			yamlString: testUseSSLAndCertDefined,
//			elastalert: &esv1alpha1.Elastalert{
//				Spec: esv1alpha1.ElastalertSpec{
//					ConfigSetting: map[string]string{
//						"config.yaml": testUseSSLAndCertDefined,
//					},
//				},
//			},
//			want: wantUseSSLAndCertDndefined,
//		},
//		{
//			name:       "test add rules folder",
//			certString: "abc",
//			yamlString: testRulesFolder,
//			elastalert: &esv1alpha1.Elastalert{
//				Spec: esv1alpha1.ElastalertSpec{
//					ConfigSetting: map[string]string{
//						"config.yaml": testRulesFolder,
//					},
//				},
//			},
//			want: wantRulesFolder,
//		},
//	}
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			var want map[string]interface{}
//			var have map[string]interface{}
//			err := PatchConfigSettings(tc.elastalert, tc.certString)
//			if err != nil {
//				panic(err)
//			}
//			if err = yaml.Unmarshal([]byte(tc.want), &want); err != nil {
//				panic(err)
//			}
//			if err = yaml.Unmarshal([]byte(tc.elastalert.Spec.ConfigSetting["config.yaml"]), &have); err != nil {
//				panic(err)
//			}
//			if !reflect.DeepEqual(have, want) {
//				t.Errorf("podspec.PatchConfigSettings() = %v, want %v", have, tc.want)
//			}
//		})
//	}
//}

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

//func TestGenerateNewConfigmap(t *testing.T) {
//	testCases := []struct {
//		name       string
//		elastalert esv1alpha1.Elastalert
//		suffx      string
//		want       corev1.ConfigMap
//	}{
//		{
//			name:  "test generate default config",
//			suffx: "-config",
//			elastalert: esv1alpha1.Elastalert{
//				ObjectMeta: metav1.ObjectMeta{
//					Name: "test-elastalert",
//				},
//				Spec: esv1alpha1.ElastalertSpec{
//					ConfigSetting: map[string]string{
//						"config.yaml": "test: configmap",
//					},
//				},
//			},
//			want: corev1.ConfigMap{
//				ObjectMeta: metav1.ObjectMeta{
//					Name: "test-elastalert-config",
//					OwnerReferences: []metav1.OwnerReference{
//						{
//							APIVersion:         "v1",
//							Kind:               "Elastalert",
//							Name:               "test-elastalert",
//							UID:                "",
//							Controller:         &varTrue,
//							BlockOwnerDeletion: &varTrue,
//						},
//					},
//				},
//				Data: map[string]string{
//					"config.yaml": "test: configmap",
//				},
//			},
//		},
//		{
//			name:  "test generate default rule",
//			suffx: "-rule",
//			elastalert: esv1alpha1.Elastalert{
//				ObjectMeta: metav1.ObjectMeta{
//					Name: "test-elastalert",
//				},
//				Spec: esv1alpha1.ElastalertSpec{
//					ConfigSetting: map[string]string{
//						"config.yaml": "test: configmap",
//					},
//					Rule: map[string]string{
//						"rule.yaml": "test: rule",
//					},
//				},
//			},
//			want: corev1.ConfigMap{
//				ObjectMeta: metav1.ObjectMeta{
//					Name: "test-elastalert-rule",
//					OwnerReferences: []metav1.OwnerReference{
//						{
//							APIVersion:         "v1",
//							Kind:               "Elastalert",
//							Name:               "test-elastalert",
//							UID:                "",
//							Controller:         &varTrue,
//							BlockOwnerDeletion: &varTrue,
//						},
//					},
//				},
//				Data: map[string]string{
//					"rule.yaml": "test: rule",
//				},
//			},
//		},
//	}
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			s := scheme.Scheme
//			s.AddKnownTypes(corev1.SchemeGroupVersion, &esv1alpha1.Elastalert{})
//			have, err := GenerateNewConfigmap(s, &tc.elastalert, tc.suffx)
//			require.NoError(t, err)
//			require.Equal(t, tc.want, *have)
//		})
//	}
//}

func TestGenerateYamlMap(t *testing.T) {
	testCases := []struct {
		name     string
		maparray []map[string]interface{}
		want     map[string]string
	}{
		{
			name: "test generate yaml map",
			maparray: []map[string]interface{}{
				{
					"name": "test-elastalert",
					"type": "any",
				},
				{
					"name": "test-elastalert2",
					"type": "aggs",
				},
			},
			want: map[string]string{
				"test-elastalert.yaml":  "1",
				"test-elastalert2.yaml": "1",
			},
		},
	}
	for _, tc := range testCases {
		have, err := GenerateYamlMap(tc.maparray)
		require.NoError(t, err)
		require.Equal(t, have, tc.want)
	}
}
