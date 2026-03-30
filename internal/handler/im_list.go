package handler

import (
	"encoding/json"
	"net/http"

	"github.com/flashclaw/flashclaw-im-channel/internal/config"
	"github.com/flashclaw/flashclaw-im-channel/internal/im"
	"github.com/flashclaw/flashclaw-im-channel/internal/oclaw"
)

// IMHandler handles all methods on /api/v1/im.
//
//   GET    /api/v1/im              — list all channels with current config
//   POST   /api/v1/im              — create or replace a channel config
//   PATCH  /api/v1/im              — partial update a channel config
//   DELETE /api/v1/im?channel=X    — delete a channel config
func IMHandler(_ *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			imList(w)
		case http.MethodPost:
			imSave(w, r, false)
		case http.MethodPatch:
			imSave(w, r, true)
		case http.MethodDelete:
			imDelete(w, r)
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

// channelsConfig mirrors the shape of openclaw's channels section.
type channelsConfig struct {
	Telegram *im.TelegramConfig `json:"telegram"`
	Slack    *im.SlackConfig    `json:"slack"`
	Line     *im.LineConfig     `json:"line"`
}

// imList queries `openclaw config get channels` to build the response.
func imList(w http.ResponseWriter) {
	resp := imListResponse{}

	channels, err := oclaw.GetAs[channelsConfig]("channels")
	if err == nil && channels != nil {
		if channels.Telegram != nil && isTelegramBound(channels.Telegram) {
			resp.Telegram = imChannelEntry{Bound: true, Config: channels.Telegram}
		}
		if channels.Slack != nil && isSlackBound(channels.Slack) {
			resp.Slack = imChannelEntry{Bound: true, Config: channels.Slack}
		}
		if channels.Line != nil && isLineBound(channels.Line) {
			resp.Line = imChannelEntry{Bound: true, Config: channels.Line}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func isTelegramBound(c *im.TelegramConfig) bool {
	return c.BotToken != "" || c.TokenFile != ""
}

func isSlackBound(c *im.SlackConfig) bool {
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
func imSave(w http.ResponseWriter, r *http.Request, patch bool) {
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
			updated, err := patchTelegramViaCLI(p)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, updated)
		} else {
			var c im.TelegramConfig
			if err := json.Unmarshal(req.Telegram, &c); err != nil {
				writeError(w, http.StatusBadRequest, "invalid telegram config: "+err.Error())
				return
			}
			if c.Enabled == nil {
				t := true
				c.Enabled = &t
			}
			if c.DmPolicy == "" {
				c.DmPolicy = "open"
			}
			if c.DmPolicy == "open" && len(c.AllowFrom) == 0 {
				c.AllowFrom = []string{"*"}
			}
			if c.GroupPolicy == "" {
				c.GroupPolicy = "open"
			}
			if err := oclaw.Set("channels.telegram", c); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if c.Enabled == nil || *c.Enabled {
				_ = oclaw.EnsureChannelEnabled("telegram")
			}
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
			updated, err := patchSlackViaCLI(p)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, updated)
		} else {
			var c im.SlackConfig
			if err := json.Unmarshal(req.Slack, &c); err != nil {
				writeError(w, http.StatusBadRequest, "invalid slack config: "+err.Error())
				return
			}
			if c.Enabled == nil {
				t := true
				c.Enabled = &t
			}
			if c.DmPolicy == "" {
				c.DmPolicy = "open"
			}
			if c.DmPolicy == "open" && len(c.AllowFrom) == 0 {
				c.AllowFrom = []string{"*"}
			}
			if c.GroupPolicy == "" {
				c.GroupPolicy = "open"
			}
			if err := oclaw.Set("channels.slack", c); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if c.Enabled == nil || *c.Enabled {
				_ = oclaw.EnsureChannelEnabled("slack")
			}
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
			updated, err := patchLineViaCLI(p)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, updated)
		} else {
			var c im.LineConfig
			if err := json.Unmarshal(req.Line, &c); err != nil {
				writeError(w, http.StatusBadRequest, "invalid line config: "+err.Error())
				return
			}
			if c.Enabled == nil {
				t := true
				c.Enabled = &t
			}
			if c.DmPolicy == "" {
				c.DmPolicy = "open"
			}
			if c.DmPolicy == "open" && len(c.AllowFrom) == 0 {
				c.AllowFrom = []string{"*"}
			}
			if c.GroupPolicy == "" {
				c.GroupPolicy = "open"
			}
			if err := oclaw.Set("channels.line", c); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if c.Enabled == nil || *c.Enabled {
				_ = oclaw.EnsureChannelEnabled("line")
			}
			writeJSON(w, http.StatusOK, c)
		}

	default:
		writeError(w, http.StatusBadRequest, "unknown channel: "+req.Channel+"; must be telegram | slack | line")
	}
}

