// Package im handles reading and writing the openclaw IM configuration
// sections inside ~/.openclaw/openclaw.json.
//
// All writes use an in-process mutex to prevent concurrent modification.
package im

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// fileMu serialises all config file accesses within this process.
var fileMu sync.Mutex

// readRaw reads and JSON-parses the openclaw config file into a generic map.
func readRaw(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return m, nil
}

// writeRaw atomically writes the config map back to path.
// It backs up the existing file, then writes to a temp file and renames.
func writeRaw(path string, m map[string]interface{}) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	// Back up before overwriting — same rotation scheme as openclaw.
	maintainConfigBackups(path)

	tmp, err := os.CreateTemp(dir, ".openclaw-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp→config: %w", err)
	}
	return nil
}

// channels returns (creating if necessary) the root "channels" map.
func channels(m map[string]interface{}) map[string]interface{} {
	if v, ok := m["channels"]; ok {
		if ch, ok := v.(map[string]interface{}); ok {
			return ch
		}
	}
	ch := map[string]interface{}{}
	m["channels"] = ch
	return ch
}

// getIMSection returns the IM-specific sub-map (e.g. channels.telegram).
// Returns nil if the section does not exist.
func getIMSection(m map[string]interface{}, key string) map[string]interface{} {
	ch := channels(m)
	if v, ok := ch[key]; ok {
		if sub, ok := v.(map[string]interface{}); ok {
			return sub
		}
	}
	return nil
}

// setIMSection replaces the named IM section entirely.
func setIMSection(m map[string]interface{}, key string, section interface{}) {
	channels(m)[key] = section
}

// deleteIMSection removes the named IM section.
func deleteIMSection(m map[string]interface{}, key string) bool {
	ch := channels(m)
	if _, exists := ch[key]; !exists {
		return false
	}
	delete(ch, key)
	return true
}

