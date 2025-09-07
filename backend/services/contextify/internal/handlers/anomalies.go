package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sirupsen/logrus"
)

// GET /anomalies?service=X&limit_anomalies=N&limit_context=N
func (h *Handlers) GetAnomaliesByService(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")
	if service == "" {
		writeError(w, http.StatusBadRequest, "missing service param")
		return
	}

	// Parse limits
	limitAnomalies := parseLimit(r.URL.Query().Get("limit_anomalies"), 20)
	limitContext := parseLimit(r.URL.Query().Get("limit_context"), 20)

	anomalies, err := h.DB.GetRecentAnomalies(service, limitAnomalies)
	if err != nil {
		logrus.WithError(err).Error("DB query failed for anomalies")
		writeError(w, http.StatusInternalServerError, "failed to fetch anomalies")
		return
	}

	contexts, err := h.DB.GetRecentEvents(service, limitContext)
	if err != nil {
		logrus.WithError(err).Error("DB query failed for context events")
		writeError(w, http.StatusInternalServerError, "failed to fetch context")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service":        service,
		"anomaly_count":  len(anomalies),
		"context_count":  len(contexts),
		"anomalies":      anomalies,
		"recent_context": contexts,
	})
}

// --- helpers ---

func parseLimit(raw string, def int) int {
	if raw == "" {
		return def
	}
	if n, err := strconv.Atoi(raw); err == nil {
		if n > 0 && n <= 500 {
			return n
		}
	}
	return def
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   msg,
		"code":    code,
		"success": false,
	})
}
