# PostOffice

自定义域名邮局系统 —— 使用自己的域名搭建专属邮件服务，支持完整的邮件收发、客户端访问和第三方 API 集成。

## 功能特性

- **SMTP 邮件收发** — 端口 25 接收外部邮件 / 端口 587 用户提交发送
- **IMAP4rev1 客户端访问** — 端口 993，支持 Thunderbird、Apple Mail 等客户端
- **内置 Web 邮件界面** — Vue 3 + TypeScript，开箱即用
- **多域名支持** — 绑定多个自定义域名
- **外部 API** — 通过 API Key 拉取指定邮箱的邮件（支持 `limit=-1` 获取全部）
- **安全特性** — DKIM 签名、SPF/DMARC 验证、速率限制、Argon2id 密码哈希
- **运维友好** — 健康检查、Prometheus 指标、结构化日志
- **一键部署** — Docker Compose 编排，开箱即用

## 架构概览

```
┌──────────────────────────────────────────────────────────┐
│                    Docker Compose                        │
│                                                          │
│  ┌──────────────────────────────────────────────────┐    │
│  │              PostOffice (Go Binary)               │    │
│  │                                                    │    │
│  │  ┌─────────┐ ┌─────────┐ ┌──────┐ ┌──────────┐   │    │
│  │  │SMTP :25 │ │SMTP :587│ │IMAP  │ │HTTP :8080│   │    │
│  │  │(入站)   │ │(提交)   │ │ :993 │ │API+WebUI │   │    │
│  │  └────┬────┘ └────┬────┘ └──┬───┘ └────┬─────┘   │    │
│  │       └────────────┴─────────┴──────────┘         │    │
│  │                      │                             │    │
│  │       ┌──────────────┼──────────────┐              │    │
│  │       ▼              ▼              ▼              │    │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐         │    │
│  │  │PostgreSQL│  │  MinIO   │  │  Redis   │         │    │
│  │  │  :5432   │  │  :9000   │  │  :6379   │         │    │
│  │  └──────────┘  └──────────┘  └──────────┘         │    │
│  └──────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────┘
```

## 项目结构

```
postoffice/
├── cmd/postoffice/main.go           # 程序入口
├── internal/
│   ├── app/app.go                   # 依赖注入与启动编排
│   ├── config/config.go             # 配置加载 (YAML + 环境变量)
│   ├── domain/
│   │   ├── model/models.go          # 领域模型
│   │   └── repository/              # 仓储接口
│   ├── smtp/                        # SMTP 服务 (收件 + 发件 + 队列)
│   ├── imap/                        # IMAP4rev1 服务
│   ├── api/                         # REST API + 中间件
│   │   ├── router.go                # 路由配置
│   │   ├── handler/                 # 请求处理器
│   │   └── middleware/              # JWT/APIKey/限流/日志/指标
│   ├── service/                     # 应用服务层
│   ├── infrastructure/
│   │   ├── postgres/                # 数据库实现 + 迁移
│   │   └── storage/                 # S3/MinIO 邮件存储
│   └── pkg/email/                   # MIME 邮件解析
├── web/                             # Vue 3 前端
├── deployments/                     # Dockerfile + docker-compose.yml
├── configs/postoffice.yaml          # 默认配置
├── .env.example                     # 环境变量模板
└── Makefile
```

## 快速开始

### 前置要求

- Docker & Docker Compose
- 一个域名（用于收发邮件）
- 服务器开放端口：25、587、993、8080（或 443）

### 部署步骤

```bash
# 1. 克隆项目
git clone https://github.com/Code9xs/ByteOryx-PostOffice.git
cd ByteOryx-PostOffice

# 2. 配置环境变量
cp .env.example .env
vim .env  # 设置域名、密码等

# 3. 启动所有服务
cd deployments
docker compose up -d --build

# 4. 等待服务就绪
docker compose ps  # 确认所有服务 healthy

# 5. 访问
# Web 界面: http://your-server:8080
# MinIO 控制台: http://your-server:9001
```

### 环境变量说明

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `MAIL_HOSTNAME` | 邮件服务器主机名 | `mail.example.com` |
| `DB_PASSWORD` | PostgreSQL 密码 | `postoffice` |
| `MINIO_ACCESS_KEY` | MinIO 访问密钥 | `minioadmin` |
| `MINIO_SECRET_KEY` | MinIO 密钥 | `minioadmin` |
| `JWT_SECRET` | JWT 签名密钥 | `change-me-in-production` |

## API 文档

### 用户认证

#### 注册
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"your-password","display_name":"Admin"}'
```

#### 登录
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"your-password"}'
```

