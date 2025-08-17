#!/bin/bash

# Test script for OpenAI embedding API routes
# Make sure to set OPENAI_API_KEY environment variable before running

BASE_URL="http://localhost:8080"

echo "Testing OpenAI Embedding API Routes"
echo "=================================="

# Test 1: Embed some text
echo "1. Embedding text samples..."
curl -X POST "$BASE_URL/embed" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "doc1",
    "text": "Machine learning is a subset of artificial intelligence that focuses on algorithms.",
    "metadata": {"category": "technology", "type": "definition"}
  }' | jq

echo -e "\n"

curl -X POST "$BASE_URL/embed" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "doc2", 
    "text": "The quick brown fox jumps over the lazy dog.",
    "metadata": {"category": "example", "type": "sentence"}
  }' | jq

echo -e "\n"

curl -X POST "$BASE_URL/embed" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "doc3",
    "text": "Artificial intelligence and machine learning algorithms are transforming technology.",
    "metadata": {"category": "technology", "type": "description"}
  }' | jq

echo -e "\n"

# Test 2: Search using text
echo "2. Searching with text query..."
curl -X POST "$BASE_URL/search/text" \
  -H "Content-Type: application/json" \
  -d '{
    "text": "What is machine learning?",
    "k": 2
  }' | jq

echo -e "\n"

# Test 3: Check database stats
echo "3. Checking database statistics..."
curl -X GET "$BASE_URL/stats" | jq

echo -e "\n"

echo "Test completed!"
