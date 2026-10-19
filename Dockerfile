# syntax=docker/dockerfile:1

# 本镜像只做打包，Go 二进制与前端产物均在本机构建后复制进来。
# 构建请使用 make docker / make docker-api，不要直接 docker build。
FROM alpine:3.22

ARG VERSION=dev
ARG WITH_FRONTEND=true

LABEL org.opencontainers.image.title="Yu" \
      org.opencontainers.image.description="网站统计 + 服务监控 + 短链/遥测" \
      org.opencontainers.image.version="${VERSION}" \
      org.remoon.yu.frontend="${WITH_FRONTEND}"

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 yu \
    && adduser -S -D -H -u 10001 -G yu yu \
    && mkdir -p /data \
    && chown yu:yu /data

WORKDIR /app

COPY dist/yu-server /app/yu-server
COPY dist/public/ /app/public/

ENV YU_ADDR=:8080 \
    YU_STATIC_DIR=/app/public \
    TZ=UTC

EXPOSE 8080

USER yu

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/api/health || exit 1

ENTRYPOINT ["/app/yu-server"]
