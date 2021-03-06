package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFreeForm(t *testing.T) {
	uiconfig := `{"es":{"password":"changeme","server-urls":"http://elasticsearch:9200","username":"elastic"}}`
	o := NewFreeForm(map[string]interface{}{
		"es": map[string]interface{}{
			"server-urls": "http://elasticsearch:9200",
			"username":    "elastic",
			"password":    "changeme",
		},
	})
	json, err := o.MarshalJSON()
	assert.NoError(t, err)
	assert.NotNil(t, json)
	assert.Equal(t, uiconfig, string(*o.json))
}

func TestFreeFormUnmarshalMarshal(t *testing.T) {
	uiconfig := `{"es":{"password":"changeme","server-urls":"http://elasticsearch:9200","username":"elastic"}}`
	o := NewFreeForm(nil)
	_ = o.UnmarshalJSON([]byte(uiconfig))
	json, err := o.MarshalJSON()
	assert.NoError(t, err)
	assert.NotNil(t, json)
	assert.Equal(t, uiconfig, string(*o.json))
}

func TestFreeFormListUnmarshalMarshal(t *testing.T) {
	testconfig := `[{"es":{"password":"changeme","server-urls":"http://elasticsearch:9200","username":"elastic"}},{"es2":{"password":"changeme","server-urls":"http://elasticsearch:9200","username":"elastic"}}]`
	o := NewFreeForm(nil)
	_ = o.UnmarshalJSON([]byte(testconfig))
	json, err := o.MarshalJSON()
	assert.NoError(t, err)
	assert.NotNil(t, json)
	assert.Equal(t, testconfig, string(*o.json))
}

func TestFreeFormNilUnmarshalMarshal(t *testing.T) {
	testconfig := ``
	o := NewFreeForm(nil)
	_ = o.UnmarshalJSON([]byte(testconfig))
	json, err := o.MarshalJSON()
	assert.NoError(t, err)
	assert.NotNil(t, json)
	assert.Equal(t, testconfig, string(*o.json))
}

func TestFreeFormIsEmptyFalse(t *testing.T) {
	o := NewFreeForm(map[string]interface{}{
		"es": map[string]interface{}{
			"server-urls": "http://elasticsearch:9200",
			"username":    "elastic",
			"password":    "changeme",
		},
	})
	assert.False(t, o.IsEmpty())
}

func TestFreeFormIsEmptyTrue(t *testing.T) {
	o := NewFreeForm(map[string]interface{}{})
	assert.True(t, o.IsEmpty())
}

func TestFreeFormIsEmptyNilTrue(t *testing.T) {
	o := NewFreeForm(nil)
	assert.True(t, o.IsEmpty())
}

func TestToMap(t *testing.T) {
	tests := []struct {
		m        map[string]interface{}
		expected map[string]interface{}
		err      string
	}{
		{expected: map[string]interface{}{}},
		{m: map[string]interface{}{"foo": "bar$"}, expected: map[string]interface{}{"foo": "bar$"}},
		{m: map[string]interface{}{"foo": true}, expected: map[string]interface{}{"foo": true}},
	}
	for _, test := range tests {
		f := NewFreeForm(test.m)
		got, err := f.GetMap()
		if test.err != "" {
			assert.EqualError(t, err, test.err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, test.expected, got)
		}
	}
}

func TestFreeForm_GetStringMap(t *testing.T) {
	tests := []struct {
		m        map[string]interface{}
		expected map[string]string
		err      string
	}{
		{expected: map[string]string{}},
		{m: map[string]interface{}{"foo": "bar$"}, expected: map[string]string{"foo": "bar$"}},
		{m: map[string]interface{}{"foo": "true"}, expected: map[string]string{"foo": "true"}},
	}
	for _, test := range tests {
		f := NewFreeForm(test.m)
		got, err := f.GetStringMap()
		if test.err != "" {
			assert.EqualError(t, err, test.err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, test.expected, got)
		}
	}
}
