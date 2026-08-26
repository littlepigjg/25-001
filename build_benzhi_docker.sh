#!/bin/bash

# LogAlert Docker 构建脚本
# 用法: ./build_benzhi_docker.sh [镜像名] [标签] [平台]

set -e

# 默认参数
IMAGE_NAME="${1:-logalert}"
TAG="${2:-latest}"
PLATFORM="${3:-linux/amd64}"

echo "=== LogAlert Docker 构建脚本 ==="
echo ""

# 检查 Docker 是否可用
if ! command -v docker &> /dev/null; then
    echo "错误: Docker 未安装或不在 PATH 中"
    exit 1
fi

# 检查 Docker daemon 是否运行
if ! docker info &> /dev/null; then
    echo "错误: Docker daemon 未运行"
    exit 1
fi

echo "✓ Docker 检查通过"

# 显示构建信息
echo ""
echo "镜像信息:"
echo "  名称: ${IMAGE_NAME}"
echo "  标签: ${TAG}"
echo "  平台: ${PLATFORM}"
echo "  Dockerfile: benzhi.Dockerfile"
echo ""

# 执行构建
echo "开始构建..."
echo ""

# 构建镜像
docker buildx build \
    -f benzhi.Dockerfile \
    --platform "${PLATFORM}" \
    -t "${IMAGE_NAME}:${TAG}" \
    --load \
    .

if [ $? -eq 0 ]; then
    echo ""
    echo "✓ 构建成功!"
    echo ""
    echo "镜像信息:"
    docker images | grep "${IMAGE_NAME}"
else
    echo ""
    echo "✗ 构建失败"
    exit 1
fi
