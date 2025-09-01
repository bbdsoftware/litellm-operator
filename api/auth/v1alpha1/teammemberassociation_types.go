/*
Copyright 2025.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TeamMemberAssociationSpec defines the desired state of TeamMemberAssociation
type TeamMemberAssociationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ConnectionRef defines how to connect to the LiteLLM instance
	// +kubebuilder:validation:Required
	ConnectionRef ConnectionRef `json:"connectionRef"`

	// MaxBudgetInTeam is the maximum budget for the user in the team
	MaxBudgetInTeam string `json:"maxBudgetInTeam,omitempty"`
	// TeamID is the ID of the team
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="TeamAlias is immutable"
	TeamAlias string `json:"teamAlias,omitempty"`
	// UserEmail is the email of the user
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="UserEmail is immutable"
	UserEmail string `json:"userEmail,omitempty"`
	// Role is the role of the user - one of "admin" or "user"
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=admin;user
	Role string `json:"role,omitempty"`
}

// TeamMemberAssociationStatus defines the observed state of TeamMemberAssociation
type TeamMemberAssociationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// TeamAlias is the alias of the team
	TeamAlias string `json:"teamAlias,omitempty"`
	// TeamID is the ID of the team
	TeamID string `json:"teamID,omitempty"`
	// UserEmail is the email of the user
	UserEmail string `json:"userEmail,omitempty"`
	// UserID is the ID of the user
	UserID string `json:"userID,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="The ready status of the association"
// +kubebuilder:printcolumn:name="Team",type="string",JSONPath=".spec.teamAlias",description="The team alias"
// +kubebuilder:printcolumn:name="User",type="string",JSONPath=".spec.userEmail",description="The user's email address"
// +kubebuilder:printcolumn:name="Role",type="string",JSONPath=".spec.role",description="The user's role in the team"
// +kubebuilder:printcolumn:name="Budget",type="string",JSONPath=".spec.maxBudgetInTeam",description="Maximum budget for user in team"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time since creation"

// TeamMemberAssociation is the Schema for the teammemberassociations API
type TeamMemberAssociation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamMemberAssociationSpec   `json:"spec,omitempty"`
	Status TeamMemberAssociationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TeamMemberAssociationList contains a list of TeamMemberAssociation
type TeamMemberAssociationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TeamMemberAssociation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TeamMemberAssociation{}, &TeamMemberAssociationList{})
}
