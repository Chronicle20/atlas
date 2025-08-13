package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
)

// BenchmarkResult represents a single benchmark result (matching parser.go)
type BenchmarkResult struct {
	Name           string  `json:"name"`
	Package        string  `json:"package"`
	Iterations     int64   `json:"iterations"`
	NsPerOp        float64 `json:"ns_per_op"`
	BytesPerOp     int64   `json:"bytes_per_op,omitempty"`
	AllocsPerOp    int64   `json:"allocs_per_op,omitempty"`
	MBPerSec       float64 `json:"mb_per_sec,omitempty"`
}

// BenchmarkSuite represents a collection of benchmark results (matching parser.go)
type BenchmarkSuite struct {
	Metadata    map[string]string `json:"metadata"`
	Benchmarks  []BenchmarkResult `json:"benchmarks"`
	GeneratedAt interface{}       `json:"generated_at"`
}

// ComparisonResult represents the result of comparing two benchmark results
type ComparisonResult struct {
	Name               string  `json:"name"`
	Package            string  `json:"package"`
	BaselineNsPerOp    float64 `json:"baseline_ns_per_op"`
	CurrentNsPerOp     float64 `json:"current_ns_per_op"`
	PerformanceChange  float64 `json:"performance_change_percent"`
	IsRegression       bool    `json:"is_regression"`
	IsImprovement      bool    `json:"is_improvement"`
	BaselineBytesPerOp int64   `json:"baseline_bytes_per_op,omitempty"`
	CurrentBytesPerOp  int64   `json:"current_bytes_per_op,omitempty"`
	BytesChange        float64 `json:"bytes_change_percent,omitempty"`
	BaselineAllocs     int64   `json:"baseline_allocs_per_op,omitempty"`
	CurrentAllocs      int64   `json:"current_allocs_per_op,omitempty"`
	AllocsChange       float64 `json:"allocs_change_percent,omitempty"`
}

// ComparisonSummary represents the overall comparison results
type ComparisonSummary struct {
	TotalBenchmarks      int                `json:"total_benchmarks"`
	RegressionsDetected  int                `json:"regressions_detected"`
	ImprovementsDetected int                `json:"improvements_detected"`
	Threshold            float64            `json:"threshold_percent"`
	HasRegressions       bool               `json:"has_regressions"`
	Results              []ComparisonResult `json:"results"`
}

