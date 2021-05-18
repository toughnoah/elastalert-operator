package podspec

import (
	esv1alpha1 "elastalert/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func GenerateCertSecret(Scheme *runtime.Scheme, e *esv1alpha1.Elastalert) (*corev1.Secret, error) {
	se := BuildCertSecret(e)
	if err := ctrl.SetControllerReference(e, se, Scheme); err != nil {
		return nil, err
	}
	return se, nil
}

func BuildCertSecret(e *esv1alpha1.Elastalert) *corev1.Secret {
	var data = map[string][]byte{}
	stringCert := e.Spec.Cert
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
