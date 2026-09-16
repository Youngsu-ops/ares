# ================= 第一阶段：编译环境 =================
# 基础镜像可通过 build-arg 覆盖（国内网络拉不到 Docker Hub 时用加速源）：
#   --build-arg BASE_REGISTRY=docker.m.daocloud.io/library/
# GitHub Actions 构建不需要，保持默认即可
ARG BASE_REGISTRY=
FROM ${BASE_REGISTRY}golang:1.24-alpine AS builder

# Go 模块代理（国内网络用 https://goproxy.cn,direct；Actions 保持默认）
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

# 【关键 1】go-sqlite3 是 CGO 驱动，必须装 gcc 并开启 CGO。
# 若设成 CGO_ENABLED=0，编译能过但运行时会直接崩溃：
#   "Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work. This is a stub"
RUN apk --no-cache add gcc musl-dev

# 【关键 2】不要硬编码 GOARCH。
# 硬写 GOARCH=amd64 时，在 arm64 机器上会让 aarch64-gcc 去交叉编译 amd64，
# 直接报 "gcc: error: unrecognized command-line option '-m64'"。
# 目标架构交给构建平台决定：本地构建 = 本机架构，CI/集群 = linux/amd64（用 --platform 指定）。
ENV GO111MODULE=on \
    CGO_ENABLED=1

WORKDIR /build

# 1. 先复制 go.mod 和 go.sum 锁文件（利用 Docker 层缓存，依赖不变时跳过下载）
COPY go.mod go.sum ./
# 2. 下载依赖
RUN go mod download

# 3. 复制项目所有剩余源码
COPY . .

# 4. 编译，-ldflags="-s -w" 剔除调试信息，体积缩减约 40%
RUN go build -ldflags="-s -w" -o ares .

# ================= 第二阶段：纯净运行环境 =================
# 注意：运行镜像的 Alpine 版本要与编译镜像一致，避免 musl 版本不匹配
ARG BASE_REGISTRY=
FROM ${BASE_REGISTRY}alpine:3.21

# ca-certificates: 请求外部 HTTPS 接口（OAuth/第三方 API）不报错
# tzdata: 正确显示日志时间（默认是 UTC）
RUN apk --no-cache add ca-certificates tzdata

ENV TZ=Asia/Shanghai
WORKDIR /app

# 从第一阶段拷贝编译好的二进制
COPY --from=builder /build/ares /app/ares
# 模板与静态资源（.dockerignore 已排除 static/uploads，避免把用户上传内容打进镜像）
COPY --from=builder /build/templates /app/templates
COPY --from=builder /build/static /app/static

# 运行时目录：data 与 static/uploads 由 PVC 挂载
RUN mkdir -p data static/uploads static/img && chown -R 1000:1000 data static

# 以非 root 运行（K8s 里配合 fsGroup: 1000 授予卷写权限）
USER 1000:1000

# Ares 默认端口
EXPOSE 8888

ENTRYPOINT ["./ares"]
