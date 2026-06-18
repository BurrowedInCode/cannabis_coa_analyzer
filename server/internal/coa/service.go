package coa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

type Service struct {
	client *anthropic.Client
	prompt string
}

func NewService(promptPath string) (*Service, error) {
	prompt, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load prompt: %w", err)
	}
	client := anthropic.NewClient()
	return &Service{client: &client, prompt: string(prompt)}, nil
}

type UploadedFile struct {
	ID       string
	Filename string
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
	seen := map[string]bool{}

	for _, t := range a.Summary {
		key := strings.ToLower(t.Name)
		if summaryDropMap[key] {
			continue
		}
		if cannonical, ok := summaryNameMap[key]; ok {
			t.Name = cannonical
		}

		if !seen[t.Name] {
			seen[t.Name] = true
			normalized = append(normalized, t)
		}
	}

	a.Summary = normalized

	cannabinoids := a.Cannabinoids[:0]
	for _, c := range a.Cannabinoids {
		if cannabinoidDropMap[strings.ToLower(c.Name)] {
			continue
		}
		if cannonical, ok := cannabinoidNameMap[strings.ToLower(c.Name)]; ok {
			c.Name = cannonical
		}
		cannabinoids = append(cannabinoids, c)
	}

	a.Cannabinoids = cannabinoids

}

func (s *Service) UploadCOA(ctx context.Context, r io.Reader, filename string, contentType string) (*UploadedFile, error) {
	meta, err := s.client.Beta.Files.Upload(ctx, anthropic.BetaFileUploadParams{
		File: anthropic.File(r, filename, contentType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to uploaded file: %w", err)
	}
	return &UploadedFile{
		ID:       meta.ID,
		Filename: filename,
	}, nil
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func (s *Service) AnalyzeCOA(ctx context.Context, fileID string) (*Analysis, error) {
	msg, err := s.client.Beta.Messages.New(context.Background(),
		anthropic.BetaMessageNewParams{
			Model:       anthropic.ModelClaudeHaiku4_5,
			MaxTokens:   1024,
			Temperature: anthropic.Float(0),
			Betas:       []anthropic.AnthropicBeta{anthropic.AnthropicBetaFilesAPI2025_04_14},
			Messages: []anthropic.BetaMessageParam{
				anthropic.NewBetaUserMessage(
					anthropic.NewBetaTextBlock(s.prompt),
					anthropic.NewBetaDocumentBlock(anthropic.BetaFileDocumentSourceParam{
						FileID: fileID,
					}),
				),
			},
		})
	if err != nil {
		return nil, fmt.Errorf("failed to analyze COA: %w", err)
	}
	for _, block := range msg.Content {
		if b, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			var analysis Analysis
			if err := json.Unmarshal([]byte(extractJSON(b.Text)), &analysis); err != nil {
				return nil, fmt.Errorf("failed to parse COA response: %w", err)
			}
			normalizeJSON(&analysis)
			return &analysis, nil
		}
	}
	return nil, fmt.Errorf("no text response from Claude")
}

func (s *Service) DeleteCOA(ctx context.Context, fileID string) error {
	_, err := s.client.Beta.Files.Delete(
		ctx,
		fileID,
		anthropic.BetaFileDeleteParams{},
	)
	if err != nil {
		return fmt.Errorf("Failed to delete COA: %w", err)
	}
	return nil
}
