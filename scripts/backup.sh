#!/bin/bash
# ============================================================
# Ares 博客数据备份脚本
# 用法: bash backup.sh
# 产物: ares-backup-YYYYMMDD-HHMMSS.tar.gz
# ============================================================

set -e

# ---- 配置（按实际情况修改）----
APP_DIR="${APP_DIR:-/root/go/src/ares}"       # 项目目录
BACKUP_DIR="${BACKUP_DIR:-/root/ares-backups}" # 备份存放目录
# --------------------------------

TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_FILE="$BACKUP_DIR/ares-backup-$TIMESTAMP.tar.gz"

echo "========================================"
echo "  Ares 数据备份"
echo "  时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo "========================================"

# 创建备份目录
mkdir -p "$BACKUP_DIR"

# 检查项目目录
if [ ! -d "$APP_DIR" ]; then
    echo "[错误] 项目目录不存在: $APP_DIR"
    echo "       如果路径不同，请用 APP_DIR=/your/path bash backup.sh"
    exit 1
fi

# 临时目录
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# ---- 1. 数据库 ----
echo ""
echo "[1/4] 备份数据库..."
if [ -f "$APP_DIR/data/blog.db" ]; then
    # 用 sqlite3 做热备份（如果有的话），否则直接复制
    if command -v sqlite3 &>/dev/null; then
        sqlite3 "$APP_DIR/data/blog.db" ".backup '$TMP_DIR/blog.db'"
        echo "  ✅ SQLite 热备份完成"
    else
        cp "$APP_DIR/data/blog.db" "$TMP_DIR/blog.db"
        echo "  ✅ 数据库文件复制完成（建议安装 sqlite3 做热备份）"
    fi
    DB_SIZE=$(du -h "$TMP_DIR/blog.db" | cut -f1)
    echo "     大小: $DB_SIZE"
else
    echo "  ⚠️  数据库文件不存在，跳过"
fi

# ---- 2. 上传文件 ----
echo ""
echo "[2/4] 备份上传文件..."
if [ -d "$APP_DIR/static/uploads" ]; then
    cp -r "$APP_DIR/static/uploads" "$TMP_DIR/uploads"
    UPLOAD_SIZE=$(du -sh "$TMP_DIR/uploads" | cut -f1)
    echo "  ✅ 上传文件备份完成，大小: $UPLOAD_SIZE"
else
    echo "  ⚠️  上传目录不存在，跳过"
fi

# ---- 3. 配置文件 ----
echo ""
echo "[3/4] 备份配置..."
# systemd 服务文件（包含环境变量配置）
if [ -f /etc/systemd/system/ares.service ]; then
    cp /etc/systemd/system/ares.service "$TMP_DIR/ares.service"
    echo "  ✅ systemd 配置已备份"
fi
# 导出当前环境变量快照（不包含密码明文，仅记录哪些变量已设置）
{
    echo "# Ares 环境变量快照 - $(date)"
    echo "# 以下变量在 systemd 中配置，迁移时需要重新设置"
    echo ""
    grep -oP 'Environment=\K[^=]+' /etc/systemd/system/ares.service 2>/dev/null | while read -r key; do
        echo "# $key = [已设置，迁移时需重新填写]"
    done
} > "$TMP_DIR/env-snapshot.txt"
echo "  ✅ 环境变量快照已记录"

# ---- 4. 版本信息 ----
echo ""
echo "[4/4] 记录版本信息..."
{
    echo "Ares 备份信息"
    echo "备份时间: $(date '+%Y-%m-%d %H:%M:%S')"
    echo "服务器: $(hostname)"
    echo "项目路径: $APP_DIR"
    echo "仓库地址: https://github.com/Youngsu-ops/ares"
    echo ""
    echo "数据清单:"
    echo "  - data/blog.db     (SQLite 数据库)"
    echo "  - static/uploads/  (上传的图片/音频/视频)"
    echo "  - ares.service     (systemd 配置)"
    echo ""
    echo "恢复步骤:"
    echo "  1. 在新服务器: git clone https://github.com/Youngsu-ops/ares.git"
    echo "  2. go build -o ares ."
    echo "  3. 解压备份: tar xzf ares-backup-*.tar.gz -C /path/to/ares/"
    echo "  4. 恢复 systemd 配置"
    echo "  5. systemctl daemon-reload && systemctl start ares"
} > "$TMP_DIR/BACKUP_INFO.txt"
echo "  ✅ 版本信息已记录"

# ---- 打包 ----
echo ""
echo "正在打包..."
tar czf "$BACKUP_FILE" -C "$TMP_DIR" .

FINAL_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
echo ""
echo "========================================"
echo "  ✅ 备份完成!"
echo "========================================"
echo ""
echo "  备份文件: $BACKUP_FILE"
echo "  文件大小: $FINAL_SIZE"
echo ""
echo "  下载到本地（在你的电脑执行）:"
echo "  scp root@服务器IP:$BACKUP_FILE ./"
echo ""
echo "  恢复数据时解压即可:"
echo "  tar xzf ares-backup-*.tar.gz -C /path/to/ares/"
echo "========================================"

# 列出备份目录中所有备份
echo ""
echo "历史备份列表:"
ls -lh "$BACKUP_DIR"/ares-backup-*.tar.gz 2>/dev/null || echo "  （无）"
