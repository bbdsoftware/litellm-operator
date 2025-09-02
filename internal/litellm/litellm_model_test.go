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

package litellm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLitellmModel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Litellm Model Suite")
}

var _ = Describe("Litellm Model", func() {
	var (
		client    *LitellmClient
		server    *httptest.Server
		ctx       context.Context
		baseURL   string
		masterKey string
	)

	BeforeEach(func() {
		ctx = context.Background()
		masterKey = "test-master-key"
	})

	AfterEach(func() {
		if server != nil {
			server.Close()
		}
	})

	Describe("CreateModel", func() {
		Context("when the request is successful", func() {
			BeforeEach(func() {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					Expect(r.Method).To(Equal("POST"))
					Expect(r.URL.Path).To(Equal("/model/new"))
					Expect(r.Header.Get("Authorization")).To(Equal("Bearer " + masterKey))
					Expect(r.Header.Get("Content-Type")).To(Equal("application/json"))

					// Read and validate request body
					var req ModelRequest
					err := json.NewDecoder(r.Body).Decode(&req)
					Expect(err).NotTo(HaveOccurred())
					Expect(req.ModelName).To(Equal("test-model"))

					// Return success response
					response := ModelResponse{
						ModelName: "test-model",
						LiteLLMParams: &UpdateLiteLLMParams{
							InputCostPerToken:  float64Ptr(0.001),
							OutputCostPerToken: float64Ptr(0.002),
							Model:              stringPtr("gpt-4"),
						},
						ModelInfo: &ModelInfo{
							ID: stringPtr("model-123"),
						},
					}

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					if err := json.NewEncoder(w).Encode(response); err != nil {
						Fail("failed to encode response: " + err.Error())
					}
				}))

				baseURL = server.URL
				client = NewLitellmClient(baseURL, masterKey)
			})

			It("should create a model successfully", func() {
				req := &ModelRequest{
					ModelName: "test-model",
					LiteLLMParams: &UpdateLiteLLMParams{
						InputCostPerToken:  float64Ptr(0.001),
						OutputCostPerToken: float64Ptr(0.002),
						Model:              stringPtr("gpt-4"),
					},
					ModelInfo: &ModelInfo{},
				}

				response, err := client.CreateModel(ctx, req)

				Expect(err).NotTo(HaveOccurred())
				Expect(response.ModelName).To(Equal("test-model"))
				Expect(response.LiteLLMParams.InputCostPerToken).To(Equal(float64Ptr(0.001)))
				Expect(response.LiteLLMParams.OutputCostPerToken).To(Equal(float64Ptr(0.002)))
				Expect(response.ModelInfo.ID).To(Equal(stringPtr("model-123")))
				Expect(response.LiteLLMParams.Model).To(Equal(stringPtr("gpt-4")))
			})
		})

		Context("when the server returns an error", func() {
			BeforeEach(func() {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					if err := json.NewEncoder(w).Encode(map[string]interface{}{
						"error": map[string]interface{}{
							"message": "Model already exists",
							"type":    "validation_error",
							"code":    "MODEL_EXISTS",
						},
					}); err != nil {
						Fail("failed to encode error response: " + err.Error())
					}
				}))

				baseURL = server.URL
				client = NewLitellmClient(baseURL, masterKey)
			})

			It("should return an error", func() {
				req := &ModelRequest{
					ModelName: "existing-model",
				}

				_, err := client.CreateModel(ctx, req)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Model already exists"))
			})
		})

		Context("when the request body cannot be marshalled", func() {
			BeforeEach(func() {
				client = NewLitellmClient("http://localhost:8080", masterKey)
			})

			It("should return a marshalling error", func() {
				// Create a request with a field that cannot be marshalled
				req := &ModelRequest{
					ModelName: "test-model",
					LiteLLMParams: &UpdateLiteLLMParams{
						VertexCredentials: make(chan int), // This cannot be marshalled
					},
				}

				_, err := client.CreateModel(ctx, req)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("json: unsupported type"))
			})
		})
	})

	Describe("UpdateModel", func() {
		Context("when the request is successful", func() {
			BeforeEach(func() {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					Expect(r.Method).To(Equal("POST"))
					Expect(r.URL.Path).To(Equal("/model/update"))
					Expect(r.Header.Get("Authorization")).To(Equal("Bearer " + masterKey))
					Expect(r.Header.Get("Content-Type")).To(Equal("application/json"))

					// Read and validate request body
					var req ModelRequest
					err := json.NewDecoder(r.Body).Decode(&req)
					Expect(err).NotTo(HaveOccurred())
					Expect(req.ModelName).To(Equal("test-model"))

					// Return success response
					response := ModelResponse{
						ModelName: "test-model",
						LiteLLMParams: &UpdateLiteLLMParams{
							InputCostPerToken:  float64Ptr(0.002),
							OutputCostPerToken: float64Ptr(0.003),
						},
					}

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					if err := json.NewEncoder(w).Encode(response); err != nil {
						Fail("failed to encode response: " + err.Error())
					}
				}))

				baseURL = server.URL
				client = NewLitellmClient(baseURL, masterKey)
			})

			It("should update a model successfully", func() {
				req := &ModelRequest{
					ModelName: "test-model",
					LiteLLMParams: &UpdateLiteLLMParams{
						InputCostPerToken:  float64Ptr(0.002),
						OutputCostPerToken: float64Ptr(0.003),
					},
				}

				response, err := client.UpdateModel(ctx, req)

				Expect(err).NotTo(HaveOccurred())
				Expect(response.ModelName).To(Equal("test-model"))
				Expect(response.LiteLLMParams.InputCostPerToken).To(Equal(float64Ptr(0.002)))
				Expect(response.LiteLLMParams.OutputCostPerToken).To(Equal(float64Ptr(0.003)))
			})
		})

		Context("when the server returns an error", func() {
			BeforeEach(func() {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)
					if err := json.NewEncoder(w).Encode(map[string]interface{}{
						"detail": map[string]interface{}{
							"error": "Model not found",
						},
					}); err != nil {
						Fail("failed to encode error response: " + err.Error())
					}
				}))

				baseURL = server.URL
				client = NewLitellmClient(baseURL, masterKey)
			})

			It("should return an error", func() {
				req := &ModelRequest{
					ModelName: "non-existent-model",
				}

				_, err := client.UpdateModel(ctx, req)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("resource does not exist"))
			})
		})
	})

	Describe("GetModel", func() {
		Context("when the request is successful", func() {
			BeforeEach(func() {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					Expect(r.Method).To(Equal("GET"))
					Expect(r.URL.Path).To(Equal("/model/"))
					Expect(r.URL.Query().Get("litellm_model_id")).To(Equal("test-model"))
					Expect(r.Header.Get("Authorization")).To(Equal("Bearer " + masterKey))

					// Return success response
					response := ModelResponse{
						ModelName: "test-model",
						LiteLLMParams: &UpdateLiteLLMParams{
							InputCostPerToken:  float64Ptr(0.001),
							OutputCostPerToken: float64Ptr(0.002),
							ApiKey:             stringPtr("sk-test-key"),
							Model:              stringPtr("gpt-4"),
						},
						ModelInfo: &ModelInfo{
							ID: stringPtr("model-123"),
						},
					}

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					if err := json.NewEncoder(w).Encode(response); err != nil {
						Fail("failed to encode response: " + err.Error())
					}
				}))

				baseURL = server.URL
				client = NewLitellmClient(baseURL, masterKey)
			})

			It("should get a model successfully", func() {
				response, err := client.GetModel(ctx, "test-model")

				Expect(err).NotTo(HaveOccurred())
				Expect(response.ModelName).To(Equal("test-model"))
				Expect(response.LiteLLMParams.InputCostPerToken).To(Equal(float64Ptr(0.001)))
				Expect(response.LiteLLMParams.OutputCostPerToken).To(Equal(float64Ptr(0.002)))
				Expect(response.LiteLLMParams.ApiKey).To(Equal(stringPtr("sk-test-key")))
				Expect(response.ModelInfo.ID).To(Equal(stringPtr("model-123")))
				Expect(response.LiteLLMParams.Model).To(Equal(stringPtr("gpt-4")))
			})
		})

		Context("when the server returns an error", func() {
			BeforeEach(func() {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)
					if err := json.NewEncoder(w).Encode(map[string]interface{}{
						"error": map[string]interface{}{
							"message": "Model not found",
							"type":    "not_found",
							"code":    "MODEL_NOT_FOUND",
						},
					}); err != nil {
						Fail("failed to encode error response: " + err.Error())
					}
				}))

				baseURL = server.URL
				client = NewLitellmClient(baseURL, masterKey)
			})

			It("should return an error", func() {
				_, err := client.GetModel(ctx, "non-existent-model")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("the requested resource does not exist"))
			})
		})
	})

	Describe("DeleteModel", func() {
		Context("when the request is successful", func() {
			BeforeEach(func() {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					Expect(r.Method).To(Equal("POST"))
					Expect(r.URL.Path).To(Equal("/model/delete"))
					Expect(r.Header.Get("Authorization")).To(Equal("Bearer " + masterKey))
					Expect(r.Header.Get("Content-Type")).To(Equal("application/json"))

					// Read and validate request body
					var reqBody map[string]string
					err := json.NewDecoder(r.Body).Decode(&reqBody)
					Expect(err).NotTo(HaveOccurred())
					Expect(reqBody["id"]).To(Equal("model-12132"))

					w.WriteHeader(http.StatusOK)
				}))

				baseURL = server.URL
				client = NewLitellmClient(baseURL, masterKey)
			})

			It("should delete a model successfully", func() {
				err := client.DeleteModel(ctx, "model-12132")

				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the server returns an error", func() {
			BeforeEach(func() {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)
					if err := json.NewEncoder(w).Encode(map[string]interface{}{
						"error": map[string]interface{}{
							"message": "Model not found",
							"type":    "not_found",
							"code":    "MODEL_NOT_FOUND",
						},
					}); err != nil {
						Fail("failed to encode error response: " + err.Error())
					}
				}))

				baseURL = server.URL
				client = NewLitellmClient(baseURL, masterKey)
			})

			It("should return an error", func() {
				err := client.DeleteModel(ctx, "non-existent-model-id")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("resource does not exist"))
			})
		})
	})

	Describe("IsModelUpdateNeeded", func() {
		var client *LitellmClient

		BeforeEach(func() {
			client = NewLitellmClient("http://localhost:8080", "test-key")
		})

		Context("when model name is different", func() {
			It("should return true", func() {
				model := &ModelResponse{
					ModelName: "old-model",
				}
				req := &ModelRequest{
					ModelName: "new-model",
				}

				result, err := client.IsModelUpdateNeeded(ctx, model, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(result).To(BeTrue())
			})
		})

		Context("when LiteLLM parameters are different", func() {
			It("should return true when input cost changes", func() {
				model := &ModelResponse{
					ModelName: "test-model",
					LiteLLMParams: &UpdateLiteLLMParams{
						InputCostPerToken: float64Ptr(0.001),
					},
				}
				req := &ModelRequest{
					ModelName: "test-model",
					LiteLLMParams: &UpdateLiteLLMParams{
						InputCostPerToken: float64Ptr(0.002),
					},
				}

				result, err := client.IsModelUpdateNeeded(ctx, model, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(result).To(BeTrue())
			})

			It("should return true when output cost changes", func() {
				model := &ModelResponse{
					ModelName: "test-model",
					LiteLLMParams: &UpdateLiteLLMParams{
						OutputCostPerToken: float64Ptr(0.001),
					},
				}
				req := &ModelRequest{
					ModelName: "test-model",
					LiteLLMParams: &UpdateLiteLLMParams{
						OutputCostPerToken: float64Ptr(0.002),
					},
				}

				result, err := client.IsModelUpdateNeeded(ctx, model, req)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.NeedsUpdate).To(BeTrue())

			})

			It("should return true when API key changes", func() {
				model := &ModelResponse{
					ModelName: "test-model",
					LiteLLMParams: &UpdateLiteLLMParams{
						ApiKey: stringPtr("old-key"),
					},
				}
				req := &ModelRequest{
					ModelName: "test-model",
					LiteLLMParams: &UpdateLiteLLMParams{
						ApiKey: stringPtr("new-key"),
					},
				}

				result, err := client.IsModelUpdateNeeded(ctx, model, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.NeedsUpdate).To(BeTrue())

			})
		})

		Context("when model is different", func() {
			It("should return true when model changes", func() {
				model := &ModelResponse{
					ModelName: "test-model",
					LiteLLMParams: &UpdateLiteLLMParams{
						Model: stringPtr("gpt-3.5"),
					},
				}
				req := &ModelRequest{
					ModelName: "test-model",
					LiteLLMParams: &UpdateLiteLLMParams{
						Model: stringPtr("gpt-4"),
					},
				}

				result, err := client.IsModelUpdateNeeded(ctx, model, req)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.NeedsUpdate).To(BeTrue())
			})

			It("should return true when modelInfo changes", func() {
				model := &ModelResponse{
					ModelName: "test-model",
					ModelInfo: &ModelInfo{
						ID: stringPtr("model-123"),
					},
				}
				req := &ModelRequest{
					ModelName: "test-model",
					ModelInfo: &ModelInfo{
						ID: stringPtr("model-000008"),
					},
				}

				result, err := client.IsModelUpdateNeeded(ctx, model, req)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.NeedsUpdate).To(BeTrue())
			})
		})

		Context("when no changes are detected", func() {
			It("should return false", func() {
				model := &ModelResponse{
					ModelName: "test-model",
					LiteLLMParams: &UpdateLiteLLMParams{
						InputCostPerToken:  float64Ptr(0.001),
						OutputCostPerToken: float64Ptr(0.002),
						ApiKey:             stringPtr("test-key"),
						Model:              stringPtr("gpt-4"),
					},
					ModelInfo: &ModelInfo{
						ID: stringPtr("model-123"),
					},
				}
				req := &ModelRequest{
					ModelName: "test-model",
					LiteLLMParams: &UpdateLiteLLMParams{
						InputCostPerToken:  float64Ptr(0.001),
						OutputCostPerToken: float64Ptr(0.002),
						ApiKey:             stringPtr("test-key"),
						Model:              stringPtr("gpt-4"),
					},
					ModelInfo: &ModelInfo{
						ID: stringPtr("model-123"),
					},
				}

				result, err := client.IsModelUpdateNeeded(ctx, model, req)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.NeedsUpdate).To(BeFalse())
			})
		})

		Context("when comparing nil values", func() {
			It("should handle nil LiteLLMParams", func() {
				model := &ModelResponse{
					ModelName: "test-model",
				}
				req := &ModelRequest{
					ModelName: "test-model",
					LiteLLMParams: &UpdateLiteLLMParams{
						InputCostPerToken: float64Ptr(0.001),
					},
				}

				result, err := client.IsModelUpdateNeeded(ctx, model, req)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.NeedsUpdate).To(BeTrue())
			})

			It("should handle nil ModelInfo", func() {
				model := &ModelResponse{
					ModelName: "test-model",
				}
				req := &ModelRequest{
					ModelName: "test-model",
					ModelInfo: &ModelInfo{
						ID: stringPtr("000000000011"),
					},
				}

				result, err := client.IsModelUpdateNeeded(ctx, model, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.NeedsUpdate).To(BeTrue())

				Expect(result).To(BeTrue())
			})
		})
	})

	Describe("ModelRequest and ModelResponse serialisation", func() {
		It("should marshal and unmarshal ModelRequest correctly", func() {
			req := &ModelRequest{
				ModelName: "test-model",
				LiteLLMParams: &UpdateLiteLLMParams{
					InputCostPerToken:  float64Ptr(0.001),
					OutputCostPerToken: float64Ptr(0.002),
					ApiKey:             stringPtr("sk-test-key"),
					ApiBase:            stringPtr("https://api.openai.com"),
					TPM:                intPtr(1000),
					RPM:                intPtr(100),
					Model:              stringPtr("gpt-4"),
				},
				ModelInfo: &ModelInfo{
					ID:     stringPtr("model-123"),
					TeamID: stringPtr("team-123"),
				},
			}

			data, err := json.Marshal(req)
			Expect(err).NotTo(HaveOccurred())

			var unmarshalled ModelRequest
			err = json.Unmarshal(data, &unmarshalled)
			Expect(err).NotTo(HaveOccurred())

			Expect(unmarshalled.ModelName).To(Equal(req.ModelName))
			Expect(unmarshalled.LiteLLMParams.InputCostPerToken).To(Equal(req.LiteLLMParams.InputCostPerToken))
			Expect(unmarshalled.LiteLLMParams.OutputCostPerToken).To(Equal(req.LiteLLMParams.OutputCostPerToken))
			Expect(unmarshalled.LiteLLMParams.ApiKey).To(Equal(req.LiteLLMParams.ApiKey))
			Expect(unmarshalled.LiteLLMParams.ApiBase).To(Equal(req.LiteLLMParams.ApiBase))
			Expect(unmarshalled.LiteLLMParams.TPM).To(Equal(req.LiteLLMParams.TPM))
			Expect(unmarshalled.LiteLLMParams.RPM).To(Equal(req.LiteLLMParams.RPM))
			Expect(unmarshalled.ModelInfo.ID).To(Equal(req.ModelInfo.ID))
			Expect(unmarshalled.LiteLLMParams.Model).To(Equal(req.LiteLLMParams.Model))
			Expect(unmarshalled.ModelInfo.TeamID).To(Equal(req.ModelInfo.TeamID))
		})

		It("should marshal and unmarshal ModelResponse correctly", func() {
			resp := &ModelResponse{
				ModelName: "test-model",
				LiteLLMParams: &UpdateLiteLLMParams{
					InputCostPerToken:  float64Ptr(0.001),
					OutputCostPerToken: float64Ptr(0.002),
					ApiKey:             stringPtr("sk-test-key"),
					Model:              stringPtr("gpt-4"),
				},
				ModelInfo: &ModelInfo{
					ID: stringPtr("model-123"),
				},
			}

			data, err := json.Marshal(resp)
			Expect(err).NotTo(HaveOccurred())

			var unmarshalled ModelResponse
			err = json.Unmarshal(data, &unmarshalled)
			Expect(err).NotTo(HaveOccurred())

			Expect(unmarshalled.ModelName).To(Equal(resp.ModelName))
			Expect(unmarshalled.LiteLLMParams.InputCostPerToken).To(Equal(resp.LiteLLMParams.InputCostPerToken))
			Expect(unmarshalled.LiteLLMParams.OutputCostPerToken).To(Equal(resp.LiteLLMParams.OutputCostPerToken))
			Expect(unmarshalled.LiteLLMParams.ApiKey).To(Equal(resp.LiteLLMParams.ApiKey))
			Expect(unmarshalled.ModelInfo.ID).To(Equal(resp.ModelInfo.ID))
			Expect(unmarshalled.LiteLLMParams.Model).To(Equal(resp.LiteLLMParams.Model))
		})
	})

	Describe("Edge cases and error handling", func() {
		Context("when the server is unreachable", func() {
			BeforeEach(func() {
				client = NewLitellmClient("http://localhost:9999", masterKey)
			})

			It("should return a connection error for CreateModel", func() {
				req := &ModelRequest{
					ModelName: "test-model",
				}

				_, err := client.CreateModel(ctx, req)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("connection refused"))
			})

			It("should return a connection error for UpdateModel", func() {
				req := &ModelRequest{
					ModelName: "test-model",
				}

				_, err := client.UpdateModel(ctx, req)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("connection refused"))
			})

			It("should return a connection error for GetModel", func() {
				_, err := client.GetModel(ctx, "test-model")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("connection refused"))
			})

			It("should return a connection error for DeleteModel", func() {
				err := client.DeleteModel(ctx, "test-model")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("connection refused"))
			})
		})

		Context("when the server returns invalid JSON", func() {
			BeforeEach(func() {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					if _, err := w.Write([]byte("invalid json")); err != nil {
						Fail("failed to write response: " + err.Error())
					}
				}))

				baseURL = server.URL
				client = NewLitellmClient(baseURL, masterKey)
			})

			It("should return an unmarshalling error for CreateModel", func() {
				req := &ModelRequest{
					ModelName: "test-model",
				}

				_, err := client.CreateModel(ctx, req)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid character"))
			})

			It("should return an unmarshalling error for UpdateModel", func() {
				req := &ModelRequest{
					ModelName: "test-model",
				}

				_, err := client.UpdateModel(ctx, req)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid character"))
			})

			It("should return an unmarshalling error for GetModel", func() {
				_, err := client.GetModel(ctx, "test-model")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid character"))
			})
		})

		Context("when the server returns an unknown error format", func() {
			BeforeEach(func() {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					if _, err := w.Write([]byte("unknown error format")); err != nil {
						Fail("failed to write response: " + err.Error())
					}
				}))

				baseURL = server.URL
				client = NewLitellmClient(baseURL, masterKey)
			})

		})
	})
})

// Helper functions for creating pointers to primitive types
// Helper functions for creating pointers to primitive types
// nolint:unused
func float64Ptr(v float64) *float64 { return &v }

// nolint:unused
func stringPtr(v string) *string { return &v }

// nolint:unused
func intPtr(v int) *int { return &v }

// nolint:unused
func boolPtr(v bool) *bool { return &v }
