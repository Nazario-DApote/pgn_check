#!/bin/bash

# PGN Check - Performance Benchmark Script
# Measures validation and correction performance on test files

# Colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# Parse arguments
TEST_FILE=""
TEST_ALL=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -a|--all)
            TEST_ALL=true
            shift
            ;;
        *)
            TEST_FILE="$1"
            shift
            ;;
    esac
done

echo -e "${CYAN}=========================================${NC}"
echo -e "${CYAN}PGN Check - Performance Benchmark${NC}"
echo -e "${CYAN}=========================================${NC}"
echo ""

# Check if executable exists
if [ ! -f "./pgn_check.exe" ] && [ ! -f "./pgn_check" ]; then
    echo -e "${YELLOW}Error: pgn_check executable not found. Building...${NC}"
    go build -o pgn_check
    if [ $? -ne 0 ]; then
        echo -e "${RED}Build failed!${NC}"
        exit 1
    fi
    echo -e "${GREEN}Build completed successfully!${NC}"
    echo ""
fi

# Determine executable name
EXE="./pgn_check"
if [ -f "./pgn_check.exe" ]; then
    EXE="./pgn_check.exe"
fi

# Function to test file performance
test_file_performance() {
    local file_path="$1"
    
    if [ ! -f "$file_path" ]; then
        echo -e "${RED}  File not found: $file_path${NC}"
        return 1
    fi
    
    local file_size_bytes=$(stat -f%z "$file_path" 2>/dev/null || stat -c%s "$file_path" 2>/dev/null)
    local file_size_mb=$(echo "scale=2; $file_size_bytes / 1048576" | bc)
    local file_name=$(basename "$file_path")
    
    echo -e "${WHITE}Testing: $file_name ($file_size_mb MB)${NC}"
    
    # Test validation
    echo -n "  Running validation..."
    local start_time=$(date +%s.%N)
    $EXE "$file_path" > /dev/null 2>&1
    local end_time=$(date +%s.%N)
    local validation_time=$(echo "$end_time - $start_time" | bc)
    local validation_time_formatted=$(printf "%.2f" $validation_time)
    local validation_speed=$(echo "scale=2; $file_size_mb / $validation_time" | bc)
    echo -e " ${GREEN}$validation_time_formatted sec ($validation_speed MB/s)${NC}"
    
    # Test correction
    local temp_output="temp_benchmark_output.pgn"
    echo -n "  Running correction..."
    start_time=$(date +%s.%N)
    $EXE -o "$temp_output" "$file_path" > /dev/null 2>&1
    end_time=$(date +%s.%N)
    local correction_time=$(echo "$end_time - $start_time" | bc)
    local correction_time_formatted=$(printf "%.2f" $correction_time)
    local correction_speed=$(echo "scale=2; $file_size_mb / $correction_time" | bc)
    echo -e " ${GREEN}$correction_time_formatted sec ($correction_speed MB/s)${NC}"
    
    # Cleanup
    [ -f "$temp_output" ] && rm -f "$temp_output"
    
    echo ""
    
    # Store results for summary
    RESULTS+=("$file_name|$file_size_mb|$validation_time_formatted|$validation_speed|$correction_time_formatted|$correction_speed")
}

# Array to store results
RESULTS=()

if [ -n "$TEST_FILE" ]; then
    # Test single file
    test_file_performance "$TEST_FILE"
elif [ "$TEST_ALL" = true ]; then
    # Test all PGN files in test_files directory
    for file in test_files/*.pgn; do
        [ -f "$file" ] && test_file_performance "$file"
    done
else
    # Test default large files
    echo -e "${YELLOW}Testing large files (use --all to test all files)...${NC}"
    echo ""
    
    for file in test_files/twic*.pgn; do
        [ -f "$file" ] && test_file_performance "$file"
    done
fi

# Display summary
if [ ${#RESULTS[@]} -gt 0 ]; then
    echo -e "${CYAN}=========================================${NC}"
    echo -e "${CYAN}Summary${NC}"
    echo -e "${CYAN}=========================================${NC}"
    echo ""
    
    # Calculate averages
    total_size=0
    total_val_speed=0
    total_cor_speed=0
    count=0
    
    for result in "${RESULTS[@]}"; do
        IFS='|' read -r name size val_time val_speed cor_time cor_speed <<< "$result"
        total_size=$(echo "$total_size + $size" | bc)
        total_val_speed=$(echo "$total_val_speed + $val_speed" | bc)
        total_cor_speed=$(echo "$total_cor_speed + $cor_speed" | bc)
        count=$((count + 1))
    done
    
    avg_val_speed=$(echo "scale=2; $total_val_speed / $count" | bc)
    avg_cor_speed=$(echo "scale=2; $total_cor_speed / $count" | bc)
    
    echo -e "${WHITE}Total data processed: $(printf "%.2f" $total_size) MB${NC}"
    echo -e "${GREEN}Average validation speed: $avg_val_speed MB/s${NC}"
    echo -e "${GREEN}Average correction speed: $avg_cor_speed MB/s${NC}"
    echo ""
    
    # Projection for large files
    echo -e "${CYAN}=========================================${NC}"
    echo -e "${CYAN}Projections for Large Files${NC}"
    echo -e "${CYAN}=========================================${NC}"
    echo ""
    
    for size in 100 500 1000 8000; do
        val_minutes=$(echo "scale=1; ($size / $avg_val_speed) / 60" | bc)
        cor_minutes=$(echo "scale=1; ($size / $avg_cor_speed) / 60" | bc)
        
        if [ $size -ge 1000 ]; then
            size_label="$(echo "scale=0; $size / 1000" | bc) GB"
        else
            size_label="$size MB"
        fi
        
        echo -e "${size_label}:"
        echo -e "${YELLOW}  Validation: $val_minutes min${NC}"
        echo -e "${YELLOW}  Correction: $cor_minutes min${NC}"
    done
    echo ""
    
    # Detailed results
    echo -e "${CYAN}=========================================${NC}"
    echo -e "${CYAN}Detailed Results${NC}"
    echo -e "${CYAN}=========================================${NC}"
    echo ""
    
    printf "%-30s %10s %10s %12s %10s %12s\n" "File" "Size(MB)" "Valid(s)" "Valid(MB/s)" "Corr(s)" "Corr(MB/s)"
    printf "%.s-" {1..90}
    echo ""
    
    for result in "${RESULTS[@]}"; do
        IFS='|' read -r name size val_time val_speed cor_time cor_speed <<< "$result"
        printf "%-30s %10s %10s %12s %10s %12s\n" "$name" "$size" "$val_time" "$val_speed" "$cor_time" "$cor_speed"
    done
fi

echo ""
echo -e "${GREEN}Benchmark completed!${NC}"
