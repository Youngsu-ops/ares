# Ares 上 K8s 部署运行手册（1 → 6 顺序执行）

> 仓库：https://github.com/Youngsu-ops/ares
> 目标：把 Ares 从裸机 systemd 部署，迁移到海外 K8s 集群

## 顺序总览

| 阶段 | 做什么 | 产物 | 本仓库对应文件 |
|------|--------|------|----------------|
| 1 | 建立镜像仓库 | 可推送的 registry 命名空间 | — |
| 2 | 制作镜像 | 可运行的容器镜像 | `Dockerfile`、`.dockerignore` |
| 3 | 构建 PV / 存储 | 持久卷声明 | `k8s/pvc.yaml` |
| 4 | 编写 Deployment 并部署 | 跑起来的 Pod | `k8s/*.yaml` |
| 5 | 测试验证 | 域名可访问 + 数据完整 | 本文档第 5 节 |
| 6 | 搭 CI/CD | push 即发布 | `.github/workflows/docker-publish.yml` |

**关于顺序的一个说明**：你的顺序是对的，属于「先手工跑通、再自动化」的引导式（bootstrap）路径。有个细节值得知道——CI/CD 本质上就是自动化第 1、2、4 步，所以第 6 步做完后，前几步会退化成「只在第一次做」。另一种路径是先写 CI/CD，让流水线产出镜像（第 2 步由流水线代劳），但那样出问题时排查链路更长。**推荐按你的顺序走。**

---

## 阶段 0：前置准备

```bash
# 本地需要
docker --version          # 构建镜像
kubectl version --client  # 操作集群
gh --version              # 可选，管理仓库

# 需要启动 Docker Desktop（daemon 未运行无法构建）
open -a Docker

# 需要准备
# ① GitHub PAT（勾选 write:packages，用于推镜像到 GHCR）
# ② 海外 K8s 集群的 kubeconfig
# ③ domains: blog.kivensu.club 的 DNS 控制权
```

---

## 阶段 1：建立镜像仓库

海外集群推荐 **GHCR**（GitHub Container Registry）：与仓库同源、免额外账号、海外集群拉取快、私有镜像免费。

```bash
# 1. 用 PAT 登录 GHCR（PAT 需勾选 write:packages + read:packages）
echo "你的_PAT" | docker login ghcr.io -u Youngsu-ops --password-stdin

# 2. 仓库不需要手动创建——首次 push 时自动创建
#    最终镜像地址形如：ghcr.io/youngsu-ops/ares:latest
```

备选方案：

| 方案 | 优点 | 缺点 |
|------|------|------|
| **GHCR**（推荐） | 与 GitHub 集成、免费、CI 免额外密钥 | 需要 PAT 带 packages 权限 |
| Docker Hub | 通用性最好 | 免费账号有拉取限流 |
| 阿里云 ACR | 国内拉取快 | 海外集群跨区延迟、需额外配置 |

---

## 阶段 2：制作镜像

### 2.1 本地构建（先跑通，验证 Dockerfile 正确）

```bash
cd /Users/youngsu/Desktop/youngsu-blog-plus

# 构建（本地验证用；国内网络需加两个 build-arg，见 2.1.1）
docker build -t ares:alpine .

# 本地跑起来验证（挂本地目录当数据卷）
docker run -d --name ares-test \
  -p 8899:8888 \
  -e BLOG_URL=http://localhost:8899 \
  -e SESSION_SECRET=$(openssl rand -hex 32) \
  -e BLOG_ADMIN_PASS=test123456 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/static/uploads:/app/static/uploads \
  ares:alpine

# 验证
curl http://localhost:8899/healthz     # 期望 {"status":"ok"}
curl -I http://localhost:8899/          # 期望 200

# 清理
docker rm -f ares-test
```

> **国内网络完整构建命令**（Docker Hub 与 proxy.golang.org 均不可达时）：
>
> ```bash
> docker build \
>   --build-arg BASE_REGISTRY=docker.m.daocloud.io/library/ \
>   --build-arg GOPROXY=https://goproxy.cn,direct \
>   -t ares:alpine .
> ```
>
> 若 `docker buildx` 报 `~/.docker/buildx ... operation not permitted`（沙箱环境），加 `DOCKER_CONFIG=/tmp/ares-docker` 重定向配置目录；注意重定向后需要把插件目录也链过去：
> `ln -sfn ~/.docker/cli-plugins /tmp/ares-docker/cli-plugins`