// patchTelegramViaCLI applies each non-nil field via individual `openclaw config set` calls,
// then reads back the result. Patched fields are overlaid to restore values that
// `config get` may redact (e.g. botToken → "***").
func patchTelegramViaCLI(p im.PatchTelegramRequest) (*im.TelegramConfig, error) {
	const base = "channels.telegram."
	if p.Enabled != nil {
		if err := oclaw.Set(base+"enabled", *p.Enabled); err != nil {
			return nil, err
		}
	}
	if p.BotToken != nil {
		if err := oclaw.Set(base+"botToken", *p.BotToken); err != nil {
			return nil, err
		}
	}
	if p.TokenFile != nil {
		if err := oclaw.Set(base+"tokenFile", *p.TokenFile); err != nil {
			return nil, err
		}
	}
	if p.DmPolicy != nil {
		if err := oclaw.Set(base+"dmPolicy", *p.DmPolicy); err != nil {
			return nil, err
		}
	}
	if p.GroupPolicy != nil {
		if err := oclaw.Set(base+"groupPolicy", *p.GroupPolicy); err != nil {
			return nil, err
		}
	}
	if p.AllowFrom != nil {
		if err := oclaw.Set(base+"allowFrom", *p.AllowFrom); err != nil {
			return nil, err
		}
	}
	if p.WebhookURL != nil {
		if err := oclaw.Set(base+"webhookUrl", *p.WebhookURL); err != nil {
			return nil, err
		}
	}
	if p.WebhookSecret != nil {
		if err := oclaw.Set(base+"webhookSecret", *p.WebhookSecret); err != nil {
			return nil, err
		}
	}
	if p.WebhookPath != nil {
		if err := oclaw.Set(base+"webhookPath", *p.WebhookPath); err != nil {
			return nil, err
		}
	}
	if p.WebhookHost != nil {
		if err := oclaw.Set(base+"webhookHost", *p.WebhookHost); err != nil {
			return nil, err
		}
	}
	if p.WebhookPort != nil {
		if err := oclaw.Set(base+"webhookPort", *p.WebhookPort); err != nil {
			return nil, err
		}
	}
	if p.ConfigWrites != nil {
		if err := oclaw.Set(base+"configWrites", *p.ConfigWrites); err != nil {
			return nil, err
		}
	}

	cur, err := oclaw.GetAs[im.TelegramConfig]("channels.telegram")
	if err != nil || cur == nil {
		cur = &im.TelegramConfig{}
	}
	// Re-apply patch values so the response reflects actual new values
	// (config get redacts sensitive fields like botToken to "***").
	if p.Enabled != nil {
		cur.Enabled = p.Enabled
	}
	if p.BotToken != nil {
		cur.BotToken = *p.BotToken
	}
	if p.TokenFile != nil {
		cur.TokenFile = *p.TokenFile
	}
	if p.DmPolicy != nil {
		cur.DmPolicy = *p.DmPolicy
	}
	if p.GroupPolicy != nil {
		cur.GroupPolicy = *p.GroupPolicy
	}
	if p.AllowFrom != nil {
		cur.AllowFrom = *p.AllowFrom
	}
	if p.WebhookURL != nil {
		cur.WebhookURL = *p.WebhookURL
	}
	if p.WebhookSecret != nil {
		cur.WebhookSecret = *p.WebhookSecret
	}
	if p.WebhookPath != nil {
		cur.WebhookPath = *p.WebhookPath
	}
	if p.WebhookHost != nil {
		cur.WebhookHost = *p.WebhookHost
	}
	if p.WebhookPort != nil {
		cur.WebhookPort = p.WebhookPort
	}
	if p.ConfigWrites != nil {
		cur.ConfigWrites = p.ConfigWrites
	}
	return cur, nil
}

