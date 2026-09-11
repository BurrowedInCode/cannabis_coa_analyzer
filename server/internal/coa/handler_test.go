package coa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubCOAAnalyzer struct {
	filenames []string
}

func (s *stubCOAAnalyzer) AnalyzeCOA(_ context.Context, _ io.Reader, filename string) (*Analysis, Usage, error) {
	s.filenames = append(s.filenames, filename)
	return &Analysis{SampleName: "Valid sample"}, Usage{}, nil
}

type stubCOAAnalysisStore struct {
	analyses []*Analysis
}

func (s *stubCOAAnalysisStore) StoreCOAAnalysis(_ context.Context, analysis *Analysis, _ Usage) error {
	s.analyses = append(s.analyses, analysis)
	return nil
}

func TestAnalyzeCOAHandlerRejectsInvalidBatchSize(t *testing.T) {
	tests := []struct {
		name      string
		fileCount int
		message   string
	}{
		{name: "no files", message: "at least one COA file is required\n"},
		{name: "too many files", fileCount: maxFiles + 1, message: "too many files\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			for i := range tt.fileCount {
				part, err := writer.CreateFormFile("coa", fmt.Sprintf("sample-%d.pdf", i))
				if err != nil {
					t.Fatal(err)
				}
				if _, err := part.Write([]byte("%PDF-1.7\n")); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}

			request := httptest.NewRequest(http.MethodPost, "/coa/analyze", &body)
			request.Header.Set("Content-Type", writer.FormDataContentType())
			response := httptest.NewRecorder()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			AnalyzeCOAHandler(logger, nil, nil).ServeHTTP(response, request)

			if response.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", response.Code, http.StatusBadRequest)
			}
			if response.Body.String() != tt.message {
				t.Errorf("body = %q, want %q", response.Body.String(), tt.message)
			}
		})
	}
}

func TestAnalyzeCOAHandlerRejectsFilesOverMaxRequestSize(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("coa", "oversized.pdf")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := part.Write(make([]byte, maxRequestSize)); err != nil {
		t.Fatal(err)
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/coa/analyze", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	AnalyzeCOAHandler(logger, nil, nil).ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestAnalyzeCOAHandlerIsolatesOversizedFileFailure(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	validPart, err := writer.CreateFormFile("coa", "valid.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validPart.Write([]byte("%PDF-1.7\nvalid")); err != nil {
		t.Fatal(err)
	}

	oversizedPart, err := writer.CreateFormFile("coa", "oversized.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := oversizedPart.Write(make([]byte, maxFileSize+1)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/coa/analyze", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	analyzer := &stubCOAAnalyzer{}
	store := &stubCOAAnalysisStore{}

	AnalyzeCOAHandler(logger, analyzer, store).ServeHTTP(response, request)

	if response.Code != http.StatusMultiStatus {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMultiStatus)
	}
	var result struct {
		Succeeded []string `json:"succeeded"`
		Failed    []string `json:"failed"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result.Succeeded) != 1 || result.Succeeded[0] != "valid.pdf" {
		t.Errorf("succeeded = %v", result.Succeeded)
	}
	if len(result.Failed) != 1 || result.Failed[0] != "oversized.pdf" {
		t.Errorf("failed = %v", result.Failed)
	}
	if len(analyzer.filenames) != 1 || analyzer.filenames[0] != "valid.pdf" {
		t.Errorf("analyzed files = %v", analyzer.filenames)
	}
	if len(store.analyses) != 1 || store.analyses[0].SampleName != "Valid sample" {
		t.Errorf("stored analyses = %#v", store.analyses)
	}
}
