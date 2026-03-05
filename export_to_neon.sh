#!/bin/bash

echo "=== 导出本地数据到Neon ==="
echo ""

# 导出数据
echo "1. 导出本地数据..."
pg_dump nagoya_properties -U lw -t properties --data-only > properties_data.sql

echo "✅ 数据已导出到 properties_data.sql"
echo ""
echo "2. 在Neon SQL Editor中运行以下步骤："
echo "   a. 先运行 migrations/001_init.sql 创建表"
echo "   b. 然后运行 properties_data.sql 导入数据"
echo ""
echo "3. 获取Neon连接字符串："
echo "   格式: postgresql://username:password@ep-xxx.region.aws.neon.tech/neondb"
echo ""

# 显示连接模板
echo "=== Streamlit Cloud Secrets 配置 ==="
echo "从Neon的Connection string解析出以下信息："
echo ""
echo "例如: postgresql://luweiy:abc123@ep-cool-darkness-123456.us-east-2.aws.neon.tech/neondb"
echo ""
echo "对应:"
echo "DB_HOST=ep-cool-darkness-123456.us-east-2.aws.neon.tech"
echo "DB_PORT=5432"
echo "DB_USER=luweiy"
echo "DB_PASSWORD=abc123"
echo "DB_NAME=neondb"
