#!/bin/bash

set -e
SERVER_URL="http://localhost:8080"
CONCURRENT_REQUESTS=50
TOTAL_REQUESTS=1000
VECTOR_DIM=128
SEARCH_K=5

gen_vector() {
  python3 -c "import random; print([round(random.uniform(-1,1),3) for _ in range($VECTOR_DIM)])"
}

insert_vector() {
    local id=$1
    local vector_data=$(gen_vector)
    response=$(curl -s -w "%{http_code}" -o /dev/null -X POST "$SERVER_URL/vectors" \
      -H "Content-Type: application/json" \
      -d "{\"vector\":{\"id\":\"concurrent_$id\",\"data\":$vector_data}}")
    if [ "$response" = "201" ] || [ "$response" = "200" ]; then
        echo -n "✓"
    else
        echo -n "✗"
    fi
}

insert_start=$(date +%s)
for ((batch=0; batch<$TOTAL_REQUESTS; batch+=$CONCURRENT_REQUESTS)); do
    pids=()
    for ((i=0; i<$CONCURRENT_REQUESTS && (batch+i)<$TOTAL_REQUESTS; i++)); do
        insert_vector $((batch + i)) &
        pids+=($!)
    done
    for pid in "${pids[@]}"; do
        wait $pid
    done
done
insert_end=$(date +%s)
insert_duration=$((insert_end - insert_start))
echo "✅ Insertion completed in $insert_duration seconds"
echo "📈 Rate: $(echo "scale=2; $TOTAL_REQUESTS / $insert_duration" | bc -l) vectors/sec"

search_count=100
query_vector=$(gen_vector)
search_start=$(date +%s)
for ((batch=0; batch<$search_count; batch+=$CONCURRENT_REQUESTS)); do
    pids=()
    for ((i=0; i<$CONCURRENT_REQUESTS && (batch+i)<$search_count; i++)); do
        curl -s -X POST "$SERVER_URL/search" \
            -H "Content-Type: application/json" \
            -d "{\"vector\":$query_vector,\"k\":$SEARCH_K}" > /dev/null &
        pids+=($!)
    done
    for pid in "${pids[@]}"; do
        wait $pid
    done
done
search_end=$(date +%s)
search_duration=$((search_end - search_start))
echo "✅ Search testing completed in $search_duration seconds"
if [ "$search_duration" -eq 0 ]; then
  echo "📈 Rate: N/A (duration < 1s)"
else
  echo "📈 Rate: $(echo "scale=2; $search_count / $search_duration" | bc -l) searches/second"
fi