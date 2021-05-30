package podspec

import (
	"elastalert/api/v1alpha1"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"testing"
)

func TestGenerateCertSecret(t *testing.T) {
	testCases := []struct {
		name       string
		elastalert v1alpha1.Elastalert
		want       v1.Secret
	}{
		{
			name: "test generate default secret",
			elastalert: v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: v1alpha1.ElastalertSpec{
					Cert: "abc",
				},
			},
			want: v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test" + DefaultCertSuffix,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "v1",
							Kind:               "Elastalert",
							Name:               "test",
							UID:                "",
							Controller:         &varTrue,
							BlockOwnerDeletion: &varTrue,
						},
					},
				},
				Data: map[string][]byte{
					DefaultElasticCertName: []byte("abc"),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := scheme.Scheme
			s.AddKnownTypes(v1.SchemeGroupVersion, &v1alpha1.Elastalert{})
			have, err := GenerateCertSecret(s, &tc.elastalert)
			require.NoError(t, err)
			require.Equal(t, tc.want, *have)
		})
	}
}
