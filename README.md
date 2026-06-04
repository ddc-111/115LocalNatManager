# 115 云盘本地管理器

本地服务 + Chrome 扩展，用于管理 115 云盘下载任务。

## 功能特性

- **磁力链接检测**：自动检测网页中的磁力链接
- **一键云下载**：直接发送磁力链接到 115 云盘
- **自动下载监控**：自动下载已完成的文件到本地
- **文件管理**：浏览、创建文件夹、删除文件
- **令牌管理**：基于 Refresh Token 的安全认证
- **跨平台**：支持 macOS、Windows 和 Linux

## 快速安装

### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/ddc-111/115LocalNatManager/master/scripts/install-mac.sh | bash
```

### Windows（管理员 PowerShell）

```powershell
irm https://raw.githubusercontent.com/ddc-111/115LocalNatManager/master/scripts/install-windows.ps1 | iex
```

### 手动安装

1. 从 [GitHub Releases](https://github.com/ddc-111/115LocalNatManager/releases) 下载最新版本
2. 解压到目录
3. 运行二进制文件：`./115manager`

## Chrome 扩展安装

1. 下载 `115cloud-extension.zip` 并解压
2. 打开 Chrome，访问 `chrome://extensions`
3. 启用右上角的"开发者模式"
4. 点击"加载已解压的扩展程序"
5. 选择解压后的 `extension` 文件夹

## 配置说明

### 设置刷新令牌

1. 从 [115 开放平台](https://open.115.com/) 获取刷新令牌
2. 点击 Chrome 扩展图标
3. 进入"设置"页面
4. 输入刷新令牌并点击"保存令牌"

### 下载设置

- **下载目录**：已完成文件的本地保存路径
- **监控间隔**：检查已完成下载的频率（默认：30 秒）

## API 接口

后端服务默认运行在 `http://localhost:11580`。

### 身份验证

| 方法 | 接口 | 说明 |
|------|------|------|
| POST | `/api/token` | 设置刷新令牌 |
| GET | `/api/token` | 获取令牌状态 |

### 文件管理

| 方法 | 接口 | 说明 |
|------|------|------|
| GET | `/api/files` | 获取文件列表 |
| GET | `/api/files/:id` | 获取文件详情 |
| PUT | `/api/files/:id` | 重命名文件 |
| POST | `/api/files/delete` | 删除文件 |
| POST | `/api/files/move` | 移动文件 |
| GET | `/api/files/search` | 搜索文件 |
| POST | `/api/folders` | 创建文件夹 |

### 云下载

| 方法 | 接口 | 说明 |
|------|------|------|
| POST | `/api/download` | 添加下载任务 |
| GET | `/api/download/tasks` | 获取任务列表 |
| DELETE | `/api/download/tasks/:hash` | 删除任务 |
| POST | `/api/download/clear` | 清空任务 |
| GET | `/api/download/quota` | 获取下载配额 |
| GET | `/api/download/monitor` | 获取监控状态 |
| POST | `/api/download/monitor` | 切换监控 |

### 配置

| 方法 | 接口 | 说明 |
|------|------|------|
| GET | `/api/config` | 获取配置 |
| PUT | `/api/config` | 更新配置 |

## 开发指南

### 环境要求

- Go 1.21+
- Chrome 浏览器

### 从源码构建

```bash
# 克隆仓库
git clone https://github.com/ddc-111/115LocalNatManager.git
cd 115LocalNatManager

# 构建后端
cd backend
go build -o ../dist/115manager .

# 运行
./dist/115manager
```

### 项目结构

```
115LocalNatManager/
├── backend/                    # Go 后端服务
│   ├── api/                    # 115 API 客户端
│   ├── config/                 # 配置管理
│   ├── handler/                # HTTP 处理器
│   ├── service/                # 业务逻辑
│   └── main.go                 # 入口文件
├── extension/                  # Chrome 扩展
│   ├── content/                # 内容脚本（磁力链检测）
│   ├── background/             # 后台服务
│   └── dashboard/              # 管理界面
├── scripts/                    # 安装脚本
└── .github/workflows/          # CI/CD
```

## 许可证

MIT License

## 贡献指南

1. Fork 本仓库
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request
