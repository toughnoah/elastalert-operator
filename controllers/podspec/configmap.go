package podspec

import (
	esv1alpha1 "elastalert/api/v1alpha1"
	"errors"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func GenerateNewConfigmap(Scheme *runtime.Scheme, e *esv1alpha1.Elastalert, suffix string) (*corev1.ConfigMap, error) {
	var data map[string]string
	switch suffix {
	case esv1alpha1.RuleSuffx:
		data = e.Spec.Rule
	case esv1alpha1.ConfigSuffx:
		data = e.Spec.ConfigSetting
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Name + suffix,
			Namespace: e.Namespace,
		},
		Data: data,
	}
	err := ctrl.SetControllerReference(e, cm, Scheme)
	if err != nil {
		return nil, err
	}
	return cm, nil
}

func PatchConfigSettings(e *esv1alpha1.Elastalert, stringCert string) error {
	var config = map[string]interface{}{}
	var bytesConfig []byte
	var err error
	if e.Spec.ConfigSetting == nil {
		return errors.New("Not found config.yaml")
	}
	if err = yaml.Unmarshal([]byte(e.Spec.ConfigSetting["config.yaml"]), &config); err != nil {
		return err
	}
	config["rules_folder"] = DefaultRulesFolder
	if config["use_ssl"] != nil && config["use_ssl"].(bool) == true && stringCert == "" {
		config["verify_certs"] = false
	}

	if config["use_ssl"] != nil && config["use_ssl"].(bool) == true && stringCert != "" {
		config["verify_certs"] = true
		config["ca_certs"] = DefaultElasticCertPath
	}

	if config["use_ssl"] != nil && config["use_ssl"].(bool) == false {
		delete(config, "ca_certs")
		delete(config, "verify_certs")
	}
	if config["verify_certs"] != nil && config["verify_certs"].(bool) == false && stringCert != "" {
		delete(config, "ca_certs")
	}

	if config["use_ssl"] == nil {
		delete(config, "verify_certs")
		delete(config, "ca_certs")
	}
	if bytesConfig, err = yaml.Marshal(config); err != nil {
		return err
	}
	e.Spec.ConfigSetting["config.yaml"] = string(bytesConfig)
	return nil
}

func ConfigMapsToMap(cms []corev1.ConfigMap) map[string]corev1.ConfigMap {
	m := map[string]corev1.ConfigMap{}
	for _, d := range cms {
		m[d.Name] = d
	}
	return m
}
