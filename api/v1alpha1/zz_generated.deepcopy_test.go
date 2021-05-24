package v1alpha1

import (
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
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

func TestElastalert_DeepCopyObject(t *testing.T) {
	ea := &Elastalert{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}
	newea := ea.DeepCopyObject()
	assert.Equal(t, newea, ea)
}

func TestElastalertSpec_DeepCopy(t *testing.T) {
	ea := &ElastalertSpec{
		PodTemplateSpec: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
		},
		Rule: []FreeForm{
			NewFreeForm(map[string]interface{}{
				"test": true,
			}),
		},
	}
	newea := ea.DeepCopy()
	assert.Equal(t, newea, ea)
}

func TestElastalertStatus_DeepCopy(t *testing.T) {
	ea := &ElastalertStatus{
		Version: "v1.0",
		Phase:   "RUNNIG",
	}
	newea := ea.DeepCopy()
	assert.Equal(t, newea, ea)
}

func TestElastalertList_DeepCopy(t *testing.T) {
	ea := &ElastalertList{
		Items: []Elastalert{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
		},
	}
	newea := ea.DeepCopy()
	assert.Equal(t, newea, ea)
}

func TestFreeForm_DeepCopyInTo(t *testing.T) {
	nf := NewFreeForm(map[string]interface{}{
		"test": 1,
	})
	newnf := new(FreeForm)
	nf.DeepCopyInto(newnf)
	assert.Equal(t, *newnf, nf)
}
func TestElastalert_DeepCopyInTo(t *testing.T) {
	ea := &Elastalert{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Status: ElastalertStatus{
			Condictions: []metav1.Condition{
				{
					Type: "test",
				},
			},
		},
	}
	newea := new(Elastalert)
	ea.DeepCopyInto(newea)
	assert.Equal(t, newea, ea)
}
