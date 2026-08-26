# Docker 缺陷验证 Implementation Plan

## Repository Research

### 项目现状
- **项目**: logalert - 日志告警服务
- **模块**: `logalert`, Go 1.22
- **入口**: `cmd/server/main.go` - 完整 HTTP 服务器，监听 8080 端口
- **配置**: `config/config.go` 支持 DefaultConfig() 回退，config.json 不存在时使用默认值
- **Dockerfile**: `benzhi.Dockerfile` - 基于 golang:1.22，使用 `go run ./cmd/server` 启动
- **构建脚本**: `build_benzhi_docker.sh` - 有交互式 `read -p` 等待确认
- **构建器**: 已配置 buildx，支持 linux/amd64 和 linux/arm64

### 缺陷信息
- **bug_id**: logalert-nil-11
- **bug_category**: nil
- **缺陷**: SafeMap.Get() 返回 (nil, true) 而非 (nil, false)
- **触发方式**: 调用 GetByID 查询不存在的 ID 或删除后遍历 timeIndex

### Dockerfile 问题
1. 使用 `go run` 而非编译后运行二进制（功能正常但效率低）
2. `go mod download && go build ./...` 会编译全部包包括测试，且会编译所有 main 包
3. 需要确保 cmd/server 被正确编译为二进制

## Files and Modules
- `benzhi.Dockerfile`: 修复为多阶段构建，编译 cmd/server 二进制
- `build_benzhi_docker.sh`: 跳过交互式确认（通过直接调用 docker 命令绕过）
- 无源代码修改（保持缺陷注入状态）

## Implementation Steps

### Step 1: 修复 Dockerfile
- 使用多阶段构建
- 第一阶段: golang:1.22 编译 cmd/server 为二进制
- 第二阶段: 轻量镜像（或保留 golang 镜像以兼容）运行二进制
- 暴露 8080 端口
- 添加健康检查

### Step 2: 构建 amd64 镜像
- 使用 docker buildx build --platform linux/amd64

### Step 3: 构建 arm64 镜像
- 使用 docker buildx build --platform linux/arm64
- 验证 arm64 兼容性

### Step 4: 运行容器
- 清理旧容器
- 启动新容器，映射 8080 端口
- 验证健康检查端点 /health

### Step 5: 容器内验证缺陷
- 在容器内运行 go test 触发缺陷
- 或通过 HTTP API 触发缺陷

### Step 6: 输出验证报告

## Dependencies and Considerations
- QEMU 需已安装以支持 arm64 模拟
- Docker daemon 需运行
- go mod download 需要网络访问（或 go.sum 已存在）
- config.json 不在 .dockerignore 中但 config 加载有默认值回退

## Validation
- Docker 构建 amd64: docker buildx build --platform linux/amd64
- Docker 构建 arm64: docker buildx build --platform linux/arm64
- 容器健康检查: curl http://localhost:8080/health
- 缺陷验证: go test . -count=1 -run '^TestRedGreen$'

## Risks
- arm64 构建可能因缺少 QEMU 而失败 → 检查 buildx builder 平台支持
- 容器内 go test 可能因缺少依赖而失败 → go mod download 已在构建时执行
- 健康检查可能因启动慢而超时 → 已设置 30 秒重试