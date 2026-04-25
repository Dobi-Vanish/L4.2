#!/bin/bash

PEERS="127.0.0.1:9000,127.0.0.1:9001,127.0.0.1:9002"
FILE="data.txt"
PATTERN="banana"
FLAGS="-n -i"

if [ ! -x "./mygrep" ]; then
    echo "Error: ./mygrep not found or not executable. Run 'go build -o mygrep ./cmd/grep' first."
    exit 1
fi

start_node() {
    local id=$1
    local title="Node $id"
    local cmd="./mygrep -id=$id -peers=\"$PEERS\" -file=\"$FILE\" -pattern=\"$PATTERN\" $FLAGS"
    if command -v gnome-terminal >/dev/null; then
        gnome-terminal --title="$title" -- bash -c "$cmd; echo 'Press Enter to exit...'; read"
    elif command -v xterm >/dev/null; then
        xterm -title "$title" -e bash -c "$cmd; echo 'Press Enter to exit...'; read"
    elif command -v konsole >/dev/null; then
        konsole --new-tab --title "$title" -e bash -c "$cmd; echo 'Press Enter to exit...'; read"
    else
        echo "No known terminal emulator found. Running node $id in background (log in node$id.log)"
        $cmd > node$id.log 2>&1 &
    fi
}

echo "Starting cluster with 3 nodes..."
start_node 0
sleep 1
start_node 1
sleep 1
start_node 2
echo "All nodes started. Coordinator (node 0) will print results in its terminal window."