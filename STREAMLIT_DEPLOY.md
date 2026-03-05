# Streamlit Cloud 部署指南

## 📖 概述

本项目的Streamlit应用需要连接到PostgreSQL数据库。由于Streamlit Cloud运行在云端，无法直接访问本地数据库，因此需要使用云端数据库。

## 🚀 部署步骤

### 1. 创建云端数据库（推荐使用 Neon）

**Neon** 是推荐的免费PostgreSQL云服务：
- ✅ 完全免费，无限制
- ✅ 自动休眠节省资源
- ✅ 支持分支功能
- ✅ 低延迟（Asia区域）

#### 创建 Neon 数据库：

1. 访问 https://neon.tech
2. 用 GitHub 账号登录
3. 点击 "Create a project"
4. 选择区域（推荐 Tokyo 或 Singapore）
5. 等待项目创建完成

#### 创建表结构：

1. 进入 Neon 项目 → "SQL Editor"
2. 复制 `migrations/001_init.sql` 的内容
3. 粘贴并运行

#### 导入本地数据：

```bash
# 导出本地数据
./export_to_neon.sh

# 或者手动导出
pg_dump nagoya_properties -U lw -t properties --data-only > properties_data.sql
```

然后在 Neon SQL Editor 中运行 `properties_data.sql`

#### 获取连接信息：

在 Neon 项目 Dashboard 点击 "Connection details"，复制连接字符串，例如：
```
postgresql://luweiy:abc123@ep-cool-darkness-123456.us-east-2.aws.neon.tech/neondb
```

解析为：
- DB_HOST: `ep-cool-darkness-123456.us-east-2.aws.neon.tech`
- DB_PORT: `5432`
- DB_USER: `luweiy`
- DB_PASSWORD: `abc123`
- DB_NAME: `neondb`

### 2. 部署到 Streamlit Cloud

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
