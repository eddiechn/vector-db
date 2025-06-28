#!/bin/bash

set -e
MILVUS_URL="http://localhost:9091"
COLLECTION="test_vectors"
VECTOR_DIM=128
CONCURRENT_REQUESTS=50
TOTAL_REQUESTS=1000
SEARCH_K=5

curl -s -X POST "$MILVUS_URL/v1/vector/collections" \
  -H "Content-Type: application/json" \
  -d "{
    \"collection_name\": \"$COLLECTION\",
    \"dimension\": $VECTOR_DIM,
    \"index_file_size\": 1024,
    \"metric_type\": \"L2\"
  }" > /dev/null

gen_vector() {
  python3 -c "import random; print([round(random.uniform(-1,1),3) for _ in range($VECTOR_DIM)])"
}

insert_vector() {
  local id=$1
  local vector_data=$(gen_vector)
  response=$(curl -s -w "%{http_code}" -o /dev/null -X POST "$MILVUS_URL/v1/vector/insert" \
    -H "Content-Type: application/json" \
    -d "{
      \"collection_name\": \"$COLLECTION\",
      \"records\": [ $vector_data ],
      \"ids\": [ $id ]
    }")
  if [ "$response" = "200" ]; then
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
    curl -s -X POST "$MILVUS_URL/v1/vector/search" \
      -H "Content-Type: application/json" \
      -d "{
        \"collection_name\": \"$COLLECTION\",
        \"query_records\": [ $query_vector ],
        \"topk\": $SEARCH_K
      }" > /dev/null &
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