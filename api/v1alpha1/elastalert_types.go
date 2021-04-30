/*

Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// +k8s:openapi-gen=true
	ElastAlertPhraseFailed = "FAILED"
	// +k8s:openapi-gen=true
	ElastAlertPhraseSucceeded = "RUNNING"

	ElastAlertAvailableReason = "NewElastAlertAvailable"

	ElastAlertAvailableType = "Progressing"

	ElastAlertAvailableStatus = "True"

	ElastAlertUnAvailableReason = "ElastAlertUnAvailable"

	ElastAlertUnAvailableType = "Stopped"

	ElastAlertUnAvailableStatus = "False"

	ActionSuccess = "success"

	ActionFailed = "failed"

	ElastAlertVersion = "v1.0"

	ConfigSuffx = "-config"

	RuleSuffx = "-rule"

	RuleMountPath = "/etc/elastalert/rules"

	ConfigMountPath = "/etc/elastalert"

	SuccessMessage = "Have a nice day!"

	FailedMessage = "Faild to apply resources"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ElastalertSpec defines the desired state of Elastalert
type ElastalertSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Elastalert rules as yaml string
	Rule               map[string]string `json:"rule,omitempty"`
	v1.PodTemplateSpec `json:",inline"`
	ConfigSetting      map[string]string `json:"config,omitempty"`
	Image              string            `json:"image,omitempty"`
	Cert               map[string]string `json:"cert,omitempty"`
}

// ElastalertStatus defines the observed state of Elastalert
type ElastalertStatus struct {
	Version     string             `json:"version,omitempty"`
	Phase       string             `json:"phase,omitempty"`
	Condictions []metav1.Condition `json:"conditions"`
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +k8s:openapi-gen=true
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="Elastalert"
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="Elastalert instance's status"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="Elastalert Version"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Elastalert is the Schema for the elastalerts API
type Elastalert struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElastalertSpec   `json:"spec,omitempty"`
	Status ElastalertStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ElastalertList contains a list of Elastalert
type ElastalertList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Elastalert `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Elastalert{}, &ElastalertList{})
}
