# Streamlit Cloud 部署完整指南

## 🚀 重新部署步骤（5分钟）

### 1. 访问 Streamlit Cloud
打开：https://share.streamlit.io/

确认你已登录，账号应该是：**953172@gmail.com** (GitHub: luweiy1321)

### 2. 创建新应用
- 点击右上角 "**New app**"
- 选择你的GitHub仓库：**luweiy1321/nagoya-property-crawler**
- 主文件：**streamlit_app.py**
- 点击 "**Advanced settings**"

### 3. 配置 Secrets（重要！）
在 "**Secrets**" 部分添加：

```
DB_HOST=ep-floral-cherry-a1xz7gdk.ap-southeast-1.aws.neon.tech
DB_PORT=5432
DB_USER=neondb_owner
DB_PASSWORD=npg_UBEigRoV6Dk5
DB_NAME=neondb
```

### 4. 部署
- 点击 "**Deploy**"
- 等待2-3分钟
- 完成！

## ✅ 部署成功后

你会得到一个公开网址，格式如：`https://你的应用名.streamlit.app`

## 🔧 故障排除

**如果显示 "数据库连接失败"：**
1. 检查Secrets是否正确复制（没有多余空格）
2. 点击应用右上角 "Rerun"

**如果显示 "Received no response"：**
1. 等待几分钟让部署完成
2. 刷新页面

**如果想删除旧应用：**
如果你有权限，在应用设置中可以删除
如果没有权限，旧应用会在30天后自动删除
