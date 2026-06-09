package coa

import (
	"context"
	"fmt"
	"io"

	"github.com/anthropics/anthropic-sdk-go"
)

type Service struct {
	client *anthropic.Client
}

func NewService() *Service {
	client := anthropic.NewClient()
	return &Service{client: &client}
}

type UploadedFile struct {
	ID       string
	Filename string
}

func (s *Service) UploadCOA(ctx context.Context, r io.Reader, filename string, contentType string) (*UploadedFile, error) {
	meta, err := s.client.Beta.Files.Upload(ctx, anthropic.BetaFileUploadParams{
		File: anthropic.File(r, filename, contentType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to uploaded file: %w", err)
	}
	fmt.Println(meta.SizeBytes)
	return &UploadedFile{
		ID:       meta.ID,
		Filename: filename,
	}, nil
}

func (s *Service) AnalyzeCOA(ctx context.Context, fileID string) (string, error) {
	msg, err := s.client.Beta.Messages.New(context.Background(),
		anthropic.BetaMessageNewParams{
			Model:     anthropic.ModelClaudeHaiku4_5,
			MaxTokens: 1024,
			Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaFilesAPI2025_04_14},
			Messages: []anthropic.BetaMessageParam{
				anthropic.NewBetaUserMessage(
					anthropic.NewBetaTextBlock("Please summarize this document for me."),
					anthropic.NewBetaDocumentBlock(anthropic.BetaFileDocumentSourceParam{
						FileID: fileID,
					}),
				),
			},
		})
	if err != nil {
		return "", fmt.Errorf("failed to analyze COA: %w", err)
	}
	for _, block := range msg.Content {
		if b, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			return b.Text, nil
		}
	}
	return "", fmt.Errorf("no text response from Claude")
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
