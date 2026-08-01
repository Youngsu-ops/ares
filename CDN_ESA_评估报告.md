# Youngsu's Blog — CDN/ESA 部署就绪评估报告

> 评估时间：2026-08-01 | 目标环境：阿里云 ESA（边缘安全加速）+ WAF

---

## 总评：⚠️ 基本可用，但需补齐 4 项关键缺口

| 维度 | 状态 | 评分 |
|------|------|------|
| 静态资源路径规范 | ✅ 相对路径 | 95/100 |
| 代理信任 / HTTPS | ✅ 已配置 | 90/100 |
| 安全性 | ✅ 全面加固 | 87/100 |
| SEO | ✅ 完整 | 85/100 |
| 环境变量化 | ✅ 全部可配置 | 90/100 |
| **缓存策略** | ❌ 无 Cache-Control | **0/100** |
| **传输压缩** | ❌ 无 Gzip/Brotli | **0/100** |
| **静态资源指纹** | ❌ 无版本号 | **0/100** |
| **综合** | ⚠️ | **≈70/100** |

---

## 逐项诊断

### ✅ 1. 静态资源路径 — 合格
所有 CSS/JS/OG Image 均使用绝对路径（`/static/css/style.css`），CDN 回源无歧义。
外部依赖仅 `highlight.js`（cdnjs.cloudflare.com），已纳入 CSP 白名单。主页 **无外部字体**，首屏轻量（CSS 仅 22KB）。

### ✅ 2. 代理信任与 HTTPS — 合格
- `TRUSTED_PROXY` 环境变量 → `SetTrustedProxies()`，ESA 回源 IP 可配
- Cookie Secure 根据 `BLOG_URL` 协议自动切换
- `isHTTPS()` 自动检测 `https://` 前缀
- 监听 `0.0.0.0:PORT`，容器/代理场景友好

### ✅ 3. 安全性 — 合格 (87/100)
CSP、X-Frame-Options、CSRF、速率限制、bluemonday HTML 清洗、文件上传 MIME 校验均已就位。

### ✅ 4. SEO — 合格
Open Graph + Twitter Card + Schema.org JSON-LD + canonical URL + robots.txt + sitemap.xml + RSS Feed，全齐。

### ✅ 5. 环境变量化 — 合格
`BLOG_PORT` / `BLOG_DB` / `BLOG_URL` / `SESSION_SECRET` / `BLOG_ADMIN_PASS` / `TRUSTED_PROXY` / `GIN_MODE` 全部可配置。

---

### ❌ 6. 静态资源缓存策略 — 缺失（核心问题）

**现象**：`r.Static("/static", "./static")` 是 Gin 原生静态文件服务，**不设置任何 Cache-Control 头**。浏览器默认不发 `If-None-Match`/`If-Modified-Since`，每次请求 22KB CSS 都要完整传输。

**影响**：
- CDN 无法识别资源可缓存时长，默认短缓存或永不缓存
- 每次回源都重新传输，浪费 ESA 回源带宽
- 用户每次访问都重新下载静态资源

**修复方案**：自定义静态文件中间件，对 `/static/*` 路径设置：
```
Cache-Control: public, max-age=31536000, immutable
ETag: <文件 hash>
```

### ❌ 7. 传输压缩 — 缺失

**现象**：服务端未启用 Gzip/Brotli，所有响应以原始大小传输。

**影响**：
- CSS 22KB → 可压缩至 5KB（节省 77%）
- HTML 页面 → 可压缩至原大小 20-30%
- 移动网络下体验差，ESA 计费按流量的话费用更高

**修复方案**：引入 Gin Gzip 中间件（`gin-contrib/gzip`），一行代码即可。

### ❌ 8. 静态资源指纹 — 缺失

**现象**：CSS 文件引用为 `<link href="/static/css/style.css">`，无 hash 后缀。

**影响**：更新 CSS 后，CDN 和浏览器继续使用旧缓存，用户看不到新样式。除非每次手动 purge CDN 缓存。

**修复方案**：编译时给 `style.css` 加 hash → `/static/css/style.a3f2b1c.css`，模板中通过构建变量注入。

---

## ESA 部署配置指南（预期配置）

### 源站环境变量
```bash
export BLOG_URL="https://your-domain.com"
export BLOG_PORT="8888"
export SESSION_SECRET="<32字节随机字符串>"
export BLOG_ADMIN_PASS="<强密码>"
export TRUSTED_PROXY="ESA回源IP段,127.0.0.1"
export GIN_MODE="release"
```

### ESA 边缘规则
| 路径 | 缓存策略 | 说明 |
|------|----------|------|
| `/static/*` | 缓存 365 天 | 静态资源长期缓存 |
| `/*.html` (隐式) | 缓存 10 分钟 | HTML 短期缓存 |
| `/sitemap.xml` | 缓存 1 小时 | 搜索引擎爬虫 |
| `/feed` | 缓存 1 小时 | RSS Feed |
| `/post/*`, `/category/*` | 缓存 5 分钟 | 文章页短期缓存 |
| `/auth/*` | 不缓存 | 认证相关 |
| `/admin/*` | 不缓存 | 管理后台 |

### ESA WAF 配置
- 初始模式：**观察模式**，运行 24h 后查看误拦截情况
- 切换为 **拦截模式** 后监控 429/403 错误率
- CC 防护阈值建议：单 IP 50次/分钟（页面浏览场景）

### CDN 安全头
ESA 边缘层建议保留源站的 Security Headers（CSP、X-Frame-Options 等），不做重写。

---

## 结论

**当前可以部署 ESA，但建议先补齐缓存头和压缩后再上线**。4 项缺口预计 30 分钟内可修复，对用户体验和带宽成本影响显著。

SQLite 数据库对博客场景足够（读多写少），ESA WAF 可覆盖管理员用户名可预测等低风险项。
