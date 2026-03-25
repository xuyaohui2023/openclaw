# flashclaw-im-channel

IM 渠道配置管理服务。为 [openclaw](https://docs.openclaw.ai) 的 Telegram、Slack、LINE 三个 IM 渠道提供统一的 REST CRUD API，通过直接读写 `openclaw.json` 配置文件并发送 SIGUSR1 信号触发 openclaw 热重载，完成渠道配置的实时生效。

---

## 目录结构

```
flashclaw-im-channel/
├── cmd/server/main.go          # 程序入口，HTTP 服务器启动
├── configs/
│   ├── config-dev.json         # 开发环境配置
│   ├── config-test.json        # 测试环境配置
│   ├── config-pre.json         # 预发环境配置
│   └── config-prod.json        # 生产环境配置
├── internal/
│   ├── config/config.go        # 配置加载（环境变量 > 配置文件 > 内置默认值）
│   ├── handler/
│   │   ├── im_list.go          # 统一 IM 路由处理（GET/POST/PATCH/DELETE）
│   │   ├── health.go           # 健康检查
│   │   └── base.go             # 公共响应工具 + notifyReload
│   ├── im/
│   │   ├── types.go            # TelegramConfig / SlackConfig / LineConfig 及 Patch 类型
│   │   ├── configfile.go       # openclaw.json 读写（互斥锁 + 原子写）
│   │   └── backup.go           # 写前备份轮转（5 个槽位）
│   ├── middleware/auth.go      # X-Api-Key 认证中间件
│   └── openclaw/
│       ├── notify.go           # NotifyReload：SIGUSR1（直接 PID 或 PID 文件）
│       ├── notify_unix.go      # SIGUSR1 实现（Linux/macOS）
│       └── notify_windows.go   # SIGUSR1 存根（Windows，仅用于编译）
├── go.mod
└── README.md
```

---

## 技术栈

| 项目 | 版本 |
|------|------|
| Go | 1.24 |
| 标准库 `net/http` | — |

> 无第三方依赖。热重载通过 SIGUSR1 实现，不依赖 WebSocket 网关。

---

## REST API

### 认证

所有 `/api/v1/im` 接口均需在请求头携带 API Key：

```
X-Api-Key: <FLASHCLAW_API_KEY>
```

认证失败返回 `401 Unauthorized`。

---

### GET /api/v1/im — 查询所有渠道配置

返回三个渠道的绑定状态。已绑定渠道（凭证非空）返回完整配置，未绑定渠道 `bound=false`，`config` 字段省略。

**响应示例：**

```json
{
  "telegram": {
    "bound": true,
    "config": {
      "botToken": "123456:ABC-DEF...",
      "dmPolicy": "pairing"
    }
  },
  "slack": {
    "bound": true,
    "config": {
      "botToken": "xoxb-...",
      "appToken": "xapp-..."
    }
  },
  "line": {
    "bound": false
  }
}
```

---

### POST /api/v1/im — 创建/完整替换渠道配置

`channel` 字段指定目标渠道，对应嵌套对象提供完整配置（全量覆盖）。

**Telegram 示例：**

```json
{
  "channel": "telegram",
  "telegram": {
    "botToken": "123456:ABC-DEF..."
  }
}
```

**Slack 示例：**

```json
{
  "channel": "slack",
  "slack": {
    "botToken": "xoxb-...",
    "appToken": "xapp-..."
  }
}
```

**LINE 示例：**

```json
{
  "channel": "line",
  "line": {
    "channelAccessToken": "your-channel-access-token",
    "channelSecret": "your-channel-secret"
  }
}
```

**响应：** `200 OK`，返回写入后的完整配置对象。

---

### PATCH /api/v1/im — 部分更新渠道配置

仅更新请求体中出现的字段，未传字段保持原值不变。

**示例：仅更新 Telegram DM 策略**

```json
{
  "channel": "telegram",
  "telegram": {
    "dmPolicy": "open"
  }
}
```

**示例：清空 Slack allowFrom 列表**

```json
{
  "channel": "slack",
  "slack": {
    "allowFrom": []
  }
}
```

> `allowFrom: []` 表示清空；省略 `allowFrom` 字段则不修改。

**响应：** `200 OK`，返回更新后的完整配置对象。

---

### DELETE /api/v1/im?channel=\<name\> — 删除渠道配置

从 `openclaw.json` 中移除指定渠道的完整配置块。

```
DELETE /api/v1/im?channel=telegram
DELETE /api/v1/im?channel=slack
DELETE /api/v1/im?channel=line
```

**响应：** `204 No Content`（成功）/ `404 Not Found`（渠道不存在）。

---

### GET /health — 健康检查（无需认证）

```json
{"status": "ok"}
```

---

## 渠道配置字段说明

### Telegram

| 字段 | 类型 | 说明 |
|------|------|------|
| `botToken` | string | BotFather 颁发的 Bot Token（如 `123456:ABC-DEF...`）。与 `tokenFile` 二选一，必须提供其中一个 |
| `tokenFile` | string | 存放 Bot Token 的文件路径（拒绝软链接）。与 `botToken` 二选一 |
| `dmPolicy` | string | DM 发起策略：`pairing`（默认）/ `allowlist` / `open` / `disabled` |
| `allowFrom` | []string | 允许发起 DM 的 Telegram 数字 User ID 列表（`allowlist` 模式生效） |
| `webhookUrl` | string | 启用 Webhook 模式的公网 URL，设置时需同时提供 `webhookSecret` |
| `webhookSecret` | string | Webhook 签名密钥 |
| `webhookPath` | string | Webhook 本地监听路径（默认 `/telegram-webhook`） |
| `webhookHost` | string | Webhook 本地监听 Host（默认 `127.0.0.1`） |
| `webhookPort` | int | Webhook 本地监听端口（默认 `8787`） |
| `enabled` | bool | 启用/禁用该渠道（默认 `true`） |
| `configWrites` | bool | 是否允许通过 Telegram 命令修改 openclaw 配置 |

### Slack

| 字段 | 类型 | 说明 |
|------|------|------|
| `botToken` | string | Slack Bot Token（`xoxb-...`）。必填 |
| `appToken` | string | App-Level Token（`xapp-...`），需 `connections:write` 权限。Socket 模式必填 |
| `signingSecret` | string | 签名密钥。HTTP 模式必填 |
| `mode` | string | 连接模式：`socket`（默认）/ `http` |
| `webhookPath` | string | HTTP 模式事件接收路径（默认 `/slack/events`） |
| `userToken` | string | 可选用户 Token（`xoxp-...`），用于扩展读操作 |
| `userTokenReadOnly` | bool | 限制 userToken 只读（默认 `true`） |
| `dmPolicy` | string | DM 策略：`pairing`（默认）/ `allowlist` / `open` / `disabled` |
| `groupPolicy` | string | 群组消息策略：`allowlist`（默认）/ `open` / `disabled` |
| `allowFrom` | []string | DM/群组全局白名单（Slack User ID 或 `["*"]` 表示所有人） |
| `replyToMode` | string | 回复线程模式：`off`（默认）/ `first` / `all` |
| `textChunkLimit` | int | 单条消息最大字符数（默认 `4000`） |
| `chunkMode` | string | 长消息分割模式（`newline` 按段落分割） |
| `mediaMaxMb` | int | 入站附件大小上限 MB（默认 `20`） |
| `ackReaction` | string | 处理中 Emoji 表情（如 `hourglass`） |
| `typingReaction` | string | 打字中临时 Emoji 表情 |
| `streaming` | string | 实时预览：`partial`（默认）/ `off` / `block` / `progress` |
| `nativeStreaming` | bool | 使用 Slack 原生流式 API（默认 `true`） |
| `enabled` | bool | 启用/禁用该渠道（默认 `true`） |

### LINE

| 字段 | 类型 | 说明 |
|------|------|------|
| `channelAccessToken` | string | LINE Developers 控制台的 Channel Access Token。与 `tokenFile` 二选一 |
| `channelSecret` | string | LINE Developers 控制台的 Channel Secret。与 `secretFile` 二选一 |
| `tokenFile` | string | 存放 Channel Access Token 的文件路径（拒绝软链接） |
| `secretFile` | string | 存放 Channel Secret 的文件路径（拒绝软链接） |
| `webhookPath` | string | Webhook 本地监听路径（默认 `/line/webhook`） |
| `dmPolicy` | string | DM 策略：`pairing`（默认）/ `allowlist` / `open` / `disabled` |
| `allowFrom` | []string | 允许发起 DM 的 LINE User ID 列表 |
| `groupPolicy` | string | 群组访问策略：`open` / `allowlist` / `disabled` |
| `groupAllowFrom` | []string | 允许在群组中交互的 LINE User ID 列表 |
| `enabled` | bool | 启用/禁用该渠道（默认 `true`） |

---

## 配置说明

### 环境选择

通过环境变量 `FLASHCLAW_ENV` 选择配置文件（默认 `dev`）：

```bash
FLASHCLAW_ENV=prod ./flashclaw-im-channel
# 加载 configs/config-prod.json
```

配置加载优先级（高 → 低）：

```
环境变量  >  configs/config-{env}.json  >  内置默认值
```

### 配置文件字段

| 字段（JSON） | 环境变量 | 默认值 | 说明 |
|-------------|---------|--------|------|
| `bind` | `FLASHCLAW_BIND` | `127.0.0.1` | HTTP 监听地址 |
| `port` | `FLASHCLAW_PORT` | `18790` | HTTP 监听端口 |
| `apiKey` | `FLASHCLAW_API_KEY` | —（**必填**） | 接口认证密钥 |
| `openclawConfigPath` | `OPENCLAW_CONFIG_PATH` | `~/.openclaw/openclaw.json` | openclaw 配置文件路径 |
| `openclawPid` | `OPENCLAW_PID` | `0` | openclaw 进程 PID（静态指定，优先级最高） |
| `openclawPidFile` | `OPENCLAW_PID_FILE` | `<openclaw.json 同目录>/openclaw.pid` | openclaw PID 文件路径（自动读取） |

`openclawConfigPath` 自动推导顺序：`$OPENCLAW_STATE_DIR` > `$CLAWDBOT_STATE_DIR` > `~/.openclaw`。

### 配置文件目录

默认读取可执行文件工作目录下的 `configs/` 目录，可通过 `FLASHCLAW_CONFIG_DIR` 覆盖：

```bash
FLASHCLAW_CONFIG_DIR=/etc/flashclaw-im-channel/configs FLASHCLAW_ENV=prod ./flashclaw-im-channel
```

---

## 热重载机制

每次写入配置后，服务通过 SIGUSR1 通知 openclaw 热重载（无需重启 openclaw）：

```
1. openclawPid > 0      → 直接向该 PID 发送 SIGUSR1
2. openclawPidFile 可读 → 读取 PID 文件获取 PID，再发送 SIGUSR1
3. 均不可用            → 记录错误，配置已写入磁盘，openclaw 重启后自动生效
```

只需确保 openclaw 启动时写入 PID 文件，本服务零配置即可自动完成热重载。

openclaw 默认 PID 文件路径：`~/.openclaw/openclaw.pid`。

若 openclaw 未自动写入 PID 文件，可在 systemd unit 中补充：

```ini
[Service]
ExecStartPost=/bin/sh -c 'echo $MAINPID > /root/.openclaw/openclaw.pid'
```

---

## 配置写入安全

每次写入 `openclaw.json` 前，服务自动执行备份轮转（与 openclaw 备份策略一致）：

```
openclaw.json        ← 当前生效配置
openclaw.json.bak    ← 最近一次备份
openclaw.json.bak.1
openclaw.json.bak.2
openclaw.json.bak.3
openclaw.json.bak.4  ← 最旧备份（共保留 5 个槽位）
```

写入流程：备份轮转 → 临时文件写入 → `rename` 原子替换（防止写入中途崩溃导致配置损坏）。

---

## 部署

### 编译

```bash
go build -o flashclaw-im-channel ./cmd/server
```

### 生产环境启动

```bash
FLASHCLAW_ENV=prod ./flashclaw-im-channel
```

### systemd 示例

```ini
[Unit]
Description=flashclaw-im-channel IM config service
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/flashclaw-im-channel
ExecStart=/opt/flashclaw-im-channel/flashclaw-im-channel
Environment=FLASHCLAW_ENV=prod
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### 与 dubbo-api-svc 对接

dubbo-api-svc 通过以下配置项访问本服务：

```yaml
# application-dev.yml
flashclaw:
  im:
    apikey: dev_2bad74bcf5a4457bb4d38261f2b01e8c

# application-test.yml
flashclaw:
  im:
    apikey: test_2bad74bcf5a4457bb4d38261f2b01e8c

# application-pre.yml
flashclaw:
  im:
    apikey: pre_2bad74bcf5a4457bb4d38261f2b01e8c

# application-prod.yml
flashclaw:
  im:
    apikey: prod_2bad74bcf5a4457bb4d38261f2b01e8c
```

本服务监听在 `127.0.0.1:18790`，dubbo-api-svc 通过内网直连。

---

## 注意事项

- `openclawConfigPath` 所在目录须对本服务进程可读写
- 各环境 `apiKey` 已在对应配置文件中配置，与 dubbo-api-svc 各环境的 `flashclaw.im.apikey` 保持一致
- 本服务与 openclaw 直接共享配置文件，热重载通过 SIGUSR1 完成，不经过任何网络调用
