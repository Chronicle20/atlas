package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// BenchmarkResult represents a single benchmark result
type BenchmarkResult struct {
	Name           string    `json:"name"`
	Package        string    `json:"package"`
	Iterations     int64     `json:"iterations"`
	NsPerOp        float64   `json:"ns_per_op"`
	BytesPerOp     int64     `json:"bytes_per_op,omitempty"`
	AllocsPerOp    int64     `json:"allocs_per_op,omitempty"`
	MBPerSec       float64   `json:"mb_per_sec,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
	CommitHash     string    `json:"commit_hash,omitempty"`
	Branch         string    `json:"branch,omitempty"`
}

// BenchmarkSuite represents a collection of benchmark results
type BenchmarkSuite struct {
	Metadata    map[string]string  `json:"metadata"`
	Benchmarks  []BenchmarkResult  `json:"benchmarks"`
	GeneratedAt time.Time          `json:"generated_at"`
}

// Regular expression to parse benchmark output
// Example: BenchmarkMapLazyEvaluation-8    	 5000000	       234 ns/op	      24 B/op	       2 allocs/op
var benchmarkRegex = regexp.MustCompile(`^Benchmark(\w+)(-\d+)?\s+(\d+)\s+([0-9.]+)\s+ns/op(?:\s+([0-9]+)\s+B/op)?(?:\s+([0-9]+)\s+allocs/op)?(?:\s+([0-9.]+)\s+MB/s)?`)

func main() {
	var results []BenchmarkResult
	scanner := bufio.NewScanner(os.Stdin)
	
	// Get git information for metadata
	commitHash := getGitCommitHash()
	branch := getGitBranch()
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip non-benchmark lines
		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}
		
		// Parse benchmark line
		matches := benchmarkRegex.FindStringSubmatch(line)
		if len(matches) < 5 {
			continue
		}
		
		result := BenchmarkResult{
			Name:       matches[1],
			Timestamp:  time.Now(),
			CommitHash: commitHash,
			Branch:     branch,
		}
		
		// Parse iterations
		if iterations, err := strconv.ParseInt(matches[3], 10, 64); err == nil {
			result.Iterations = iterations
		}
		
		// Parse ns/op
		if nsPerOp, err := strconv.ParseFloat(matches[4], 64); err == nil {
			result.NsPerOp = nsPerOp
		}
		
		// Parse bytes per op (optional)
		if len(matches) > 5 && matches[5] != "" {
			if bytesPerOp, err := strconv.ParseInt(matches[5], 10, 64); err == nil {
				result.BytesPerOp = bytesPerOp
			}
		}
		
		// Parse allocs per op (optional)
		if len(matches) > 6 && matches[6] != "" {
			if allocsPerOp, err := strconv.ParseInt(matches[6], 10, 64); err == nil {
				result.AllocsPerOp = allocsPerOp
			}
		}
		
		// Parse MB/s (optional)
		if len(matches) > 7 && matches[7] != "" {
			if mbPerSec, err := strconv.ParseFloat(matches[7], 64); err == nil {
				result.MBPerSec = mbPerSec
			}
		}
		
		// Try to extract package information from GOPACKAGE env var or assume model package
		result.Package = getPackageFromBenchmarkName(result.Name)
		
		results = append(results, result)
	}
	
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
	
	// Create benchmark suite
	suite := BenchmarkSuite{
		Metadata: map[string]string{
			"commit_hash": commitHash,
			"branch":      branch,
			"go_version":  getGoVersion(),
		},
		Benchmarks:  results,
		GeneratedAt: time.Now(),
	}
	
	// Output JSON
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(suite); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

// getGitCommitHash returns the current git commit hash
func getGitCommitHash() string {
	if hash := os.Getenv("GITHUB_SHA"); hash != "" {
		return hash
	}
	// Fallback to local git command would go here
	return "unknown"
}

// getGitBranch returns the current git branch
func getGitBranch() string {
	if ref := os.Getenv("GITHUB_REF"); ref != "" {
		// Extract branch name from refs/heads/branch-name
		if strings.HasPrefix(ref, "refs/heads/") {
			return strings.TrimPrefix(ref, "refs/heads/")
		}
	}
	// Fallback to local git command would go here
	return "unknown"
}

// getGoVersion returns the Go version
func getGoVersion() string {
	// This would typically run `go version` command
	return "1.24.x"
}

// getPackageFromBenchmarkName attempts to determine the package from benchmark name
func getPackageFromBenchmarkName(name string) string {
	// Simple heuristic based on benchmark name patterns
	if strings.Contains(name, "Async") || strings.Contains(name, "Await") {
		return "async"
	}
	return "model"
}