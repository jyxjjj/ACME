# DESMG ACME Service

一个轻量级的 ACME 客户端与证书分发 HTTP JSON API，设计用于自动化 SSL/TLS 证书的申请、续期并向受信任的边缘/应用服务器分发证书。

本项目的初衷是解决在跨国/跨地区通过 rsync 等方式同步证书时遇到的网络延迟、丢包和连接失败问题：由一台主机管理 ACME 流程并通过 HTTPS API 将证书安全下发给其他主机。

## 主要功能

- 自动注册/加载 ACME 账户（使用 P-384 EC 密钥）。
- 使用 Cloudflare DNS（DNS-01）自动完成域名验证并申请证书（使用 certmagic + acmez）。
- 保存账户与证书到本地磁盘（默认路径见下）。
- 提供带授权验证的 HTTPS JSON API：`/cert` 获取证书 PEM，`/key` 获取私钥 PEM。
- 简单的守护进程：启动后会立即运行一次证书管理任务，并在每天 01:00 触发检查/续期流程。

## 快速前提

请确保你已经准备好：

- 一个 Cloudflare 帐号并为目标域启用了 Cloudflare DNS 管理权限（项目使用 Cloudflare 的 API Token）。
- Cloudflare Access（Zero Trust）已配置 —— API 的访问使用 Cloudflare Access JWT（详见下文）。
- 在运行机器上安装 Go（用于编译），或使用仓库提供的二进制文件。

## 环境变量（必填）

在启动程序前，需设置以下环境变量：

- `CF_ZT_ORG_NAME`：Cloudflare Zero Trust 域名（例如 `myteam`），用于 JWT 验证的证书地址：`https://{org}.cloudflareaccess.com/cdn-cgi/access/certs`。
- `CF_ZT_AUD`：Cloudflare Access 策略的 Client ID / Audience，用于验证 JWT 的 ClientID（Verifier 的 ClientID）。
- `ACME_EMAIL`：注册 ACME 账户使用的邮箱（必须为合法邮箱）。
- `CF_API_TOKEN`：Cloudflare API Token，用于通过 DNS-01 自动创建/删除 TXT 记录。
- `CERT_DOMAIN`：要管理的域名，支持多个域名以逗号分隔（例如：`example.com,www.example.com`）。第一个域名也用于存放 cert 目录名。

可选环境变量：

- `ADDR`：监听地址，默认 `0.0.0.0`。
- `PORT`：监听端口，默认 `9504`。

安全提示：私钥与账户 key 会以文件形式保存在磁盘（权限 0600），请确保文件系统与访问权限安全。

## 默认行为与注意事项

- 代码默认使用 Let's Encrypt 的 Staging CA（在代码中 `CADirectory = certmagic.LetsEncryptStagingCA`）。
- 在生产环境请在编译/配置时将其改为 `certmagic.LetsEncryptProductionCA` 或修改源码后重新编译。
- 程序默认每天会在 01:00 检查并执行一次证书管理（renew/obtain）。启动时会立即执行一次任务。

## 编译与运行

示例（在仓库根目录）：

```bash
make
```

在生产上，建议使用 systemd 管理（示例 service 文件请根据项目或系统自行创建）。示例命令（将二进制与 service 文件放到合适位置）：

```bash
ln -sf /www/server/acme/acme.service /etc/systemd/system/acme.service
systemctl daemon-reload
systemctl enable --now acme.service
systemctl status acme.service
```

## HTTP API（证书分发）

程序使用 Gin 提供两个只读接口：

- `GET /cert` —— 返回证书 PEM（完整链），JSON 格式。
- `GET /key`  —— 返回私钥 PEM，JSON 格式。

安全：两个接口都需要通过 Cloudflare Access 发放的 JWT（请求头名：`Cf-Access-Jwt-Assertion`）。程序会使用 `CF_ZT_ORG_NAME` 与 `CF_ZT_AUD` 来验证 JWT。

