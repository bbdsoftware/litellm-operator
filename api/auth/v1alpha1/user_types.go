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

// UserSpec defines the desired state of User
type UserSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// ConnectionRef defines how to connect to the LiteLLM instance
	// +kubebuilder:validation:Required
	ConnectionRef ConnectionRef `json:"connectionRef"`

	// Aliases is the model aliases for the user
	Aliases map[string]string `json:"aliases,omitempty"`
	// AllowedCacheControls is the list of allowed cache control values
	AllowedCacheControls []string `json:"allowedCacheControls,omitempty"`
	// AutoCreateKey is whether to automatically create a key for the user
	AutoCreateKey bool `json:"autoCreateKey,omitempty"`
	// Blocked is whether the user is blocked
	Blocked bool `json:"blocked,omitempty"`
	// BudgetDuration - Budget is reset at the end of specified duration. If not set, budget is never reset. You can set duration as seconds ("30s"), minutes ("30m"), hours ("30h"), days ("30d"), months ("1mo").
	BudgetDuration string `json:"budgetDuration,omitempty"`
	// Duration is the duration for the key auto-created on /user/new
	Duration string `json:"duration,omitempty"`
	// Guardrails is the list of active guardrails for the user
	Guardrails []string `json:"guardrails,omitempty"`
	// KeyAlias is the optional alias of the key if autoCreateKey is true
	KeyAlias string `json:"keyAlias,omitempty"`
	// MaxBudget is the maximum budget for the user
	MaxBudget string `json:"maxBudget,omitempty"`
	// MaxParallelRequests is the maximum number of parallel requests for the user
	MaxParallelRequests int `json:"maxParallelRequests,omitempty"`
	// Metadata is the metadata of the user
	Metadata map[string]string `json:"metadata,omitempty"`
	// ModelMaxBudget is the model specific maximum budget
	ModelMaxBudget map[string]string `json:"modelMaxBudget,omitempty"`
	// ModelRPMLimit is the model specific maximum requests per minute
	ModelRPMLimit map[string]string `json:"modelRPMLimit,omitempty"`
	// ModelTPMLimit is the model specific maximum tokens per minute
	ModelTPMLimit map[string]string `json:"modelTPMLimit,omitempty"`
	// Models is the list of models that the user is allowed to use
	Models []string `json:"models,omitempty"`
	// Permissions is the user-specific permissions
	Permissions map[string]string `json:"permissions,omitempty"`
	// RPMLimit is the maximum requests per minute for the user
	RPMLimit int `json:"rpmLimit,omitempty"`
	// SendInviteEmail is whether to send an invite email to the user - NOTE: the user endpoint will return an error if email alerting is not configured and this is enabled, but the user will still be created.
	SendInviteEmail bool `json:"sendInviteEmail,omitempty"`
	// SoftBudget - alert when user exceeds this budget, doesn't block requests
	SoftBudget string `json:"softBudget,omitempty"`
	// Spend is the amount spent by user
	Spend string `json:"spend,omitempty"`
	// SSOUserID is the id of the user in the SSO provider
	SSOUserID string `json:"ssoUserID,omitempty"`
	// Teams is the list of teams that the user is a member of
	Teams []string `json:"teams,omitempty"`
	// TPMLimit is the maximum tokens per minute for the user
	TPMLimit int `json:"tpmLimit,omitempty"`
	// UserAlias is the alias of the user
	UserAlias string `json:"userAlias,omitempty"`
	// UserEmail is the email of the user
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="UserEmail is immutable"
	UserEmail string `json:"userEmail,omitempty"`
	// UserID is the ID of the user. If not set, a unique ID will be generated.
	UserID string `json:"userID,omitempty"`
	// UserRole is the role of the user - one of "proxy_admin", "proxy_admin_viewer", "internal_user", "internal_user_viewer"
	// +kubebuilder:validation:Enum=proxy_admin;proxy_admin_viewer;internal_user;internal_user_viewer
	UserRole string `json:"userRole,omitempty"`
}