// patchSlackViaCLI applies each non-nil field via individual `openclaw config set` calls,
// then reads back the result with patched fields overlaid.
func patchSlackViaCLI(p im.PatchSlackRequest) (*im.SlackConfig, error) {
	const base = "channels.slack."
	if p.Enabled != nil {
		if err := oclaw.Set(base+"enabled", *p.Enabled); err != nil {
			return nil, err
		}
	}
	if p.Mode != nil {
		if err := oclaw.Set(base+"mode", *p.Mode); err != nil {
			return nil, err
		}
	}
	if p.BotToken != nil {
		if err := oclaw.Set(base+"botToken", *p.BotToken); err != nil {
			return nil, err
		}
	}
	if p.AppToken != nil {
		if err := oclaw.Set(base+"appToken", *p.AppToken); err != nil {
			return nil, err
		}
	}
	if p.SigningSecret != nil {
		if err := oclaw.Set(base+"signingSecret", *p.SigningSecret); err != nil {
			return nil, err
		}
	}
	if p.WebhookPath != nil {
		if err := oclaw.Set(base+"webhookPath", *p.WebhookPath); err != nil {
			return nil, err
		}
	}
	if p.UserToken != nil {
		if err := oclaw.Set(base+"userToken", *p.UserToken); err != nil {
			return nil, err
		}
	}
	if p.UserTokenReadOnly != nil {
		if err := oclaw.Set(base+"userTokenReadOnly", *p.UserTokenReadOnly); err != nil {
			return nil, err
		}
	}
	if p.DmPolicy != nil {
		if err := oclaw.Set(base+"dmPolicy", *p.DmPolicy); err != nil {
			return nil, err
		}
	}
	if p.GroupPolicy != nil {
		if err := oclaw.Set(base+"groupPolicy", *p.GroupPolicy); err != nil {
			return nil, err
		}
	}
	if p.AllowFrom != nil {
		if err := oclaw.Set(base+"allowFrom", *p.AllowFrom); err != nil {
			return nil, err
		}
	}
	if p.DangerouslyAllowNameMatching != nil {
		if err := oclaw.Set(base+"dangerouslyAllowNameMatching", *p.DangerouslyAllowNameMatching); err != nil {
			return nil, err
		}
	}
	if p.ReplyToMode != nil {
		if err := oclaw.Set(base+"replyToMode", *p.ReplyToMode); err != nil {
			return nil, err
		}
	}
	if p.TextChunkLimit != nil {
		if err := oclaw.Set(base+"textChunkLimit", *p.TextChunkLimit); err != nil {
			return nil, err
		}
	}
	if p.ChunkMode != nil {
		if err := oclaw.Set(base+"chunkMode", *p.ChunkMode); err != nil {
			return nil, err
		}
	}
	if p.MediaMaxMb != nil {
		if err := oclaw.Set(base+"mediaMaxMb", *p.MediaMaxMb); err != nil {
			return nil, err
		}
	}
	if p.AckReaction != nil {
		if err := oclaw.Set(base+"ackReaction", *p.AckReaction); err != nil {
			return nil, err
		}
	}
	if p.TypingReaction != nil {
		if err := oclaw.Set(base+"typingReaction", *p.TypingReaction); err != nil {
			return nil, err
		}
	}
	if p.Streaming != nil {
		if err := oclaw.Set(base+"streaming", *p.Streaming); err != nil {
			return nil, err
		}
	}
	if p.NativeStreaming != nil {
		if err := oclaw.Set(base+"nativeStreaming", *p.NativeStreaming); err != nil {
			return nil, err
		}
	}
	if p.ConfigWrites != nil {
		if err := oclaw.Set(base+"configWrites", *p.ConfigWrites); err != nil {
			return nil, err
		}
	}

	cur, err := oclaw.GetAs[im.SlackConfig]("channels.slack")
	if err != nil || cur == nil {
		cur = &im.SlackConfig{}
	}
	if p.Enabled != nil {
		cur.Enabled = p.Enabled
	}
	if p.Mode != nil {
		cur.Mode = *p.Mode
	}
	if p.BotToken != nil {
		cur.BotToken = *p.BotToken
	}
	if p.AppToken != nil {
		cur.AppToken = *p.AppToken
	}
	if p.SigningSecret != nil {
		cur.SigningSecret = *p.SigningSecret
	}
	if p.WebhookPath != nil {
		cur.WebhookPath = *p.WebhookPath
	}
	if p.UserToken != nil {
		cur.UserToken = *p.UserToken
	}
	if p.UserTokenReadOnly != nil {
		cur.UserTokenReadOnly = p.UserTokenReadOnly
	}
	if p.DmPolicy != nil {
		cur.DmPolicy = *p.DmPolicy
	}
	if p.GroupPolicy != nil {
		cur.GroupPolicy = *p.GroupPolicy
	}
	if p.AllowFrom != nil {
		cur.AllowFrom = *p.AllowFrom
	}
	if p.DangerouslyAllowNameMatching != nil {
		cur.DangerouslyAllowNameMatching = p.DangerouslyAllowNameMatching
	}
	if p.ReplyToMode != nil {
		cur.ReplyToMode = *p.ReplyToMode
	}
	if p.TextChunkLimit != nil {
		cur.TextChunkLimit = p.TextChunkLimit
	}
	if p.ChunkMode != nil {
		cur.ChunkMode = *p.ChunkMode
	}
	if p.MediaMaxMb != nil {
		cur.MediaMaxMb = p.MediaMaxMb
	}
	if p.AckReaction != nil {
		cur.AckReaction = *p.AckReaction
	}
	if p.TypingReaction != nil {
		cur.TypingReaction = *p.TypingReaction
	}
	if p.Streaming != nil {
		cur.Streaming = *p.Streaming
	}
	if p.NativeStreaming != nil {
		cur.NativeStreaming = p.NativeStreaming
	}
	if p.ConfigWrites != nil {
		cur.ConfigWrites = p.ConfigWrites
	}
	return cur, nil
}

