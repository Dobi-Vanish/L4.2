#!/bin/bash

set -e

TEST_FILE="data.txt"
PATTERN="banana"
FLAGS="-n -i"
PEERS="127.0.0.1:9000,127.0.0.1:9001,127.0.0.1:9002"
BINARY="./grep"

if ! command -v grep >/dev/null; then
    echo "ERROR: grep not found in PATH. Please install grep."
    exit 1
fi

if [ ! -x "$BINARY" ]; then
    echo "ERROR: $BINARY not found or not executable. Run 'go build -o grep ./cmd/grep' first."
    exit 1
fi

echo "=== Comparing mygrep (distributed) with original grep ==="
echo
echo "[1] Running original grep..."
START=$(date +%s%N)
grep $FLAGS "$PATTERN" "$TEST_FILE" > original_output.txt 2>/dev/null
END=$(date +%s%N)
ORIG_TIME=$(echo "scale=3; ($END - $START) / 1000000000" | bc)
echo "Original grep finished in $ORIG_TIME seconds"

echo
echo "[2] Running distributed mygrep (3 nodes) in background..."
START=$(date +%s%N)

$BINARY -id=0 -peers="$PEERS" -file="$TEST_FILE" -pattern="$PATTERN" $FLAGS > node0.log 2>&1 &
$BINARY -id=1 -peers="$PEERS" -file="$TEST_FILE" -pattern="$PATTERN" $FLAGS > node1.log 2>&1 &
$BINARY -id=2 -peers="$PEERS" -file="$TEST_FILE" -pattern="$PATTERN" $FLAGS > node2.log 2>&1 &

while kill -0 $! 2>/dev/null; do
    sleep 0.5
done
wait

END=$(date +%s%N)
MYGREP_TIME=$(echo "scale=3; ($END - $START) / 1000000000" | bc)
echo "Distributed mygrep finished in $MYGREP_TIME seconds"

grep -E '^[0-9a-zA-Z]' node0.log > mygrep_output.txt

echo
echo "[3] Comparing outputs..."
if diff -q original_output.txt mygrep_output.txt >/dev/null; then
    echo "SUCCESS: Outputs are identical."
else
    echo "FAILURE: Outputs differ."
    echo "=== Original grep output ==="
    cat original_output.txt
    echo "=== Mygrep output ==="
    cat mygrep_output.txt
fi

echo
echo "=== Summary ==="
echo "Original grep:    $ORIG_TIME sec"
echo "Distributed mygrep: $MYGREP_TIME sec"

rm -f node0.log node1.log node2.log original_output.txt mygrep_output.txt