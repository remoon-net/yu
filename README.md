# Yu

Tianji 精简版：网站统计 + 服务监控。前后端分离，后端单体 Go 服务，前端 SvelteKit 静态站点。

## 功能

- 网站统计：PV/UV、会话、跳出率、停留时长、页面、来源/UTM、访客画像、实时访客、自定义事件、公开看板
- 服务监控：HTTP / TCP / Ping / DNS 探测、可用率与延迟分位、静默、SSL 到期提醒
- 告警：连续失败阈值、恢复通知、邮件 / Webhook（HMAC 签名）、维护窗口
- 事件：自动故障记录、手动事件、状态页展示
- 状态页：公开访问、组件分组、访问密码
- 多用户与工作区、邀请制注册、API Key

## 技术栈

- 后端：Go + chi + pgx + PostgreSQL，内嵌 SQL 迁移
- 前端：SvelteKit 3 + Svelte 5 + TypeScript + Luxon + ECharts，`adapter-static` 静态构建
- 部署：静态文件由 Caddy 托管，`/api/*` 反代到 Go

## 目录

```text
backend/    Go 服务、迁移、采集脚本
frontend/   SvelteKit 控制台
design.md   设计文档
```

## 快速开始

1. 准备 PostgreSQL 15+：

```bash
createdb yu
```

2. 启动后端：

```bash
export YU_DATABASE_URL='postgres://yu:yu@127.0.0.1:5432/yu?sslmode=disable'
export YU_BOOTSTRAP_EMAIL='admin@example.com'
export YU_BOOTSTRAP_PASSWORD='change-me-please'
go run ./backend/cmd/server
```

首次启动会自动执行迁移并创建管理员与默认工作区。

3. 启动前端：

```bash
cd frontend
npm install
YU_API=http://127.0.0.1:8080 npm run dev
```

访问 Vite 输出的地址，使用上面的管理员账号登录。

## 生产部署

```bash
go build -o yu-server ./backend/cmd/server
cd frontend && npm ci && npm run build
```

Caddy 示例：

```caddy
yu.example.com {
    handle /api/* {
        reverse_proxy 127.0.0.1:8080
    }
    handle {
        root * /var/www/yu
        try_files {path} /index.html
        file_server
    }
}
```

`/collect.js` 也由 Go 提供，若 API 单独走子域名，请同步在 Caddy 中反代。

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `YU_ADDR` | `:8080` | 监听地址 |
| `YU_DATABASE_URL` | 本地 yu 库 | PostgreSQL 连接串 |
| `YU_PUBLIC_BASE_URL` | `http://localhost:8080` | 公开访问地址，用于安装脚本与邀请链接 |
| `YU_ALLOWED_ORIGINS` | 空 | 允许跨域的控制台来源，逗号分隔 |
| `YU_SECURE_COOKIES` | `false` | HTTPS 部署时设为 `true` |
| `YU_SESSION_TTL` | `720h` | 会话有效期 |
| `YU_BOOTSTRAP_EMAIL` / `YU_BOOTSTRAP_PASSWORD` | 空 | 首次启动创建管理员 |
| `YU_BOOTSTRAP_WORKSPACE` | `默认工作区` | 初始工作区名称 |
| `YU_PROBE_CONCURRENCY` | `32` | 探测并发上限 |

## 采集

在站点设置页复制脚本：

```html
<script defer src="https://yu.example.com/collect.js" data-token="站点 token"></script>
```

自定义事件：

```js
window.yu.event('signup', { plan: 'pro' });
```

## 邮件通知

在工作区「告警与事件 → 通知渠道」中添加「邮件」渠道，直接填写 SMTP 主机、端口、用户名、密码、发件人与收件人，每个渠道可以使用不同的 SMTP 服务器。

环境变量中的 SMTP 配置作为兜底：渠道未填写 `host` 时使用全局配置。

## API Key

工作区设置中创建，使用 `Authorization: Bearer <token>` 调用 API（支持 `api_key` 认证，写请求无需 CSRF）。

## 注意事项

- Ping 探针使用 ICMP，需要 `net.ipv4.ping_group_range` 允许当前用户或二进制的 `CAP_NET_RAW` 权限。
- 数据永久保留：`events` 与 `monitor_checks` 按月分区，后台会自动创建未来两个月分区。
- 多实例部署时，调度器通过 PostgreSQL advisory lock 保证只有一个实例执行探测。
- DNS 使用系统解析器，可通过监控配置中的 `server` 指定 DNS 服务器。
