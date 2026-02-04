#!/bin/bash

# Script to validate all PGN files in a directory
# Usage: ./validate_all.sh <directory> [-o output_directory]

if [ $# -lt 1 ]; then
    echo "Usage: $0 <directory> [-o output_directory]"
    echo "Example: $0 ./test_files"
    echo "         $0 ./test_files -o ./corrected_files"
    exit 1
fi

input_dir="$1"
output_dir=""
auto_correct=false

# Check if -o flag is provided
if [ $# -eq 3 ] && [ "$2" == "-o" ]; then
    output_dir="$3"
    auto_correct=true
    mkdir -p "$output_dir"
fi

# Check if directory exists
if [ ! -d "$input_dir" ]; then
    echo "Error: Directory '$input_dir' not found"
    exit 1
fi

# Find all .pgn files
pgn_files=$(find "$input_dir" -type f -name "*.pgn")

if [ -z "$pgn_files" ]; then
    echo "No PGN files found in '$input_dir'"
    exit 0
fi

# Count total files
total_files=$(echo "$pgn_files" | wc -l)
current=0
valid_count=0
invalid_count=0

echo "========================================="
echo "PGN Validator - Batch Processing"
echo "========================================="
echo "Directory: $input_dir"
echo "Total files: $total_files"
if [ "$auto_correct" = true ]; then
    echo "Output directory: $output_dir"
fi
echo "========================================="
echo ""

# Process each file
while IFS= read -r file; do
    ((current++))
    filename=$(basename "$file")
    
    echo "[$current/$total_files] Processing: $filename"
    
    if [ "$auto_correct" = true ]; then
        # Validate and correct
        output_file="$output_dir/$filename"
        error_output=$(./pgn_check.exe -o "$output_file" "$file" 2>&1)
        result=$?
    else
        # Validate only
        error_output=$(./pgn_check.exe "$file" 2>&1)
        result=$?
    fi
    
    if [ $result -eq 0 ]; then
        ((valid_count++))
        echo "  ✓ Valid"
    else
        ((invalid_count++))
        echo "  ✗ Errors found"
        # Display errors and warnings if present
        if echo "$error_output" | grep -qE "(Error:|Warning:)"; then
            echo "$error_output" | grep -E "(Error:|Warning:|Line [0-9]+:)" | sed 's/^/    /'
        fi
    fi
    echo ""
done <<< "$pgn_files"

# Summary
echo "========================================="
echo "Summary"
echo "========================================="
echo "Total processed: $total_files"
echo "Valid files: $valid_count"
echo "Files with errors: $invalid_count"
if [ "$auto_correct" = true ]; then
    echo "Corrected files saved to: $output_dir"
fi
echo "========================================="