### 2.1.1 三个必踩的坑（全部已实测复现并修复）

| # | 坑 | 现象 | 解法 |
|---|----|------|------|
| 1 | **`CGO_ENABLED=0` 是致命的** | 编译成功，运行时报 `Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work. This is a stub`，进程崩溃 → K8s 里表现为 CrashLoopBackOff | 必须 `CGO_ENABLED=1` + 装 `gcc musl-dev`（Alpine）/ `gcc`（Debian） |
| 2 | **硬编码 `GOARCH=amd64`** | 在 arm64 机器上构建报 `gcc: error: unrecognized command-line option '-m64'` | 不要写 `GOOS`/`GOARCH`，交给构建平台决定；集群用 `--platform linux/amd64` |
| 3 | **GHCR 镜像名必须全小写** | 推送报 `invalid reference format`（用户名 `Youngsu-ops` 含大写） | CI 里加一步 `echo "IMAGE_NAME=${GITHUB_REPOSITORY,,}" >> $GITHUB_ENV` |

**坑 1 的验证过程（值得记牢）**：

```bash
# 编译期完全正常，看不出任何问题
CGO_ENABLED=0 go build -ldflags="-s -w" -o ares .
# → 退出码 0，产物 15MB，"看起来"完全正常

# 但一运行就原形毕露
./ares
# → Failed to connect database: Binary was compiled with 'CGO_ENABLED=0',
#   go-sqlite3 requires cgo to work. This is a stub
# → 进程退出
```

> 这类「编译通过、运行崩溃」的问题在容器里最难排查：镜像构建成功、流水线绿灯，部署到集群却一直重启。

### 2.1.2 基础镜像与体积

| 方案 | 体积 | 说明 |
|------|------|------|
| Debian 两阶段（Debian slim 运行） | 172MB | 兼容性最好 |
| **Alpine 两阶段（推荐）** | **39MB** | 体积小 77%，需 `gcc musl-dev` 编译，musl 运行 |
| 纯 Go 驱动 + `CGO_ENABLED=0` | ~20MB | 需把 `go-sqlite3` 换成 `modernc.org/sqlite`（改 go.mod，当前未采用） |

**Alpine 编译镜像与运行镜像的版本要对齐**（当前都用 `alpine:3.21` 系），否则可能出现 musl 版本不匹配。

### 2.2 推送镜像

### 2.2 推送镜像

```bash
docker tag ares:local ghcr.io/youngsu-ops/ares:latest
docker tag ares:local ghcr.io/youngsu-ops/ares:v1.0.0
docker push ghcr.io/youngsu-ops/ares:latest
docker push ghcr.io/youngsu-ops/ares:v1.0.0
```

### 2.3 或直接交给 CI 构建（跳过 2.1/2.2，先跳到阶段 6 再回来）

本仓库已有 `.github/workflows/docker-publish.yml`，push 到 main 分支即自动构建并推送 GHCR。此时阶段 2 变成「等流水线跑完」。

---

## 阶段 3：构建 PV / 存储

海外托管集群（EKS / GKE / AKS / DOKS）**都有默认 StorageClass**，会自动动态供给 PV，所以你只需要写 PVC：

```bash
# 先确认集群有默认存储类
kubectl get storageclass
# 带 (default) 标记的那个就是，无需手动建 PV

kubectl apply -f k8s/pvc.yaml
kubectl get pvc            # 期望 ares-data 与 ares-uploads 都是 Bound
```

**为什么是两个 PVC（而不是一个大 PVC 加 subPath）**：

| 方案 | 问题 |
|------|------|
| 单个 PVC + `subPath` 挂两处 | subPath 目录首次创建时属主是 root，非 root 容器写不进去，SQLite 直接报错 |
| 单个 PVC 双挂载（不带 subPath） | `blog.db` 会同时出现在 `/static/uploads/` 下，等于把用户密码哈希挂到公网可下载路径 |
| **两个 PVC（当前方案）** | 各自独立挂载，无权限坑、无泄漏面 |