func main() {
	var (
		baselineFile = flag.String("baseline", "", "Path to baseline benchmark results JSON file")
		currentFile  = flag.String("current", "", "Path to current benchmark results JSON file")
		threshold    = flag.Float64("threshold", 20.0, "Regression threshold percentage")
		outputFile   = flag.String("output", "", "Path to output comparison results JSON file")
	)
	flag.Parse()

	if *baselineFile == "" || *currentFile == "" {
		fmt.Fprintf(os.Stderr, "Error: Both -baseline and -current flags are required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Load baseline results
	baselineResults, err := loadBenchmarkSuite(*baselineFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading baseline: %v\n", err)
		os.Exit(2)
	}

	// Load current results
	currentResults, err := loadBenchmarkSuite(*currentFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading current results: %v\n", err)
		os.Exit(2)
	}

	// Compare results
	summary := compareBenchmarks(baselineResults, currentResults, *threshold)

	// Output results
	if *outputFile != "" {
		if err := saveSummary(summary, *outputFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving summary: %v\n", err)
			os.Exit(2)
		}
	} else {
		// Output to stdout
		printSummary(summary)
	}

	// Exit with code 1 if regressions detected, 0 otherwise
	if summary.HasRegressions {
		os.Exit(1)
	}
}

// loadBenchmarkSuite loads benchmark results from a JSON file
func loadBenchmarkSuite(filename string) (*BenchmarkSuite, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	var suite BenchmarkSuite
	if err := json.NewDecoder(file).Decode(&suite); err != nil {
		return nil, fmt.Errorf("failed to decode JSON from %s: %w", filename, err)
	}

	return &suite, nil
}

// compareBenchmarks compares current results against baseline
func compareBenchmarks(baseline, current *BenchmarkSuite, threshold float64) *ComparisonSummary {
	summary := &ComparisonSummary{
		Threshold: threshold,
		Results:   []ComparisonResult{},
	}

	// Create lookup map for current results
	currentMap := make(map[string]BenchmarkResult)
	for _, result := range current.Benchmarks {
		key := fmt.Sprintf("%s/%s", result.Package, result.Name)
		currentMap[key] = result
	}

	// Compare each baseline benchmark with current
	for _, baselineResult := range baseline.Benchmarks {
		key := fmt.Sprintf("%s/%s", baselineResult.Package, baselineResult.Name)
		currentResult, exists := currentMap[key]
		
		if !exists {
			// Benchmark was removed - this might be worth noting but not an error
			continue
		}

		comparison := compareIndividualBenchmarks(baselineResult, currentResult, threshold)
		summary.Results = append(summary.Results, comparison)

		if comparison.IsRegression {
			summary.RegressionsDetected++
		}
		if comparison.IsImprovement {
			summary.ImprovementsDetected++
		}
	}

	summary.TotalBenchmarks = len(summary.Results)
	summary.HasRegressions = summary.RegressionsDetected > 0

	// Sort results by performance change (worst regressions first)
	sort.Slice(summary.Results, func(i, j int) bool {
		return summary.Results[i].PerformanceChange > summary.Results[j].PerformanceChange
	})

	return summary
}

// compareIndividualBenchmarks compares two individual benchmark results
func compareIndividualBenchmarks(baseline, current BenchmarkResult, threshold float64) ComparisonResult {
	comparison := ComparisonResult{
		Name:            current.Name,
		Package:         current.Package,
		BaselineNsPerOp: baseline.NsPerOp,
		CurrentNsPerOp:  current.NsPerOp,
	}

	// Calculate performance change percentage
	// Positive percentage = performance regression (slower)
	// Negative percentage = performance improvement (faster)
	if baseline.NsPerOp > 0 {
		comparison.PerformanceChange = ((current.NsPerOp - baseline.NsPerOp) / baseline.NsPerOp) * 100
	}

	comparison.IsRegression = comparison.PerformanceChange > threshold
	comparison.IsImprovement = comparison.PerformanceChange < -threshold

	// Compare memory allocations if available
	if baseline.BytesPerOp > 0 && current.BytesPerOp > 0 {
		comparison.BaselineBytesPerOp = baseline.BytesPerOp
		comparison.CurrentBytesPerOp = current.BytesPerOp
		comparison.BytesChange = ((float64(current.BytesPerOp) - float64(baseline.BytesPerOp)) / float64(baseline.BytesPerOp)) * 100
	}

	// Compare allocation count if available
	if baseline.AllocsPerOp > 0 && current.AllocsPerOp > 0 {
		comparison.BaselineAllocs = baseline.AllocsPerOp
		comparison.CurrentAllocs = current.AllocsPerOp
		comparison.AllocsChange = ((float64(current.AllocsPerOp) - float64(baseline.AllocsPerOp)) / float64(baseline.AllocsPerOp)) * 100
	}

	return comparison
}

// saveSummary saves the comparison summary to a JSON file
func saveSummary(summary *ComparisonSummary, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(summary)
}

// printSummary prints a human-readable summary to stdout
func printSummary(summary *ComparisonSummary) {
	fmt.Printf("Benchmark Comparison Summary\n")
	fmt.Printf("============================\n")
	fmt.Printf("Total Benchmarks: %d\n", summary.TotalBenchmarks)
	fmt.Printf("Regressions Detected: %d\n", summary.RegressionsDetected)
	fmt.Printf("Improvements Detected: %d\n", summary.ImprovementsDetected)
	fmt.Printf("Regression Threshold: %.1f%%\n\n", summary.Threshold)

	if summary.RegressionsDetected > 0 {
		fmt.Printf("⚠️  Performance Regressions:\n")
		fmt.Printf("============================\n")
		
		for _, result := range summary.Results {
			if result.IsRegression {
				fmt.Printf("• %s/%s: %.1f%% slower (%.2f → %.2f ns/op)\n",
					result.Package, result.Name, result.PerformanceChange,
					result.BaselineNsPerOp, result.CurrentNsPerOp)
				
				if result.BytesChange != 0 {
					fmt.Printf("  Memory: %.1f%% change (%d → %d B/op)\n",
						result.BytesChange, result.BaselineBytesPerOp, result.CurrentBytesPerOp)
				}
				
				if result.AllocsChange != 0 {
					fmt.Printf("  Allocations: %.1f%% change (%d → %d allocs/op)\n",
						result.AllocsChange, result.BaselineAllocs, result.CurrentAllocs)
				}
				fmt.Printf("\n")
			}
		}
	}

	if summary.ImprovementsDetected > 0 {
		fmt.Printf("✅ Performance Improvements:\n")
		fmt.Printf("============================\n")
		
		for _, result := range summary.Results {
			if result.IsImprovement {
				fmt.Printf("• %s/%s: %.1f%% faster (%.2f → %.2f ns/op)\n",
					result.Package, result.Name, math.Abs(result.PerformanceChange),
					result.BaselineNsPerOp, result.CurrentNsPerOp)
			}
		}
	}
}