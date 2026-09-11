package llm

import "context"

type JSONSchema struct {
	Name       string
	Definition map[string]any
}

type Document struct {
	Filename string
	MIMEType string
	Data     []byte
}

type Request struct {
	Prompt         string
	Document       Document
	ResponseSchema *JSONSchema
}

type Usage struct {
	InputTokens  int64
	OutputTokens int64
	CostUSD      float64
}

type Response struct {
	Content string
	Usage   Usage
}

type Client interface {
	Complete(context.Context, Request) (Response, error)
}
