# ─── LexAGI Dockerfile ─────────────────────────────────
# 多阶段构建：编译 → 运行时
# 商业标准：非 root、最小镜像、健康检查

# ─── 阶段1：构建 ─────────────────────────────────────
FROM golang:1.23-alpine AS builder

WORKDIR /app

# 安装构建依赖
RUN apk add --no-cache git ca-certificates tzdata

# 复制 go mod
COPY go.mod go.sum* ./
RUN go mod download && go mod verify

# 复制源码
COPY . .

# 编译（静态链接，无 CGO）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o bin/lexagi ./cmd/lexagi

# ─── 阶段2：运行时 ───────────────────────────────────
FROM alpine:3.19

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata wget

# 非 root 用户（安全要求）
RUN addgroup -g 1000 lexagi && \
    adduser -u 1000 -G lexagi -s /bin/sh -D lexagi

WORKDIR /app

# 从构建阶段复制二进制
COPY --from=builder /app/bin/lexagi .
COPY --from=builder /app/prompts ./prompts
COPY --from=builder /app/db/migrations ./db/migrations

# 权限
RUN chown -R lexagi:lexagi /app
USER lexagi

# 端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --spider -q http://localhost:8080/healthz || exit 1

# 启动
CMD ["./lexagi"]
