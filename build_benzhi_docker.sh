#!/bin/bash

# LogAlert Docker 构建脚本
# 用法: ./build_benzhi_docker.sh [镜像名] [标签] [平台]

set -e

# 默认参数
IMAGE_NAME="${1:-logalert}"
TAG="${2:-latest}"
PLATFORM="${3:-linux/amd64}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}=== LogAlert Docker 构建脚本 ===${NC}"
echo ""

# 检查 Docker 是否可用
if ! command -v docker &> /dev/null; then
    echo -e "${RED}错误: Docker 未安装或不在 PATH 中${NC}"
    echo "请访问 https://docs.docker.com/get-docker/ 安装 Docker"
    exit 1
fi

# 检查 Docker daemon 是否运行
if ! docker info &> /dev/null; then
    echo -e "${RED}错误: Docker daemon 未运行${NC}"
    echo "请启动 Docker 服务: sudo systemctl start docker"
    exit 1
fi

echo -e "${GREEN}✓ Docker 检查通过${NC}"

# 显示构建信息
echo ""
echo "镜像信息:"
echo "  名称: ${IMAGE_NAME}"
echo "  标签: ${TAG}"
echo "  平台: ${PLATFORM}"
echo "  Dockerfile: benzhi.Dockerfile"
echo ""

# 确认（仅在交互式终端时）
if [ -t 0 ]; then
    read -p "开始构建? [Y/n] " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]] && [ -n "$REPLY" ]; then
        echo "构建已取消"
        exit 0
    fi
fi

# 执行构建
echo ""
echo -e "${YELLOW}开始构建...${NC}"
echo ""

# 使用 buildx 构建（支持跨平台）
docker buildx build \
    -f benzhi.Dockerfile \
    --platform "${PLATFORM}" \
    -t "${IMAGE_NAME}:${TAG}" \
    --load \
    .

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✓ 构建成功!${NC}"
    echo ""
    echo "镜像信息:"
    docker images | grep "${IMAGE_NAME}"
    echo ""
    echo "运行示例:"
    echo "  # 前台运行 (查看日志)"
    echo "  docker run --rm -p 8080:8080 ${IMAGE_NAME}:${TAG}"
    echo ""
    echo "  # 后台运行"
    echo "  docker run -d --name logalert -p 8080:8080 ${IMAGE_NAME}:${TAG}"
    echo ""
    echo "  # 查看健康状态"
    echo "  curl http://localhost:8080/health"
    echo ""
    echo "  # 查看服务信息"
    echo "  curl http://localhost:8080/info"
else
    echo ""
    echo -e "${RED}✗ 构建失败${NC}"
    exit 1
fi
