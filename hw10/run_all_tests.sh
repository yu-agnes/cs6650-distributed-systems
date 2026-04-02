#!/bin/bash

# ─────────────────────────────────────────────────────────
# Run all load test experiments:
#   Leader-Follower: 3 W/R configs × 4 read/write ratios = 12 runs
#   Leaderless:      1 config     × 4 read/write ratios =  4 runs
#   Total: 16 runs
#
# Usage: chmod +x run_all_tests.sh && ./run_all_tests.sh
# ─────────────────────────────────────────────────────────

DURATION=30       # seconds per run
CONCURRENCY=10    # concurrent workers
KEYS=20           # number of distinct keys

WRITE_PCTS=(1 10 50 90)

mkdir -p results

echo "============================================"
echo "  Leader-Follower Load Tests"
echo "============================================"

# --- W=5 R=1 ---
echo ""
echo ">>> Starting Leader-Follower W=5 R=1 ..."
cd "$(dirname "$0")"
docker compose -f docker-compose-leader.yml down 2>/dev/null
W=5 R=1 docker compose -f docker-compose-leader.yml up --build -d
sleep 5  # wait for containers to start

for WP in "${WRITE_PCTS[@]}"; do
    echo "  Running write=${WP}% read=$((100-WP))% ..."
    go run ./loadtest/ \
        -mode=leader -write-pct=${WP} -duration=${DURATION} \
        -concurrency=${CONCURRENCY} -keys=${KEYS} \
        -out="results/leader_W5R1_w${WP}"
done

docker compose -f docker-compose-leader.yml down

# --- W=1 R=5 ---
echo ""
echo ">>> Starting Leader-Follower W=1 R=5 ..."
W=1 R=5 docker compose -f docker-compose-leader.yml up --build -d
sleep 5

for WP in "${WRITE_PCTS[@]}"; do
    echo "  Running write=${WP}% read=$((100-WP))% ..."
    go run ./loadtest/ \
        -mode=leader -write-pct=${WP} -duration=${DURATION} \
        -concurrency=${CONCURRENCY} -keys=${KEYS} \
        -out="results/leader_W1R5_w${WP}"
done

docker compose -f docker-compose-leader.yml down

# --- W=3 R=3 ---
echo ""
echo ">>> Starting Leader-Follower W=3 R=3 ..."
W=3 R=3 docker compose -f docker-compose-leader.yml up --build -d
sleep 5

for WP in "${WRITE_PCTS[@]}"; do
    echo "  Running write=${WP}% read=$((100-WP))% ..."
    go run ./loadtest/ \
        -mode=leader -write-pct=${WP} -duration=${DURATION} \
        -concurrency=${CONCURRENCY} -keys=${KEYS} \
        -out="results/leader_W3R3_w${WP}"
done

docker compose -f docker-compose-leader.yml down

echo ""
echo "============================================"
echo "  Leaderless Load Tests"
echo "============================================"

echo ""
echo ">>> Starting Leaderless W=5 R=1 ..."
docker compose -f docker-compose-leaderless.yml up --build -d
sleep 5

for WP in "${WRITE_PCTS[@]}"; do
    echo "  Running write=${WP}% read=$((100-WP))% ..."
    go run ./loadtest/ \
        -mode=leaderless -write-pct=${WP} -duration=${DURATION} \
        -concurrency=${CONCURRENCY} -keys=${KEYS} \
        -out="results/leaderless_w${WP}"
done

docker compose -f docker-compose-leaderless.yml down

echo ""
echo "============================================"
echo "  All tests completed!"
echo "  Results in: results/"
echo "============================================"
ls -la results/
