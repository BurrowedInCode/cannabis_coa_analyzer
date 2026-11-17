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
	"slices"
	"sync"
	"testing"
)

type stubCOAAnalyzer struct {
	filenames    []string
	failFileName string
	sampleNames  map[string]string
	mutex        sync.Mutex
}

func (s *stubCOAAnalyzer) AnalyzeCOA(_ context.Context, _ io.Reader, filename string) (*Analysis, Usage, error) {
	s.mutex.Lock()
	s.filenames = append(s.filenames, filename)
	s.mutex.Unlock()
	if filename == s.failFileName {
		return nil, Usage{}, fmt.Errorf("failed to analyze file %s", filename)
	}
	sampleName := "Valid sample"
	if name, ok := s.sampleNames[filename]; ok {
		sampleName = name
	}
	return &Analysis{SampleName: sampleName}, Usage{}, nil
}

type stubCOAAnalysisStore struct {
	analyses       []*Analysis
	failSampleName string
	mutex          sync.Mutex
}

func (s *stubCOAAnalysisStore) StoreCOAAnalysis(_ context.Context, analysis *Analysis, _ Usage) error {
	s.mutex.Lock()
	s.analyses = append(s.analyses, analysis)
	s.mutex.Unlock()
	if analysis.SampleName == s.failSampleName {
		return fmt.Errorf("failed to store sample %s", analysis.SampleName)
	}
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

func TestAnalyzeCOAHandlerIsolatesAnalyzerFailure(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, filename := range []string{"valid.pdf", "fails.pdf"} {
		part, err := writer.CreateFormFile("coa", filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte("%PDF-1.7\nvalid")); err != nil {
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
	analyzer := &stubCOAAnalyzer{failFileName: "fails.pdf"}
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
	if len(result.Failed) != 1 || result.Failed[0] != "fails.pdf" {
		t.Errorf("failed = %v", result.Failed)
	}
	if len(analyzer.filenames) != 2 || !slices.Contains(analyzer.filenames, "valid.pdf") || !slices.Contains(analyzer.filenames, "fails.pdf") {
		t.Errorf("analyzed files = %v", analyzer.filenames)
	}
	if len(store.analyses) != 1 {
		t.Fatalf("stored analysis count = %d, want 1", len(store.analyses))
	}
	if store.analyses[0] == nil || store.analyses[0].SampleName != "Valid sample" {
		t.Errorf("stored analysis = %#v, want valid sample", store.analyses[0])
	}
}

func TestAnalyzeCOAHandlerIsolatesStoreFailure(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, filename := range []string{"valid.pdf", "fails.pdf"} {
		part, err := writer.CreateFormFile("coa", filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte("%PDF-1.7\nvalid")); err != nil {
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
	analyzer := &stubCOAAnalyzer{
		sampleNames: map[string]string{"fails.pdf": "Storage failure sample"},
	}
	store := &stubCOAAnalysisStore{failSampleName: "Storage failure sample"}

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
	if len(result.Failed) != 1 || result.Failed[0] != "fails.pdf" {
		t.Errorf("failed = %v", result.Failed)
	}
	if len(analyzer.filenames) != 2 || !slices.Contains(analyzer.filenames, "valid.pdf") || !slices.Contains(analyzer.filenames, "fails.pdf") {
		t.Errorf("analyzed files = %v", analyzer.filenames)
	}
	if len(store.analyses) != 2 {
		t.Fatalf("storage attempt count = %d, want 2", len(store.analyses))
	}
	var attemptedSamples []string
	for _, analysis := range store.analyses {
		if analysis == nil {
			t.Fatal("storage received a nil analysis")
		}
		attemptedSamples = append(attemptedSamples, analysis.SampleName)
	}
	if !slices.Contains(attemptedSamples, "Valid sample") || !slices.Contains(attemptedSamples, "Storage failure sample") {
		t.Errorf("storage attempts = %v, want both samples", attemptedSamples)
	}
}
