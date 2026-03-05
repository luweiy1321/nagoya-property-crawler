# 日本名古屋房产爬虫系统

基于 Go + Chromedp 的日本房产信息爬取系统，支持爬取 SUUMO、HOMES、at-home 三大房产网站，使用 PostgreSQL 存储，并提供 Bootstrap 网页展示界面。

## 功能特性

- 🏠 **多网站爬取**：支持 SUUMO、HOMES、at-home 三大房产网站
- 📊 **PostgreSQL 存储**：完整的数据存储和去重机制
- 🌐 **Web 界面**：基于 Bootstrap 的响应式网页界面
- ⏰ **定时任务**：每两天自动爬取更新
- 🔍 **高级筛选**：支持价格、面积、房型等多维度筛选
- 📥 **数据导出**：支持 CSV/JSON 格式导出
- 🚀 **高性能**：并发爬取，连接池管理

## 项目结构

```
nagoya-property-crawler/
├── cmd/
│   ├── crawler/          # 爬虫主程序
│   │   └── main.go
│   └── server/           # Web服务器
│       └── main.go
├── internal/
│   ├── crawler/
│   │   ├── suumo.go      # SUUMO专用爬虫
│   │   ├── homes.go      # HOMES专用爬虫
│   │   └── athome.go     # at-home专用爬虫
│   ├── models/
│   │   └── property.go   # 数据模型定义
│   ├── database/
│   │   └── postgres.go   # PostgreSQL操作
│   ├── web/
│   │   ├── handlers.go   # HTTP处理器
│   │   └── templates/    # HTML模板
│   ├── config/
│   │   └── config.go     # 配置管理
│   └── scheduler/
│       └── scheduler.go  # 定时任务
├── migrations/
│   └── 001_init.sql      # 数据库迁移
├── static/               # 静态资源
│   ├── css/
│   └── js/
├── config.yaml           # 配置文件
├── go.mod
└── README.md
```

## 快速开始

### 1. 环境要求

- Go 1.24+
- PostgreSQL 12+
- Chrome/Chromium 浏览器

### 2. 安装依赖

```bash
# 安装 Chrome (macOS)
brew install --cask google-chrome

# 安装 Chrome (Ubuntu)
sudo apt-get install chromium-browser

# 下载 Go 依赖
go mod download
```

### 3. 数据库设置

```bash
# 创建数据库
psql -U postgres -c "CREATE DATABASE nagoya_properties;"

# 运行迁移脚本
psql -U postgres -d nagoya_properties -f migrations/001_init.sql
```

### 4. 配置文件

编辑 `config.yaml` 配置数据库连接：

```yaml
database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "your_password"
  dbname: "nagoya_properties"
```

### 5. 运行爬虫

```bash
# 爬取所有来源
go run cmd/crawler/main.go

# 爬取特定来源
go run cmd/crawler/main.go -source=suumo

# 爬取特定类型
go run cmd/crawler/main.go -type=rent

# 指定页数
go run cmd/crawler/main.go -pages=10
```

### 6. 启动 Web 服务器

```bash
go run cmd/server/main.go
# 访问 http://localhost:8080
```

## Web 界面功能

- **首页**：统计概览 + 筛选表单
- **列表页**：房产卡片列表 + 分页
- **详情页**：单个房产完整信息
- **下载页**：CSV/JSON 导出

## API 接口

```
GET /api/stats       # 获取统计数据
GET /api/properties  # 获取房产列表（支持分页和筛选）
GET /api/health      # 健康检查
```

## 筛选参数

| 参数 | 说明 | 示例 |
|------|------|------|
| source | 数据源 | suumo, homes, athome |
| type | 房源类型 | rent, sale |
| min_price | 最低价格（日元） | 50000 |
| max_price | 最高价格（日元） | 100000 |
| min_area | 最小面积（㎡） | 20 |
| max_area | 最大面积（㎡） | 80 |
| layout | 房型 | 1LDK, 2DK, etc. |
| station | 最近车站 | 名古屋 |
| limit | 每页数量 | 20 |
| offset | 偏移量 | 0 |

## 定时任务

系统支持定时自动爬取，默认每2天运行一次。可在 `config.yaml` 中配置：

```yaml
scheduler:
  cron: "0 0 0 */2 * *"  # 每2天凌晨执行
  enabled: true
```

## 注意事项

1. **网站结构变化**：CSS 选择器可能失效，需要定期维护
2. **反爬虫**：建议使用代理，控制爬取频率
3. **法律合规**：仅供学习研究使用，遵守 robots.txt
4. **数据准确性**：价格等信息需人工验证

## 开发

```bash
# 运行测试
go test ./...

# 格式化代码
go fmt ./...

# 代码检查
go vet ./...
```

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！
