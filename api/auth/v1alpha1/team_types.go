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
	"github.com/bbdsoftware/litellm-operator/internal/interfaces"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type TeamMemberWithRole struct {
	// UserID is the ID of the user
	UserID string `json:"userID,omitempty"`
	// UserEmail is the email of the user
	UserEmail string `json:"userEmail,omitempty"`
	// Role is the role of the user - one of "admin" or "user"
	Role string `json:"role,omitempty"`
}

// ConnectionRef defines how to connect to a LiteLLM instance
type ConnectionRef struct {
	// SecretRef references a secret containing connection details
	SecretRef *SecretRef `json:"secretRef,omitempty"`

	// InstanceRef references a LiteLLM instance
	InstanceRef *InstanceRef `json:"instanceRef,omitempty"`
}

// SecretRef references a secret containing connection details
type SecretRef struct {
	// Name is the name of the secret
	Name string `json:"name"`

	// Keys defines the keys in the secret that contain connection details
	Keys SecretKeys `json:"keys"`
}

// SecretKeys defines the keys in a secret for connection details
type SecretKeys struct {
	// MasterKey is the key in the secret containing the master key
	MasterKey string `json:"masterKey"`

	// URL is the key in the secret containing the LiteLLM URL
	URL string `json:"url"`
}

// GetMasterKey returns the master key field name
func (s SecretKeys) GetMasterKey() string {
	return s.MasterKey
}

// GetURL returns the URL field name
func (s SecretKeys) GetURL() string {
	return s.URL
}

// InstanceRef references a LiteLLM instance
type InstanceRef struct {
	// Name is the name of the LiteLLM instance
	Name string `json:"name"`

	// Namespace is the namespace of the LiteLLM instance (defaults to the same namespace as the Team)
	Namespace string `json:"namespace,omitempty"`
}

// GetSecretRef returns the SecretRef if it exists
func (c ConnectionRef) GetSecretRef() interface{} {
	return c.SecretRef
}

// GetInstanceRef returns the InstanceRef if it exists
func (c ConnectionRef) GetInstanceRef() interface{} {
	return c.InstanceRef
}

// HasSecretRef checks if SecretRef is specified
func (c ConnectionRef) HasSecretRef() bool {
	return c.SecretRef != nil
}

// HasInstanceRef checks if InstanceRef is specified
func (c ConnectionRef) HasInstanceRef() bool {
	return c.InstanceRef != nil
}

// GetSecretName returns the secret name
func (s *SecretRef) GetSecretName() string {
	return s.Name
}

// GetNamespace returns the namespace (empty for auth SecretRef)
func (s *SecretRef) GetNamespace() string {
	return ""
}

// GetKeys returns the keys structure
func (s *SecretRef) GetKeys() interfaces.KeysInterface {
	return s.Keys
}

// HasKeys returns true since auth SecretRef always has keys
func (s *SecretRef) HasKeys() bool {
	return true
}

// GetInstanceName returns the instance name
func (i *InstanceRef) GetInstanceName() string {
	return i.Name
}

// GetNamespace returns the namespace
func (i *InstanceRef) GetNamespace() string {
	return i.Namespace
}

// TeamSpec defines the desired state of Team
type TeamSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ConnectionRef defines how to connect to the LiteLLM instance
	// +kubebuilder:validation:Required
	ConnectionRef ConnectionRef `json:"connectionRef"`

	// Blocked is a flag indicating if the team is blocked or not - will stop all calls from keys with this team_id
	Blocked bool `json:"blocked,omitempty"`
	// BudgetDuration - Budget is reset at the end of specified duration. If not set, budget is never reset. You can set duration as seconds ("30s"), minutes ("30m"), hours ("30h"), days ("30d"), months ("1mo").
	BudgetDuration string `json:"budgetDuration,omitempty"`
	// Guardrails are guardrails for the team
	Guardrails []string `json:"guardrails,omitempty"`
	// MaxBudget is the maximum budget for the team
	MaxBudget string `json:"maxBudget,omitempty"`
	// Metadata is the metadata of the team
	Metadata map[string]string `json:"metadata,omitempty"`
	// ModelAliases are model aliases for the team
	ModelAliases map[string]string `json:"modelAliases,omitempty"`
	// Models is the list of models that are associated with the team. All keys for this team_id will have at most, these models. If empty, assumes all models are allowed.
	Models []string `json:"models,omitempty"`
	// OrganizationID is the ID of the organization that the team belongs to. If not set, the team will be created with no organization.
	OrganizationID string `json:"organizationID,omitempty"`
	// RPMLimit is the maximum requests per minute limit for the team - all keys associated with this team_id will have at max this RPM limit
	RPMLimit int `json:"rpmLimit,omitempty"`
	// Tags for tracking spend and/or doing tag-based routing. Requires Enterprise license
	Tags []string `json:"tags,omitempty"`
	// TeamAlias is the alias of the team
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="TeamAlias is immutable"
	TeamAlias string `json:"teamAlias,omitempty"`
	// TeamID is the ID of the team. If not set, a unique ID will be generated.
	TeamID string `json:"teamID,omitempty"`
	// TeamMemberPermissions is the list of routes that non-admin team members can access. Example: ["/key/generate", "/key/update", "/key/delete"]
	TeamMemberPermissions []string `json:"teamMemberPermissions,omitempty"`
	// TPMLimit is the maximum tokens per minute limit for the team - all keys with this team_id will have at max this TPM limit
	TPMLimit int `json:"tpmLimit,omitempty"`
}

