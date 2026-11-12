package llm

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	openrouter "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/components"
	"github.com/OpenRouterTeam/go-sdk/optionalnullable"
)

type OpenRouterConfig struct {
	APIKey     string
	Model      string
	BaseURL    string
	HTTPClient *http.Client
}

type OpenRouterClient struct {
	client *openrouter.OpenRouter
	model  string
}

func NewOpenRouterClient(config OpenRouterConfig) (*OpenRouterClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenRouter API key is required")
	}
	if config.Model == "" {
		return nil, fmt.Errorf("LLM model is required")
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Minute}
	}
	options := []openrouter.SDKOption{
		openrouter.WithSecurity(config.APIKey),
		openrouter.WithClient(httpClient),
		openrouter.WithTimeout(5 * time.Minute),
	}
	if config.BaseURL != "" {
		options = append(options, openrouter.WithServerURL(strings.TrimRight(config.BaseURL, "/")))
	}

	return &OpenRouterClient{
		client: openrouter.New(options...),
		model:  config.Model,
	}, nil
}

func (c *OpenRouterClient) Complete(ctx context.Context, request Request) (Response, error) {
	mimeType := request.Document.MIMEType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	fileData := "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(request.Document.Data)

	chatRequest := components.ChatRequest{
		Model: openrouter.Pointer(c.model),
		Messages: []components.ChatMessages{
			components.CreateChatMessagesUser(components.ChatUserMessage{
				Role: components.ChatUserMessageRoleUser,
				Content: components.CreateChatUserMessageContentArrayOfChatContentItems([]components.ChatContentItems{
					components.CreateChatContentItemsText(components.ChatContentText{Text: request.Prompt}),
					components.CreateChatContentItemsFile(components.ChatContentFile{File: components.ChatContentFileFile{
						Filename: openrouter.Pointer(request.Document.Filename),
						FileData: openrouter.Pointer(fileData),
					}}),
				}),
			}),
		},
		MaxTokens: optionalnullable.From(openrouter.Pointer[int64](1024)),
	}
	if request.ResponseSchema != nil {
		responseFormat := components.CreateResponseFormatJSONSchema(components.ChatFormatJSONSchemaConfig{
			JSONSchema: components.ChatJSONSchemaConfig{
				Name:   request.ResponseSchema.Name,
				Schema: request.ResponseSchema.Definition,
				Strict: optionalnullable.From(new(true)),
			},
		})
		chatRequest.ResponseFormat = &responseFormat
		chatRequest.Provider = optionalnullable.From(new(components.ProviderPreferences{
			RequireParameters: optionalnullable.From(new(true)),
		}))
	}

	result, err := c.client.Chat.Send(ctx, chatRequest, nil)
	if err != nil {
		return Response{}, fmt.Errorf("OpenRouter completion failed: %w", err)
	}
	if result == nil || result.ChatResult == nil || len(result.ChatResult.Choices) == 0 {
		return Response{}, fmt.Errorf("OpenRouter response contained no choices")
	}

	choice := result.ChatResult.Choices[0]

	if choice.FinishReason == nil {
		return Response{}, fmt.Errorf("OpenRouter completion ended without a finish reason")
	}

	if *choice.FinishReason != components.ChatFinishReasonEnumStop {
		return Response{}, fmt.Errorf(
			"OpenRouter completion ended with finish reason %q",
			*choice.FinishReason,
		)
	}

	content, ok := choice.Message.Content.GetOrZero()
	if !ok || content.Str == nil || *content.Str == "" {
		return Response{}, fmt.Errorf("OpenRouter response contained no text")
	}

	usage := Usage{}
	if result.ChatResult.Usage != nil {
		usage.InputTokens = result.ChatResult.Usage.PromptTokens
		usage.OutputTokens = result.ChatResult.Usage.CompletionTokens
		usage.CostUSD, _ = result.ChatResult.Usage.Cost.GetOrZero()
	}

	return Response{Content: *content.Str, Usage: usage}, nil
}