**只有两种情况需要手动建 PV**：
1. 集群无默认 StorageClass（自建集群 Kubeadm）
2. 要用既有的 NFS / Ceph / 云盘快照恢复

手动 PV 示例（NFS 场景）：

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: ares-pv
spec:
  capacity:
    storage: 5Gi
  accessModes: [ReadWriteOnce]
  persistentVolumeReclaimPolicy: Retain   # 关键：删 PVC 不删数据
  nfs:
    server: 10.0.0.10
    path: /data/ares
```

> **务必把 `reclaimPolicy` 设为 `Retain`**——默认的 `Delete` 会在删 PVC 时连数据一起干掉。

---

## 阶段 4：编写 Deployment 并部署

### 4.1 创建密钥（不要提交到 git）

```bash
cp k8s/secret.example.yaml k8s/secret.yaml
# 编辑填入真实值
SESSION_SECRET=$(openssl rand -hex 32)   # 生成后填入
kubectl apply -f k8s/secret.yaml
```

> `SESSION_SECRET` 迁移时必须沿用旧值，否则所有登录态失效。

### 4.2 修改部署参数

编辑 `k8s/deployment.yaml`：
- `image:` 改成 `ghcr.io/youngsu-ops/ares:latest`
- `BLOG_URL` 改成正式域名
- 如 GHCR 镜像是私有的，需要 imagePullSecret

### 4.3 部署

```bash
# 顺序：PVC(已在阶段3) → Secret → Service → Deployment → Ingress
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/ingress.yaml

# 观察
kubectl get pods -w
kubectl logs -f deploy/ares
kubectl get ingress ares       # 拿 ADDRESS（Ingress 外网 IP）
```

### 4.4 关键设计约束（务必理解）

| 约束 | 原因 | 应对 |
|------|------|------|
| `replicas: 1` | SQLite 单写者，多副本会写坏库 | 保持 1，不要配 HPA |
| `strategy: Recreate` | RWO PVC 无法被两个 Pod 同时挂载 | 先停后起，短暂中断可接受 |
| 不配 HPA | 存储层不支持横向扩展 | 后期换 PostgreSQL + 对象存储再扩容 |

---

## 阶段 5：测试验证

### 5.1 集群内自检

```bash
kubectl exec -it deploy/ares -- curl -s localhost:8888/healthz
kubectl describe pod -l app=ares | grep -A5 Events
```

### 5.2 外网访问 + HTTPS

```bash
# DNS 把 blog.kivensu.club 指向 Ingress 的 ADDRESS
dig +short blog.kivensu.club

# 证书是否签发（cert-manager 约 1-2 分钟）
kubectl get certificate ares-tls
kubectl get secret ares-tls

curl -I https://blog.kivensu.club/
```

### 5.3 功能验证清单

```
□ 首页正常渲染（时间轴动画、CSS 加载）
□ 登录成功（验证 BLOG_URL 协议与访问协议一致，否则 cookie Secure 问题复发）
□ 后台可发布文章
□ 上传图片成功
□ 评论提交成功
□ /robots.txt 和 /sitemap.xml 正常
□ 静态资源 Cache-Control 头正常
```

### 5.4 数据持久化验证（最容易出错的一环）

```bash
# 把旧服务器数据导入 PVC
kubectl cp ./blog.db ares-<pod>:/app/data/blog.db
kubectl cp ./uploads/. ares-<pod>:/app/static/uploads/

# 关键：删 Pod 验证数据不丢
kubectl delete pod -l app=ares
kubectl get pods -w
# 新 Pod 起来后再次访问，文章/用户还在 = 持久化正确
```

### 5.5 回滚演练

```bash
kubectl rollout undo deploy/ares
kubectl rollout status deploy/ares
```

---

## 阶段 6：搭建 CI/CD

CI/CD 要回答三个问题：**何时触发**、**怎么打包**、**推送到哪里**。

```
        push 到 main / 打 v* tag / 手动点按钮
                        │
                        ▼
            GitHub Actions（ubuntu-latest，原生 amd64）
                        │  读 Dockerfile → 两阶段构建
                        ▼
                 镜像 ares:latest / :sha / :v1.0.0
                        │  docker login ghcr.io（用内置 GITHUB_TOKEN）
                        ▼
        ghcr.io/youngsu-ops/ares  ← 集群从这里拉取
