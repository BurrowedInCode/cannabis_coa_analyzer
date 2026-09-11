package coa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/BurrowedInCode/cannabis_coa_analyzer/internal/llm"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

const pdfMIMEType = "application/pdf"

type Service struct {
	client    llm.Client
	schema    llm.JSONSchema
	validator *jsonschema.Schema
	prompt    string
}

func NewService(client llm.Client, promptPath string, schemaPath string) (*Service, error) {
	prompt, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load prompt: %w", err)
	}

	var schemaDefinition map[string]any
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema: %w", err)
	}

	if err := json.Unmarshal(schemaData, &schemaDefinition); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	if err := compiler.AddResource("coa_analysis.json", schemaDefinition); err != nil {
		return nil, fmt.Errorf("failed to load schema validator: %w", err)
	}
	validator, err := compiler.Compile("coa_analysis.json")
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema validator: %w", err)
	}

	return &Service{
		client: client,
		schema: llm.JSONSchema{
			Name:       "coa_analysis",
			Definition: schemaDefinition,
		},
		validator: validator,
		prompt:    string(prompt),
	}, nil
}

var summaryNameMap = map[string]string{
	"foreign materials":   "Foreign Matter",
	"filth and foreign":   "Foreign Matter",
	"moisture":            "Moisture Content",
	"pathogenic":          "Microbials",
	"microbiology (qpcr)": "Microbials",
}

var summaryDropMap = map[string]bool{
	"total contaminant load": true,
	"label claim":            true,
	"tcl":                    true,
}

var requiredSummaryNames = map[string]bool{
	"Microbials":        true,
	"Pesticides":        true,
	"Heavy Metals":      true,
	"Mycotoxins":        true,
	"Foreign Matter":    true,
	"Moisture Content":  true,
	"Residual Solvents": true,
	"Water Activity":    true,
}

var cannabinoidNameMap = map[string]string{
	"total active thc": "Total THC",
	"total active cbd": "Total CBD",
}

var cannabinoidDropMap = map[string]bool{
	"total cbg": true,
	"total cbn": true,
}

func normalizeJSON(a *Analysis) {
	normalized := a.Summary[:0]

	for _, t := range a.Summary {
		key := strings.ToLower(t.Name)
		if summaryDropMap[key] {
			continue
		}
		if cannonical, ok := summaryNameMap[key]; ok {
			t.Name = cannonical
		}

		normalized = append(normalized, t)
	}

	a.Summary = normalized

	cannabinoids := a.Cannabinoids[:0]
	for _, c := range a.Cannabinoids {
		key := strings.ToLower(c.Name)
		if cannabinoidDropMap[key] {
			continue
		}
		if cannonical, ok := cannabinoidNameMap[key]; ok {
			c.Name = cannonical
		}
		cannabinoids = append(cannabinoids, c)
	}

	a.Cannabinoids = cannabinoids

}

func validateSummary(summary []TestSummary) error {
	if len(summary) != len(requiredSummaryNames) {
		return fmt.Errorf("summary must contain exactly %d categories", len(requiredSummaryNames))
	}

	seen := make(map[string]bool, len(summary))
	for _, item := range summary {
		if !requiredSummaryNames[item.Name] {
			return fmt.Errorf("unexpected summary category %q", item.Name)
		}
		if seen[item.Name] {
			return fmt.Errorf("duplicate summary category %q", item.Name)
		}
		seen[item.Name] = true
	}

	return nil
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

type Usage = llm.Usage

func (s *Service) AnalyzeCOA(ctx context.Context, r io.Reader, filename string) (*Analysis, Usage, error) {
	document, err := io.ReadAll(io.LimitReader(r, maxFileSize+1))
	if err != nil {
		return nil, Usage{}, fmt.Errorf("failed to read COA: %w", err)
	}

	if len(document) > maxFileSize {
		return nil, Usage{}, fmt.Errorf(
			"COA exceeds %d MiB limit",
			maxFileSize>>20,
		)
	}

	if !bytes.HasPrefix(document, []byte("%PDF-")) {
		return nil, Usage{}, fmt.Errorf("file is not a valid PDF")
	}

	if detected := http.DetectContentType(document); detected != pdfMIMEType {
		return nil, Usage{}, fmt.Errorf("file is not a valid PDF")
	}

	response, err := s.client.Complete(ctx, llm.Request{
		Prompt:         s.prompt,
		ResponseSchema: &s.schema,
		Document: llm.Document{
			Filename: filename,
			MIMEType: pdfMIMEType,
			Data:     document,
		},
	})

	if err != nil {
		return nil, Usage{}, fmt.Errorf("failed to analyze COA: %w", err)
	}

	responseJSON := extractJSON(response.Content)
	instance, err := jsonschema.UnmarshalJSON(strings.NewReader(responseJSON))
	if err != nil {
		return nil, response.Usage, fmt.Errorf("failed to parse COA response: %w", err)
	}
	if err := s.validator.Validate(instance); err != nil {
		return nil, response.Usage, fmt.Errorf("COA response failed schema validation: %w", err)
	}

	var analysis Analysis
	decoder := json.NewDecoder(strings.NewReader(responseJSON))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&analysis); err != nil {
		return nil, response.Usage, fmt.Errorf("failed to parse COA response: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, response.Usage, fmt.Errorf("failed to parse COA response: unexpected trailing data")
	}
	normalizeJSON(&analysis)
	if err := validateSummary(analysis.Summary); err != nil {
		return nil, response.Usage, fmt.Errorf("COA response failed semantic validation: %w", err)
	}
	return &analysis, response.Usage, nil
}
