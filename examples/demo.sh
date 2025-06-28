#!/bin/bash

# VectorDB Demo Script
# This script demonstrates the core functionality of VectorDB

set -e

BASE_URL="http://localhost:8080"

echo "🚀 VectorDB Demo Script"
echo "======================="

# Check if server is running
echo "📡 Checking server status..."
if curl -s "${BASE_URL}/health" > /dev/null; then
    echo "✅ Server is running"
else
    echo "❌ Server is not running. Please start it with: ./vectordb"
    exit 1
fi

# Get initial stats
echo ""
echo "📊 Initial database stats:"
curl -s "${BASE_URL}/stats" | python3 -m json.tool

# Insert some example vectors
echo ""
echo "📝 Inserting example vectors..."

# Vector 1: Text-like embedding
curl -s -X POST "${BASE_URL}/vectors" \
    -H "Content-Type: application/json" \
    -d '{
        "vector": {
            "id": "document_1",
            "data": [0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8]
        },
        "metadata": {
            "type": "document",
            "category": "technology",
            "title": "Introduction to Machine Learning"
        }
    }' | python3 -m json.tool

# Vector 2: Similar document
curl -s -X POST "${BASE_URL}/vectors" \
    -H "Content-Type: application/json" \
    -d '{
        "vector": {
            "id": "document_2", 
            "data": [0.15, 0.25, 0.35, 0.45, 0.55, 0.65, 0.75, 0.85]
        },
        "metadata": {
            "type": "document",
            "category": "technology", 
            "title": "Deep Learning Fundamentals"
        }
    }' | python3 -m json.tool

# Vector 3: Different category
curl -s -X POST "${BASE_URL}/vectors" \
    -H "Content-Type: application/json" \
    -d '{
        "vector": {
            "id": "document_3",
            "data": [0.9, 0.1, 0.8, 0.2, 0.7, 0.3, 0.6, 0.4]
        },
        "metadata": {
            "type": "document",
            "category": "sports",
            "title": "Olympic Swimming Records"
        }
    }' | python3 -m json.tool

echo ""
echo "✅ Inserted 3 vectors"

# List vectors
echo ""
echo "📋 Listing vectors:"
curl -s "${BASE_URL}/vectors?limit=10" | python3 -m json.tool

# Perform similarity search
echo ""
echo "🔍 Searching for vectors similar to document_1:"
curl -s -X POST "${BASE_URL}/search" \
    -H "Content-Type: application/json" \
    -d '{
        "vector": [0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8],
        "k": 3,
        "distance_metric": 0
    }' | python3 -m json.tool

# Get a specific vector
echo ""
echo "📖 Getting specific vector (document_1):"
curl -s "${BASE_URL}/vectors/document_1" | python3 -m json.tool

# Get updated stats
echo ""
echo "📊 Updated database stats:"
curl -s "${BASE_URL}/stats" | python3 -m json.tool

# Test different distance metrics
echo ""
echo "🔬 Testing different distance metrics:"

echo "Cosine Similarity (metric 0):"
curl -s -X POST "${BASE_URL}/search" \
    -H "Content-Type: application/json" \
    -d '{
        "vector": [0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8],
        "k": 2,
        "distance_metric": 0
    }' | python3 -c "import sys, json; data=json.load(sys.stdin); print(f'Results: {len(data[\"results\"])} vectors found')"

echo "Euclidean Distance (metric 1):"
curl -s -X POST "${BASE_URL}/search" \
    -H "Content-Type: application/json" \
    -d '{
        "vector": [0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8],
        "k": 2,
        "distance_metric": 1
    }' | python3 -c "import sys, json; data=json.load(sys.stdin); print(f'Results: {len(data[\"results\"])} vectors found')"

# Clean up - delete a vector
echo ""
echo "🗑️  Deleting document_3:"
curl -s -X DELETE "${BASE_URL}/vectors/document_3" | python3 -m json.tool

# Final stats
echo ""
echo "📊 Final database stats:"
curl -s "${BASE_URL}/stats" | python3 -m json.tool

echo ""
echo "🎉 Demo completed successfully!"
echo "💡 Visit http://localhost:8080/ for the web dashboard"
echo "📚 Visit http://localhost:8080/api-docs for full API documentation"
