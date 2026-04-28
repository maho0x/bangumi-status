# Bangumi Status

独立的 Bangumi 可用性监控系统，从多个地区的探针节点并发检测 bgm.tv、bangumi.tv、chii.in 及 Bangumi API 的可用状态，聚合结果后展示在状态页面并通过 Telegram 推送告警。

An independent availability monitor for [Bangumi](https://bgm.tv), probing bgm.tv, bangumi.tv, chii.in, next.bgm.tv, and api.bgm.tv from multiple geographic regions.

---

## 架构 / Architecture

```
[probe @ region A] ──┐
[probe @ region B] ──┼──► [aggregator] ──► PostgreSQL
[probe @ region C] ──┘         │
                                ├──► Status Page (SPA)
                                ├──► JSON / Atom API
                                └──► Telegram notifications
```

两个二进制文件共享 `internal/` 包：

- **`cmd/aggregator`** — 中心 HTTP 服务器，接收探针上报、存入 PostgreSQL、服务前端 SPA 和 JSON/Atom API、发送 Telegram 告警。
- **`cmd/probe`** — 轻量探针，部署在各远端节点，每 60 秒执行一次检测并 POST 结果到聚合器。

Two binaries sharing the `internal/` packages:

- **`cmd/aggregator`** — Central HTTP server. Receives probe results, stores them in PostgreSQL, serves the SPA frontend and JSON/Atom APIs, and sends Telegram notifications on outages.
- **`cmd/probe`** — Lightweight agent deployed to remote hosts. Checks all components every 60 s and POSTs results to the aggregator's `/api/ingest`.

## 功能 / Features

- 多区域探针，基于 **≥3 探针仲裁**防止单点误报
- 同时检测 Guest（公开访问）和 Auth（需登录）两种状态
- 延迟 >5s 标记为 Degraded，无响应标记为 Down
- 30 天历史正常率图表 + 14 天事件列表
- Atom Feed 订阅
- 中英双语 UI，每 30 秒自动刷新
- Telegram 机器人告警（连续 2 次异常后触发，含固定状态消息）
- 在线人数图表（从 bangumi.tv 抓取）

Multi-region probes with **≥3-probe quorum** to eliminate false positives. Detects both Guest and Authenticated access. 30-day uptime history, 14-day incident list, Atom feed, bilingual UI, and Telegram alerts.

## 快速开始 / Getting Started

### 前置依赖 / Prerequisites

- Go 1.21+
- PostgreSQL 14+

### 编译 / Build

```bash
bash deploy/build.sh
# 输出 / outputs: dist/aggregator-linux-amd64, dist/probe-linux-amd64, dist/probe-linux-arm64
```

### 部署聚合器 / Deploy Aggregator

1. 将 `dist/aggregator-linux-amd64` 复制到服务器

2. 创建环境变量文件 `/etc/bangumi-status/aggregator.env`：

```env
INGEST_SECRET=<长随机字符串 / long random string>
DB_DSN=postgres://user:pass@localhost:5432/bangumi_status?sslmode=disable

# 可选 / optional
TELEGRAM_BOT_TOKEN=<bot token>
TELEGRAM_CHAT_ID=<chat id>
STATUS_PAGE_URL=https://your-domain.example

# 第三方探针 token：每个第三方一个 token，绑定一个 PROBE_ID 前缀
# 格式：token:prefix;token2:prefix2
# 第三方上报时 PROBE_ID 必须以分配给他的前缀开头，可在前缀内自由命名
# Per-operator tokens for third-party nodes — bound to a probe-id prefix
INGEST_SECRETS=tok_aaa:alice-;tok_bbb:bob-
```

3. 参考 `deploy/aggregator.service` 配置 systemd 服务

### 部署探针 / Deploy Probe

1. 将 `dist/probe-linux-amd64`（或 arm64）复制到探针主机

2. 创建环境变量文件 `/etc/bangumi-status/probe.env`：

```env
PROBE_ID=tokyo-1
PROBE_REGION=jp
AGGREGATOR_URL=https://your-domain.example/api/ingest
INGEST_SECRET=<与聚合器相同 / same as aggregator>

# 检测登录状态需要 / required for Auth checks
BGM_COOKIE=<bangumi cookie string>
BGM_API_TOKEN=<bangumi API token>
BGM_USER_AGENT=Mozilla/5.0 ...
```

3. 参考 `deploy/probe.service` 配置 systemd 服务

### 使用部署脚本 / Using Deploy Scripts

```bash
# 部署聚合器
bash deploy/deploy-aggregator.sh

# 部署探针到远端主机
bash deploy/deploy-probe.sh <ssh-host> <probe-id> <region> [arch]
# 例如 / e.g.:
bash deploy/deploy-probe.sh my-server tokyo-1 jp amd64

# 一键构建并部署所有节点
INGEST_SECRET=... bash deploy/deploy-all.sh
```

## API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/ingest` | Probe ingestion (Bearer token required) |
| `GET` | `/api/status` | Current status (20s HTTP cache) |
| `GET` | `/api/probes` | Registered probe list |
| `GET` | `/api/health` | Store stats |
| `GET` | `/api/feed.atom` | Atom feed (last 50 incidents) |
| `GET` | `/` | SPA frontend |

## 贡献探针节点 / Contributing a Probe Node

欢迎贡献你自己的服务器作为探针节点！详细步骤见 [CONTRIBUTING-PROBE.md](CONTRIBUTING-PROBE.md)。

第三方节点使用各自的 Bangumi 账号凭据和独立的 ingest token，可单独吊销。如不愿意提供凭据，也可以以 guest-only 模式运行（仅检测公开页面）。

Want to contribute a probe node? See [CONTRIBUTING-PROBE.md](CONTRIBUTING-PROBE.md). Third-party nodes use their own Bangumi credentials and a per-probe ingest token, or can run in guest-only mode without any credentials.

## License

MIT
