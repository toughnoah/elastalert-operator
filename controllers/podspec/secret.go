package podspec

import (
	esv1alpha1 "elastalert/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GenerateCertSecret(e *esv1alpha1.Elastalert) *corev1.Secret {
	var data = map[string][]byte{}
	stringCert := e.Spec.Cert[DefaultElasticCertName]
	data[DefaultElasticCertName] = []byte(stringCert)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultCertName,
			Namespace: e.Namespace,
		},
		Data: data,
	}
	return secret
}
