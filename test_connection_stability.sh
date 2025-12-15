#!/bin/bash

# Test script for connection stability
# Runs multiple examples sequentially to verify no connection issues

set -e

echo "=== Testing Connection Stability ==="
echo "This script will run multiple examples sequentially"
echo "Press Ctrl+C to stop"
echo ""

# Array of example directories
examples=(
    "control"
    "symbols"
    "timedate"
    "arrays"
    "structs"
)

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

success_count=0
failure_count=0

# Run each example
for example in "${examples[@]}"; do
    echo -e "${YELLOW}Running example: $example${NC}"
    
    if go run "examples/$example/main.go" 2>&1; then
        echo -e "${GREEN}✓ $example completed successfully${NC}"
        ((success_count++))
    else
        echo -e "${RED}✗ $example failed${NC}"
        ((failure_count++))
    fi
    
    # Small delay between runs
    echo "Waiting 2 seconds before next example..."
    sleep 2
    echo ""
done

echo ""
echo "=== Test Summary ==="
echo -e "Successful: ${GREEN}$success_count${NC}"
echo -e "Failed: ${RED}$failure_count${NC}"

if [ $failure_count -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed${NC}"
    exit 1
fi
