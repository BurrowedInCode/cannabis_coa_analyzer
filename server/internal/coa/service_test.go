package coa

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurrowedInCode/cannabis_coa_analyzer/internal/llm"
)

type stubLLMClient struct {
	request  llm.Request
	response llm.Response
}

const validAnalysisResponse = `{
	"laboratory": {
		"name": "Test Lab",
		"address": "123 Main St",
		"phone": "555-0100",
		"certification": "CERT-1"
	},
	"sample_name": "Blue Dream",
	"seed_to_sale_number": "ABC123",
	"sample_matrix": "Flower",
	"test_date": "2026-01-02T00:00:00Z",
	"overall_pass": true,
	"cannabinoids": [{"name":"total active thc","value":21.5,"unit":"%"}],
	"terpenes": [],
	"summary": [
		{"name":"Microbials","status":"Pass"},
		{"name":"Pesticides","status":"Pass"},
		{"name":"Heavy Metals","status":"Pass"},
		{"name":"Mycotoxins","status":"Pass"},
		{"name":"Foreign Matter","status":"Pass"},
		{"name":"Moisture Content","status":"Pass"},
		{"name":"Residual Solvents","status":"Not Tested"},
		{"name":"Water Activity","status":"Pass"}
	]
}`

func (c *stubLLMClient) Complete(_ context.Context, request llm.Request) (llm.Response, error) {
	c.request = request
	return c.response, nil
}

func testSchemaPath(t *testing.T) string {
	t.Helper()
	schemaPath := filepath.Join("..", "..", "schemas", "coa_analysis_v2.json")
	if _, err := os.Stat(schemaPath); err != nil {
		t.Fatal(err)
	}
	return schemaPath
}

func TestServiceAnalyzeCOA(t *testing.T) {
	promptPath := filepath.Join(t.TempDir(), "prompt.md")
	if err := os.WriteFile(promptPath, []byte("extract the COA"), 0o600); err != nil {
		t.Fatal(err)
	}

	client := &stubLLMClient{response: llm.Response{
		Content: "```json\n" + validAnalysisResponse + "\n```",
		Usage:   llm.Usage{InputTokens: 100, OutputTokens: 20, CostUSD: 0.001},
	}}

	service, err := NewService(client, promptPath, testSchemaPath(t))
	if err != nil {
		t.Fatal(err)
	}

	analysis, usage, err := service.AnalyzeCOA(
		context.Background(),
		strings.NewReader("%PDF-1.7\npdf-data"),
		"sample.pdf",
	)
	if err != nil {
		t.Fatal(err)
	}

	if client.request.Prompt != "extract the COA" || string(client.request.Document.Data) != "%PDF-1.7\npdf-data" {
		t.Errorf("request = %#v", client.request)
	}
	if client.request.ResponseSchema == nil || client.request.ResponseSchema.Name != "coa_analysis" {
		t.Fatalf("response schema = %#v", client.request.ResponseSchema)
	}
	if client.request.ResponseSchema.Definition["type"] != "object" {
		t.Errorf("schema definition = %#v", client.request.ResponseSchema.Definition)
	}
	if client.request.Document.Filename != "sample.pdf" || client.request.Document.MIMEType != "application/pdf" {
		t.Errorf("document = %#v", client.request.Document)
	}
	if analysis.SampleName != "Blue Dream" {
		t.Errorf("sample name = %q", analysis.SampleName)
	}
	if len(analysis.Cannabinoids) != 1 || analysis.Cannabinoids[0].Name != "Total THC" {
		t.Errorf("cannabinoids = %#v", analysis.Cannabinoids)
	}
	if usage.InputTokens != 100 || usage.OutputTokens != 20 || usage.CostUSD != 0.001 {
		t.Errorf("usage = %#v", usage)
	}
}

