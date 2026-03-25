package im

// TelegramConfig mirrors the openclaw channels.telegram config shape.
// Reference: https://docs.openclaw.ai/channels/telegram
type TelegramConfig struct {
	// Enabled activates or deactivates the channel on startup.
	Enabled *bool `json:"enabled,omitempty"`
	// BotToken is the bot token from BotFather (e.g. "123456:ABC-DEF...").
	BotToken string `json:"botToken,omitempty"`
	// TokenFile is the path to a regular file containing the bot token (symlinks rejected).
	// Use as an alternative to BotToken.
	TokenFile string `json:"tokenFile,omitempty"`
	// DmPolicy controls who can initiate DMs: pairing | allowlist | open | disabled.
	DmPolicy string `json:"dmPolicy,omitempty"`
	// AllowFrom lists Telegram numeric user IDs allowed to send DMs (used with allowlist policy).
	AllowFrom []string `json:"allowFrom,omitempty"`
	// WebhookURL enables webhook mode. Requires WebhookSecret when set.
	WebhookURL string `json:"webhookUrl,omitempty"`
	// WebhookSecret is required when WebhookURL is configured.
	WebhookSecret string `json:"webhookSecret,omitempty"`
	// WebhookPath is the local HTTP path for the webhook (default: /telegram-webhook).
	WebhookPath string `json:"webhookPath,omitempty"`
	// WebhookHost is the bind host for the webhook server (default: 127.0.0.1).
	WebhookHost string `json:"webhookHost,omitempty"`
	// WebhookPort is the bind port for the webhook server (default: 8787).
	WebhookPort *int `json:"webhookPort,omitempty"`
	// ConfigWrites controls whether /config set is allowed from Telegram chat commands.
	ConfigWrites *bool `json:"configWrites,omitempty"`
}

// SlackConfig mirrors the openclaw channels.slack config shape.
// Reference: https://docs.openclaw.ai/channels/slack
type SlackConfig struct {
	// Enabled activates or deactivates the channel on startup.
	Enabled *bool `json:"enabled,omitempty"`

	// --- Connection ---

	// Mode is the connection mode: socket | http (default: socket).
	Mode string `json:"mode,omitempty"`
	// BotToken is the Slack bot token (xoxb-...). Required for both modes.
	// Env fallback: SLACK_BOT_TOKEN (default account only).
	BotToken string `json:"botToken,omitempty"`
	// AppToken is the app-level token (xapp-...) with connections:write scope.
	// Required for socket mode. Env fallback: SLACK_APP_TOKEN (default account only).
	AppToken string `json:"appToken,omitempty"`
	// SigningSecret validates incoming HTTP event payloads. Required for http mode.
	SigningSecret string `json:"signingSecret,omitempty"`
	// WebhookPath is the HTTP endpoint for Slack event deliveries (default: /slack/events).
	// Each multi-account http-mode account must use a unique path.
	WebhookPath string `json:"webhookPath,omitempty"`

	// --- User token ---

	// UserToken is an optional user token (xoxp-...) for extended read operations.
	// No env fallback — must be set in config.
	UserToken string `json:"userToken,omitempty"`
	// UserTokenReadOnly restricts the user token to read-only operations (default: true).
	UserTokenReadOnly *bool `json:"userTokenReadOnly,omitempty"`

	// --- Access control ---

	// DmPolicy controls who can initiate DMs: pairing | allowlist | open | disabled (default: pairing).
	DmPolicy string `json:"dmPolicy,omitempty"`
	// GroupPolicy controls channel message handling: open | allowlist | disabled (default: allowlist).
	GroupPolicy string `json:"groupPolicy,omitempty"`
	// AllowFrom is the global DM/channel allowlist (Slack user IDs, or ["*"] for everyone).
	AllowFrom []string `json:"allowFrom,omitempty"`
	// DangerouslyAllowNameMatching enables username/display-name based routing (use sparingly).
	DangerouslyAllowNameMatching *bool `json:"dangerouslyAllowNameMatching,omitempty"`

	// --- Message delivery ---

	// ReplyToMode controls reply threading: off | first | all (default: off).
	ReplyToMode string `json:"replyToMode,omitempty"`
	// TextChunkLimit is the maximum characters per outbound message (default: 4000).
	TextChunkLimit *int `json:"textChunkLimit,omitempty"`
	// ChunkMode controls how long responses are split: "newline" splits on paragraph boundaries.
	ChunkMode string `json:"chunkMode,omitempty"`
	// MediaMaxMb is the inbound attachment size cap in MB (default: 20).
	MediaMaxMb *int `json:"mediaMaxMb,omitempty"`

	// --- Reactions & streaming ---

	// AckReaction is the emoji shortcode shown while a request is being processed.
	AckReaction string `json:"ackReaction,omitempty"`
	// TypingReaction is a temporary reaction displayed during processing.
	TypingReaction string `json:"typingReaction,omitempty"`
	// Streaming controls live response previews: off | partial | block | progress (default: partial).
	Streaming string `json:"streaming,omitempty"`
	// NativeStreaming uses Slack's native streaming API when available (default: true).
	NativeStreaming *bool `json:"nativeStreaming,omitempty"`

	// --- Misc ---

	// ConfigWrites enables automatic channel config migration on channel_id_changed events.
	ConfigWrites *bool `json:"configWrites,omitempty"`
}

