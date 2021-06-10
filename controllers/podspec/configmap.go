package podspec

import (
	esv1alpha1 "elastalert/api/v1alpha1"
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func GenerateNewConfigmap(Scheme *runtime.Scheme, e *esv1alpha1.Elastalert, suffix string) (*corev1.ConfigMap, error) {
	var data = make(map[string]string)
	var err error
	switch suffix {
	case esv1alpha1.RuleSuffx:
		data, err = GenerateYamlMap(e.Spec.Rule)
		if err != nil {
			log.Error(
				err,
				"Failed to generate rules configmaps",
				"Elastalert.Namespace", e.Namespace,
				"Configmaps.Namespace", e.Namespace,
			)
			return nil, err
		}
	case esv1alpha1.ConfigSuffx:
		rawmap, err := e.Spec.ConfigSetting.GetMap()
		out, err := yaml.Marshal(rawmap)
		if err != nil {
			log.Error(
				err,
				"Failed to generate config.yaml configmaps",
				"Elastalert.Namespace", e.Namespace,
				"Configmaps.Namespace", e.Namespace,
			)
			return nil, err
		}
		data["config.yaml"] = string(out)
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Name + suffix,
			Namespace: e.Namespace,
		},
		Data: data,
	}
	err = ctrl.SetControllerReference(e, cm, Scheme)
	if err != nil {
		log.Error(
			err,
			"Failed to generate configmaps",
			"Elastalert.Namespace", e.Namespace,
			"Configmaps.Namespace", e.Namespace,
		)
		return nil, err
	}
	return cm, nil
}

func PatchConfigSettings(e *esv1alpha1.Elastalert, stringCert string) error {
	config, err := e.Spec.ConfigSetting.GetMap()
	if config == nil || err != nil {
		return errors.New("get config failed")
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
	e.Spec.ConfigSetting = esv1alpha1.NewFreeForm(config)
	return nil
}

func ConfigMapsToMap(cms []corev1.ConfigMap) map[string]corev1.ConfigMap {
	m := map[string]corev1.ConfigMap{}
	for _, d := range cms {
		m[d.Name] = d
	}
	return m
}

func GenerateYamlMap(ruleArray []esv1alpha1.FreeForm) (map[string]string, error) {
	var data = map[string]string{}
	for _, v := range ruleArray {
		m, err := v.GetMap()
		if err != nil {
			return nil, err
		}
		key := fmt.Sprintf("%s.yaml", m["name"])
		out, err := yaml.Marshal(m)
		if err != nil {
			return nil, err
		}
		data[key] = string(out)

	}
	return data, nil
}

func PatchAlertSettings(e *esv1alpha1.Elastalert) error {
	var ruleArray []esv1alpha1.FreeForm
	alert, err := e.Spec.Alert.GetMap()
	if err != nil {
		return err
	}
	if alert == nil {
		return nil
	}
	for _, v := range e.Spec.Rule {
		rule, err := v.GetMap()
		if err != nil {
			return err
		}
		if rule["alert"] == nil {
			MergeInterfaceMap(rule, alert)
		}
		ruleArray = append(ruleArray, esv1alpha1.NewFreeForm(rule))
	}
	e.Spec.Rule = ruleArray

	return nil
}
