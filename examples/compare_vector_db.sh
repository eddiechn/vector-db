#!/bin/bash

set -e

GO_SCRIPT="./examples/concurrency_test_go.sh"
MILVUS_SCRIPT="./examples/concurrency_test_milvus.sh"

# Concurrency levels to test
CONCURRENCY_LEVELS=(1 3 5 8 10 25 50)

# Total requests for each test
TOTAL_REQUESTS=1000

# Output CSV header
echo "concurrency,db,insert_rate,search_rate"

for CONC in "${CONCURRENCY_LEVELS[@]}"; do
  # Update concurrency in both scripts (and total requests)
  sed -i '' "s/^CONCURRENT_REQUESTS=.*/CONCURRENT_REQUESTS=$CONC/" "$GO_SCRIPT"
  sed -i '' "s/^CONCURRENT_REQUESTS=.*/CONCURRENT_REQUESTS=$CONC/" "$MILVUS_SCRIPT"
  sed -i '' "s/^TOTAL_REQUESTS=.*/TOTAL_REQUESTS=$TOTAL_REQUESTS/" "$GO_SCRIPT"
  sed -i '' "s/^TOTAL_REQUESTS=.*/TOTAL_REQUESTS=$TOTAL_REQUESTS/" "$MILVUS_SCRIPT"

  # Run Go Vector DB test
  GO_OUTPUT=$(bash "$GO_SCRIPT")
  GO_INSERT_RATE=$(echo "$GO_OUTPUT" | grep -m1 "vectors/sec" | awk '{print $(NF-1)}')
  GO_SEARCH_RATE=$(echo "$GO_OUTPUT" | grep -m1 "searches/second" | awk '{print $(NF-1)}')
  echo "$CONC,go,$GO_INSERT_RATE,$GO_SEARCH_RATE"

  # Run Milvus test
  MILVUS_OUTPUT=$(bash "$MILVUS_SCRIPT")
  MILVUS_INSERT_RATE=$(echo "$MILVUS_OUTPUT" | grep -m1 "vectors/sec" | awk '{print $(NF-1)}')
  MILVUS_SEARCH_RATE=$(echo "$MILVUS_OUTPUT" | grep -m1 "searches/second" | awk '{print $(NF-1)}')
  echo "$CONC,milvus,$MILVUS_INSERT_RATE,$MILVUS_SEARCH_RATE"
done