返回 JWT Token，后续请求通过 `Authorization: Bearer <token>` 认证。

### 域名管理

```bash
# 添加域名
curl -X POST http://localhost:8080/api/v1/domains \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"yourdomain.com"}'

# 查看域名列表
curl http://localhost:8080/api/v1/domains \
  -H "Authorization: Bearer <token>"
```

### 邮箱管理

```bash
# 创建邮箱 (自动创建 INBOX/Sent/Drafts/Trash/Junk 文件夹)
curl -X POST http://localhost:8080/api/v1/mailboxes \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"local_part":"hello","domain_id":"<domain-uuid>"}'
```

### 外部 API（API Key 认证）

#### 创建 API Key
```bash
curl -X POST http://localhost:8080/api/v1/apikeys \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"my-app","mailbox_ids":["<mailbox-uuid>"]}'
```

API Key 格式：`po_` + 32 位随机字符串，**仅在创建时返回一次**，请妥善保存。

#### 拉取邮件
```bash
# 获取最近 10 封邮件
curl http://localhost:8080/api/v1/external/mailboxes/hello@yourdomain.com/messages?limit=10 \
  -H "Authorization: Bearer po_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

# 获取所有邮件
curl http://localhost:8080/api/v1/external/mailboxes/hello@yourdomain.com/messages?limit=-1 \
  -H "Authorization: Bearer po_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

# 仅获取未读邮件
curl "http://localhost:8080/api/v1/external/mailboxes/hello@yourdomain.com/messages?unseen_only=true" \
  -H "Authorization: Bearer po_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

# 按时间过滤 + 分页
curl "http://localhost:8080/api/v1/external/mailboxes/hello@yourdomain.com/messages?since=2026-01-01T00:00:00Z&offset=0&limit=20" \
  -H "Authorization: Bearer po_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

**参数说明：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `limit` | int | 获取数量，`-1` 为全部，默认 20 |
| `offset` | int | 分页偏移，默认 0 |
| `folder` | string | 文件夹名，默认 `INBOX` |
| `since` | string | 起始时间 (RFC3339)，可选 |
| `unseen_only` | bool | 仅未读邮件，默认 false |

#### 响应示例
```json
{
  "total": 150,
  "messages": [
    {
      "id": "uuid",
      "message_id": "<abc@example.com>",
      "from_address": "sender@other.com",
      "to_addresses": ["hello@yourdomain.com"],
      "subject": "Hello",
      "date": "2026-05-22T10:00:00Z",
      "size_bytes": 4096,
      "is_seen": false,
      "body_text": "plain text body...",
      "body_html": "<p>html body...</p>",
      "attachments": [
        {
          "filename": "doc.pdf",
          "size": 102400,
          "content_type": "application/pdf",
          "download_url": "/api/v1/external/messages/uuid/attachments/0"
        }
      ]
    }
  ]
}
```

### 运维端点

```bash
# 健康检查
curl http://localhost:8080/health

# Prometheus 指标
curl http://localhost:8080/metrics
```

## DNS 配置

添加域名后，需要在 DNS 服务商配置以下记录：

| 类型 | 名称 | 值 | 说明 |
|------|------|------|------|
| MX | @ | `10 mail.yourdomain.com` | 邮件路由 |
| A | mail | `<服务器IP>` | 邮件服务器地址 |
| TXT | @ | `v=spf1 a mx -all` | SPF 验证 |
| TXT | `postoffice._domainkey` | *(创建域名时返回)* | DKIM 签名验证 |
| TXT | `_dmarc` | `v=DMARC1; p=quarantine` | DMARC 策略 |

## 邮件客户端配置

使用 Thunderbird / Apple Mail / Outlook 等客户端：

| 设置 | 值 |
|------|------|
| IMAP 服务器 | `mail.yourdomain.com` |
| IMAP 端口 | 993 (SSL/TLS) |
| SMTP 服务器 | `mail.yourdomain.com` |
| SMTP 端口 | 587 (STARTTLS) |
| 用户名 | 完整邮箱地址 |
| 密码 | 注册时设置的密码 |

## 本地开发

```bash
# 启动依赖服务
cd deployments && docker compose up -d postgres redis minio

# 构建
make build

# 运行
make run

# 前端开发 (热重载)
cd web && npm run dev

# 测试
make test

# 代码检查
make lint
```

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go 1.22+ |
| 前端 | Vue 3 + TypeScript + Vite |
| 数据库 | PostgreSQL 16 |
| 对象存储 | MinIO (S3 兼容) |
| 缓存/队列 | Redis 7 |
| HTTP 框架 | Gin |
| 部署 | Docker Compose |

## License

MIT
