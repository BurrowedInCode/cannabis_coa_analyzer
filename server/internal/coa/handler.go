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

type job struct {
	fh    *multipart.FileHeader
	ctx   context.Context
	errCh chan<- error
	wg    *sync.WaitGroup
}

type WorkerPool struct {
	jobs   chan job
	logger *slog.Logger
	svc    *Service
	store  *Store
}

func NewWorkerPool(logger *slog.Logger, svc *Service, store *Store, n int) *WorkerPool {
	wp := &WorkerPool{
		jobs:   make(chan job),
		logger: logger,
		svc:    svc,
		store:  store,
	}

	for range n {
		go func() {
			for j := range wp.jobs {
				processCOA(j.ctx, logger, svc, store, j.fh, j.errCh)
				j.wg.Done()
			}
		}()
	}

	return wp
}

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

func AnalyzeCOAHandler(logger *slog.Logger, wp *WorkerPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			logger.Error("failed to parse form", "error", err)
			http.Error(w, "failed to parse form", http.StatusBadRequest)
			return
		}

		files := r.MultipartForm.File["coa"]

		var wg sync.WaitGroup
		errCh := make(chan error, len(files))

		for _, fh := range files {
			wg.Add(1)
			wp.jobs <- job{fh: fh, ctx: r.Context(), errCh: errCh, wg: &wg}
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
