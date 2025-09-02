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

// ErrNotFound is returned when the LiteLLM service responds with a 404
var ErrNotFound = errors.New("litellm: resource not found")

func NewLitellmClient(baseURL, masterKey string) *LitellmClient {
	return &LitellmClient{
		baseURL:   baseURL,
		masterKey: masterKey,
	}
}

func (l *LitellmClient) TestConnection(ctx context.Context) error {
	_, err := l.makeRequest(ctx, "GET", "/", nil)
	if err != nil {
		return err
	}
	return nil
}

func (l *LitellmClient) makeRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	log := log.FromContext(ctx)

	// Helper function to pretty print JSON if valid
	prettyPrintJSON := func(data []byte) string {
		if len(data) == 0 {
			return ""
		}
		var jsonObj interface{}
		if err := json.Unmarshal(data, &jsonObj); err != nil {
			// Not valid JSON, return as string
			return string(data)
		}
		prettyJSON, err := json.MarshalIndent(jsonObj, "", "  ")
		if err != nil {
			// Fallback to original string if pretty printing fails
			return string(data)
		}
		return string(prettyJSON)
	}

	// Debug logging for request details
	log.V(1).Info("Making request to Litellm",
		"method", method,
		"path", path,
		"baseURL", l.baseURL,
		"fullURL", l.baseURL+path,
		"bodySize", len(body),
	)

	httpReq, err := http.NewRequest(method, l.baseURL+path, bytes.NewBuffer(body))
	if err != nil {
		log.Error(err, "Failed to create request", "body", prettyPrintJSON(body))
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+l.masterKey)

	// Debug logging for request headers
	log.V(1).Info("Request headers",
		"content-type", httpReq.Header.Get("Content-Type"),
		"authorization", "Bearer [REDACTED]",
		"user-agent", httpReq.Header.Get("User-Agent"),
	)

	defer func() {
		if closeErr := httpReq.Body.Close(); closeErr != nil {
			log.Error(closeErr, "Failed to close request body")
		}
	}()

	log.V(1).Info("Sending request to Litellm")
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		log.Error(err, "Failed to send request to Litellm", "body", prettyPrintJSON(body))
		return nil, err
	}

	// Debug logging for response details
	log.V(1).Info("Received response from Litellm",
		"statusCode", httpResp.StatusCode,
		"status", httpResp.Status,
		"contentLength", httpResp.ContentLength,
	)

	// Debug logging for response headers
	log.V(1).Info("Response headers",
		"content-type", httpResp.Header.Get("Content-Type"),
		"content-length", httpResp.Header.Get("Content-Length"),
	)

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		log.Error(err, "Failed to read response body")
		return nil, err
	}

	// Debug log response body
	log.V(1).Info("Response body", "body", prettyPrintJSON(respBody))

	// Handle different HTTP status codes
	switch httpResp.StatusCode {
	case 200:
		log.V(1).Info("Request completed successfully")
		return respBody, nil
	case 400:
		log.V(1).Info("Bad request (400)", "statusCode", httpResp.StatusCode)
		litellmError, err := processLitellmError(log, "Bad request", respBody)
		if err != nil {
			log.Error(err, "Failed to parse error response body")
			return nil, fmt.Errorf("bad request: invalid request parameters")
		}
		return nil, fmt.Errorf("bad request: %s", litellmError.Message)
	case 401:
		log.V(1).Info("Unauthorized (401)", "statusCode", httpResp.StatusCode)
		return nil, fmt.Errorf("unauthorized: invalid or missing authentication credentials")
	case 403:
		log.V(1).Info("Forbidden (403)", "statusCode", httpResp.StatusCode)
		litellmError, err := processLitellmError(log, "Forbidden", respBody)
		if err != nil {
			log.Error(err, "Failed to parse error response body")
			return nil, fmt.Errorf("forbidden: insufficient permissions")
		}
		return nil, fmt.Errorf("forbidden: %s", litellmError.Message)
	case 404:
		log.V(1).Info("Not found (404)", "statusCode", httpResp.StatusCode)
		return nil, fmt.Errorf("%w: the requested resource does not exist", ErrNotFound)
	case 409:
		log.V(1).Info("Conflict (409)", "statusCode", httpResp.StatusCode)
		litellmError, err := processLitellmError(log, "Conflict", respBody)
		if err != nil {
			log.Error(err, "Failed to parse error response body")
			return nil, fmt.Errorf("conflict: resource already exists or state conflict")
		}
		return nil, fmt.Errorf("conflict: %s", litellmError.Message)
	case 422:
		log.V(1).Info("Unprocessable entity (422)", "statusCode", httpResp.StatusCode)
		litellmError, err := processLitellmError(log, "Unprocessable entity", respBody)
		if err != nil {
			log.Error(err, "Failed to parse error response body")
			return nil, fmt.Errorf("unprocessable entity: validation failed")
		}
		return nil, fmt.Errorf("unprocessable entity: %s", litellmError.Message)
	case 429:
		log.V(1).Info("Too many requests (429)", "statusCode", httpResp.StatusCode)
		return nil, fmt.Errorf("rate limited: too many requests, please try again later")
	case 500:
		log.V(1).Info("Internal server error (500)", "statusCode", httpResp.StatusCode)
		litellmError, err := processLitellmError(log, "Internal server error", respBody)
		if err != nil {
			log.Error(err, "Failed to parse error response body")
			return nil, fmt.Errorf("internal server error: service temporarily unavailable")
		}
		return nil, fmt.Errorf("internal server error: %s", litellmError.Message)
	case 502:
		log.V(1).Info("Bad gateway (502)", "statusCode", httpResp.StatusCode)
		return nil, fmt.Errorf("bad gateway: upstream service unavailable")
	case 503:
		log.V(1).Info("Service unavailable (503)", "statusCode", httpResp.StatusCode)
		return nil, fmt.Errorf("service unavailable: please try again later")
	case 504:
		log.V(1).Info("Gateway timeout (504)", "statusCode", httpResp.StatusCode)
		return nil, fmt.Errorf("gateway timeout: request timed out")
	default:
		log.V(1).Info("Request failed with unexpected status code", "statusCode", httpResp.StatusCode)
		litellmError, err := processLitellmError(log, "Request failed", respBody)
		if err != nil {
			log.Error(err, "Failed to parse error response body")
			return nil, fmt.Errorf("request failed with status %d: unexpected error", httpResp.StatusCode)
		}
		return nil, fmt.Errorf("request failed with status %d: %s", httpResp.StatusCode, litellmError.Message)
	}
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