// patchLineViaCLI applies each non-nil field via individual `openclaw config set` calls,
// then reads back the result with patched fields overlaid.
func patchLineViaCLI(p im.PatchLineRequest) (*im.LineConfig, error) {
	const base = "channels.line."
	if p.Enabled != nil {
		if err := oclaw.Set(base+"enabled", *p.Enabled); err != nil {
			return nil, err
		}
	}
	if p.ChannelAccessToken != nil {
		if err := oclaw.Set(base+"channelAccessToken", *p.ChannelAccessToken); err != nil {
			return nil, err
		}
	}
	if p.ChannelSecret != nil {
		if err := oclaw.Set(base+"channelSecret", *p.ChannelSecret); err != nil {
			return nil, err
		}
	}
	if p.TokenFile != nil {
		if err := oclaw.Set(base+"tokenFile", *p.TokenFile); err != nil {
			return nil, err
		}
	}
	if p.SecretFile != nil {
		if err := oclaw.Set(base+"secretFile", *p.SecretFile); err != nil {
			return nil, err
		}
	}
	if p.WebhookPath != nil {
		if err := oclaw.Set(base+"webhookPath", *p.WebhookPath); err != nil {
			return nil, err
		}
	}
	if p.DmPolicy != nil {
		if err := oclaw.Set(base+"dmPolicy", *p.DmPolicy); err != nil {
			return nil, err
		}
	}
	if p.AllowFrom != nil {
		if err := oclaw.Set(base+"allowFrom", *p.AllowFrom); err != nil {
			return nil, err
		}
	}
	if p.GroupPolicy != nil {
		if err := oclaw.Set(base+"groupPolicy", *p.GroupPolicy); err != nil {
			return nil, err
		}
	}
	if p.GroupAllowFrom != nil {
		if err := oclaw.Set(base+"groupAllowFrom", *p.GroupAllowFrom); err != nil {
			return nil, err
		}
	}

	cur, err := oclaw.GetAs[im.LineConfig]("channels.line")
	if err != nil || cur == nil {
		cur = &im.LineConfig{}
	}
	if p.Enabled != nil {
		cur.Enabled = p.Enabled
	}
	if p.ChannelAccessToken != nil {
		cur.ChannelAccessToken = *p.ChannelAccessToken
	}
	if p.ChannelSecret != nil {
		cur.ChannelSecret = *p.ChannelSecret
	}
	if p.TokenFile != nil {
		cur.TokenFile = *p.TokenFile
	}
	if p.SecretFile != nil {
		cur.SecretFile = *p.SecretFile
	}
	if p.WebhookPath != nil {
		cur.WebhookPath = *p.WebhookPath
	}
	if p.DmPolicy != nil {
		cur.DmPolicy = *p.DmPolicy
	}
	if p.AllowFrom != nil {
		cur.AllowFrom = *p.AllowFrom
	}
	if p.GroupPolicy != nil {
		cur.GroupPolicy = *p.GroupPolicy
	}
	if p.GroupAllowFrom != nil {
		cur.GroupAllowFrom = *p.GroupAllowFrom
	}
	return cur, nil
}

// --- DELETE ---

func imDelete(w http.ResponseWriter, r *http.Request) {
	channel := r.URL.Query().Get("channel")
	if channel == "" {
		writeError(w, http.StatusBadRequest, "query param 'channel' is required (telegram | slack | line)")
		return
	}

	var path string
	switch channel {
	case "telegram":
		path = "channels.telegram"
	case "slack":
		path = "channels.slack"
	case "line":
		path = "channels.line"
	default:
		writeError(w, http.StatusBadRequest, "unknown channel: "+channel+"; must be telegram | slack | line")
		return
	}

	if err := oclaw.Unset(path); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
