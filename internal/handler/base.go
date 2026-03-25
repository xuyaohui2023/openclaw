// Package handler implements the HTTP request handlers for each IM channel.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/flashclaw/flashclaw-im-channel/internal/config"
	"github.com/flashclaw/flashclaw-im-channel/internal/openclaw"
)

// notifyReload calls openclaw to reload its configuration after a write.
// Errors are logged but do not fail the HTTP response.
func notifyReload(cfg *config.Config) {
	err := openclaw.NotifyReload(cfg.OpenclawPID, cfg.OpenclawPIDFile)
	if err != nil {
		// Best-effort: log and continue. The config file has already been
		// written correctly; openclaw will pick it up on next restart.
		_ = err // TODO: wire in a proper logger if needed
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
