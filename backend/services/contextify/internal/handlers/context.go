package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// GET /context/{trace_id}?limit=N
func (h *Handlers) GetContextByTraceID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	traceID := vars["trace_id"]

	if traceID == "" {
		writeError(w, http.StatusBadRequest, "missing trace_id")
		return
	}

	// Parse optional limit param
	limit := 50 // default
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			if parsed > 0 && parsed <= 500 {
				limit = parsed
			}
		}
	}

	events, err := h.DB.GetContextsByTraceID(traceID, limit)
	if err != nil {
		logrus.WithError(err).Error("DB query failed for trace_id")
		writeError(w, http.StatusInternalServerError, "failed to fetch context")
		return
	}

	if len(events) == 0 {
		writeError(w, http.StatusNotFound, "no records found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"trace_id": traceID,
		"count":    len(events),
		"events":   events,
	})
}