// unmarshalSection decodes a raw map into a typed struct.
func unmarshalSection[T any](raw map[string]interface{}) (*T, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// --- Telegram ---

// GetTelegram reads the current channels.telegram section.
// Returns an empty config (not an error) if the section is absent.
func GetTelegram(path string) (*TelegramConfig, error) {
	fileMu.Lock()
	defer fileMu.Unlock()
	m, err := readRaw(path)
	if err != nil {
		return nil, err
	}
	sec := getIMSection(m, "telegram")
	if sec == nil {
		return &TelegramConfig{}, nil
	}
	return unmarshalSection[TelegramConfig](sec)
}

// SetTelegram replaces channels.telegram entirely.
func SetTelegram(path string, cfg TelegramConfig) error {
	fileMu.Lock()
	defer fileMu.Unlock()
	m, err := readRaw(path)
	if err != nil {
		return err
	}
	setIMSection(m, "telegram", cfg)
	return writeRaw(path, m)
}

// PatchTelegram applies a partial update to channels.telegram.
// Only fields with non-nil pointers in req are applied.
func PatchTelegram(path string, req PatchTelegramRequest) (*TelegramConfig, error) {
	fileMu.Lock()
	defer fileMu.Unlock()
	m, err := readRaw(path)
	if err != nil {
		return nil, err
	}
	sec := getIMSection(m, "telegram")
	if sec == nil {
		sec = map[string]interface{}{}
	}
	cur, err := unmarshalSection[TelegramConfig](sec)
	if err != nil {
		return nil, err
	}
	if req.Enabled != nil {
		cur.Enabled = req.Enabled
	}
	if req.BotToken != nil {
		cur.BotToken = *req.BotToken
	}
	if req.TokenFile != nil {
		cur.TokenFile = *req.TokenFile
	}
	if req.DmPolicy != nil {
		cur.DmPolicy = *req.DmPolicy
	}
	if req.AllowFrom != nil {
		cur.AllowFrom = *req.AllowFrom
	}
	if req.WebhookURL != nil {
		cur.WebhookURL = *req.WebhookURL
	}
	if req.WebhookSecret != nil {
		cur.WebhookSecret = *req.WebhookSecret
	}
	if req.WebhookPath != nil {
		cur.WebhookPath = *req.WebhookPath
	}
	if req.WebhookHost != nil {
		cur.WebhookHost = *req.WebhookHost
	}
	if req.WebhookPort != nil {
		cur.WebhookPort = req.WebhookPort
	}
	if req.ConfigWrites != nil {
		cur.ConfigWrites = req.ConfigWrites
	}
	setIMSection(m, "telegram", cur)
	return cur, writeRaw(path, m)
}

// DeleteTelegram removes channels.telegram.
func DeleteTelegram(path string) error {
	fileMu.Lock()
	defer fileMu.Unlock()
	m, err := readRaw(path)
	if err != nil {
		return err
	}
	if !deleteIMSection(m, "telegram") {
		return fmt.Errorf("channels.telegram not found")
	}
	return writeRaw(path, m)
}

// --- Slack ---

// GetSlack reads the current channels.slack section.
func GetSlack(path string) (*SlackConfig, error) {
	fileMu.Lock()
	defer fileMu.Unlock()
	m, err := readRaw(path)
	if err != nil {
		return nil, err
	}
	sec := getIMSection(m, "slack")
	if sec == nil {
		return &SlackConfig{}, nil
	}
	return unmarshalSection[SlackConfig](sec)
}

// SetSlack replaces channels.slack entirely.
func SetSlack(path string, cfg SlackConfig) error {
	fileMu.Lock()
	defer fileMu.Unlock()
	m, err := readRaw(path)
	if err != nil {
		return err
	}
	setIMSection(m, "slack", cfg)
	return writeRaw(path, m)
}

// PatchSlack applies a partial update to channels.slack.
func PatchSlack(path string, req PatchSlackRequest) (*SlackConfig, error) {
	fileMu.Lock()
	defer fileMu.Unlock()
	m, err := readRaw(path)
	if err != nil {
		return nil, err
	}
	sec := getIMSection(m, "slack")
	if sec == nil {
		sec = map[string]interface{}{}
	}
	cur, err := unmarshalSection[SlackConfig](sec)
	if err != nil {
		return nil, err
	}
	if req.Enabled != nil {
		cur.Enabled = req.Enabled
	}
	if req.Mode != nil {
		cur.Mode = *req.Mode
	}
	if req.BotToken != nil {
		cur.BotToken = *req.BotToken
	}
	if req.AppToken != nil {
		cur.AppToken = *req.AppToken
	}
	if req.SigningSecret != nil {
		cur.SigningSecret = *req.SigningSecret
	}
	if req.WebhookPath != nil {
		cur.WebhookPath = *req.WebhookPath
	}
	if req.UserToken != nil {
		cur.UserToken = *req.UserToken
	}
	if req.UserTokenReadOnly != nil {
		cur.UserTokenReadOnly = req.UserTokenReadOnly
	}
	if req.DmPolicy != nil {
		cur.DmPolicy = *req.DmPolicy
	}
	if req.GroupPolicy != nil {
		cur.GroupPolicy = *req.GroupPolicy
	}
	if req.AllowFrom != nil {
		cur.AllowFrom = *req.AllowFrom
	}
	if req.DangerouslyAllowNameMatching != nil {
		cur.DangerouslyAllowNameMatching = req.DangerouslyAllowNameMatching
	}
	if req.ReplyToMode != nil {
		cur.ReplyToMode = *req.ReplyToMode
	}
	if req.TextChunkLimit != nil {
		cur.TextChunkLimit = req.TextChunkLimit
	}
	if req.ChunkMode != nil {
		cur.ChunkMode = *req.ChunkMode
	}
	if req.MediaMaxMb != nil {
		cur.MediaMaxMb = req.MediaMaxMb
	}
	if req.AckReaction != nil {
		cur.AckReaction = *req.AckReaction
	}
	if req.TypingReaction != nil {
		cur.TypingReaction = *req.TypingReaction
	}
	if req.Streaming != nil {
		cur.Streaming = *req.Streaming
	}
	if req.NativeStreaming != nil {
		cur.NativeStreaming = req.NativeStreaming
	}
	if req.ConfigWrites != nil {
		cur.ConfigWrites = req.ConfigWrites
	}
	setIMSection(m, "slack", cur)
	return cur, writeRaw(path, m)
}

// DeleteSlack removes channels.slack.
func DeleteSlack(path string) error {
	fileMu.Lock()
	defer fileMu.Unlock()
	m, err := readRaw(path)
	if err != nil {
		return err
	}
	if !deleteIMSection(m, "slack") {
		return fmt.Errorf("channels.slack not found")
	}
	return writeRaw(path, m)
}

// --- Line ---

// GetLine reads the current channels.line section.
func GetLine(path string) (*LineConfig, error) {
	fileMu.Lock()
	defer fileMu.Unlock()
	m, err := readRaw(path)
	if err != nil {
		return nil, err
	}
	sec := getIMSection(m, "line")
	if sec == nil {
		return &LineConfig{}, nil
	}
	return unmarshalSection[LineConfig](sec)
}

// SetLine replaces channels.line entirely.
func SetLine(path string, cfg LineConfig) error {
	fileMu.Lock()
	defer fileMu.Unlock()
	m, err := readRaw(path)
	if err != nil {
		return err
	}
	setIMSection(m, "line", cfg)
	return writeRaw(path, m)
}

// PatchLine applies a partial update to channels.line.
func PatchLine(path string, req PatchLineRequest) (*LineConfig, error) {
	fileMu.Lock()
	defer fileMu.Unlock()
	m, err := readRaw(path)
	if err != nil {
		return nil, err
	}
	sec := getIMSection(m, "line")
	if sec == nil {
		sec = map[string]interface{}{}
	}
	cur, err := unmarshalSection[LineConfig](sec)
	if err != nil {
		return nil, err
	}
	if req.Enabled != nil {
		cur.Enabled = req.Enabled
	}
	if req.ChannelAccessToken != nil {
		cur.ChannelAccessToken = *req.ChannelAccessToken
	}
	if req.ChannelSecret != nil {
		cur.ChannelSecret = *req.ChannelSecret
	}
	if req.TokenFile != nil {
		cur.TokenFile = *req.TokenFile
	}
	if req.SecretFile != nil {
		cur.SecretFile = *req.SecretFile
	}
	if req.WebhookPath != nil {
		cur.WebhookPath = *req.WebhookPath
	}
	if req.DmPolicy != nil {
		cur.DmPolicy = *req.DmPolicy
	}
	if req.AllowFrom != nil {
		cur.AllowFrom = *req.AllowFrom
	}
	if req.GroupPolicy != nil {
		cur.GroupPolicy = *req.GroupPolicy
	}
	if req.GroupAllowFrom != nil {
		cur.GroupAllowFrom = *req.GroupAllowFrom
	}
	setIMSection(m, "line", cur)
	return cur, writeRaw(path, m)
}

// DeleteLine removes channels.line.
func DeleteLine(path string) error {
	fileMu.Lock()
	defer fileMu.Unlock()
	m, err := readRaw(path)
	if err != nil {
		return err
	}
	if !deleteIMSection(m, "line") {
		return fmt.Errorf("channels.line not found")
	}
	return writeRaw(path, m)
}
