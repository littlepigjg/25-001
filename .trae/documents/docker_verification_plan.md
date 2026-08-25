# Docker 缺陷验证实施计划

## 仓库研究结论

### 项目概况
- **项目**: logalert — 日志告警系统（Go 1.22）
- **模块路径**: `logalert`
- **入口**: `cmd/server/main.go`（HTTP 服务，监听 8080 端口）
- **缺陷类型**: timezone — 时间窗口使用UTC而非本地时区
- **缺陷ID**: logalert-other-30

### 关键文件
| 文件 | 用途 |
|------|------|
| `benzhi.Dockerfile` | Docker 构建文件，基于 golang:1.22 |
| `build_benzhi_docker.sh` | Docker 构建脚本（需交互输入，将直接使用 docker 命令） |
| `red_green_test.go` | 缺陷验证测试（package main，设置 CST 时区验证告警不触发） |
| `cmd/server/main.go` | HTTP 服务入口，提供 `/health` 健康检查端点 |
| `验收.txt` | 缺陷详情说明 |

### Dockerfile 分析
- 基础镜像: `golang:1.22`
- 工作目录: `/app`
- 构建: `go mod download && go build ./...`
- 启动: `go run ./cmd/server`（监听 8080 端口）

### 缺陷测试机制
- `red_green_test.go` 设置 `time.Local = time.FixedZone("CST", 8*3600)`
- 创建 3 条 ERROR 日志 + 1 条阈值为 3 的规则
- 调用 `EvaluateRule`，因 UTC 时区缺陷导致查询窗口偏移 8 小时，返回 nil
- 输出 `RED (红灯，缺陷未修复)`

### Docker 环境注意事项
- 容器默认时区为 UTC，但测试显式设置 `time.Local`，故不受影响
- Dockerfile 复制整个项目（包括测试文件），可在容器内直接运行 `go test`
- `config.json` 在 .dockerignore 中被排除，但 main.go 有默认配置回退逻辑

## 实施步骤

### Step 1: 构建 Docker 镜像（amd64）
```bash
docker buildx build \
    -f benzhi.Dockerfile \
    --platform linux/amd64 \
    -t logalert:latest \
    --load .
```

### Step 2: 构建 Docker 镜像（arm64）
```bash
docker buildx build \
    -f benzhi.Dockerfile \
    --platform linux/arm64 \
    -t logalert:arm64 \
    --load .
```

### Step 3: 运行容器
```bash
docker rm -f test-verify 2>/dev/null || true
docker run -d --name test-verify \
    -p 8080:8080 \
    -v $(pwd):/app \
    logalert:latest
```

### Step 4: 容器内验证编译与静态检查
```bash
docker exec test-verify /bin/bash -c "cd /app && go build ./... && echo 'BUILD OK' && go vet ./... && echo 'VET OK'"
```

### Step 5: 验证服务可运行
```bash
for i in $(seq 1 30); do
  if curl -s http://localhost:8080/health >/dev/null 2>&1; then
    echo 'RUN OK (service healthy)'
    break
  fi
  sleep 1
done
```

### Step 6: 容器内缺陷验证
```bash
docker exec test-verify /bin/bash -c "cd /app && go test . -count=1 -run '^TestRedGreen$' -v"
```

### Step 7: 生成验证报告
汇总所有结果，输出缺陷验证报告。

### Step 8: 清理
```bash
docker rm -f test-verify 2>/dev/null || true
```

## 风险与处理
1. **交互脚本**: `build_benzhi_docker.sh` 有 `read -p` 交互，改用直接 `docker buildx build` 命令
2. **arm64 镜像加载**: `--load` 可能需要配合 `docker buildx build`，已在步骤中包含
3. **容器内 go 工具链**: Dockerfile 使用 `golang:1.22` 基础镜像，包含完整 Go 工具链
4. **端口冲突**: 若 8080 端口已占用，需先清理旧容器
5. **时区影响**: 容器默认 UTC，但测试显式设置 CST，不受影响

## 验证标准
- Docker 构建 (amd64): PASS — 镜像成功构建
- Docker 构建 (arm64): PASS — 镜像成功构建
- 容器内编译: PASS — `go build ./...` 成功
- 容器内 vet: PASS — `go vet ./...` 无警告
- 服务可运行: PASS — `/health` 返回 200
- 缺陷验证: RED — `TestRedGreen` 输出 "RED (红灯，缺陷未修复)"
