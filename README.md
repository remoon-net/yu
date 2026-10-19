# Yu

Tianji 精简版：网站统计 + 服务监控 + 短链/遥测，前后端分离。后端单体 Go 服务，前端 SvelteKit 静态站点。

## 功能

**网站统计**

- PV / UV、新访客 / 回访、DAU / WAU / MAU、会话数、跳出率、平均停留，带本期 vs 上期环比
- 会话时长分布、工作区概览：跨站点汇总 PV / UV / 会话与趋势，站点卡片展示区间数据
- 热门页面、入口页、退出页
- 来源、UTM（source / medium / campaign / term / content）、渠道归类
- 自动追踪出站点击、文件下载、滚动深度与站内搜索（作为自定义事件）
- 访客画像：设备类型、手机品牌、浏览器、操作系统、屏幕、语言、国家/地区/城市
- 实时访客、自定义事件与属性、公开看板（可设密码）
- 目标转化（事件/页面）、2-5 步漏斗分析
- 留存分析（按天/周队列热力图）
- 维度筛选与可保存分段（页面、来源域名、国家、设备、浏览器、UTM Source）
- 原始事件 CSV 导出

**服务监控**

- HTTP / TCP / Ping / DNS 探测，支持自定义请求方法、Header 与请求体，状态码、响应体、JSONPath、耗时断言
- 可用率、P50/P95/P99、状态时间轴
- 连续失败告警、恢复通知、静默期、维护窗口抑制
- SSL 到期提醒、故障事件与手动 incident、公开状态页
- 状态徽章 `/monitor/<workspace>/<monitor>/badge.svg`，可嵌入 README 或站点

**短链与遥测**

- 短链 `/s/<标识>`：302 跳转、UTM 透传、二维码、过期与启停、点击趋势与来源分布
- 遥测像素 `/t/<标识>.gif`：1×1 透明图埋点，统计无法注入脚本的页面的曝光

**平台能力**

- 多用户与工作区（owner / admin / member），邀请制注册
- API Key + REST API，深色 / 浅色主题
- PostgreSQL 存储，事件按月分区，原始数据永久保留

## 技术栈

- 后端：Go + chi + pgx + PostgreSQL，内嵌 SQL 迁移
- 前端：SvelteKit 3 + Svelte 5 + TypeScript + Luxon + ECharts，`adapter-static` 静态构建
- 部署：静态文件由 Caddy 托管，`/api/*`、`/s/*`、`/t/*`、`/monitor/*` 反代到 Go

## 目录

```text
backend/
  cmd/server/            启动入口
  internal/httpapi/      HTTP 路由与 handler
  internal/store/        数据访问（pgx + 手写 SQL）
  internal/analytics/    采集、UA 解析、渠道判定
  internal/monitor/      探针、调度器、断言
  internal/auth/         会话、密码、邀请
  internal/mailer/       SMTP 发送
  migrations/            内嵌 SQL 迁移
  scripts/               演示数据脚本
frontend/
  src/lib/api/           API 封装与类型
  src/lib/components/    通用组件与图表
  src/routes/            页面路由
  static/collect.js      采集脚本
  build/                 adapter-static 产物
docker-compose.yml       PostgreSQL + Yu 一键启动
Dockerfile               镜像打包（只复制本机构建产物）
Makefile                 构建、镜像与检查
design.md                设计文档
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
pnpm install
YU_API=http://127.0.0.1:8080 pnpm run dev
```

访问 Vite 输出的地址，使用上面的管理员账号登录。

本地开发和生产使用同一域名，采集脚本由前端静态目录提供（`/collect.js`）。若前后端端口不同，请把 `YU_PUBLIC_BASE_URL` 设为前端地址，安装代码里的脚本 URL 才会正确；同时把前端来源加入 `YU_ALLOWED_ORIGINS`，否则浏览器会拦截跨域请求：

```bash
export YU_ALLOWED_ORIGINS='http://localhost:5173'
```

## 生产部署

```bash
go build -o yu-server ./backend/cmd/server
cd frontend && pnpm ci && pnpm run build
```

Caddy 示例：

```caddy
yu.example.com {
    handle /api/* {
        reverse_proxy 127.0.0.1:8080
    }
    # 短链跳转与遥测像素由 Go 处理
    handle /s/* {
        reverse_proxy 127.0.0.1:8080
    }
    handle /t/* {
        reverse_proxy 127.0.0.1:8080
    }
    handle /monitor/* {
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

### Docker 镜像

镜像采用「本机构建、Docker 只打包」的方式：Go 二进制与前端 `build/` 产物先在宿主机生成，再复制进镜像，容器内不再执行 npm / go 构建。

```bash
# 含前端静态文件（默认）：镜像 shynome/yu:<git describe>，同时打 shynome/yu:latest
make docker

# 仅后端 API：镜像 shynome/yu:<git describe>-api
make docker-api

