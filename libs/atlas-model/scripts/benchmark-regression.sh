#!/bin/bash

# Benchmark Regression Detection Script
# This script runs benchmarks and compares results against baseline to detect regressions

set -e

# Configuration
BASELINE_DIR="benchmarks/baselines"
CURRENT_RESULTS="benchmarks/current.json"
COMPARISON_RESULTS="benchmarks/comparison.json"
THRESHOLD_PERCENT=20  # Alert if performance degrades by more than 20%

# Ensure benchmark directories exist
mkdir -p benchmarks
mkdir -p "$BASELINE_DIR"

# Function to run benchmarks and output JSON
run_benchmarks() {
    echo "Running benchmarks..."
    go test -bench=. -benchmem -benchtime=3s -count=5 ./... | \
    go run scripts/benchmark-parser/benchmark-parser.go > "$CURRENT_RESULTS"
    
    if [[ ! -f "$CURRENT_RESULTS" ]]; then
        echo "Error: Failed to generate benchmark results"
        exit 1
    fi
    
    echo "Benchmarks completed. Results saved to $CURRENT_RESULTS"
}

# Function to get safe branch name (replace slashes with underscores)
get_safe_branch_name() {
    git rev-parse --abbrev-ref HEAD | sed 's/\//_/g'
}

# Function to check if baseline exists
baseline_exists() {
    local branch_name=$(get_safe_branch_name)
    local baseline_file="$BASELINE_DIR/${branch_name}.json"
    [[ -f "$baseline_file" ]]
}

# Function to save current results as baseline
save_baseline() {
    local branch_name=$(get_safe_branch_name)
    local baseline_file="$BASELINE_DIR/${branch_name}.json"
    
    echo "Saving baseline for branch: $branch_name"
    cp "$CURRENT_RESULTS" "$baseline_file"
    echo "Baseline saved to $baseline_file"
}

# Function to compare against baseline
compare_with_baseline() {
    local branch_name=$(get_safe_branch_name)
    local baseline_file="$BASELINE_DIR/${branch_name}.json"
    
    echo "Comparing against baseline: $baseline_file"
    
    go run scripts/benchmark-comparer/benchmark-comparer.go \
        -baseline="$baseline_file" \
        -current="$CURRENT_RESULTS" \
        -threshold="$THRESHOLD_PERCENT" \
        -output="$COMPARISON_RESULTS"
    
    local comparison_exit_code=$?
    
    if [[ $comparison_exit_code -eq 0 ]]; then
        echo "✅ No significant performance regressions detected"
        return 0
    elif [[ $comparison_exit_code -eq 1 ]]; then
        echo "⚠️  Performance regressions detected!"
        echo "See $COMPARISON_RESULTS for details"
        
        # Display regression summary
        if [[ -f "$COMPARISON_RESULTS" ]]; then
            echo ""
            echo "Regression Summary:"
            echo "=================="
            cat "$COMPARISON_RESULTS"
        fi
        
        return 1
    else
        echo "❌ Error occurred during benchmark comparison"
        return 2
    fi
}

# Main execution
main() {
    case "${1:-run}" in
        "run")
            echo "Starting benchmark regression detection..."
            run_benchmarks
            
            if baseline_exists; then
                compare_with_baseline
            else
                echo "No baseline found. This will be saved as the new baseline."
                save_baseline
            fi
            ;;
        "baseline")
            echo "Creating new baseline..."
            run_benchmarks
            save_baseline
            ;;
        "compare-only")
            if [[ -f "$CURRENT_RESULTS" ]]; then
                compare_with_baseline
            else
                echo "No current results found. Run benchmarks first."
                exit 1
            fi
            ;;
        *)
            echo "Usage: $0 [run|baseline|compare-only]"
            echo "  run:          Run benchmarks and compare (default)"
            echo "  baseline:     Create new baseline"
            echo "  compare-only: Compare existing results"
            exit 1
            ;;
    esac
}

main "$@"