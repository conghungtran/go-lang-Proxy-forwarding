#!/bin/bash

# Script to start the proxy-forward server
echo "Starting Proxy Forward Server..."
echo "================================="

# Check if binary exists
if [ ! -f "./bin/proxy-forward" ]; then
    echo "Error: Binary file './bin/proxy-forward' not found!"
    echo "Please run: go build -o bin/proxy-forward main.go"
    exit 1
fi

# Check if config files exist
if [ ! -f "./list_proxy.txt" ]; then
    echo "Error: Config file './list_proxy.txt' not found!"
    exit 1
fi

# Display proxy credentials info
if [ -f "./proxy_credentials.txt" ]; then
    echo "Authentication credentials loaded from ./proxy_credentials.txt"
    echo "Check this file for username/password for each proxy port."
    echo ""
fi

# Run the proxy server
echo "Executing: ./bin/proxy-forward"
echo ""

./bin/proxy-forward
