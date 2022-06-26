/*
Copyright 2022.

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

package v1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KamenevMonitorSpec defines the desired state of KamenevMonitor
type KamenevMonitorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Maximum time a delployment is allowed to be running before its destroyed
	MaxLifeTime time.Duration `json:"maxLifeTime,omitempty"`
	// Maximum am amount of resources allowed to be deployed in the cluster
	MaxResources int `json:"maxResources,omitempty"`
}

// KamenevMonitorStatus defines the observed state of KamenevMonitor
type KamenevMonitorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KamenevMonitor is the Schema for the kamenevmonitors API
type KamenevMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KamenevMonitorSpec   `json:"spec,omitempty"`
	Status KamenevMonitorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KamenevMonitorList contains a list of KamenevMonitor
type KamenevMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KamenevMonitor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KamenevMonitor{}, &KamenevMonitorList{})
}
