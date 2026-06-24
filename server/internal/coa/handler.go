package coa

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

func AnalyzeCOAHandler(logger *slog.Logger, svc *Service, store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, header, err := r.FormFile("coa")
		if err != nil {
			http.Error(w, "missing coa file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		uploaded, err := svc.UploadCOA(r.Context(), file, header.Filename, header.Header.Get("Content-Type"))
		if err != nil {
			logger.Error("Failed to upload COA", "error", err)
			http.Error(w, "failed to upload file", http.StatusInternalServerError)
			return
		}
		result, err := svc.AnalyzeCOA(r.Context(), uploaded.ID)
		if err != nil {
			logger.Error("failed to analyze COA", "error", err)
			http.Error(w, "failed to analyze file", http.StatusInternalServerError)
			return
		}

		if err := svc.DeleteCOA(r.Context(), uploaded.ID); err != nil {
			logger.Error("Failed to delete COA", "error", err)
		}

		err = store.StoreCOAAnalysis(r.Context(), result)

		if err != nil {
			logger.Error("failed to store analysis", "error", err)
			http.Error(w, "failed to store analysis", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
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
