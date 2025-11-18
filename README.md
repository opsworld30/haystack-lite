# Haystack-Lite

> 基于 Facebook Haystack 设计的轻量级文件存储系统

[![Go Version](https://img.shields.io/badge/Go-1.25-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

## 简介

Haystack-Lite 是一个高性能的文件存储系统，专为海量小文件存储场景设计。通过将多个小文件聚合到大文件中，配合内存索引，实现了高效的文件存储和检索。

### 核心特性

- 🚀 **零配置启动** - 默认使用 SQLite，开箱即用
- 📦 **聚合存储** - 多个小文件存储在单个 Volume 文件中，减少磁盘碎片
- ⚡ **高性能** - 内存索引 + 顺序写入，O(1) 查找复杂度
- 🔄 **自动轮转** - Volume 达到上限自动创建新文件
- 🔒 **数据安全** - CRC32 校验 + Cookie 验证，确保数据完整性
- 💾 **双数据库** - 支持 SQLite（开发）和 MySQL（生产）
- 🗜️ **后台压缩** - 自动回收已删除文件空间
- 📤 **分片上传** - 支持大文件分片上传和断点续传
- 🌐 **多协议支持** - REST API、S3 兼容 API、WebDAV 协议
- 📊 **监控指标** - Prometheus 指标导出，健康检查接口

## 快速开始

### 使用 Make（推荐）

```bash
# 初始化配置
make init

# 安装依赖
make install

# 运行服务
make run

# 查看所有命令
make help
```

### 手动运行

```bash
# 1. 安装依赖
go mod download

# 2. 初始化配置
cp configs/config.example.yaml configs/config.yaml

# 3. 运行服务
go run main.go

# 4. 使用自定义配置
go run main.go -config=/path/to/config.yaml
```

### 测试 API

```bash
# 上传文件
curl -F "file=@test.txt" http://localhost:8080/file

# 下载文件
curl http://localhost:8080/file/1 -o downloaded.txt

# 查看状态
curl http://localhost:8080/status

# 健康检查
curl http://localhost:8080/health
```

## API 接口

### 基础文件操作

| 方法   | 路径                  | 功能         |
| ------ | --------------------- | ------------ |
| POST   | `/file`               | 上传文件     |
| GET    | `/file/:id`           | 下载文件     |
| GET    | `/file/:id/info`      | 获取文件信息 |
| GET    | `/file/:id/preview`   | 在线预览     |
| DELETE | `/file/:id`           | 删除文件     |
| GET    | `/files`              | 列出所有文件 |

### 批量操作

| 方法 | 路径                    | 功能     |
| ---- | ----------------------- | -------- |
| POST | `/files/batch`          | 批量上传 |
| POST | `/files/batch/download` | 批量下载 |
| POST | `/files/batch/delete`   | 批量删除 |

### 分片上传

| 方法   | 路径                                  | 功能             |
| ------ | ------------------------------------- | ---------------- |
| POST   | `/upload/init`                        | 初始化分片上传   |
| POST   | `/upload/:upload_id/chunk/:chunk_index` | 上传分片         |
| POST   | `/upload/:upload_id/complete`         | 完成上传         |
| GET    | `/upload/:upload_id/progress`         | 查询上传进度     |
| DELETE | `/upload/:upload_id`                  | 取消上传         |
| GET    | `/uploads`                            | 列出所有上传任务 |

### S3 兼容 API

| 方法   | 路径                | 功能       |
| ------ | ------------------- | ---------- |
| PUT    | `/s3/:bucket/*key`  | 上传对象   |
| GET    | `/s3/:bucket/*key`  | 下载/列出对象 |
| HEAD   | `/s3/:bucket/*key`  | 获取对象元数据 |
| DELETE | `/s3/:bucket/*key`  | 删除对象   |

### WebDAV 协议

| 方法     | 路径            | 功能         |
| -------- | --------------- | ------------ |
| PROPFIND | `/webdav/*path` | 列出目录     |
| GET      | `/webdav/*path` | 下载文件     |
| PUT      | `/webdav/*path` | 上传文件     |
| DELETE   | `/webdav/*path` | 删除文件     |
| MKCOL    | `/webdav/*path` | 创建目录     |

### 系统管理

| 方法 | 路径                  | 功能             |
| ---- | --------------------- | ---------------- |
| GET  | `/status`             | 系统状态         |
| GET  | `/health`             | 健康检查         |
| GET  | `/health/live`        | 存活检查         |
| GET  | `/health/ready`       | 就绪检查         |
| GET  | `/metrics`            | Prometheus 指标  |
| GET  | `/compaction/stats`   | 压缩统计         |
| POST | `/compaction/run`     | 手动触发压缩     |

详细文档见 [docs/API.md](docs/API.md)

## 配置说明

### 数据库选择

编辑 `configs/config.yaml` 切换数据库：

```yaml
database:
  type: "sqlite"  # 开发环境，零配置
  # type: "mysql"  # 生产环境，高性能
```

**SQLite（默认）**
- 零配置，开箱即用
- 适合单机部署
- QPS < 1000

**MySQL（推荐生产环境）**
- 需要外部 MySQL 服务
- 支持高并发
- QPS > 1000

### 存储配置

```yaml
storage:
  data_dir: "./data"              # 数据目录
  max_volume_size: 1073741824     # Volume 最大大小（1GB）
  volume_file_ext: ".dat"         # Volume 文件扩展名
  sync_interval: 60               # 同步间隔（秒）
  read_only: false                # 只读模式
```

### 压缩配置

```yaml
compaction:
  enabled: true                   # 启用自动压缩
  interval: 3600                  # 检查间隔（秒）
  deleted_threshold: 0.3          # 删除率阈值（30%）
  min_volume_size: 10485760       # 最小压缩体积（10MB）
```

详细配置见 [docs/CONFIG.md](docs/CONFIG.md)

## 系统架构

### 分层设计

```
┌─────────────────────────────────────────┐
│   HTTP API 层（Gin）                    │
│   REST / S3 / WebDAV                    │
├─────────────────────────────────────────┤
│   Store 管理层                          │
│   Volume 管理 / 文件路由 / ID 分配      │
├─────────────────────────────────────────┤
│   存储引擎层                            │
│   Needle 操作 / 索引管理 / CRC 校验     │
├─────────────────────────────────────────┤
│   物理存储层                            │
│   Volume 文件（.dat）+ 数据库           │
└─────────────────────────────────────────┘
```

### 核心概念

- **Needle**：单个文件单元，包含 ID、Cookie、数据、CRC32
- **Volume**：大文件（.dat），包含多个 Needle
- **Store**：管理多个 Volume，负责文件路由和 ID 分配
- **Database**：存储元数据，支持索引重建

### 数据存储

```
data/
├── volume_1.dat          # Volume 文件（聚合存储）
├── volume_2.dat
├── haystack.db           # SQLite 数据库（元数据）
└── chunks/               # 分片上传临时目录
```

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

## 文档导航

### 快速入门
- [快速开始](docs/QUICKSTART.md) - 5 分钟上手指南
- [API 文档](docs/API.md) - REST API 接口说明

### 高级功能
- [新功能文档](docs/NEW_FEATURES.md) - 批量操作、分片上传、后台压缩
- [S3 API](docs/S3_API.md) - S3 兼容 API 使用指南
- [WebDAV](docs/WEBDAV.md) - WebDAV 协议支持
- [文件预览](docs/PREVIEW.md) - 在线预览功能

### 配置与部署
- [配置说明](docs/CONFIG.md) - 配置文件详解
- [架构设计](docs/ARCHITECTURE.md) - 系统设计和技术选型

### 版本历史
- [更新日志](docs/CHANGELOG.md) - 版本更新记录

## 开发指南

### Make 命令

```bash
make help       # 查看所有命令
make init       # 初始化配置文件
make install    # 安装依赖
make build      # 编译程序
make run        # 运行服务（前台）
make run-bg     # 运行服务（后台）
make stop       # 停止后台服务
make status     # 查看运行状态
make logs       # 查看日志
make test       # 运行测试
make fmt        # 格式化代码
make vet        # 代码检查
make clean      # 清理文件
```

### 代码规范

提交代码前请确保：

```bash
go fmt ./...    # 格式化代码
go vet ./...    # 静态检查
make test       # 运行测试
```

### 性能指标

| 数据库  | 适用场景   | QPS      | 部署方式 |
| ------- | ---------- | -------- | -------- |
| SQLite  | 开发/测试  | < 1000   | 单机     |
| MySQL   | 生产环境   | > 1000   | 分布式   |

### 技术栈

- **语言**：Go 1.25
- **框架**：Gin（HTTP）、GORM（ORM）
- **数据库**：SQLite、MySQL
- **配置**：YAML

## 功能特性

### ✅ 已实现

#### 核心功能
- [x] 文件上传、下载、删除
- [x] 文件元数据（文件名、MIME、MD5、时间戳）
- [x] 批量操作（批量上传、下载、删除）
- [x] 分片上传（大文件分片上传）
- [x] 断点续传（上传断点续传）
- [x] 后台压缩（自动回收已删除文件空间）

#### 多协议支持
- [x] REST API（标准 HTTP 接口）
- [x] S3 兼容 API（支持 AWS CLI 和 SDK）
- [x] WebDAV 协议（可挂载为网络磁盘）

#### 运维功能
- [x] 健康检查（liveness/readiness）
- [x] Prometheus 指标导出
- [x] 优雅关闭（30 秒超时）
- [x] 日志管理
- [x] 文件预览（在线预览文档、图片、视频）

#### 数据库支持
- [x] SQLite（零配置，适合开发）
- [x] MySQL（高性能，适合生产）

### 🚧 规划中

#### 性能优化
- [ ] 下载断点续传（Range 请求）
- [ ] 连接池优化
- [ ] Redis 缓存层
- [ ] 异步写入
- [ ] 批量写入
- [ ] 读写分离

#### 安全增强
- [ ] JWT/OAuth2 认证
- [ ] API 限流
- [ ] 文件加密存储
- [ ] 访问日志审计
- [ ] 细粒度权限控制

#### 分布式支持
- [ ] 多节点部署
- [ ] 负载均衡
- [ ] 数据复制
- [ ] 故障转移
- [ ] 一致性哈希

#### 其他功能
- [ ] 图片处理（缩略图、裁剪、水印）
- [ ] CDN 集成
- [ ] 配置热重载

详细使用文档见 [docs/NEW_FEATURES.md](docs/NEW_FEATURES.md)

## 贡献

欢迎提交 Issue 和 Pull Request！

在提交 PR 前，请确保：

- 代码通过 `go fmt` 格式化
- 代码通过 `go vet` 检查
- 添加必要的测试
- 更新相关文档

## 许可证

MIT License
