#!/bin/bash

# 迁移数据到云端数据库的脚本

echo "=== 名古屋房产数据库迁移工具 ==="
echo ""
echo "推荐免费云数据库服务："
echo "1. Supabase - https://supabase.com (免费500MB)"
echo "2. Neon - https://neon.tech (免费无限制)"
echo "3. Railway - https://railway.app (免费$5额度)"
echo ""

read -p "选择服务 (1/2/3): " choice

case $choice in
  1)
    echo "=== Supabase 设置 ==="
    echo "1. 访问 https://supabase.com 并注册"
    echo "2. 创建新项目"
    echo "3. 进入 SQL Editor，运行以下命令创建表："
    echo ""
    cat migrations/001_init.sql
    echo ""
    echo "4. 获取连接信息：Settings > Database > Connection string > URI"
    echo ""
    read -p "输入数据库URI: " db_uri

    echo "5. 导出本地数据..."
    pg_dump nagoya_properties -U lw -t properties > properties_backup.sql

    echo "6. 导入到云端数据库..."
    psql "$db_uri" < properties_backup.sql

    echo "✅ 迁移完成！"
    echo ""
    echo "在 Streamlit Cloud Secrets 中添加："
    echo "DB_HOST=$(echo $db_uri | grep -oP '@\K[^:]*')"
    echo "DB_PORT=5432"
    echo "DB_USER=postgres"
    echo "DB_PASSWORD=your_password"
    echo "DB_NAME=postgres"
    ;;
  2)
    echo "=== Neon 设置 ==="
    echo "1. 访问 https://neon.tech 并注册"
    echo "2. 创建新项目"
    echo "3. 在 SQL Editor 运行 migrations/001_init.sql"
    echo "4. 获取连接字符串"
    echo ""
    echo "然后手动运行数据迁移"
    ;;
  *)
    echo "无效选择"
    exit 1
    ;;
esac