func TestServiceRejectsInvalidPDF(t *testing.T) {
	promptPath := filepath.Join(t.TempDir(), "prompt.md")
	if err := os.WriteFile(promptPath, []byte("extract"), 0o600); err != nil {
		t.Fatal(err)
	}

	client := &stubLLMClient{}
	service, err := NewService(client, promptPath, testSchemaPath(t))
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = service.AnalyzeCOA(
		context.Background(),
		strings.NewReader("not a PDF"),
		"fake.pdf",
	)
	if err == nil || !strings.Contains(err.Error(), "not a valid PDF") {
		t.Fatalf("error = %v", err)
	}

	if client.request.Document.Data != nil {
		t.Fatal("LLM client was called for an invalid PDF")
	}
}

func TestServiceRejectsOversizedPDF(t *testing.T) {
	promptPath := filepath.Join(t.TempDir(), "prompt.md")
	if err := os.WriteFile(promptPath, []byte("extract"), 0o600); err != nil {
		t.Fatal(err)
	}

	document := make([]byte, maxFileSize+1)
	copy(document, "%PDF-1.7\n")
	client := &stubLLMClient{}
	service, err := NewService(client, promptPath, testSchemaPath(t))
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = service.AnalyzeCOA(context.Background(), bytes.NewReader(document), "sample.pdf")
	if err == nil || !strings.Contains(err.Error(), "exceeds 10 MiB limit") {
		t.Fatalf("error = %v", err)
	}
	if client.request.Document.Data != nil {
		t.Fatal("LLM client was called for an oversized PDF")
	}
}

func TestServiceRejectsInvalidResponse(t *testing.T) {
	promptPath := filepath.Join(t.TempDir(), "prompt.md")
	if err := os.WriteFile(promptPath, []byte("extract"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		content   string
		wantError string
	}{
		{
			name:      "unknown field",
			content:   strings.Replace(validAnalysisResponse, `{`, `{"unexpected":true,`, 1),
			wantError: "additional properties",
		},
		{name: "missing fields", content: `{}`, wantError: "missing properties"},
		{
			name:      "invalid status",
			content:   strings.Replace(validAnalysisResponse, `"Pass"`, `"Unknown"`, 1),
			wantError: "value must be one of",
		},
		{
			name:      "missing summary category",
			content:   strings.Replace(validAnalysisResponse, `{"name":"Microbials","status":"Pass"},`, ``, 1),
			wantError: "minItems: got 7, want 8",
		},
		{
			name:      "unknown summary category",
			content:   strings.Replace(validAnalysisResponse, `"Water Activity"`, `"Unknown"`, 1),
			wantError: "value must be one of",
		},
		{
			name:      "duplicate summary category",
			content:   strings.Replace(validAnalysisResponse, `"Water Activity"`, `"Pesticides"`, 1),
			wantError: `duplicate summary category "Pesticides"`,
		},
		{
			name:      "invalid date",
			content:   strings.Replace(validAnalysisResponse, `2026-01-02T00:00:00Z`, `not-a-date`, 1),
			wantError: "date-time",
		},
		{name: "malformed JSON", content: `{`, wantError: "unexpected EOF"},
		{name: "trailing data", content: `{} {}`, wantError: "invalid character after top-level value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &stubLLMClient{response: llm.Response{Content: tt.content}}
			service, err := NewService(client, promptPath, testSchemaPath(t))
			if err != nil {
				t.Fatal(err)
			}

			_, _, err = service.AnalyzeCOA(
				context.Background(),
				strings.NewReader("%PDF-1.7\npdf-data"),
				"sample.pdf",
			)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want error containing %q", err, tt.wantError)
			}
		})
	}
}

func TestNewServiceRejectsInvalidSchema(t *testing.T) {
	promptPath := filepath.Join(t.TempDir(), "prompt.md")
	if err := os.WriteFile(promptPath, []byte("extract"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		schema    string
		wantError string
	}{
		{name: "malformed JSON", schema: `{`, wantError: "failed to parse schema"},
		{name: "invalid schema", schema: `{"type":"invalid"}`, wantError: "failed to compile schema validator"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schemaPath := filepath.Join(t.TempDir(), "schema.json")
			if err := os.WriteFile(schemaPath, []byte(tt.schema), 0o600); err != nil {
				t.Fatal(err)
			}

			_, err := NewService(&stubLLMClient{}, promptPath, schemaPath)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want error containing %q", err, tt.wantError)
			}
		})
	}
}
