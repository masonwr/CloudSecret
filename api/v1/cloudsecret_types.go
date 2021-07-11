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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CloudSecretSpec defines the desired state of CloudSecret
type CloudSecretSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Data key secret path mapping
	Data map[string]string `json:"data,omitempty"`

	// SyncPeriod defines in seconds the delay before
	// the secret is again reconciled, in essense the
	// polling interval.
	SyncPeriod uint64 `json:"syncPeriod,omitempty"`
}

// CloudSecretStatus defines the observed state of CloudSecret
type CloudSecretStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// SecretResolution map the state of each secret, either RESOLVED or FAILED
	SecretResolution map[string]string `json:"secretResolution,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CloudSecret is the Schema for the cloudsecrets API
type CloudSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudSecretSpec   `json:"spec,omitempty"`
	Status CloudSecretStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CloudSecretList contains a list of CloudSecret
type CloudSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudSecret{}, &CloudSecretList{})
}

// GetChildSecretKey returns a key suitable for searching for a
// child secret
func (c *CloudSecret) GetChildSecretKey() types.NamespacedName {
	return types.NamespacedName{
		Name:      c.GetName(),
		Namespace: c.GetNamespace(),
	}
}

// InitChildSecret initializes a new k8s secret with *this*
// cloud secret set as its owner
func (c *CloudSecret) InitChildSecret() corev1.Secret {
	var secret corev1.Secret

	secret.SetName(c.GetName())
	secret.SetNamespace(c.GetNamespace())
	secret.SetOwnerReferences([]metav1.OwnerReference{
		metav1.OwnerReference{
			APIVersion: c.APIVersion,
			Kind:       c.Kind,
			Name:       c.Name,
			UID:        c.UID,
		},
	})

	return secret
}
