#!/usr/bin/env bash
# 构建评测用 Docker 镜像（双架构 linux/amd64 与 linux/arm64）。
# 用法：bash build_benzhi_docker.sh <镜像名> <平台>
#   平台可选：amd64 / arm64 / all（默认 all）
set -euo pipefail

IMAGE_NAME="${1:-my-project}"
PLATFORM="${2:-all}"

case "$PLATFORM" in
  amd64) PLATFORMS="linux/amd64" ;;
  arm64) PLATFORMS="linux/arm64" ;;
  all)   PLATFORMS="linux/amd64,linux/arm64" ;;
  *) echo "未知平台: $PLATFORM (amd64|arm64|all)" >&2; exit 1 ;;
esac

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "构建镜像 $IMAGE_NAME 平台 $PLATFORMS"
docker buildx build \
  --file benzhi.Dockerfile \
  --platform "$PLATFORMS" \
  --tag "$IMAGE_NAME:latest" \
  --load \
  .

echo "完成：$IMAGE_NAME"
