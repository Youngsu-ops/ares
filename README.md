# Ares

> 记录技术与生活的点滴，探索无限可能

基于 **Go + Gin** 构建的轻量级个人博客系统，采用苹果未来金属科技 UI 风格，支持 Markdown 写作、付费专栏、评论系统和完整的 SEO 优化。项目代号 **Ares**（阿瑞斯），寓意如战神般坚定、果敢地守护每一篇创作。

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://go.dev/)
[![Gin](https://img.shields.io/badge/Gin-1.10-blue?logo=go)](https://gin-gonic.com/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

---

## 功能概览

### 核心功能
- **时间轴首页** — 倒序文章时间轴，带穿梭动画与悬停日期气泡
- **Markdown 编辑器** — 三模式（分屏/编辑/预览）+ 全屏 + 图片拖拽/粘贴上传
- **付费专栏** — 板块关联文章，统一定价，购买后解锁阅读
- **评论系统** — 登录可评 + 审核机制 + XSS 防护（bluemonday）
- **管理后台** — 文章/分类/板块/评论全生命周期管理
- **用户系统** — 注册/登录 + bcrypt 加密 + 角色分离

### SEO 优化
- Open Graph / Twitter Card meta 标签
- Schema.org JSON-LD 结构化数据（Article + WebSite + BreadcrumbList）
- 动态 Sitemap.xml + Robots.txt
- Canonical URL + RSS Feed
- 完整的 meta description / keywords

### 安全加固
- CSP (Content-Security-Policy) 响应头
- CSRF 令牌校验（Origin/Referer）
- 速率限制（登录 5次/分钟/ip，注册 3次/分钟/ip）
- bluemonday HTML 二次清洗（Markdown + 评论）
- 文件上传 MIME 类型 + 扩展名双重校验
- Cookie Secure/SameSite/HttpOnly
- 可配置的代理信任（CDN/ESA 回源场景）

---

## 技术栈

| 层级 | 技术 |
|------|------|
| 语言 | Go 1.24+ |
| Web 框架 | Gin |
| ORM | GORM (SQLite) |
| 模板引擎 | Go `html/template` |
| 密码加密 | bcrypt |
| Markdown | goldmark + bluemonday |
| 代码高亮 | highlight.js |
| 数据库 | SQLite |

---

## 项目结构

```
.
├── main.go                  # 入口：路由注册、中间件链
├── config/
│   └── config.go            # 配置加载（环境变量驱动）
├── database/
│   └── db.go                # 数据库初始化 + 默认管理员创建
├── handlers/
│   ├── admin.go             # 后台管理（文章 CRUD）
│   ├── auth.go              # 认证（登录/注册/登出）
│   ├── comment.go           # 评论管理
│   ├── post.go              # 公共页面（首页/文章/分类/搜索/关于）
│   ├── seo.go               # SEO 路由（robots.txt/sitemap.xml）
│   ├── seo_data.go          # SEO 模板数据注入
│   ├── section.go           # 付费板块
│   └── upload.go            # 文件上传
├── middleware/
│   ├── auth.go              # 认证/鉴权中间件
│   ├── security.go          # 安全中间件（速率限制/CSP/CSRF）
│   └── section.go           # 付费板块访问控制
├── models/                  # 数据模型定义
├── utils/
│   └── markdown.go          # Markdown 渲染 + HTML 清洗
├── templates/               # Go 模板文件
│   ├── seo.html             # 共享 SEO meta 片段
│   ├── timeline.html        # 首页时间轴
│   ├── post.html            # 文章详情页
│   ├── admin/               # 管理后台模板
│   ├── auth/                # 认证页面
│   └── section/             # 付费板块页面
├── static/
│   └── css/style.css        # 全局样式（苹果金属科技风）
└── data/
    └── blog.db              # SQLite 数据库（gitignore）
```

---

## 快速开始

### 前置要求
- Go 1.24+
- 无需外部数据库依赖（SQLite 内嵌）

### 本地运行

```bash
# 1. 克隆项目
git clone https://github.com/Youngsu-ops/ares.git
cd ares

# 2. 安装依赖
go mod download

# 3. 启动服务
BLOG_PORT=8888 go run main.go
```

访问 http://localhost:8888

### 编译运行

```bash
go build -o blog .
BLOG_PORT=8888 ./blog
```

---

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `BLOG_PORT` | `8888` | 服务端口 |
| `BLOG_DB` | `data/blog.db` | 数据库路径 |
| `BLOG_URL` | `http://localhost:8888` | 站点完整 URL（用于 SEO/HTTPS 检测） |
| `BLOG_NAME` | `Youngsu's Blog` | 站点名称 |
| `BLOG_KEYWORDS` | `技术,编程,Go,...` | SEO 关键词 |
| `BLOG_OG_IMAGE` | `/static/img/og-default.png` | Open Graph 默认图片 |
| `BLOG_ADMIN_USER` | `admin` | 管理员用户名 |
| `BLOG_ADMIN_PASS` | （随机生成） | 管理员密码（⚠️ 生产环境必须设置） |
| `BLOG_ADMIN_PATH` | `admin` | 后台路径前缀 |
| `SESSION_SECRET` | （随机生成） | 会话加密密钥（⚠️ 生产环境必须设置） |
| `GIN_MODE` | `release` | Gin 运行模式（`debug`/`release`/`test`） |
| `TRUSTED_PROXY` | （空） | CDN/代理 IP 段，逗号分隔 |

### 生产环境最小配置

```bash
export BLOG_URL="https://your-domain.com"
export SESSION_SECRET="$(openssl rand -hex 32)"
export BLOG_ADMIN_PASS="your-strong-password"
export TRUSTED_PROXY="10.0.0.0/8,172.16.0.0/12"
./blog
```

---

## ESA / CDN 部署

项目已为 CDN 边缘加速场景做好准备：

- 静态资源使用**相对路径**引用，CDN 回源无歧义
- 支持 `TRUSTED_PROXY` 配置代理信任（回源 IP 获取）
- Cookie Secure 根据 `BLOG_URL` 协议自动切换
- `isHTTPS()` 自动检测 `https://` 前缀

### ESA 边缘缓存建议

| 路径 | 缓存策略 |
|------|----------|
| `/static/*` | 缓存 365 天 |
| `/*.html` | 缓存 5-10 分钟 |
| `/sitemap.xml` / `/feed` | 缓存 1 小时 |
| `/auth/*` / `/admin/*` | 不缓存 |

### ESA WAF 建议

- 初始模式：**观察模式**，运行 24h 后评估误拦截
- 切换为拦截模式后监控 429/403 错误率
- 保留源站安全响应头（CSP、X-Frame-Options 等）

---

## API 路由

### 公开路由
```
GET  /                   首页时间轴
GET  /post/:slug         文章详情
GET  /category/:slug     分类列表
GET  /search?q=          搜索
GET  /about              关于页面
GET  /feed               RSS Feed
GET  /sections           板块列表
GET  /section/:slug      板块详情
GET  /robots.txt         SEO
GET  /sitemap.xml        SEO
```

### 认证路由（含速率限制）
```
GET/POST  /auth/login     登录（5次/分钟）
GET/POST  /auth/register  注册（3次/分钟）
GET       /auth/logout    登出
```

### 需要登录
```
POST  /post/:slug/comment     发表评论
POST  /section/:slug/purchase 购买板块
```

### 管理后台（需要 admin 角色）
```
GET/POST/DELETE  /admin/posts      文章管理
POST             /admin/upload     文件上传
GET/POST/DELETE  /admin/categories 分类管理
GET/POST/DELETE  /admin/sections   板块管理
GET/POST/DELETE  /admin/comments   评论审核
```

---

## 管理员

首次启动时自动创建管理员账号：

- 用户名：`admin`（可通过 `BLOG_ADMIN_USER` 修改）
- 密码：由 `BLOG_ADMIN_PASS` 环境变量指定，未设置时随机生成并输出到启动日志

```bash
# 查看密码
BLOG_PORT=8888 go run main.go 2>&1 | grep 管理员
```

---

## 安全审计

当前安全评分：**87/100**

| 检查项 | 状态 |
|--------|------|
| CSP 响应头 | ✅ 已配置 |
| CSRF 防护 | ✅ Origin/Referer 校验 |
| 速率限制 | ✅ 登录/注册限流 |
| XSS 防护 | ✅ bluemonday 清洗 |
| 文件上传安全 | ✅ MIME + 扩展名双重校验 |
| Cookie 安全 | ✅ Secure/SameSite/HttpOnly |
| 密码存储 | ✅ bcrypt 哈希 |
| 代理信任 | ✅ 可配置 |
| 会话密钥 | ✅ 环境变量 + 随机回退 |

---

## License

MIT © Youngsu
