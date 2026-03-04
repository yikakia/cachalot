#!/bin/bash

set -e

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "🚀 Starting full test suite..."
echo ""

# 主模块测试
echo -e "${YELLOW}[1/4] Testing root modules...${NC}"
if go test -v -race ./...; then
    echo -e "${GREEN}✓ Root modules passed${NC}"
else
    echo -e "${RED}✗ Root modules failed${NC}"
    exit 1
fi
echo ""

# Redis 存储测试
echo -e "${YELLOW}[2/4] Testing stores/redis...${NC}"
if (cd stores/redis && go test -v -race .); then
    echo -e "${GREEN}✓ Redis store passed${NC}"
else
    echo -e "${RED}✗ Redis store failed${NC}"
    exit 1
fi
echo ""

# Ristretto 存储测试
echo -e "${YELLOW}[3/4] Testing stores/ristretto...${NC}"
if (cd stores/ristretto && go test -v -race .); then
    echo -e "${GREEN}✓ Ristretto store passed${NC}"
else
    echo -e "${RED}✗ Ristretto store failed${NC}"
    exit 1
fi
echo ""

# FreeCache 存储测试
echo -e "${YELLOW}[4/4] Testing stores/freecache...${NC}"
if (cd stores/freecache && go test -v -race .); then
    echo -e "${GREEN}✓ FreeCache store passed${NC}"
else
    echo -e "${RED}✗ FreeCache store failed${NC}"
    exit 1
fi
echo ""

echo -e "${GREEN}✅ All tests passed!${NC}"