# 指定仓库名 / 构建后推送
make push IMAGE=registry.example.com/shynome/yu
```

常用变量：`IMAGE`（镜像仓库名）、`WITH_FRONTEND=0`（等价于 `make docker-api`）、`TAG_LATEST=0`（不额外打 latest）、`GOARCH=arm64`（交叉编译）、`VERSION=...`（覆盖版本号）。

版本号来自 `git describe --tags --always --dirty`，通过 ldflags 注入二进制；`GET /api/health` 可查看：

```json
{"status":"ok","version":"v1.2.3","commit":"1a2b3c4","buildTime":"2026-10-19T06:00:00Z","static":true}
```

容器内前端产物位于 `/app/public`（由 `YU_STATIC_DIR` 指定），Go 服务直接托管页面、`/collect.js`、`/s/*`、`/t/*` 与 `/monitor/*` 徽章，不再需要 Caddy。未打包前端时目录为空，服务自动退化为纯 API。

使用 compose 一键启动（PostgreSQL + Yu）：

```bash
make docker
docker compose up -d
```

或使用已有 PostgreSQL：

```bash
docker run -d --name yu \
  -p 8080:8080 \
  -e YU_DATABASE_URL='postgres://yu:yu@172.17.0.1:5432/yu?sslmode=disable' \
  -e YU_PUBLIC_BASE_URL='https://yu.example.com' \
  -e YU_SECURE_COOKIES=true \
  -e YU_BOOTSTRAP_EMAIL='admin@example.com' \
  -e YU_BOOTSTRAP_PASSWORD='change-me-please' \
  --cap-add=NET_RAW \
  shynome/yu:latest
```

> 容器以非 root 用户运行。Ping 探测需要 ICMP 权限，可用 `--cap-add=NET_RAW`，或在宿主机设置 `net.ipv4.ping_group_range`；不需要 Ping 时可省略。

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `YU_ADDR` | `:8080` | 监听地址 |
| `YU_DATABASE_URL` | 本地 yu 库 | PostgreSQL 连接串 |
| `YU_PUBLIC_BASE_URL` | `http://localhost:8080` | 对外访问地址（页面与邀请链接），也是安装脚本里的域名 |
| `YU_ALLOWED_ORIGINS` | 空 | 允许跨域的控制台来源，逗号分隔 |
| `YU_SECURE_COOKIES` | `false` | HTTPS 部署时设为 `true` |
| `YU_SESSION_TTL` | `720h` | 会话有效期 |
| `YU_LOG_LEVEL` | `info` | 日志等级：`debug` / `info` / `warn` / `error` |
| `YU_STATIC_DIR` | 空 | 内置前端静态资源目录，Docker 镜像中为 `/app/public` |
| `YU_BOOTSTRAP_EMAIL` / `YU_BOOTSTRAP_PASSWORD` | 空 | 首次启动创建管理员 |
| `YU_BOOTSTRAP_WORKSPACE` | `默认工作区` | 初始工作区名称 |
| `YU_PROBE_CONCURRENCY` | `32` | 探测并发上限 |
| `YU_GEOIP_DB` | 空 | MaxMind GeoLite2 City 数据库路径，启用后记录国家/地区/城市 |

## 地理与流量过滤

- 下载 [GeoLite2-City.mmdb](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data/)，设置 `YU_GEOIP_DB=/path/to/GeoLite2-City.mmdb`；Docker 部署时需把文件挂载进容器
- 未配置 GeoIP 时，国家/地区/城市字段保持为空，不影响其他统计
- 流量过滤在「站点设置 → 流量过滤」中配置，按站点生效；默认关闭，不丢弃机器人与内网流量
- 需要时可在面板中开启机器人过滤，并按行或逗号填写排除 IP/CIDR（支持 IPv4/IPv6，最多 64 项）
- 开启地址排除后，经反向代理部署时请确保传递 `X-Forwarded-For`
- 地理信息在请求时实时查询，历史事件未保存 IP，无法补算

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

## 遥测（像素上报）

用于**无法注入 `collect.js`** 的页面：第三方平台文章、README、邮件、落地页。嵌入一张 1×1 透明图，页面加载时记录一次曝光到对应站点的 `events`（`type=pixel`）。

工作区「遥测像素」页创建后会给出嵌入代码：

```html
<img src="https://你的域名/t/<标识>.gif" width="1" height="1" alt="" />
```

- 无论标识是否存在、是否停用，都返回 200 + GIF（避免别人页面出现破图，也避免被枚举）
- 停用或过期的像素只是不再记录事件
- 支持在图片 URL 上追加 `utm_source` / `utm_medium` / `utm_campaign`，用于区分同一篇内容的多个投放位置
- 详情面板与短链共用：曝光趋势、来源、渠道、UTM、设备、浏览器与最近明细

已知偏差（行业通病）：

- 邮件客户端常预加载图片，曝光会偏高
- 微信、知乎等平台会代理图片，referrer 与 IP 变成平台侧，来源分析会失真

## 邮件通知

在工作区「告警与事件 → 通知渠道」中添加「邮件」渠道，直接填写 SMTP 主机、端口、用户名、密码、发件人与收件人，每个渠道可以使用不同的 SMTP 服务器。支持 STARTTLS；使用 465 端口的 SSL/TLS 服务时勾选「使用 SSL/TLS（隐式加密）」即可。

## API Key

工作区设置中创建，使用 `Authorization: Bearer <token>` 调用 API（支持 `api_key` 认证，写请求无需 CSRF）。

## 测试与检查

```bash
# 一次跑完（gofmt / go build / go test / 前端检查）
make check

# 或分开执行
# 后端
go build ./...
go test ./...
gofmt -l backend

# 前端
cd frontend
pnpm run check
pnpm run build
```

`go test` 目前覆盖纯逻辑：User-Agent 解析与渠道判定、短链/像素的标识与目标校验、监控的 JSONPath 与 HTTP 配置校验。

## 注意事项

- Ping 探针使用 ICMP，需要 `net.ipv4.ping_group_range` 允许当前用户或二进制的 `CAP_NET_RAW` 权限。
- 数据永久保留：`events` 与 `monitor_checks` 按月分区，后台会自动创建未来两个月分区。
- 多实例部署时，调度器通过 PostgreSQL advisory lock 保证只有一个实例执行探测。
- DNS 使用系统解析器，可通过监控配置中的 `server` 指定 DNS 服务器。
