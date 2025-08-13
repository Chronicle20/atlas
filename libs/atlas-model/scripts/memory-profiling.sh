#!/bin/bash

# Memory Profiling Script for CI
# This script runs tests and benchmarks with memory profiling enabled

set -e

# Configuration
PROFILE_DIR="profiles"
CURRENT_TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BRANCH_NAME=$(git rev-parse --abbrev-ref HEAD | sed 's/\//_/g')

# Ensure profile directory exists
mkdir -p "$PROFILE_DIR"

echo "Starting memory profiling for branch: $BRANCH_NAME"

# Function to run tests with memory profiling
run_tests_with_memory_profile() {
    echo "Running tests with memory profiling..."
    
    # Find all packages with tests
    local packages=$(go list ./... 2>/dev/null | grep -v vendor)
    local profile_count=0
    
    for package in $packages; do
        # Check if package has tests
        if go test -list . "$package" >/dev/null 2>&1; then
            profile_count=$((profile_count + 1))
            local package_name=$(basename "$package")
            
            echo "  Profiling package: $package"
            go test -memprofile="$PROFILE_DIR/test_memory_${package_name}_${BRANCH_NAME}_${CURRENT_TIMESTAMP}.memprof" \
                    -memprofilerate=1 \
                    -v "$package" 2>/dev/null || echo "    Warning: Could not profile $package"
        fi
    done
    
    if [ $profile_count -gt 0 ]; then
        echo "✅ Generated $profile_count test memory profiles in $PROFILE_DIR/"
    else
        echo "⚠️  No test packages found for memory profiling"
    fi
}

# Function to run benchmarks with memory profiling
run_benchmarks_with_memory_profile() {
    echo "Running benchmarks with memory profiling..."
    
    # Find all packages with benchmarks
    local packages=$(go list ./... 2>/dev/null | grep -v vendor)
    local profile_count=0
    
    for package in $packages; do
        # Check if package has benchmarks by running a dry run
        if go test -bench=. -run=^$ -short "$package" >/dev/null 2>&1; then
            profile_count=$((profile_count + 1))
            local package_name=$(basename "$package")
            
            echo "  Profiling benchmarks in package: $package"
            go test -bench=. \
                    -benchmem \
                    -memprofile="$PROFILE_DIR/benchmark_memory_${package_name}_${BRANCH_NAME}_${CURRENT_TIMESTAMP}.memprof" \
                    -memprofilerate=1 \
                    -benchtime=100ms \
                    -count=1 \
                    "$package" 2>/dev/null || echo "    Warning: Could not profile benchmarks in $package"
        fi
    done
    
    if [ $profile_count -gt 0 ]; then
        echo "✅ Generated $profile_count benchmark memory profiles in $PROFILE_DIR/"
    else
        echo "⚠️  No benchmark packages found for memory profiling"
    fi
}

# Function to generate memory profile summary
generate_memory_summary() {
    echo "Generating memory profile summary..."
    
    # Find all test profiles for this timestamp
    local test_profiles=$(find "$PROFILE_DIR" -name "test_memory_*_${BRANCH_NAME}_${CURRENT_TIMESTAMP}.memprof" -type f | sort)
    local benchmark_profiles=$(find "$PROFILE_DIR" -name "benchmark_memory_*_${BRANCH_NAME}_${CURRENT_TIMESTAMP}.memprof" -type f | sort)
    
    # Generate combined summary for test profiles
    if [[ -n "$test_profiles" ]]; then
        echo ""
        echo "Test Memory Profile Summary:"
        echo "============================"
        
        local summary_file="$PROFILE_DIR/test_memory_summary_${BRANCH_NAME}_${CURRENT_TIMESTAMP}.txt"
        echo "Memory usage across all test packages:" > "$summary_file"
        echo "=======================================" >> "$summary_file"
        echo "" >> "$summary_file"
        
        for profile in $test_profiles; do
            local package_name=$(basename "$profile" | sed "s/test_memory_//; s/_${BRANCH_NAME}_${CURRENT_TIMESTAMP}.memprof//")
            echo "Package: $package_name" >> "$summary_file"
            echo "-------------------" >> "$summary_file"
            go tool pprof -text -top5 "$profile" >> "$summary_file" 2>/dev/null || echo "Could not generate summary for $package_name" >> "$summary_file"
            echo "" >> "$summary_file"
        done
        
        cat "$summary_file"
    fi
    
    # Generate combined summary for benchmark profiles
    if [[ -n "$benchmark_profiles" ]]; then
        echo ""
        echo "Benchmark Memory Profile Summary:"
        echo "================================="
        
        local summary_file="$PROFILE_DIR/benchmark_memory_summary_${BRANCH_NAME}_${CURRENT_TIMESTAMP}.txt"
        echo "Memory usage across all benchmark packages:" > "$summary_file"
        echo "===========================================" >> "$summary_file"
        echo "" >> "$summary_file"
        
        for profile in $benchmark_profiles; do
            local package_name=$(basename "$profile" | sed "s/benchmark_memory_//; s/_${BRANCH_NAME}_${CURRENT_TIMESTAMP}.memprof//")
            echo "Package: $package_name" >> "$summary_file"
            echo "-------------------" >> "$summary_file"
            go tool pprof -text -top5 "$profile" >> "$summary_file" 2>/dev/null || echo "Could not generate summary for $package_name" >> "$summary_file"
            echo "" >> "$summary_file"
        done
        
        cat "$summary_file"
    fi
}

# Function to cleanup old profiles (keep only last 5)
cleanup_old_profiles() {
    echo "Cleaning up old profiles..."
    
    # Keep only the 5 most recent test profiles for this branch
    find "$PROFILE_DIR" -name "test_memory_${BRANCH_NAME}_*.memprof" -type f | sort -r | tail -n +6 | xargs -r rm -f
    find "$PROFILE_DIR" -name "test_memory_summary_${BRANCH_NAME}_*.txt" -type f | sort -r | tail -n +6 | xargs -r rm -f
    
    # Keep only the 5 most recent benchmark profiles for this branch
    find "$PROFILE_DIR" -name "benchmark_memory_${BRANCH_NAME}_*.memprof" -type f | sort -r | tail -n +6 | xargs -r rm -f
    find "$PROFILE_DIR" -name "benchmark_memory_summary_${BRANCH_NAME}_*.txt" -type f | sort -r | tail -n +6 | xargs -r rm -f
    
    echo "✅ Old profiles cleaned up"
}

# Main execution
main() {
    case "${1:-both}" in
        "tests")
            echo "Running tests with memory profiling..."
            run_tests_with_memory_profile
            generate_memory_summary
            ;;
        "benchmarks")
            echo "Running benchmarks with memory profiling..."
            run_benchmarks_with_memory_profile
            generate_memory_summary
            ;;
        "both")
            echo "Running both tests and benchmarks with memory profiling..."
            run_tests_with_memory_profile
            run_benchmarks_with_memory_profile
            generate_memory_summary
            ;;
        "cleanup")
            echo "Cleaning up old profiles..."
            cleanup_old_profiles
            ;;
        *)
            echo "Usage: $0 [tests|benchmarks|both|cleanup]"
            echo "  tests:      Run tests with memory profiling"
            echo "  benchmarks: Run benchmarks with memory profiling"
            echo "  both:       Run both tests and benchmarks (default)"
            echo "  cleanup:    Clean up old profiles"
            exit 1
            ;;
    esac
    
    # Always cleanup after profiling
    if [[ "$1" != "cleanup" ]]; then
        cleanup_old_profiles
    fi
    
    echo "Memory profiling completed successfully! 🎯"
}

main "$@"