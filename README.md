# Haystack-Lite

> 基于 Haystack 设计思想的轻量级文件存储系统

[![Go Version](https://img.shields.io/badge/Go-1.25-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

## 特性

- 🚀 **零配置启动** - 默认使用 SQLite，开箱即用
- 📦 **聚合存储** - 多个小文件存储在一个大文件中
- ⚡ **高性能** - 内存索引 + 顺序写入，O(1) 查找
- 🔄 **自动轮转** - Volume 达到上限自动创建新文件
- 🔒 **数据安全** - CRC 校验 + Cookie 验证
- 💾 **双数据库** - 支持 SQLite（开发）和 MySQL（生产）

## 快速开始

```bash
# 1. 克隆项目
git clone <repo-url>
cd haystack-lite

# 2. 安装依赖
go mod download

# 3. 运行服务
go run main.go

# 4. 测试 API
curl -F "file=@test.txt" http://localhost:8080/file
```

## API

| 方法   | 路径        | 功能     |
| ------ | ----------- | -------- |
| POST   | `/file`     | 上传文件 |
| GET    | `/file/:id` | 下载文件 |
| DELETE | `/file/:id` | 删除文件 |
| GET    | `/status`   | 查看状态 |

详细文档见 [docs/API.md](docs/API.md)

## 配置

编辑 `configs/config.yaml` 切换数据库：

```yaml
database:
  type: "sqlite" # 或 "mysql"
```

详细配置见 [docs/CONFIG.md](docs/CONFIG.md)

## 架构

```
┌─────────────────┐
│   HTTP API      │  Gin 框架
├─────────────────┤
│  Store 管理层   │  Volume 管理、文件路由
├─────────────────┤
│  存储引擎层     │  Needle 操作、索引管理
├─────────────────┤
│  物理存储层     │  .dat 文件 + 数据库
└─────────────────┘
```

**数据存储：**

- 文件数据：`./data/volume_*.dat`
- 元数据：`./data/haystack.db` (SQLite) 或 MySQL

详细设计见 [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)

## 项目结构

```
haystack-lite/
├── main.go              # 程序入口
├── internal/            # 私有代码（不可被外部 import）
│   ├── api/             # HTTP 接口层
│   ├── config/          # 配置管理
│   └── storage/         # 存储引擎
├── configs/             # 配置文件
│   ├── config.yaml
│   └── config.example.yaml
├── scripts/             # 脚本文件
│   └── test.sh
├── docs/                # 文档
└── data/                # 数据目录
```

> 本项目遵循 [Standard Go Project Layout](https://github.com/golang-standards/project-layout)

## 文档

- [快速开始](docs/QUICKSTART.md) - 5 分钟上手指南
- [API 文档](docs/API.md) - 接口使用说明
- [新功能文档](docs/NEW_FEATURES.md) - 批量操作、分片上传、后台压缩
- [S3 API](docs/S3_API.md) - S3 兼容 API 使用指南
- [WebDAV](docs/WEBDAV.md) - WebDAV 协议支持
- [文件预览](docs/PREVIEW.md) - 在线预览功能
- [配置说明](docs/CONFIG.md) - 配置文件详解
- [架构设计](docs/ARCHITECTURE.md) - 系统设计和技术选型

## 开发

```bash
make help       # 查看所有命令
make init       # 初始化配置
make run        # 运行服务
make build      # 编译程序
make test       # 运行测试
```

## 性能

- **SQLite**: 适合单机部署，QPS < 1000
- **MySQL**: 适合生产环境，QPS > 1000

## 功能列表

### ✅ 已实现功能

- [x] **后台压缩（Compaction）** - 自动清理已删除文件，回收磁盘空间
- [x] **文件元数据** - 支持文件名、MIME 类型、MD5、创建时间
- [x] **批量操作** - 批量上传、下载、删除
- [x] **文件分片上传** - 支持大文件分片上传
- [x] **断点续传** - 支持上传断点续传

详细使用文档见 [docs/NEW_FEATURES.md](docs/NEW_FEATURES.md)

### 待开发功能

#### 核心功能

- [ ] 下载断点续传 - 支持 Range 请求

#### 性能优化

- [ ] 连接池优化 - 优化数据库连接池配置
- [ ] 缓存层 - 集成 Redis 缓存热点数据
- [ ] 异步写入 - 异步写入数据库，提高吞吐量
- [ ] 批量写入 - 批量写入数据库，减少 I/O
- [ ] 读写分离 - 支持 MySQL 主从读写分离

### 安全增强

- [ ] 认证授权 - JWT/OAuth2 认证
- [ ] API 限流 - 基于 IP/用户的限流
- [ ] 文件加密 - 支持文件加密存储
- [ ] 访问日志 - 记录所有文件访问日志
- [ ] 权限控制 - 细粒度的文件访问权限

### 运维支持

- [x] 监控指标 - Prometheus 指标导出
- [x] 健康检查 - HTTP 健康检查接口（liveness/readiness）
- [x] 优雅关闭 - 优雅关闭服务，等待请求完成（30秒超时）
- [ ] 配置热重载 - 支持配置文件热重载
- [x] 日志管理 - 日志输出和查看

### 分布式支持

- [ ] 多节点部署 - 支持多节点水平扩展
- [ ] 负载均衡 - 节点间负载均衡
- [ ] 数据复制 - 跨节点数据复制
- [ ] 故障转移 - 自动故障检测和转移
- [ ] 一致性哈希 - 数据分片和路由

### 其他功能

- [ ] 图片处理 - 缩略图、裁剪、水印
- [ ] CDN 集成 - 支持 CDN 加速
- [x] **S3 兼容** - 兼容 S3 API，支持 AWS CLI 和 SDK
- [x] **WebDAV 支持** - 支持 WebDAV 协议，可挂载为网络磁盘
- [x] **文件预览** - 在线预览文档、图片、视频

## 贡献

欢迎提交 Issue 和 Pull Request！

在提交 PR 前，请确保：

- 代码通过 `go fmt` 格式化
- 代码通过 `go vet` 检查
- 添加必要的测试
- 更新相关文档

## 许可证

MIT License