确保下游请求通过 Cloudflare Access 或在可信网络中使用。

请参阅： [https://developers.cloudflare.com/cloudflare-one/identity/service-tokens/](https://developers.cloudflare.com/cloudflare-one/identity/service-tokens/)

返回格式（统一 JSON）：

```json
{
 "code": "int",
 "msg": "string",
 "data": "string|object|array|null"
}
```

常见返回码（来自代码）：

- `0x000000` (`ErrSuccess`) — SUCCESS，`data` 为字符串类型的 PEM 内容。
- `0x400001` (`ErrUnauthorized`) — 未授权（JWT 验证失败或缺失）。
- `0x500003` (`ErrServerMisconfig`) — 服务器配置错误（缺少环境变量等）。
- `0x400004` (`ErrFileNotFound`) — 请求的文件不存在。
- `0x500000` (`ErrReadFileFailed`) — 读取文件失败。

示例：使用 curl 获取证书

```bash
# Documentation: https://developers.cloudflare.com/cloudflare-one/identity/service-tokens/

# 获取证书
curl -fSsL -H "CF-Access-Client-Id: <your-service-id>.access" -H "CF-Access-Client-Secret: <your-service-secret>" "http://127.0.0.1:9504/cert" | jq -r '.data'

# 获取私钥（注意：私钥应受严格保护）
curl -fSsL -H "CF-Access-Client-Id: <your-service-id>.access" -H "CF-Access-Client-Secret: <your-service-secret>" "http://127.0.0.1:9504/key" | jq -r '.data'
```

注意：API 返回的 `data` 字段在成功时为 PEM 文本（字符串）。请勿在不安全的通道或日志中泄露私钥。

## 日志与调试

- 程序使用 `logrus` 输出日志（默认级别 Debug），日志输出到标准输出（systemd 管理时可通过 `journalctl` 查看）。
- 启动后会输出 banner 与初始化日志，例如账户加载、证书保存路径等。

示例：使用 journalctl 查看 service 日志

```bash
journalctl -efu acme.service
```

## 如何切换到生产 CA

代码目前默认使用 Let's Encrypt Staging CA（便于测试，避免触发限额）。如果要切换到生产 CA：

1. 修改源码中 `CADirectory` 的值为 `certmagic.LetsEncryptProductionCA`（文件：`src/acme.go`）。
2. 重新构建并部署二进制文件。

注意：生产环境请谨慎操作，确保 DNS/Access 配置正确并遵守 Let's Encrypt 限额政策。

## 贡献与许可

本项目遵循仓库中的 `LICENSE` 文件（AGPL v3）。欢迎提交 Issue / PR，提议改进（例如支持更多 DNS 提供商、增加证书续期策略或将 CADirectory 通过环境变量配置等）。

## 鸣谢

本文档与代码中使用或参考了以下第三方项目、仓库、站点：

- acmez — ACME 客户端库（用于与 ACME CA 交互，项目中使用 `mholt/acmez`）：[https://github.com/mholt/acmez](https://github.com/mholt/acmez)
- certmagic — Caddy 团队的证书管理库（用于证书策略与 DNS-01 支持）：[https://github.com/caddyserver/certmagic](https://github.com/caddyserver/certmagic)
- rsync — 文件同步与分发工具（常用于在多机间同步证书）：[https://rsync.samba.org/](https://rsync.samba.org/)
- libdns/cloudflare — Cloudflare DNS 提供商适配器（用于 certmagic 的 DNS-01）：[https://github.com/libdns/cloudflare](https://github.com/libdns/cloudflare)
- logrus — 结构化日志库（用于程序日志输出）：[https://github.com/sirupsen/logrus](https://github.com/sirupsen/logrus)
- gin — HTTP Web 框架（用于实现 HTTP API）：[https://github.com/gin-gonic/gin](https://github.com/gin-gonic/gin)
