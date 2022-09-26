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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EphemeralNamespaceSpec defines the desired state of EphemeralNamespace
type EphemeralNamespaceSpec struct {
	// Template is the namespace template
	Template corev1.NamespaceSpec `json:"template,omitempty"`
	// Maximum time a delployment is allowed to be running before its destroyed
	MaxLifeTime time.Duration `json:"maxLifeTime,omitempty"`
}

// EphemeralNamespaceStatus defines the observed state of EphemeralNamespace
type EphemeralNamespaceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// EphemeralNamespace is the Schema for the ephemeralnamespaces API
type EphemeralNamespace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EphemeralNamespaceSpec   `json:"spec,omitempty"`
	Status EphemeralNamespaceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EphemeralNamespaceList contains a list of EphemeralNamespace
type EphemeralNamespaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EphemeralNamespace `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EphemeralNamespace{}, &EphemeralNamespaceList{})
}
