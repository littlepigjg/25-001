# LogAlert - 实时日志聚合与告警规则引擎

基于 Go 实现的实时日志聚合与告警规则引擎，一款轻量级的日志监控与告警服务，完成日志接收、存储、查询、告警规则管理和统计分析等核心功能。

## 项目简介

LogAlert 是一个纯 Go 标准库实现的实时日志聚合与告警规则引擎，主要功能包括：

- **日志接收**：通过 HTTP POST 接收多服务日志条目，支持按级别（INFO/WARN/ERROR/DEBUG/FATAL）、来源、关键词存储
- **日志查询**：支持按时间范围、级别、关键词等多条件查询日志
- **告警规则管理**：支持告警规则的 CRUD 操作，可按来源、级别、时间窗口、阈值灵活配置
- **定时扫描**：基于时间窗口的告警规则定时扫描，触发后记录告警事件
- **告警管理**：支持告警事件的确认、解决、删除等操作
- **统计分析**：按来源、级别、小时等维度统计日志量和错误率趋势
- **健康检查**：提供 /health、/ready、/info 等健康检查端点

## 目录结构

```
logalert/
├── cmd/
│   └── server/
│       └── main.go              # 应用入口，HTTP 服务器启动
├── internal/
│   ├── config/
│   │   └── config.go            # 应用配置管理
│   ├── handler/
│   │   ├── log_handler.go       # 日志 HTTP 处理
│   │   ├── rule_handler.go      # 规则 HTTP 处理
│   │   ├── alert_handler.go     # 告警 HTTP 处理
│   │   ├── stats_handler.go     # 统计 HTTP 处理
│   │   ├── health_handler.go    # 健康检查处理
│   │   ├── query_handler.go     # 查询处理
│   │   ├── config_handler.go    # 配置管理处理
│   │   └── middleware.go        # HTTP 中间件
│   ├── model/
│   │   ├── log_entry.go         # 日志条目模型
│   │   ├── log_level.go         # 日志级别枚举
│   │   ├── alert_rule.go        # 告警规则模型
│   │   ├── alert_event.go       # 告警事件模型
│   │   ├── query.go             # 查询条件模型
│   │   ├── statistics.go        # 统计模型
│   │   ├── errors.go            # 错误类型
│   │   ├── helpers.go           # 辅助函数
│   │   ├── source_config.go     # 来源配置模型
│   │   ├── validator.go         # 校验器
│   │   ├── policy.go            # 策略模型
│   │   └── system.go            # 系统信息模型
│   ├── service/
│   │   ├── log_service.go       # 日志业务逻辑
│   │   ├── rule_service.go      # 规则业务逻辑
│   │   ├── alert_service.go     # 告警业务逻辑
│   │   ├── stats_service.go     # 统计业务逻辑
│   │   ├── scheduler_service.go # 调度服务
│   │   ├── cleanup_service.go   # 清理服务
│   │   ├── window_manager.go    # 时间窗口管理
│   │   ├── rate_limiter.go      # 速率限制器
│   │   └── aggregation_service.go # 聚合服务
│   └── store/
│       ├── store_interface.go   # 存储接口定义
│       ├── memory_log_store.go  # 内存日志存储
│       ├── memory_rule_store.go # 内存规则存储
│       └── memory_alert_store.go # 内存告警存储
├── pkg/
│   ├── logger/
│   │   ├── logger.go            # 结构化日志
│   │   └── sink.go              # 日志输出目标
│   ├── response/
│   │   ├── response.go          # 统一响应格式
│   │   └── time.go              # 时间工具
│   ├── timeutil/
│   │   ├── time_window.go       # 时间窗口
│   │   └── time_range.go        # 时间范围
│   ├── jsonutil/
│   │   ├── json.go              # JSON 读写
│   │   └── validate.go          # JSON 校验
│   ├── httputil/
│   │   ├── middleware.go        # HTTP 中间件
│   │   └── client.go            # HTTP 客户端
│   ├── syncutil/
│   │   ├── safe_map.go          # 线程安全 Map
│   │   └── waitgroup.go         # 同步原语
│   └── errors/
│       ├── errors.go            # 增强错误类型
│       └── codes.go             # 错误码映射
├── web/
│   └── index.html               # 前端管理页面
├── go.mod
├── go.sum
├── BUG_CATALOG.md               # 缺陷候选清单
├── benzhi.Dockerfile            # Docker 构建文件
├── build_benzhi_docker.sh       # 构建脚本
└── BENZHI_README.md             # 本文档
```

## API 文档

