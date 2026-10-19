# Changelog

本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### 新增

- 服务监控的 HTTP 探测支持自定义请求方法、Header 与请求体，创建和编辑监控时均可配置
- HTTP 探测补齐高级选项：期望状态码、响应体包含/不包含、JSONPath 断言、耗时阈值与重定向设置

## [v0.0.4] - 2026-10-19

### 变更

- 前端依赖安装改用 pnpm：`pnpm ci` + `pnpm run build`，新增 `pnpm-lock.yaml` 并移除 `package-lock.json`

## [v0.0.3] - 2026-10-19

### 新增

- 新增监控状态徽章 `GET /monitor/{workspaceID}/{monitorID}/badge.svg`，可嵌入 README 或站点；支持 `?showDetail=1` 显示最近延迟。监控详情页新增「徽章」入口，可预览并复制图片地址与 Markdown 片段

### 变更

- README 的 Caddy 示例补充 `/monitor/*` 反代说明

## [v0.0.2] - 2026-10-19

### 新增

- 新增 `YU_LOG_LEVEL` 环境变量，可配置日志等级（`debug` / `info` / `warn` / `error`），默认 `info`；`docker-compose.yml` 已同步该变量

## [v0.0.1] - 2026-10-19

首个发布版本。

### 新增

- **网站统计**：PV / UV、会话数、跳出率、平均停留，热门页面、入口页、退出页，来源 / UTM / 渠道，访客画像，实时访客，自定义事件，多站点与公开看板
- **服务监控**：HTTP / TCP / Ping / DNS 探测，状态码、响应体、JSONPath 与耗时断言，可用率与 P50/P95/P99，SSL 到期提醒
- **告警与状态页**：邮件 / Webhook 通知，连续失败与恢复告警，静默期与维护窗口，公开状态页与 incident
- **短链与遥测**：`/s/<标识>` 短链跳转、UTM 透传、二维码、点击统计，`/t/<标识>.gif` 像素曝光统计
- **平台能力**：多用户与工作区、邀请制注册、API Key、PostgreSQL 存储、事件按月分区、内嵌 SQL 迁移
- **部署**：Docker 镜像与 Makefile，本机构建产物打包、Git 版本注入、镜像内置前端静态托管
