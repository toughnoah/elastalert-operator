package podspec

import (
	esv1alpha1 "elastalert/api/v1alpha1"
	"gopkg.in/yaml.v2"
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

func TestPatchConfigSettings(t *testing.T) {
	testCases := []struct {
		name       string
		yamlString string
		certString string
		elastalert *esv1alpha1.Elastalert
		want       string
	}{
		{
			name:       "test use ssl and cert undefined",
			certString: "",
			yamlString: testUseSSLAndCertUndefined,
			elastalert: &esv1alpha1.Elastalert{
				Spec: esv1alpha1.ElastalertSpec{
					ConfigSetting: map[string]string{
						"config.yaml": testUseSSLAndCertUndefined,
					},
				},
			},
			want: wantUseSSLAndCertUndefined,
		},
		{
			name:       "test not use ssl",
			certString: "",
			yamlString: testNotUseSSL,
			elastalert: &esv1alpha1.Elastalert{
				Spec: esv1alpha1.ElastalertSpec{
					ConfigSetting: map[string]string{
						"config.yaml": testNotUseSSL,
					},
				},
			},
			want: wantNotUseSSL,
		},
		{
			name:       "test use ssl and cert defined",
			certString: "abc",
			yamlString: testUseSSLAndCertDefined,
			elastalert: &esv1alpha1.Elastalert{
				Spec: esv1alpha1.ElastalertSpec{
					ConfigSetting: map[string]string{
						"config.yaml": testUseSSLAndCertDefined,
					},
				},
			},
			want: wantUseSSLAndCertDndefined,
		},
		{
			name:       "test add rules folder",
			certString: "abc",
			yamlString: testRulesFolder,
			elastalert: &esv1alpha1.Elastalert{
				Spec: esv1alpha1.ElastalertSpec{
					ConfigSetting: map[string]string{
						"config.yaml": testRulesFolder,
					},
				},
			},
			want: wantRulesFolder,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var want map[string]interface{}
			var have map[string]interface{}
			err := PatchConfigSettings(tc.elastalert, tc.certString)
			if err != nil {
				panic(err)
			}
			if err = yaml.Unmarshal([]byte(tc.want), &want); err != nil {
				panic(err)
			}
			if err = yaml.Unmarshal([]byte(tc.elastalert.Spec.ConfigSetting["config.yaml"]), &have); err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(have, want) {
				t.Errorf("podspec.PatchConfigSettings() = %v, want %v", have, tc.want)
			}
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
