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

const workers = 5

func processCOA(ctx context.Context, logger *slog.Logger, svc *Service, store *Store, fh *multipart.FileHeader, errCh chan<- error) {
	file, err := fh.Open()
	if err != nil {
		errCh <- err
		return
	}
	defer file.Close()

	uploaded, err := svc.UploadCOA(ctx, file, fh.Filename, fh.Header.Get("Content-Type"))
	if err != nil {
		logger.Error("Failed to upload COA", "error", err)
		errCh <- err
		return
	}

	result, err := svc.AnalyzeCOA(ctx, uploaded.ID)
	if err != nil {
		logger.Error("failed to analyze COA", "error", err)
		errCh <- err
		return
	}

	if err := svc.DeleteCOA(ctx, uploaded.ID); err != nil {
		logger.Error("Failed to delete COA", "error", err)
	}

	err = store.StoreCOAAnalysis(ctx, result)

	if err != nil {
		logger.Error("failed to store analysis", "error", err)
		errCh <- err
		return
	}
}

func AnalyzeCOAHandler(logger *slog.Logger, svc *Service, store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			logger.Error("failed to parse form", "error", err)
			http.Error(w, "failed to parse form", http.StatusBadRequest)
			return
		}

		files := r.MultipartForm.File["coa"]

		var wg sync.WaitGroup
		jobs := make(chan *multipart.FileHeader, len(files))
		errCh := make(chan error, len(files))

		for _, fh := range files {
			wg.Add(1)
			jobs <- fh
		}
		close(jobs)

		for range workers {
			go func() {
				for fh := range jobs {
					processCOA(r.Context(), logger, svc, store, fh, errCh)
					wg.Done()
				}
			}()
		}

		wg.Wait()

		close(errCh)

		for err := range errCh {
			if err != nil {
				http.Error(w, "failed to process COA", http.StatusInternalServerError)
				return
			}
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
