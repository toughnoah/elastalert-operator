package podspec

import (
	"elastalert/api/v1alpha1"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
				Spec: v1alpha1.ElastalertSpec{
					Cert: map[string]string{
						DefaultElasticCertName: "abc",
					},
				},
			},
			want: v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: DefaultCertName,
				},
				Data: map[string][]byte{
					DefaultElasticCertName: []byte("abc"),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			have := *GenerateCertSecret(&tc.elastalert)
			require.Equal(t, tc.want, have)
		})
	}
}