// LineConfig mirrors the openclaw channels.line config shape.
// Reference: https://docs.openclaw.ai/channels/line
type LineConfig struct {
	// Enabled activates or deactivates the channel on startup.
	Enabled *bool `json:"enabled,omitempty"`
	// ChannelAccessToken is the channel access token from LINE Developers console.
	ChannelAccessToken string `json:"channelAccessToken,omitempty"`
	// ChannelSecret is the channel secret from LINE Developers console.
	ChannelSecret string `json:"channelSecret,omitempty"`
	// TokenFile is the path to a file containing the channel access token (symlinks rejected).
	TokenFile string `json:"tokenFile,omitempty"`
	// SecretFile is the path to a file containing the channel secret (symlinks rejected).
	SecretFile string `json:"secretFile,omitempty"`
	// WebhookPath is the local HTTP path for LINE webhook events (default: /line/webhook).
	WebhookPath string `json:"webhookPath,omitempty"`
	// DmPolicy controls who can initiate DMs: pairing | allowlist | open | disabled.
	DmPolicy string `json:"dmPolicy,omitempty"`
	// AllowFrom lists LINE user IDs allowed to send DMs (used with allowlist policy).
	AllowFrom []string `json:"allowFrom,omitempty"`
	// GroupPolicy controls group message access: open | allowlist | disabled.
	GroupPolicy string `json:"groupPolicy,omitempty"`
	// GroupAllowFrom lists LINE user IDs allowed to interact in groups.
	GroupAllowFrom []string `json:"groupAllowFrom,omitempty"`
}

// PatchTelegramRequest is the body for PATCH /api/v1/im when channel=telegram.
// Only non-nil fields are applied; absent fields are left unchanged.
type PatchTelegramRequest struct {
	Enabled       *bool      `json:"enabled"`
	BotToken      *string    `json:"botToken"`
	TokenFile     *string    `json:"tokenFile"`
	DmPolicy      *string    `json:"dmPolicy"`
	AllowFrom     *[]string  `json:"allowFrom"`
	WebhookURL    *string    `json:"webhookUrl"`
	WebhookSecret *string    `json:"webhookSecret"`
	WebhookPath   *string    `json:"webhookPath"`
	WebhookHost   *string    `json:"webhookHost"`
	WebhookPort   *int       `json:"webhookPort"`
	ConfigWrites  *bool      `json:"configWrites"`
}

// PatchSlackRequest is the body for PATCH /api/v1/im when channel=slack.
// Only non-nil fields are applied; absent fields are left unchanged.
type PatchSlackRequest struct {
	Enabled                      *bool      `json:"enabled"`
	Mode                         *string    `json:"mode"`
	BotToken                     *string    `json:"botToken"`
	AppToken                     *string    `json:"appToken"`
	SigningSecret                *string    `json:"signingSecret"`
	WebhookPath                  *string    `json:"webhookPath"`
	UserToken                    *string    `json:"userToken"`
	UserTokenReadOnly            *bool      `json:"userTokenReadOnly"`
	DmPolicy                     *string    `json:"dmPolicy"`
	GroupPolicy                  *string    `json:"groupPolicy"`
	AllowFrom                    *[]string  `json:"allowFrom"`
	DangerouslyAllowNameMatching *bool      `json:"dangerouslyAllowNameMatching"`
	ReplyToMode                  *string    `json:"replyToMode"`
	TextChunkLimit               *int       `json:"textChunkLimit"`
	ChunkMode                    *string    `json:"chunkMode"`
	MediaMaxMb                   *int       `json:"mediaMaxMb"`
	AckReaction                  *string    `json:"ackReaction"`
	TypingReaction               *string    `json:"typingReaction"`
	Streaming                    *string    `json:"streaming"`
	NativeStreaming               *bool      `json:"nativeStreaming"`
	ConfigWrites                 *bool      `json:"configWrites"`
}

// PatchLineRequest is the body for PATCH /api/v1/im when channel=line.
// Only non-nil fields are applied; absent fields are left unchanged.
type PatchLineRequest struct {
	Enabled            *bool      `json:"enabled"`
	ChannelAccessToken *string    `json:"channelAccessToken"`
	ChannelSecret      *string    `json:"channelSecret"`
	TokenFile          *string    `json:"tokenFile"`
	SecretFile         *string    `json:"secretFile"`
	WebhookPath        *string    `json:"webhookPath"`
	DmPolicy           *string    `json:"dmPolicy"`
	AllowFrom          *[]string  `json:"allowFrom"`
	GroupPolicy        *string    `json:"groupPolicy"`
	GroupAllowFrom     *[]string  `json:"groupAllowFrom"`
}
