package v1alpha1

import (
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestFreeForm_DeepCopy(t *testing.T) {
	nf := NewFreeForm(map[string]interface{}{
		"test": 1,
	})
	newnf := nf.DeepCopy()
	assert.Equal(t, *newnf, nf)
}

func TestElastalert_DeepCopy(t *testing.T) {
	ea := &Elastalert{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}
	newea := ea.DeepCopy()
	assert.Equal(t, newea, ea)
}
