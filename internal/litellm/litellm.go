package litellm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type LitellmClient struct {
	baseURL   string
	masterKey string
}

func NewLitellmClient(baseURL, masterKey string) *LitellmClient {
	return &LitellmClient{
		baseURL:   baseURL,
		masterKey: masterKey,
	}
}

func (l *LitellmClient) makeRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	log := log.FromContext(ctx)

	httpReq, err := http.NewRequest(method, l.baseURL+path, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+l.masterKey)

	defer func() {
		if closeErr := httpReq.Body.Close(); closeErr != nil {
			log.Error(closeErr, "Failed to close request body")
		}
	}()

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
		litellmError, err := processLitellmError(log, "Request failed", respBody)
		if err != nil {
			log.Error(err, "Failed to parse error response body")
			return nil, err
		}
		return nil, fmt.Errorf("litellm request failed: %s", litellmError.Message)
	}

	return respBody, nil
}

// various ways in which litellm can return an error
type litellmError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

type litellmErrorDetail struct {
	Error string `json:"error"`
}

// logLitellmError logs an errorJSON object
func logLitellmError(log logr.Logger, errorJSON litellmError, message string) {
	log.Error(errors.New(errorJSON.Message), message, "error_code", errorJSON.Code, "error_type", errorJSON.Type, "error_param", errorJSON.Param)
}

// processLitellmError processes and logs the error message from the litellm service
func processLitellmError(log logr.Logger, message string, body []byte) (litellmError, error) {
	var errorResponse struct {
		Error  *litellmError       `json:"error,omitempty"`
		Detail *litellmErrorDetail `json:"detail,omitempty"`
	}
	if err := json.Unmarshal(body, &errorResponse); err != nil {
		return litellmError{}, err
	}

	if errorResponse.Error != nil {
		logLitellmError(log, *errorResponse.Error, message)
		return *errorResponse.Error, nil
	}

	if errorResponse.Detail != nil {
		errorDetail := litellmError{
			Message: errorResponse.Detail.Error,
		}

		logLitellmError(log, errorDetail, message)
		return errorDetail, nil
	}

	log.Error(errors.New("unknown error"), string(body))
	return litellmError{}, nil
}
