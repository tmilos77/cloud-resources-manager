/*
Copyright 2023.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GcpVpcPeeringSpec defines the desired state of GcpVpcPeering
type GcpVpcPeeringSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of GcpVpcPeering. Edit gcpvpcpeering_types.go to remove/update
	RemoteProject string `json:"remoteProject,omitempty"`
	RemoteVpc     string `json:"remoteVpc,omitempty"`
}

// GcpVpcPeeringStatus defines the observed state of GcpVpcPeering
type GcpVpcPeeringStatus struct {
	State StatusState `json:"state,omitempty"`

	// List of status conditions to indicate the status of a Peering.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// GcpVpcPeering is the Schema for the gcpvpcpeerings API
type GcpVpcPeering struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec    GcpVpcPeeringSpec   `json:"spec,omitempty"`
	Status  GcpVpcPeeringStatus `json:"status,omitempty"`
	Outcome *Outcome            `json:"outcome,omitempty"`
}

func (peering *GcpVpcPeering) GetSpec() any {
	return peering.Spec
}

func (peering *GcpVpcPeering) GetSourceRef() SourceRef {
	return SourceRef{
		TypeMeta: metav1.TypeMeta{
			Kind:       peering.Kind,
			APIVersion: peering.APIVersion,
		},
		Name: peering.Name,
	}
}

func (peering *GcpVpcPeering) GetOutcome() *Outcome {
	return peering.Outcome
}

func (peering *GcpVpcPeering) GetConditions() *[]metav1.Condition {
	return &peering.Status.Conditions
}

func (peering *GcpVpcPeering) GetStatusState() StatusState {
	return peering.Status.State
}

func (peering *GcpVpcPeering) SetStatusState(statusState StatusState) {
	peering.Status.State = statusState
}

//+kubebuilder:object:root=true

// GcpVpcPeeringList contains a list of GcpVpcPeering
type GcpVpcPeeringList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GcpVpcPeering `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GcpVpcPeering{}, &GcpVpcPeeringList{})
}