// TeamStatus defines the observed state of Team
type TeamStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Blocked is a flag indicating if the team is blocked or not
	Blocked bool `json:"blocked,omitempty"`
	// BudgetDuration - Budget is reset at the end of specified duration. If not set, budget is never reset.
	BudgetDuration string `json:"budgetDuration,omitempty"`
	// BudgetResetAt is the date and time when the budget will be reset
	BudgetResetAt string `json:"budgetResetAt,omitempty"`
	// CreatedAt is the date and time when the team was created
	CreatedAt string `json:"createdAt,omitempty"`
	// LiteLLMModelTable is the model table for the team
	LiteLLMModelTable string `json:"liteLLMModelTable,omitempty"`
	// MaxBudget is the maximum budget for the team
	MaxBudget string `json:"maxBudget,omitempty"`
	// MaxParallelRequests is the maximum number of parallel requests allowed
	MaxParallelRequests int `json:"maxParallelRequests,omitempty"`
	// MembersWithRole is the list of members with role
	MembersWithRole []TeamMemberWithRole `json:"membersWithRole,omitempty"`
	// ModelID is the ID of the model
	ModelID string `json:"modelID,omitempty"`
	// Models is the list of models that are associated with the team. All keys for this team_id will have at most, these models.
	Models []string `json:"models,omitempty"`
	// OrganizationID is the ID of the organization that the team belongs to
	OrganizationID string `json:"organizationID,omitempty"`
	// RPMLimit is the maximum requests per minute limit for the team - all keys associated with this team_id will have at max this RPM limit
	RPMLimit int `json:"rpmLimit,omitempty"`
	// Spend is the current spend of the team
	Spend string `json:"spend,omitempty"`
	// Tags for tracking spend and/or doing tag-based routing. Requires Enterprise license
	Tags []string `json:"tags,omitempty"`
	// TeamAlias is the alias of the team
	TeamAlias string `json:"teamAlias,omitempty"`
	// TeamID is the ID of the team
	TeamID string `json:"teamID,omitempty"`
	// TeamMemberPermissions is the list of routes that non-admin team members can access
	TeamMemberPermissions []string `json:"teamMemberPermissions,omitempty"`
	// TPMLimit is the maximum tokens per minute limit for the team - all keys with this team_id will have at max this TPM limit
	TPMLimit int `json:"tpmLimit,omitempty"`
	// UpdatedAt is the date and time when the team was last updated
	UpdatedAt string `json:"updatedAt,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="The ready status of the team"
// +kubebuilder:printcolumn:name="Alias",type="string",JSONPath=".spec.teamAlias",description="The team alias"
// +kubebuilder:printcolumn:name="Organisation",type="string",JSONPath=".spec.organizationID",description="The organisation ID"
// +kubebuilder:printcolumn:name="Blocked",type="boolean",JSONPath=".spec.blocked",description="Whether the team is blocked"
// +kubebuilder:printcolumn:name="Members",type="integer",JSONPath=".status.membersWithRole[*]",description="Number of team members"
// +kubebuilder:printcolumn:name="Budget",type="string",JSONPath=".spec.maxBudget",description="Maximum budget for the team"
// +kubebuilder:printcolumn:name="Spend",type="string",JSONPath=".status.spend",description="Current team spend"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time since creation"

// Team is the Schema for the teams API
type Team struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamSpec   `json:"spec,omitempty"`
	Status TeamStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TeamList contains a list of Team
type TeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Team `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Team{}, &TeamList{})
}
