package litellm

import (
	"context"
	"encoding/json"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type LitellmVirtualKey interface {
	GenerateVirtualKey(ctx context.Context, req *VirtualKeyRequest) (VirtualKeyResponse, error)
	DeleteVirtualKey(ctx context.Context, keyAlias string) error
	CheckVirtualKeyExists(ctx context.Context, keyAlias string) (bool, error)
}

type VirtualKeyRequest struct {
	KeyAlias       string            `json:"key_alias,omitempty"`
	UserID         string            `json:"user_id,omitempty"`
	TeamID         string            `json:"team_id,omitempty"`
	MaxBudget      string            `json:"max_budget,omitempty"`
	BudgetDuration string            `json:"budget_duration,omitempty"`
	Models         []string          `json:"models,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type VirtualKeyResponse struct {
	CreatedAt      string            `json:"created_at"`
	UpdatedAt      string            `json:"updated_at"`
	KeyAlias       string            `json:"key_alias"`
	KeyName        string            `json:"key_name"`
	UserID         string            `json:"user_id"`
	Expires        string            `json:"expires"`
	SecretKey      string            `json:"key"`
	TokenID        string            `json:"token_id"`
	MaxBudget      float64           `json:"max_budget"`
	BudgetDuration string            `json:"budget_duration"`
	Models         []string          `json:"models"`
	Metadata       map[string]string `json:"metadata"`
}

// GenerateVirtualKey generates a new virtual key for the Litellm service
func (l *LitellmClient) GenerateVirtualKey(ctx context.Context, req *VirtualKeyRequest) (VirtualKeyResponse, error) {
	log := log.FromContext(ctx)

	body, err := json.Marshal(req)
	if err != nil {
		log.Error(err, "Failed to marshal virtual key request payload")
		return VirtualKeyResponse{}, err
	}

	response, err := l.makeRequest(ctx, "POST", "/key/generate", body)
	if err != nil {
		log.Error(err, "Failed to create virtual key in Litellm")
		return VirtualKeyResponse{}, err
	}

	var virtualKeyResponse VirtualKeyResponse
	if err := json.Unmarshal(response, &virtualKeyResponse); err != nil {
		log.Error(err, "Failed to unmarshal virtual key response from Litellm")
		return VirtualKeyResponse{}, err
	}

	return virtualKeyResponse, nil
}

// DeleteVirtualKey deletes a virtual key from the Litellm service
func (l *LitellmClient) DeleteVirtualKey(ctx context.Context, keyAlias string) error {
	log := log.FromContext(ctx)

	body := []byte(`{"key_aliases": ["` + keyAlias + `"]}`)

	if _, err := l.makeRequest(ctx, "POST", "/key/delete", body); err != nil {
		log.Error(err, "Failed to delete virtual key in Litellm")
		return err
	}

	return nil
}

// CheckVirtualKeyExists checks if a virtual key already exists in the Litellm service
func (l *LitellmClient) CheckVirtualKeyExists(ctx context.Context, keyAlias string) (bool, error) {
	log := log.FromContext(ctx)

	body, err := l.makeRequest(ctx, "GET", "/key/list?key_alias="+keyAlias, nil)
	if err != nil {
		log.Error(err, "Failed to check if virtual key exists")
		return false, err
	}

	var response struct {
		Keys []struct {
			KeyAlias string `json:"key_alias"`
			KeyName  string `json:"key_name"`
		} `json:"keys"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Error(err, "Failed to unmarshal response from Litellm")
		return false, err
	}

	// Check if any key exists with the given alias
	// Since key aliases are unique, we only need to check the first key if any exists
	return len(response.Keys) > 0 && response.Keys[0].KeyAlias == keyAlias, nil
}
