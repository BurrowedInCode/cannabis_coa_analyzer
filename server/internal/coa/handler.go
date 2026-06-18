package coa

import (
	"encoding/json"
	"log/slog"
	"net/http"
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
