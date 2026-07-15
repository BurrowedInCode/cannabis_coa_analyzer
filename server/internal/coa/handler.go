package coa

import (
	"context"
	"encoding/json"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"
	"sync"
)

type coaResult struct {
	filename string
	err      error
}

func processCOA(ctx context.Context, logger *slog.Logger, svc *Service, store *Store, fh *multipart.FileHeader) coaResult {
	file, err := fh.Open()
	if err != nil {
		return coaResult{filename: fh.Filename, err: err}
	}
	defer file.Close()

	uploaded, err := svc.UploadCOA(ctx, file, fh.Filename, fh.Header.Get("Content-Type"))
	if err != nil {
		logger.Error("Failed to upload COA", "error", err)
		return coaResult{filename: fh.Filename, err: err}
	}

	result, usage, err := svc.AnalyzeCOA(ctx, uploaded.ID)
	if err != nil {
		logger.Error("failed to analyze COA", "error", err)
		return coaResult{filename: fh.Filename, err: err}
	}

	logger.Info("COA analyzed",
		"file", fh.Filename,
		"input_tokens", usage.InputTokens,
		"output_tokens", usage.OutputTokens,
		"cost_usd", usage.CostUSD(),
	)

	if err := store.StoreCOAAnalysis(ctx, result, usage); err != nil {
		logger.Error("failed to store analysis", "error", err)
		return coaResult{filename: fh.Filename, err: err}
	}

	if err := svc.DeleteCOA(ctx, uploaded.ID); err != nil {
		logger.Error("Failed to delete COA", "error", err)
	}

	return coaResult{filename: fh.Filename}
}

func AnalyzeCOAHandler(logger *slog.Logger, svc *Service, store *Store) http.HandlerFunc {
	sem := make(chan struct{}, 10)
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			logger.Error("failed to parse form", "error", err)
			http.Error(w, "failed to parse form", http.StatusBadRequest)
			return
		}

		files := r.MultipartForm.File["coa"]
		var wg sync.WaitGroup
		resCh := make(chan coaResult, len(files))

		for _, fh := range files {
			wg.Add(1)
			sem <- struct{}{}
			go func(fh *multipart.FileHeader) {
				defer wg.Done()
				defer func() { <-sem }()
				resCh <- processCOA(r.Context(), logger, svc, store, fh)
			}(fh)
		}

		wg.Wait()
		close(resCh)

		var succeeded, failed []string
		for res := range resCh {
			if res.err != nil {
				failed = append(failed, res.filename)
				logger.Error("processing failed", "file", res.filename, "error", res.err)
			} else {
				succeeded = append(succeeded, res.filename)
			}
		}

		if len(failed) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMultiStatus)
			json.NewEncoder(w).Encode(map[string][]string{
				"succeeded": succeeded,
				"failed":    failed,
			})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func GetAllCOAAnalysesHandler(logger *slog.Logger, store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limitString := r.URL.Query().Get("limit")
		offsetString := r.URL.Query().Get("offset")

		limitInt, err := strconv.Atoi(limitString)

		if err != nil || limitInt <= 0 {
			limitInt = 20
		}

		offsetInt, err := strconv.Atoi(offsetString)

		if err != nil || offsetInt < 0 {
			offsetInt = 0
		}

		analyses, err := store.GetAllCOAAnalyses(r.Context(), limitInt, offsetInt)

		if err != nil {
			logger.Error("failed to fetch analyses", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(analyses); err != nil {
			logger.Error("failed to encode response", "error", err)
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

func GetCOAAnalysisHandler(logger *slog.Logger, store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		analysisID := r.PathValue("id")

		if analysisID == "" {
			http.Error(w, "missing analysis id", http.StatusBadRequest)
			return
		}

		analysis, err := store.GetCOAAnalysis(r.Context(), analysisID)

		if err != nil {
			logger.Error("failed to fetch analysis", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(analysis); err != nil {
			logger.Error("failed to encode response", "error", err)
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

func UpdateCOAAnalysisHandler(logger *slog.Logger, store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		analysisID := r.PathValue("id")

		if analysisID == "" {
			http.Error(w, "missing analysis id", http.StatusBadRequest)
			return
		}

		var a Analysis
		if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if err := store.UpdateCOAAnalysis(r.Context(), analysisID, &a); err != nil {
			logger.Error("failed to update analysis", "error", err)
			http.Error(w, "failed to update analysis", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
