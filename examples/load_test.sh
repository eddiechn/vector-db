#!/bin/bash

# VectorDB Load Testing Script
# This script tests the performance and concurrent capabilities of VectorDB

set -e

BASE_URL="http://localhost:8080"
TOTAL_VECTORS=1000
CONCURRENT_REQUESTS=10
DIMENSIONS=128

echo "⚡ VectorDB Load Testing"
echo "======================="
echo "Total vectors to insert: $TOTAL_VECTORS"
echo "Concurrent requests: $CONCURRENT_REQUESTS"
echo "Vector dimensions: $DIMENSIONS"
echo ""

# Check if server is running
if ! curl -s "${BASE_URL}/health" > /dev/null; then
    echo "❌ Server is not running. Please start it with: ./vectordb"
    exit 1
fi

# Function to generate random vector data
generate_vector() {
    local id=$1
    local data=""
    for i in $(seq 1 $DIMENSIONS); do
        # Generate random float between -1 and 1
        val=$(echo "scale=3; ($RANDOM / 16384) - 1" | bc -l)
        if [ $i -eq 1 ]; then
            data="$val"
        else
            data="$data, $val"
        fi
    done
    echo "[$data]"
}

# Function to insert a single vector
insert_vector() {
    local id=$1
    local vector_data=$(generate_vector $id)
    
    curl -s -X POST "${BASE_URL}/vectors" \
        -H "Content-Type: application/json" \
        -d "{
            \"vector\": {
                \"id\": \"load_test_$id\",
                \"data\": $vector_data
            },
            \"metadata\": {
                \"type\": \"load_test\",
                \"batch\": $((id / 100)),
                \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"
            }
        }" > /dev/null
    
    if [ $? -eq 0 ]; then
        echo -n "✓"
    else
        echo -n "✗"
    fi
}

# Get initial stats
echo "📊 Initial stats:"
initial_stats=$(curl -s "${BASE_URL}/stats")
initial_count=$(echo $initial_stats | python3 -c "import sys, json; print(json.load(sys.stdin)['vector_count'])")
echo "Vectors in database: $initial_count"
echo ""

# Load testing - Insert vectors
echo "🚀 Starting load test (inserting $TOTAL_VECTORS vectors)..."
start_time=$(date +%s)

# Insert vectors in batches
for ((batch=0; batch<$TOTAL_VECTORS; batch+=$CONCURRENT_REQUESTS)); do
    echo -n "Batch $((batch/CONCURRENT_REQUESTS + 1)): "
    
    # Start concurrent requests
    pids=()
    for ((i=0; i<$CONCURRENT_REQUESTS && (batch+i)<$TOTAL_VECTORS; i++)); do
        insert_vector $((batch + i)) &
        pids+=($!)
    done
    
    # Wait for all requests to complete
    for pid in "${pids[@]}"; do
        wait $pid
    done
    
    echo " ($((batch + CONCURRENT_REQUESTS)) / $TOTAL_VECTORS)"
done

end_time=$(date +%s)
insert_duration=$((end_time - start_time))
echo ""
echo "✅ Insertion completed in $insert_duration seconds"
echo "📈 Rate: $(echo "scale=2; $TOTAL_VECTORS / $insert_duration" | bc -l) vectors/second"

# Verify insertions
echo ""
echo "🔍 Verifying insertions..."
final_stats=$(curl -s "${BASE_URL}/stats")
final_count=$(echo $final_stats | python3 -c "import sys, json; print(json.load(sys.stdin)['vector_count'])")
inserted_count=$((final_count - initial_count))
echo "Vectors inserted: $inserted_count / $TOTAL_VECTORS"

if [ $inserted_count -eq $TOTAL_VECTORS ]; then
    echo "✅ All vectors inserted successfully"
else
    echo "⚠️  Some insertions may have failed"
fi

# Search performance test
echo ""
echo "🔍 Testing search performance..."
search_start=$(date +%s)

# Generate a random query vector
query_vector=$(generate_vector "query")

# Perform multiple searches
search_count=100
echo "Performing $search_count search queries..."

for ((i=1; i<=search_count; i++)); do
    curl -s -X POST "${BASE_URL}/search" \
        -H "Content-Type: application/json" \
        -d "{
            \"vector\": $query_vector,
            \"k\": 10,
            \"distance_metric\": 0
        }" > /dev/null
    
    if [ $((i % 10)) -eq 0 ]; then
        echo -n "."
    fi
done

search_end=$(date +%s)
search_duration=$((search_end - search_start))
echo ""
echo "✅ Search testing completed in $search_duration seconds"
echo "📈 Rate: $(echo "scale=2; $search_count / $search_duration" | bc -l) searches/second"

# Final statistics
echo ""
echo "📊 Final performance statistics:"
curl -s "${BASE_URL}/stats" | python3 -c "
import sys, json
data = json.load(sys.stdin)
print(f'Total vectors: {data[\"vector_count\"]}')
print(f'Total search requests: {data[\"search_requests\"]}')
print(f'Total insert requests: {data[\"insert_requests\"]}')
print(f'Average latency: {data[\"average_latency\"]}')
print(f'Memory usage: {data[\"memory_usage_bytes\"] / 1024 / 1024:.2f} MB')
"

# Cleanup option
echo ""
echo "🧹 Cleanup (remove test vectors)? [y/N]"
read -r response
if [[ "$response" =~ ^[Yy]$ ]]; then
    echo "Cleaning up test vectors..."
    
    # Delete test vectors
    cleanup_count=0
    for ((i=0; i<$TOTAL_VECTORS; i++)); do
        curl -s -X DELETE "${BASE_URL}/vectors/load_test_$i" > /dev/null
        if [ $? -eq 0 ]; then
            cleanup_count=$((cleanup_count + 1))
        fi
        
        if [ $((i % 100)) -eq 0 ]; then
            echo "Cleaned up $i vectors..."
        fi
    done
    
    echo "✅ Cleaned up $cleanup_count test vectors"
fi

echo ""
echo "🎉 Load testing completed!"
echo "💡 Check the dashboard at http://localhost:8080/ for real-time stats"
