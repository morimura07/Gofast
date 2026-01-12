package respond

import (
	"encoding/json"
	"errors"
	"gofast/pkg"
	"log/slog"
	"net/http"
)

func JSON(w http.ResponseWriter, r *http.Request, data any, err error) {
	ctx := r.Context()
	if err != nil {
		var unauthorizedError pkg.UnauthorizedError
		var forbiddenError pkg.ForbiddenError
		var internalError pkg.InternalError
		var badRequestError pkg.BadRequestError
		var notFoundError pkg.NotFoundError
		var validationErrors pkg.ValidationErrors
		switch {
		case errors.As(err, &unauthorizedError):
			slog.ErrorContext(ctx, "Unauthorized", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		case errors.As(err, &forbiddenError):
			slog.ErrorContext(ctx, "Forbidden", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
			return
		case errors.As(err, &internalError):
			slog.ErrorContext(ctx, "Internal error", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
			return
		case errors.As(err, &badRequestError):
			slog.ErrorContext(ctx, "Bad request error", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "bad request"})
			return
		case errors.As(err, &notFoundError):
			slog.ErrorContext(ctx, "Not found error", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
			return
		case errors.As(err, &validationErrors):
			slog.ErrorContext(ctx, "Validation error", "fields", validationErrors)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]any{"fields": validationErrors})
			return
		default:
			slog.ErrorContext(ctx, "Error", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
	}
	if data == nil || data == struct{}{} {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	encodedData, err := json.Marshal(data)
	if err != nil {
		slog.ErrorContext(ctx, "Error writing response", "error", err)
		http.Error(w, "Error writing response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(encodedData)
	if err != nil {
		slog.ErrorContext(ctx, "Error writing response", "error", err)
		return
	}
}
