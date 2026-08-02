# Ares 数据备份与迁移指南

## 需要备份什么？

Ares 的数据分三类，优先级从高到低：

| 数据 | 路径 | 说明 | 丢失后果 |
|------|------|------|----------|
| 🔴 数据库 | `data/blog.db` | 用户账号、文章、评论、分类、订单 | **不可恢复** |
| 🔴 上传文件 | `static/uploads/` | 文章中的图片、音频、视频 | **不可恢复** |
| 🟡 配置 | `/etc/systemd/system/ares.service` | 环境变量（密钥、密码、域名） | 需重新配置 |
| 🟢 源代码 | GitHub 仓库 | 已推送到 `Youngsu-ops/ares` | 随时可 clone |

---

## 一、备份（在服务器上执行）

### 方式 A：一键脚本（推荐）

```bash
# 下载备份脚本到服务器
curl -o /root/backup.sh https://raw.githubusercontent.com/Youngsu-ops/ares/main/scripts/backup.sh
chmod +x /root/backup.sh

# 执行备份
bash /root/backup.sh
```

产物：`/root/ares-backups/ares-backup-YYYYMMDD-HHMMSS.tar.gz`

### 方式 B：手动备份

```bash
# 创建临时目录
mkdir -p /tmp/ares-backup

# 1. 数据库（先停服务避免写冲突，或用 sqlite3 热备份）
sqlite3 /root/go/src/ares/data/blog.db ".backup '/tmp/ares-backup/blog.db'"

# 2. 上传文件
cp -r /root/go/src/ares/static/uploads /tmp/ares-backup/uploads

# 3. systemd 配置
cp /etc/systemd/system/ares.service /tmp/ares-backup/

# 4. 打包
cd /tmp
tar czf ares-backup-$(date +%Y%m%d).tar.gz ares-backup/
```

### 下载到本地保存

```bash
# 在你的 Mac 上执行
scp root@8.145.38.177:/root/ares-backups/ares-backup-*.tar.gz ~/Desktop/
```

---

## 二、迁移到新服务器

### Step 1：在新服务器安装环境

```bash
# 安装 Go 1.24
wget https://go.dev/dl/go1.24.6.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.24.6.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
go version  # 确认输出 go1.24.6

# 安装 gcc（SQLite 需要）
yum install -y gcc  # CentOS
# apt install -y gcc  # Ubuntu
```

### Step 2：拉取代码并编译

```bash
cd /opt
git clone https://github.com/Youngsu-ops/ares.git
cd ares
go build -o ares .
chmod +x ares
```

### Step 3：恢复数据

```bash
# 上传备份文件到新服务器后解压
cd /opt/ares
tar xzf /path/to/ares-backup-*.tar.gz --strip-components=1

# 确认数据恢复
ls -la data/blog.db
ls -la static/uploads/
```

### Step 4：配置 systemd

```bash
# 恢复 systemd 配置（注意修改路径如果不同）
cat > /etc/systemd/system/ares.service << 'EOF'
[Unit]
Description=Ares Blog
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/ares
Environment=BLOG_PORT=8888
Environment=BLOG_URL=https://blog.kivensu.club
Environment=SESSION_SECRET=原来的密钥
Environment=BLOG_ADMIN_PASS=原来的密码
Environment=BLOG_ADMIN_PATH=admin
Environment=GIN_MODE=release
ExecStart=/opt/ares/ares
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable ares
systemctl start ares
systemctl status ares
```

### Step 5：验证

```bash
# 本地访问
curl http://localhost:8888/

# 检查数据
curl -s http://localhost:8888/ | grep -o '<title>.*</title>'
```

### Step 6：更新 DNS / ESA 回源

将阿里云 ESA 的回源地址改为新服务器 IP，DNS 解析指向新 IP。

---

## 三、定期自动备份（可选）

设置 crontab 每天凌晨 3 点自动备份：

```bash
# 编辑 crontab
crontab -e

# 添加以下行（每天凌晨 3 点备份，保留最近 30 天）
0 3 * * * /root/backup.sh >> /var/log/ares-backup.log 2>&1
# 自动清理 30 天前的备份
0 4 * * * find /root/ares-backups -name "*.tar.gz" -mtime +30 -delete
```

---

## 四、快速迁移 Checklist

```
□ 1. 旧服务器执行 backup.sh，生成 tar.gz
□ 2. scp 下载备份到本地
□ 3. 新服务器安装 Go 1.24 + gcc
□ 4. 新服务器 git clone + go build
□ 5. 解压备份到新项目目录
□ 6. 创建 systemd 服务（填入原密钥和密码）
□ 7. systemctl start ares && curl 验证
□ 8. 更新 ESA 回源 IP
□ 9. 确认域名访问正常
□ 10. 旧服务器可安全释放
```

> **注意**：`SESSION_SECRET` 必须保持一致，否则所有已登录用户的 session 会失效，需要重新登录。`BLOG_ADMIN_PASS` 也需要保持一致，否则管理员密码会变。
