# 贡献一个探针节点 / Contributing a Probe Node

欢迎贡献你自己的服务器作为 Bangumi Status 的探针节点！多一个地理位置的视角能让仲裁更可靠，也能更快发现局部网络异常。

We welcome third-party probe nodes. Extra geographic vantage points make the quorum more reliable and surface region-specific outages faster.

---

## 简要流程 / Overview

1. 联系维护者获得专属 `INGEST_SECRET`
2. 在你的服务器上构建 probe 二进制
3. 准备你自己的 Bangumi 凭据（**用你自己的账号**）
4. 运行 `scripts/setup-probe.sh` 完成安装

---

## 1. 联系维护者 / Get Credentials from the Maintainer

通过 issue / 邮件 / Telegram 联系维护者，简单说明：
- 你的代号或组织名（用于生成 namespace 前缀，例如 `alice` `acme`）
- 大概打算部署几个节点、地理位置（用于公开展示）

维护者会回复：
- 一个分配给你的 `INGEST_SECRET`（token）
- 一个 `PROBE_ID` **前缀**，例如 `alice-`

之后你**自己决定**节点叫什么名字，只需以前缀开头即可：`alice-paris-1`、`alice-aws-tokyo`、`alice-anywhere` 都可以。加新节点不用再联系维护者。

`REGION` 必须是 **ISO 3166-1 alpha-2 国家代码**（小写两字母），如 `jp` 日本 / `cn` 中国 / `us` 美国 / `de` 德国 / `fr` 法国 / `sg` 新加坡 / `hk` 香港 / `tw` 台湾 / `kr` 韩国 / `gb` 英国。完整列表见 [ISO 3166-1](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)。aggregator 会拒绝非法值。

> **安全约束**：你的 token 只能上报以分配前缀开头的 `PROBE_ID`，不能冒用维护者或其他第三方的节点身份。

---

## 2. 获取 probe 二进制 / Get the Probe Binary

**推荐：直接下载预编译二进制（无需 Go 环境）**

前往 [GitHub Releases](https://github.com/maho0x/bangumi-status/releases/latest) 下载对应架构的文件：

```bash
# x86_64 (amd64)
curl -fsSL https://github.com/maho0x/bangumi-status/releases/latest/download/probe-linux-amd64 -o probe
# 或 ARM64 (如树莓派、Oracle ARM 实例等)
curl -fsSL https://github.com/maho0x/bangumi-status/releases/latest/download/probe-linux-arm64 -o probe

chmod +x probe
```

**备选：自行编译（需要 Go 1.21+）**

```bash
git clone https://github.com/maho0x/bangumi-status.git
cd bangumi-status
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o probe ./cmd/probe
```

---

## 3. 获取 Bangumi 凭据 / Obtain Bangumi Credentials

**重要：使用你自己的 Bangumi 账号**，不要向维护者索要 cookie。Bangumi 一般不封号，但你应当为自己的凭据负责。

> Use **your own** Bangumi account credentials. Do NOT ask the maintainer for theirs.

### `BGM_API_TOKEN`
访问 https://next.bgm.tv/demo/access-token ，登录后创建一个 personal access token，复制它的值。

### `BGM_COOKIE`
浏览器登录 https://bangumi.tv （或 bgm.tv / chii.in），在 DevTools 的 **Network** 面板里点任意一个页面请求，把 **`Cookie:`** 请求头的值**整行完整复制**出来。

### `BGM_USER_AGENT`
**一定要和当前登入设备的相同！！！**

### 完全不提供凭据也可以 / Or Skip Auth Entirely

如果你不想交出 cookie，可以同时省略 `BGM_COOKIE` 和 `BGM_API_TOKEN`（必须**两个都不填**才算 guest-only）。probe 会自动进入 **guest-only 模式**，只检测公开页面的可达性，不参与登录态检测。这种节点对公开页面的多地仲裁仍然有价值。

---

## 4. 安装 / Install

### 方式一：Docker（推荐）

```bash
docker run -d \
  --name bangumi-probe \
  --restart unless-stopped \
  -e PROBE_ID=alice-paris-1 \
  -e REGION=fr \
  -e AGG_URL=https://bgm-status.ry.mk/api/ingest \
  -e INGEST_SECRET=<你拿到的 token> \
  -e BGM_COOKIE='chii_auth=...; chii_sid=...' \
  -e BGM_USER_AGENT=<你的 USER_AGENT> \
  -e BGM_API_TOKEN=<你的 API token> \
  ghcr.io/maho0x/bangumi-status-probe:latest
```

查看日志：`docker logs -f bangumi-probe`

### 方式二：systemd

把构建好的 `probe` 二进制和仓库里的 `scripts/setup-probe.sh` 一起放到目标服务器，然后以 root 运行：

```bash
sudo \
  PROBE_ID=alice-paris-1 \
  REGION=fr \
  AGG_URL=https://bgm-status.ry.mk/api/ingest \
  INGEST_SECRET=<你拿到的 token> \
  BGM_COOKIE='chii_auth=...; chii_sid=...' \
  BGM_USER_AGENT=<你的 USER_AGENT> \
  BGM_API_TOKEN=<你的 API token> \
  bash scripts/setup-probe.sh
```

> `PROBE_ID` 必须以分配给你的前缀开头（例如前缀是 `alice-`，那么 `alice-paris-1` / `alice-aws-tokyo` 都可以）。`REGION` 必须是节点**实际所在国家**的 ISO 3166-1 alpha-2 代码（小写两字母），不能填 `eu` `asia` 这类大区。

脚本会：
- 把 `probe` 安装到 `/opt/bangumi-status/probe`
- 写凭据到 `/etc/bangumi-status/probe.env`（`chmod 600`）
- 安装并启动 `bangumi-probe.service`

systemd 服务以 `DynamicUser`（无固定 UID 的临时非特权用户）运行，并启用了完整沙箱：`ProtectSystem=strict`、`PrivateTmp`、`MemoryDenyWriteExecute`、`RestrictAddressFamilies=AF_INET AF_INET6`、`CapabilityBoundingSet=` 等。probe 进程**无法**写入文件系统、访问内核接口或获取任何额外权限。

查看日志：

```bash
journalctl -u bangumi-probe -f
```

正常运行时每 60 秒会打印一行 `tick: ok=N degraded=N down=N online=N`。

---

## 卸载 / Uninstall

```bash
sudo systemctl disable --now bangumi-probe.service
sudo rm -f /etc/systemd/system/bangumi-probe.service
sudo rm -rf /opt/bangumi-status /etc/bangumi-status
sudo systemctl daemon-reload
```

---

## 安全 / Security

- `INGEST_SECRET` 绑定到分配给你的 `PROBE_ID` 前缀，**只能上报前缀下的节点**，无法冒用维护者或其他第三方的节点身份；泄露后联系维护者吊销并换发，不影响他人
- `BGM_COOKIE` 和 `BGM_API_TOKEN` 仅用于本地 HTTP 请求，**不会**上报给 aggregator——只有探测结果（HTTP 状态码、延迟、错误信息）会被上传
- 二进制开源，可自行 `go build` 后审计 `internal/check/check.go` 确认行为