// UserStatus defines the observed state of User
type UserStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// AllowedCacheControls is the list of allowed cache control values
	AllowedCacheControls []string `json:"allowedCacheControls,omitempty"`
	// AllowedRoutes is the list of allowed routes
	AllowedRoutes []string `json:"allowedRoutes,omitempty"`
	// Aliases is the model aliases for the user
	Aliases map[string]string `json:"aliases,omitempty"`
	// Blocked is whether the user is blocked
	Blocked bool `json:"blocked,omitempty"`
	// BudgetDuration - Budget is reset at the end of specified duration
	BudgetDuration string `json:"budgetDuration,omitempty"`
	// BudgetID is the ID of the budget
	BudgetID string `json:"budgetID,omitempty"`
	// Config is the user-specific config
	Config map[string]string `json:"config,omitempty"`
	// CreatedAt is the date and time when the user was created
	CreatedAt string `json:"createdAt,omitempty"`
	// CreatedBy is the user who created this user
	CreatedBy string `json:"createdBy,omitempty"`
	// Duration is the duration for the key
	Duration string `json:"duration,omitempty"`
	// EnforcedParams is the list of enforced parameters
	EnforcedParams []string `json:"enforcedParams,omitempty"`
	// Expires is the date and time when the user will expire
	Expires string `json:"expires,omitempty"`
	// Guardrails is the list of active guardrails
	Guardrails []string `json:"guardrails,omitempty"`
	// KeyAlias is the alias of the key
	KeyAlias string `json:"keyAlias,omitempty"`
	// KeyName is the name of the key
	KeyName string `json:"keyName,omitempty"`
	// KeySecretRef is the reference to the secret containing the user key
	KeySecretRef string `json:"keySecretRef,omitempty"`
	// LiteLLMBudgetTable is the budget table name
	LiteLLMBudgetTable string `json:"litellmBudgetTable,omitempty"`
	// MaxBudget is the maximum budget for the user
	MaxBudget string `json:"maxBudget,omitempty"`
	// MaxParallelRequests is the maximum number of parallel requests
	MaxParallelRequests int `json:"maxParallelRequests,omitempty"`
	// ModelMaxBudget is the model specific maximum budget
	ModelMaxBudget map[string]string `json:"modelMaxBudget,omitempty"`
	// ModelRPMLimit is the model specific maximum requests per minute
	ModelRPMLimit map[string]string `json:"modelRPMLimit,omitempty"`
	// ModelTPMLimit is the model specific maximum tokens per minute
	ModelTPMLimit map[string]string `json:"modelTPMLimit,omitempty"`
	// Models is the list of models that the user is allowed to use
	Models []string `json:"models,omitempty"`
	// Permissions is the user-specific permissions
	Permissions map[string]string `json:"permissions,omitempty"`
	// RPMLimit is the maximum requests per minute
	RPMLimit int `json:"rpmLimit,omitempty"`
	// Spend is the amount spent by user
	Spend string `json:"spend,omitempty"`
	// Tags for tracking spend and/or doing tag-based routing. Requires Enterprise license
	Tags []string `json:"tags,omitempty"`
	// Teams is the list of teams that the user is a member of
	Teams []string `json:"teams,omitempty"`
	// Token is the user's token
	Token string `json:"token,omitempty"`
	// TPMLimit is the maximum tokens per minute
	TPMLimit int `json:"tpmLimit,omitempty"`
	// UpdatedAt is the date and time when the user was last updated
	UpdatedAt string `json:"updatedAt,omitempty"`
	// UpdatedBy is the user who last updated this user
	UpdatedBy string `json:"updatedBy,omitempty"`
	// UserAlias is the alias of the user
	UserAlias string `json:"userAlias,omitempty"`
	// UserEmail is the email of the user
	UserEmail string `json:"userEmail,omitempty"`
	// UserID is the unique user id
	UserID string `json:"userID,omitempty"`
	// UserRole is the role of the user
	UserRole string `json:"userRole,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="The ready status of the user"
// +kubebuilder:printcolumn:name="Email",type="string",JSONPath=".spec.userEmail",description="The user's email address"
// +kubebuilder:printcolumn:name="Role",type="string",JSONPath=".spec.userRole",description="The user's role"
// +kubebuilder:printcolumn:name="Blocked",type="boolean",JSONPath=".spec.blocked",description="Whether the user is blocked"
// +kubebuilder:printcolumn:name="Budget",type="string",JSONPath=".spec.maxBudget",description="Maximum budget for the user"
// +kubebuilder:printcolumn:name="Spend",type="string",JSONPath=".status.spend",description="Current user spend"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time since creation"

// User is the Schema for the users API
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec,omitempty"`
	Status UserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserList contains a list of User
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

// GetConditions returns the conditions slice
func (u *User) GetConditions() []metav1.Condition {
	return u.Status.Conditions
}

// SetConditions sets the conditions slice
func (u *User) SetConditions(conditions []metav1.Condition) {
	u.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
