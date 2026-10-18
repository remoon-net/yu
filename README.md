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

本地开发和生产使用同一域名，采集脚本由前端静态目录提供（`/collect.js`）。若前后端端口不同，请把 `YU_PUBLIC_BASE_URL` 设为前端地址，安装代码里的脚本 URL 才会正确。

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
    # 短链跳转由 Go 处理
    handle /s/* {
        reverse_proxy 127.0.0.1:8080
    }
    handle {
        root * /var/www/yu
        try_files {path} /index.html
        file_server
    }
}
```

采集脚本随前端构建产物发布（`frontend/static/collect.js` → `build/collect.js`），与页面同域，由上面的静态托管直接提供。建议给它短缓存：

```caddy
@collect path /collect.js
header @collect Cache-Control "public, max-age=300"
```

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `YU_ADDR` | `:8080` | 监听地址 |
| `YU_DATABASE_URL` | 本地 yu 库 | PostgreSQL 连接串 |
| `YU_PUBLIC_BASE_URL` | `http://localhost:8080` | 对外访问地址（页面与邀请链接），也是安装脚本里的域名 |
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

单页应用（SvelteKit / React Router / Vue Router 等 history 路由）的软导航会自动上报，无需额外配置。
如需在框架导航钩子里补充上报，可手动调用（自动去重）：

```js
window.yu.pageview();
```

SvelteKit 也可以放在根布局里，让每次导航结束后补报一次（适合自定义多段式导航）：

```svelte
<script>
	import { afterNavigate } from '$app/navigation';
	afterNavigate(() => window.yu?.pageview());
</script>
```

> 采集脚本会在 history 变更后延迟到下一轮任务上报，确保拿到 SvelteKit 更新后的 `<title>`。

自定义事件：

```js
window.yu.event('signup', { plan: 'pro' });
```

## 短链追踪

工作区「短链」页可创建 `https://你的域名/s/<标识>`，点击会记录到对应站点的 `events`（`type=link_click`），带访问量 / 独立访客 / 最近点击统计，支持过期时间、启用停用与二维码。

- 短链必须归属一个站点，点击不计入站点 PV
- 访问时会把原链接的 query 参数（含 UTM）拼接到目标地址后跳转
- 过期或停用返回 410 页面
- 列表会标注目标地址是「站内」还是「站外」；详情面板可看点击趋势、来源、渠道、UTM、设备与浏览器分布及最近 50 条明细
- 目标是站外时同样能统计点击（在服务端跳转时记录），但落地页之后的行为无法追踪
- 渠道判定以来源站为准，从自己站点分享出去记为 referral，不会被当成内部跳转

本地想看演示效果可执行（会生成 3 条 `demo-` 短链与 90 天点击数据）：

```bash
docker exec -i yu-pg psql -U yu -d yu < backend/scripts/seed_short_link_demo.sql
```

清理演示数据：

```sql
delete from events where type = 'link_click' and name like 'demo-%';
delete from short_links where slug like 'demo-%';
```

## 邮件通知

在工作区「告警与事件 → 通知渠道」中添加「邮件」渠道，直接填写 SMTP 主机、端口、用户名、密码、发件人与收件人，每个渠道可以使用不同的 SMTP 服务器。

## API Key

工作区设置中创建，使用 `Authorization: Bearer <token>` 调用 API（支持 `api_key` 认证，写请求无需 CSRF）。

## 注意事项

- Ping 探针使用 ICMP，需要 `net.ipv4.ping_group_range` 允许当前用户或二进制的 `CAP_NET_RAW` 权限。
- 数据永久保留：`events` 与 `monitor_checks` 按月分区，后台会自动创建未来两个月分区。
- 多实例部署时，调度器通过 PostgreSQL advisory lock 保证只有一个实例执行探测。
- DNS 使用系统解析器，可通过监控配置中的 `server` 指定 DNS 服务器。