### 日志 API

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/logs` | 创建日志条目 |
| POST | `/api/logs/batch` | 批量创建日志 |
| GET | `/api/logs` | 查询日志（支持 source, level, keyword, start_time, end_time, limit, offset, order 参数） |
| GET | `/api/logs/{id}` | 获取单条日志 |
| DELETE | `/api/logs/{id}` | 删除日志 |
| DELETE | `/api/logs?source=xxx` | 删除指定来源的日志 |

**请求体示例 (POST /api/logs)**:
```json
{
  "level": "ERROR",
  "source": "order-service",
  "message": "Database connection timeout",
  "keywords": ["database", "timeout"]
}
```

**响应体示例**:
```json
{
  "code": 0,
  "message": "created",
  "data": {
    "id": "abc123...",
    "timestamp": "2024-01-01T12:00:00Z",
    "level": "ERROR",
    "source": "order-service",
    "message": "Database connection timeout",
    "keywords": ["database", "timeout"]
  }
}
```

### 规则 API

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/rules` | 创建告警规则 |
| GET | `/api/rules` | 列出所有规则 |
| GET | `/api/rules/{id}` | 获取单条规则 |
| PUT | `/api/rules/{id}` | 更新规则 |
| DELETE | `/api/rules/{id}` | 删除规则 |
| POST | `/api/rules/{id}/enable` | 启用规则 |
| POST | `/api/rules/{id}/disable` | 禁用规则 |

**请求体示例 (POST /api/rules)**:
```json
{
  "name": "订单服务错误监控",
  "source": "order-service",
  "level": "ERROR",
  "window_minutes": 5,
  "threshold": 10
}
```

### 告警 API

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/alerts` | 列出所有告警 |
| GET | `/api/alerts?active=true` | 仅列出活跃告警 |
| GET | `/api/alerts/{id}` | 获取单条告警 |
| POST | `/api/alerts/{id}/acknowledge` | 确认告警 |
| POST | `/api/alerts/{id}/resolve` | 解决告警 |
| DELETE | `/api/alerts/{id}` | 删除告警 |
| GET | `/api/alerts/active/count` | 统计活跃告警数 |

### 统计 API

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/stats` | 获取日志统计 |
| GET | `/api/stats/hourly?hours=24` | 获取每小时趋势 |
| GET | `/api/stats/sources?limit=10` | 获取来源统计 |
| GET | `/api/stats/daily?date=2024-01-01` | 获取日报 |
| GET | `/api/stats/error-rate?hours=24` | 获取错误率趋势 |

### 健康检查 API

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/health` | 健康检查（返回状态、运行时信息、统计数据） |
| GET | `/ready` | 就绪检查（检查所有依赖是否就绪） |
| GET | `/info` | 服务信息（版本、配置、API列表） |

## 本地运行

### 前提条件

- Go 1.22+
- 操作系统：Linux/macOS/Windows

### 运行步骤

```bash
# 克隆或进入项目目录
cd logalert

# 编译项目
go build ./...

# 运行服务
go run ./cmd/server

# 服务启动后访问:
# - API: http://localhost:8080/api/logs
# - 健康检查: http://localhost:8080/health
# - 就绪检查: http://localhost:8080/ready
# - 服务信息: http://localhost:8080/info
# - 前端页面: http://localhost:8080/
```

### 测试示例

```bash
# 创建一条日志
curl -X POST http://localhost:8080/api/logs \
  -H "Content-Type: application/json" \
  -d '{"level":"ERROR","source":"test-service","message":"Test error message"}'

# 查询日志
curl "http://localhost:8080/api/logs?source=test-service&limit=10"

# 创建告警规则
curl -X POST http://localhost:8080/api/rules \
  -H "Content-Type: application/json" \
  -d '{"name":"测试规则","source":"test-service","level":"ERROR","window_minutes":1,"threshold":1}'

# 查看统计
curl http://localhost:8080/api/stats

# 查看健康状态
curl http://localhost:8080/health

# 查看就绪状态
curl http://localhost:8080/ready
```

## Docker 构建和运行

### 使用构建脚本

```bash
# 默认构建
./build_benzhi_docker.sh

# 自定义参数
./build_benzhi_docker.sh my-logalert v1.0 linux/amd64
```

### 手动构建

```bash
# 构建镜像
docker build -f benzhi.Dockerfile -t logalert:latest .

# 运行容器
docker run -d --name logalert -p 8080:8080 logalert:latest

# 查看日志
docker logs -f logalert

# 停止并删除
docker stop logalert && docker rm logalert
```

## 测试命令

```bash
# 编译检查
go build ./...

# 静态代码检查
go vet ./...

# 运行带竞态检测的测试
go test -race -count=1 ./...

# 运行所有测试并显示覆盖率
go test -cover -count=1 ./...

# 基准测试
go test -bench=. ./...
```

## 技术栈

- **语言**: Go 1.22
- **框架**: 纯标准库 (net/http)
- **存储**: 内存存储 (可扩展)
- **特性**:
  - 优雅关闭 (SIGINT/SIGTERM)
  - 结构化日志
  - 中间件支持 (日志记录、CORS、恢复、速率限制)
  - 参数校验和错误处理
  - 并发安全的数据存储
  - 时间窗口工具
  - 统一响应格式
