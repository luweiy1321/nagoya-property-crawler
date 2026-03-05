# Streamlit Cloud 部署指南

## 📖 概述

本项目的Streamlit应用需要连接到PostgreSQL数据库。由于Streamlit Cloud运行在云端，无法直接访问本地数据库，因此需要使用云端数据库。

## 🚀 部署步骤

### 1. 创建云端数据库（推荐）

#### 选项A: Supabase（推荐）
1. 访问 https://supabase.com
2. 注册并创建新项目
3. 进入 SQL Editor，运行 `migrations/001_init.sql` 创建表
4. Settings > Database > 获取连接信息

#### 选项B: Neon
1. 访问 https://neon.tech
2. 注册并创建新项目
3. 在 SQL Editor 运行 `migrations/001_init.sql`
4. 获取连接字符串

#### 选项C: Railway
1. 访问 https://railway.app
2. New Project > Provision PostgreSQL
3. 获取连接信息

### 2. 迁移数据

从本地数据库导出数据：
```bash
pg_dump nagoya_properties -U lw -t properties > properties_backup.sql
```

导入到云端数据库：
```bash
psql "your-cloud-database-uri" < properties_backup.sql
```

### 3. 部署到 Streamlit Cloud

1. 访问 https://streamlit.io/cloud
2. 点击 "New app"
3. 选择仓库：`luweiy1321/nagoya-property-crawler`
4. 主文件：`streamlit_app.py`

### 4. 配置 Secrets

在 Streamlit Cloud 应用设置中添加以下 Secrets：

```toml
DB_HOST="your-host.supab.co"
DB_PORT="5432"
DB_USER="postgres"
DB_PASSWORD="your-password"
DB_NAME="postgres"
```

点击 "Deploy" 即可！

## 🔄 自动更新数据

如果你想持续更新云端数据库：

1. **本地爬虫 + 云端数据库**

修改 `config.yaml` 中的数据库配置：
```yaml
database:
  host: "your-host.supab.co"
  port: 5432
  user: "postgres"
  password: "your-password"
  dbname: "postgres"
  sslmode: "require"
```

2. **GitHub Actions 自动爬取**（待实现）

## 📱 本地运行

```bash
pip install -r requirements.txt
streamlit run streamlit_app.py
```

访问: http://localhost:8501
