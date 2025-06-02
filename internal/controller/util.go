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

package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// finalizerName is the name of the finalizer used by the litellm operator
const finalizerName = "litellm-operator.litellm.ai/finalizer"

// litellmBaseURL is the base URL for the litellm service
var litellmBaseURL = os.Getenv("LITELLM_BASE_URL")

// litellmMasterKey is the master key for authenticating with the litellm service
var litellmMasterKey = os.Getenv("LITELLM_MASTER_KEY")

type errorJSON struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

// makeLitellmRequest handles making HTTP requests to the LiteLLM service
func makeLitellmRequest(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	log := log.FromContext(ctx)

	httpReq, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+litellmMasterKey)

	defer httpReq.Body.Close()

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		log.Error(err, "Failed to send request to Litellm")
		return nil, err
	}

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		log.Error(err, "Failed to read response body")
		return nil, err
	}

	if httpResp.StatusCode != 200 {
		errorJSON, err := processLitellmError(log, "Request failed", respBody)
		if err != nil {
			log.Error(err, "Failed to parse error response body")
			return nil, err
		}
		return nil, fmt.Errorf("litellm request failed: %s", errorJSON.Message)
	}

	return respBody, nil
}

// logLitellmError logs an errorJSON object
func logLitellmError(log logr.Logger, errorJSON errorJSON, message string) {
	log.Error(errors.New(errorJSON.Message), message, "error_code", errorJSON.Code, "error_type", errorJSON.Type, "error_param", errorJSON.Param)
}

// processLitellmError processes and logs the error message from the litellm service
func processLitellmError(log logr.Logger, message string, body []byte) (errorJSON, error) {
	var errorResponse struct {
		Error errorJSON `json:"error"`
	}
	if err := json.Unmarshal(body, &errorResponse); err != nil {
		return errorJSON{}, err
	}
	logLitellmError(log, errorResponse.Error, message)
	return errorResponse.Error, nil
}

// ensureMetadata ensures that the metadata contains the managed_by metadata
func ensureMetadata(metadata map[string]string) map[string]string {
	operatorMetadata := map[string]string{
		"managed_by": "litellm-operator",
	}
	for k, v := range metadata {
		operatorMetadata[k] = v
	}
	return operatorMetadata
}
