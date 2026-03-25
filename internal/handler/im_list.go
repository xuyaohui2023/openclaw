package handler

import (
	"encoding/json"
	"net/http"

	"github.com/flashclaw/flashclaw-im-channel/internal/config"
	"github.com/flashclaw/flashclaw-im-channel/internal/im"
)

// IMHandler handles all methods on /api/v1/im.
//
//   GET    /api/v1/im              — list all channels with current config
//   POST   /api/v1/im              — create or replace a channel config
//   PATCH  /api/v1/im              — partial update a channel config
//   DELETE /api/v1/im?channel=X    — delete a channel config
func IMHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			imList(cfg, w, r)
		case http.MethodPost:
			imSave(cfg, w, r, false)
		case http.MethodPatch:
			imSave(cfg, w, r, true)
		case http.MethodDelete:
			imDelete(cfg, w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

// --- GET ---

// imChannelEntry is a single channel entry in the list response.
// Config is the actual stored config (included only when bound=true).
type imChannelEntry struct {
	Bound  bool        `json:"bound"`
	Config interface{} `json:"config,omitempty"`
}

type imListResponse struct {
	Telegram imChannelEntry `json:"telegram"`
	Slack    imChannelEntry `json:"slack"`
	Line     imChannelEntry `json:"line"`
}

func imList(cfg *config.Config, w http.ResponseWriter, _ *http.Request) {
	resp := imListResponse{}

	if tg, err := im.GetTelegram(cfg.OpenclawConfigPath); err == nil {
		if isTelegramBound(tg) {
			resp.Telegram = imChannelEntry{Bound: true, Config: tg}
		}
	}
	if sl, err := im.GetSlack(cfg.OpenclawConfigPath); err == nil {
		if isSlackBound(sl) {
			resp.Slack = imChannelEntry{Bound: true, Config: sl}
		}
	}
	if ln, err := im.GetLine(cfg.OpenclawConfigPath); err == nil {
		if isLineBound(ln) {
			resp.Line = imChannelEntry{Bound: true, Config: ln}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func isTelegramBound(c *im.TelegramConfig) bool {
	return c.BotToken != "" || c.TokenFile != ""
}

func isSlackBound(c *im.SlackConfig) bool {
	// Socket mode requires botToken + appToken; http mode requires botToken + signingSecret.
	return c.BotToken != "" || c.AppToken != "" || c.SigningSecret != ""
}

func isLineBound(c *im.LineConfig) bool {
	return c.ChannelAccessToken != "" || c.TokenFile != ""
}

// --- POST / PATCH ---

// imSaveRequest is the body for POST and PATCH.
// The "channel" field selects which IM to configure.
// The channel-specific config is carried in the matching nested object.
type imSaveRequest struct {
	Channel  string          `json:"channel"`
	Telegram json.RawMessage `json:"telegram,omitempty"`
	Slack    json.RawMessage `json:"slack,omitempty"`
	Line     json.RawMessage `json:"line,omitempty"`
}

// imSave handles both POST (full replace) and PATCH (partial update).
func imSave(cfg *config.Config, w http.ResponseWriter, r *http.Request, patch bool) {
	var req imSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.Channel == "" {
		writeError(w, http.StatusBadRequest, "channel is required (telegram | slack | line)")
		return
	}

	switch req.Channel {
	case "telegram":
		if req.Telegram == nil {
			writeError(w, http.StatusBadRequest, "telegram config is required when channel=telegram")
			return
		}
		if patch {
			var p im.PatchTelegramRequest
			if err := json.Unmarshal(req.Telegram, &p); err != nil {
				writeError(w, http.StatusBadRequest, "invalid telegram config: "+err.Error())
				return
			}
			updated, err := im.PatchTelegram(cfg.OpenclawConfigPath, p)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			notifyReload(cfg)
			writeJSON(w, http.StatusOK, updated)
		} else {
			var c im.TelegramConfig
			if err := json.Unmarshal(req.Telegram, &c); err != nil {
				writeError(w, http.StatusBadRequest, "invalid telegram config: "+err.Error())
				return
			}
			if err := im.SetTelegram(cfg.OpenclawConfigPath, c); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			notifyReload(cfg)
			writeJSON(w, http.StatusOK, c)
		}

	case "slack":
		if req.Slack == nil {
			writeError(w, http.StatusBadRequest, "slack config is required when channel=slack")
			return
		}
		if patch {
			var p im.PatchSlackRequest
			if err := json.Unmarshal(req.Slack, &p); err != nil {
				writeError(w, http.StatusBadRequest, "invalid slack config: "+err.Error())
				return
			}
			updated, err := im.PatchSlack(cfg.OpenclawConfigPath, p)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			notifyReload(cfg)
			writeJSON(w, http.StatusOK, updated)
		} else {
			var c im.SlackConfig
			if err := json.Unmarshal(req.Slack, &c); err != nil {
				writeError(w, http.StatusBadRequest, "invalid slack config: "+err.Error())
				return
			}
			if err := im.SetSlack(cfg.OpenclawConfigPath, c); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			notifyReload(cfg)
			writeJSON(w, http.StatusOK, c)
		}

	case "line":
		if req.Line == nil {
			writeError(w, http.StatusBadRequest, "line config is required when channel=line")
			return
		}
		if patch {
			var p im.PatchLineRequest
			if err := json.Unmarshal(req.Line, &p); err != nil {
				writeError(w, http.StatusBadRequest, "invalid line config: "+err.Error())
				return
			}
			updated, err := im.PatchLine(cfg.OpenclawConfigPath, p)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			notifyReload(cfg)
			writeJSON(w, http.StatusOK, updated)
		} else {
			var c im.LineConfig
			if err := json.Unmarshal(req.Line, &c); err != nil {
				writeError(w, http.StatusBadRequest, "invalid line config: "+err.Error())
				return
			}
			if err := im.SetLine(cfg.OpenclawConfigPath, c); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			notifyReload(cfg)
			writeJSON(w, http.StatusOK, c)
		}

	default:
		writeError(w, http.StatusBadRequest, "unknown channel: "+req.Channel+"; must be telegram | slack | line")
	}
}

// --- DELETE ---

func imDelete(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	channel := r.URL.Query().Get("channel")
	if channel == "" {
		writeError(w, http.StatusBadRequest, "query param 'channel' is required (telegram | slack | line)")
		return
	}

	var err error
	switch channel {
	case "telegram":
		err = im.DeleteTelegram(cfg.OpenclawConfigPath)
	case "slack":
		err = im.DeleteSlack(cfg.OpenclawConfigPath)
	case "line":
		err = im.DeleteLine(cfg.OpenclawConfigPath)
	default:
		writeError(w, http.StatusBadRequest, "unknown channel: "+channel+"; must be telegram | slack | line")
		return
	}

	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	notifyReload(cfg)
	w.WriteHeader(http.StatusNoContent)
}
