#!/bin/bash

set -e

TEST_FILE="data.txt"
PATTERN="banana"
FLAGS="-n -i"
PEERS="127.0.0.1:9000,127.0.0.1:9001,127.0.0.1:9002"
BINARY="./mygrep"

if ! command -v grep >/dev/null; then
    echo "ERROR: grep not found in PATH. Please install grep."
    exit 1
fi

if [[ ! -x "$BINARY" ]]; then
    echo "ERROR: $BINARY not found or not executable. Run 'go build -o mygrep ./cmd/grep' first."
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
echo "[2] Running distributed mygrep (3 nodes)..."

# Запускаем узлы последовательно с задержками
echo "Starting node 0 (coordinator)..."
$BINARY -id=0 -peers="$PEERS" -file="$TEST_FILE" -pattern="$PATTERN" $FLAGS > node0.log 2>&1 &
PID0=$!
sleep 2  # Даём координатору время поднять сервер

echo "Starting node 1..."
$BINARY -id=1 -peers="$PEERS" -file="$TEST_FILE" -pattern="$PATTERN" $FLAGS > node1.log 2>&1 &
PID1=$!
sleep 1

echo "Starting node 2..."
$BINARY -id=2 -peers="$PEERS" -file="$TEST_FILE" -pattern="$PATTERN" $FLAGS > node2.log 2>&1 &
PID2=$!

echo "Waiting for all nodes to complete..."
# Ждём завершения всех трёх узлов
wait $PID0 $PID1 $PID2

END=$(date +%s%N)
MYGREP_TIME=$(echo "scale=3; ($END - $START) / 1000000000" | bc)
echo "Distributed mygrep finished in $MYGREP_TIME seconds"

# Извлекаем вывод координатора (узел 0) из лога
# Вывод находится после строки "stop signal received, collecting all lines and exiting"
# Берём строки, которые начинаются с цифры (номера строк)
grep -E '^[0-9]+:' node0.log | sort > mygrep_output.txt

echo
echo "[3] Comparing outputs..."
# Сортируем оба вывода перед сравнением (порядок строк может отличаться)
grep $FLAGS "$PATTERN" "$TEST_FILE" | sort > original_output_sorted.txt
sort mygrep_output.txt -o mygrep_output_sorted.txt

if diff -q original_output_sorted.txt mygrep_output_sorted.txt >/dev/null; then
    echo "SUCCESS: Outputs are identical."
    echo
    echo "=== Results (found lines) ==="
    cat mygrep_output_sorted.txt
else
    echo "FAILURE: Outputs differ."
    echo "=== Original grep output ==="
    cat original_output_sorted.txt
    echo "=== Mygrep output ==="
    cat mygrep_output_sorted.txt
fi

echo
echo "=== Summary ==="
echo "Original grep:    $ORIG_TIME sec"
echo "Distributed mygrep: $MYGREP_TIME sec"

# Очистка
rm -f node0.log node1.log node2.log original_output.txt original_output_sorted.txt mygrep_output.txt mygrep_output_sorted.txt