```

### 6.1 何时触发（`on:`）

| 触发条件 | 说明 | 产出 tag |
|----------|------|----------|
| push 到 `main` | 日常迭代，自动构建 | `latest` + `sha-xxxxxx` |
| push tag `v*`（如 `v1.0.0`） | 正式发布 | `1.0.0` + `latest` |
| Actions 页面手动点 | 需要重新构建时 | 同 main |

### 6.2 怎么打包（`Dockerfile` + `build-push-action`）

流水线本身不写编译逻辑，它只是调用仓库根目录的 `Dockerfile`：

```yaml
with:
  context: .
  file: ./Dockerfile
  platforms: linux/amd64      # 关键：不指定可能产出 arm64，集群跑不起来
  push: true
  tags: ${{ steps.meta.outputs.tags }}
  cache-from: type=gha        # 复用 Actions 缓存，二次构建 ~40s
  cache-to: type=gha,mode=max
```

> **引用语法注意**：`id: meta` 定义的是 **step id**，引用时必须写 `steps.meta.outputs.tags`。写成 `id.meta.outputs.tags` 会解析为空字符串，导致 tags 为空、推送失败。

### 6.3 推送到哪里（GHCR）

```
ghcr.io/<小写仓库名>:<tag>
例：ghcr.io/youngsu-ops/ares:latest
```

- **认证**：`secrets.GITHUB_TOKEN` 是 Actions 内置的，不需要手动配 secret
- **权限**：必须声明 `packages: write`，否则 403
- **小写要求**：GHCR 不接受大写仓库名，用户名 `Youngsu-ops` 必须先转小写
- **首次推送后**：镜像默认私有。在 GitHub → Packages → ares → Package settings 里改成 public，或给集群配 imagePullSecret

### 6.4 启用步骤

1. GitHub 仓库 → Settings → Actions → General → Workflow permissions 选 **Read and write**
2. push 代码 → Actions 页面看构建日志
3. 构建成功后确认镜像：`docker pull ghcr.io/youngsu-ops/ares:latest`
4. `k8s/deployment.yaml` 里的 image 已是 `ghcr.io/youngsu-ops/ares:latest`

### 6.5 进阶：GitOps 自动发布（可选）

```bash
# ArgoCD 监听仓库，k8s/ 清单变更自动同步到集群
argocd repo add https://github.com/Youngsu-ops/ares.git
argocd app create ares \
  --repo https://github.com/Youngsu-ops/ares.git \
  --path k8s --dest-server https://kubernetes.default.svc \
  --dest-namespace default --sync-policy automated
```

### 6.6 镜像 tag 策略

| 场景 | tag | 说明 |
|------|-----|------|
| 开发迭代 | `latest` | 需配 `imagePullPolicy: Always`，否则节点复用旧镜像 |
| 生产发布 | `v1.0.0` | 语义化版本，可精确回滚 |
| 问题追溯 | `sha-xxxxxx` | 每个提交一个镜像，便于定位是哪次改动引入的 |

---

## 附：从旧服务器迁移的完整动作

```bash
# ① 旧服务器备份（脚本已在仓库）
bash <(curl -s https://raw.githubusercontent.com/Youngsu-ops/ares/main/scripts/backup.sh)
# ② 下载备份
scp root@8.145.38.177:/root/ares-backups/ares-backup-*.tar.gz ./
# ③ 解压取出 blog.db 和 uploads/
tar xzf ares-backup-*.tar.gz
# ④ 在阶段 5.4 导入 PVC
# ⑤ 域名 DNS 切到 Ingress IP
# ⑥ 观察 24h，确认无误后释放旧服务器
```

> `SESSION_SECRET` 和 `BLOG_ADMIN_PASS` 沿用旧值，用户无需重新登录。
