#!/bin/bash

# OpenAI API 测试脚本
# 测试 https://code.newcli.com/codex/v1 端点

API_URL="https://code.newcli.com/codex/v1"
MODEL="gpt-5.1-codex-max"

echo "=========================================="
echo "OpenAI API 测试脚本"
echo "=========================================="
echo "API URL: $API_URL"
echo "Model: $MODEL"
echo ""

# 需要 API Key
if [ -z "$OPENAI_API_KEY" ]; then
    echo "请设置 OPENAI_API_KEY 环境变量"
    echo "用法: OPENAI_API_KEY=sk-xxx ./test_openai_api.sh"
    exit 1
fi

echo "API Key: ${OPENAI_API_KEY:0:8}...${OPENAI_API_KEY: -4}"
echo ""

# 测试 1: 列出可用模型
echo "=========================================="
echo "测试 1: 列出可用模型 (GET /models)"
echo "=========================================="
curl -s -X GET "${API_URL}/models" \
    -H "Authorization: Bearer $OPENAI_API_KEY" \
    -H "Content-Type: application/json" | head -c 2000
echo ""
echo ""

# 测试 2: Chat Completions
echo "=========================================="
echo "测试 2: Chat Completions (POST /chat/completions)"
echo "=========================================="
echo "请求模型: $MODEL"
echo ""

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/chat/completions" \
    -H "Authorization: Bearer $OPENAI_API_KEY" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "'"$MODEL"'",
        "messages": [
            {"role": "user", "content": "Say hello in one word"}
        ],
        "max_tokens": 50
    }')

HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

echo "HTTP Status: $HTTP_CODE"
echo "Response:"
echo "$BODY" | head -c 1000
echo ""
echo ""

# 测试 3: 测试其他可能的模型名称
echo "=========================================="
echo "测试 3: 测试其他模型名称"
echo "=========================================="

for model in "gpt-4" "gpt-4o" "gpt-4-turbo" "gpt-3.5-turbo"; do
    echo -n "测试模型 $model: "
    RESP=$(curl -s -w "%{http_code}" -X POST "${API_URL}/chat/completions" \
        -H "Authorization: Bearer $OPENAI_API_KEY" \
        -H "Content-Type: application/json" \
        -d '{
            "model": "'"$model"'",
            "messages": [{"role": "user", "content": "hi"}],
            "max_tokens": 5
        }')
    HTTP=$(echo "$RESP" | tail -c 3)
    echo "HTTP $HTTP"
done

echo ""
echo "=========================================="
echo "测试完成"
echo "=========================================="
