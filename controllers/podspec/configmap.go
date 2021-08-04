package podspec

import (
	"errors"
	"fmt"
	esv1alpha1 "github.com/toughnoah/elastalert-operator/api/v1alpha1"
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
		rawMap, err := e.Spec.ConfigSetting.GetMap()
		out, err := yaml.Marshal(rawMap)
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

// PatchConfigSettings TODO should change to Chain Of Responsibility
func PatchConfigSettings(e *esv1alpha1.Elastalert, stringCert string) error {
	config, err := e.Spec.ConfigSetting.GetMap()
	if err != nil {
		return errors.New("get config failed")
	}
	rawConfig := &RawConfig{
		config: config,
		cert:   stringCert,
	}
	drHandler := &DefaultRulesFolderHandler{}
	useSSLHandler := &UseSSLHandler{}
	drHandler.setNext(useSSLHandler)

	addCertHandler := &AddCertHandler{}
	useSSLHandler.setNext(addCertHandler)

	verifyCertHandler := &VerifyCertHandler{}
	addCertHandler.setNext(verifyCertHandler)

	drHandler.handle(rawConfig)
	if rawConfig.err != nil {
		return rawConfig.err
	}
	e.Spec.ConfigSetting = esv1alpha1.NewFreeForm(rawConfig.config)
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

type RawConfig struct {
	config map[string]interface{}
	cert   string
	err    error
	useSSL bool
}

type handler interface {
	handle(config *RawConfig)
	setNext(handler handler)
}

type DefaultRulesFolderHandler struct {
	next handler
}

func (d *DefaultRulesFolderHandler) handle(raw *RawConfig) {
	if raw.err != nil {
		return
	}
	if raw.config == nil {
		raw.err = errors.New("get config map failed")
	} else {
		raw.config["rules_folder"] = DefaultRulesFolder
	}
	d.next.handle(raw)
}

func (d *DefaultRulesFolderHandler) setNext(next handler) {
	d.next = next
}

type UseSSLHandler struct {
	next handler
}

func (u *UseSSLHandler) handle(raw *RawConfig) {
	if raw.err != nil {
		return
	}
	if raw.config["use_ssl"] != nil {
		useSSL, ok := raw.config["use_ssl"].(bool)
		if !ok {
			raw.err = errors.New("error type for 'use_ssl', want bool")
		} else {
			raw.useSSL = useSSL
		}
	} else {
		raw.useSSL = false
	}
	u.next.handle(raw)
}
func (u *UseSSLHandler) setNext(next handler) {
	u.next = next
}

type AddCertHandler struct {
	next handler
}

func (a *AddCertHandler) handle(raw *RawConfig) {
	if raw.err != nil {
		return
	}
	if raw.useSSL {
		if raw.cert != "" {
			raw.config["verify_certs"] = true
			raw.config["ca_certs"] = DefaultElasticCertPath
		} else {
			raw.config["verify_certs"] = false
		}
	} else {
		delete(raw.config, "verify_certs")
		delete(raw.config, "ca_certs")
	}
	a.next.handle(raw)
}
func (a *AddCertHandler) setNext(next handler) {
	a.next = next
}

type VerifyCertHandler struct {
	next handler
}

func (v *VerifyCertHandler) handle(raw *RawConfig) {
	if raw.err != nil {
		return
	}
	if raw.config["verify_certs"] != nil {
		vc, ok := raw.config["verify_certs"].(bool)
		if !ok {
			raw.err = errors.New("error type for 'verify_certs', want bool")
		} else if vc == false && raw.cert != "" {
			delete(raw.config, "ca_certs")
		}
	}
}
func (v *VerifyCertHandler) setNext(next handler) {
	v.next = next
}
