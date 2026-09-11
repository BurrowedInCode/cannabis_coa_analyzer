package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenRouterClientComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/chat/completions" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("authorization = %q", got)
		}

		var request struct {
			Model       string   `json:"model"`
			MaxTokens   *int64   `json:"max_tokens"`
			Temperature *float64 `json:"temperature"`
			Provider    *struct {
				RequireParameters bool `json:"require_parameters"`
			} `json:"provider"`
			ResponseFormat *struct {
				Type       string `json:"type"`
				JSONSchema struct {
					Name   string         `json:"name"`
					Schema map[string]any `json:"schema"`
					Strict bool           `json:"strict"`
				} `json:"json_schema"`
			} `json:"response_format"`
			Messages []struct {
				Content []struct {
					Type string `json:"type"`
					File *struct {
						Filename string `json:"filename"`
						FileData string `json:"file_data"`
					} `json:"file"`
				} `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		if request.Model != "google/gemini-2.5-flash" {
			t.Errorf("model = %q", request.Model)
		}
		if request.MaxTokens == nil || *request.MaxTokens != 1024 {
			t.Errorf("max tokens = %v", request.MaxTokens)
		}
		if request.Temperature != nil {
			t.Errorf("temperature = %v, want omitted", *request.Temperature)
		}
		if request.ResponseFormat == nil || request.ResponseFormat.Type != "json_schema" {
			t.Fatalf("response format = %#v", request.ResponseFormat)
		}
		if request.ResponseFormat.JSONSchema.Name != "coa_analysis" || !request.ResponseFormat.JSONSchema.Strict {
			t.Errorf("JSON schema config = %#v", request.ResponseFormat.JSONSchema)
		}
		if request.ResponseFormat.JSONSchema.Schema["type"] != "object" {
			t.Errorf("JSON schema = %#v", request.ResponseFormat.JSONSchema.Schema)
		}
		if request.Provider == nil || !request.Provider.RequireParameters {
			t.Errorf("provider preferences = %#v", request.Provider)
		}
		if len(request.Messages) != 1 || len(request.Messages[0].Content) != 2 {
			t.Errorf("content = %#v", request.Messages)
			return
		}
		file := request.Messages[0].Content[1].File
		if file == nil || file.Filename != "sample.pdf" {
			t.Errorf("file = %#v", file)
			return
		}
		if file.FileData != "data:application/pdf;base64,cGRm" {
			t.Errorf("file data = %q", file.FileData)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"gen-test",
			"provider":"Google",
			"model":"google/gemini-2.5-flash",
			"object":"chat.completion",
			"created":1786233600,
			"choices":[{
				"index":0,
				"finish_reason":"stop",
				"message":{"role":"assistant","content":"{\"sample_name\":\"Blue Dream\"}"}
			}],
			"usage":{"prompt_tokens":120,"completion_tokens":20,"cost":0.0012}
		}`))
	}))
	defer server.Close()

	client, err := NewOpenRouterClient(OpenRouterConfig{
		APIKey:  "test-key",
		Model:   "google/gemini-2.5-flash",
		BaseURL: server.URL + "/api/v1/",
	})
	if err != nil {
		t.Fatal(err)
	}

	response, err := client.Complete(context.Background(), Request{
		Prompt: "Extract this COA",
		ResponseSchema: &JSONSchema{
			Name:       "coa_analysis",
			Definition: map[string]any{"type": "object"},
		},
		Document: Document{
			Filename: "sample.pdf",
			MIMEType: "application/pdf",
			Data:     []byte("pdf"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if response.Content != `{"sample_name":"Blue Dream"}` {
		t.Errorf("content = %q", response.Content)
	}
	if response.Usage.InputTokens != 120 || response.Usage.OutputTokens != 20 || response.Usage.CostUSD != 0.0012 {
		t.Errorf("usage = %#v", response.Usage)
	}
}

func TestOpenRouterClientAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limit exceeded"}}`))
	}))
	defer server.Close()

	client, err := NewOpenRouterClient(OpenRouterConfig{
		APIKey:  "test-key",
		Model:   "google/gemini-2.5-flash",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Complete(context.Background(), Request{})
	if err == nil || !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Fatalf("error = %v", err)
	}
}

func TestOpenRouterClientRejectsUnsuccessfulFinishReason(t *testing.T) {
	tests := []struct {
		name         string
		finishReason string
		wantError    string
	}{
		{name: "missing", wantError: "finish reason"},
		{name: "length", finishReason: "length", wantError: `finish reason "length"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				choice := map[string]any{
					"index":   0,
					"message": map[string]any{"role": "assistant", "content": `{}`},
				}
				if tt.finishReason != "" {
					choice["finish_reason"] = tt.finishReason
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(map[string]any{
					"id":       "gen-test",
					"provider": "test-provider",
					"model":    "test-model",
					"object":   "chat.completion",
					"created":  1786233600,
					"choices":  []any{choice},
				}); err != nil {
					t.Errorf("encode response: %v", err)
				}
			}))
			defer server.Close()

			client, err := NewOpenRouterClient(OpenRouterConfig{
				APIKey:  "test-key",
				Model:   "test-model",
				BaseURL: server.URL,
			})
			if err != nil {
				t.Fatal(err)
			}

			_, err = client.Complete(context.Background(), Request{})
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want error containing %q", err, tt.wantError)
			}
		})
	}
}

func TestNewOpenRouterClientRequiresConfiguration(t *testing.T) {
	if _, err := NewOpenRouterClient(OpenRouterConfig{Model: "model"}); err == nil {
		t.Error("expected missing API key error")
	}
	if _, err := NewOpenRouterClient(OpenRouterConfig{APIKey: "key"}); err == nil {
		t.Error("expected missing model error")
	}
}
