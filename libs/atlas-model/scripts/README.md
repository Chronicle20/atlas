# Benchmark Regression Detection

This directory contains tools for detecting performance regressions in benchmark results across commits and pull requests.

## Overview

The benchmark regression detection system consists of three main components:

1. **benchmark-regression.sh** - Main script that orchestrates benchmark runs and comparisons
2. **benchmark-parser.go** - Parses Go benchmark output into structured JSON format
3. **benchmark-comparer.go** - Compares current benchmark results against baselines to detect regressions

## How It Works

### Baseline Creation
- On the `main` branch, benchmarks are run and saved as baselines
- Baselines are stored in `benchmarks/baselines/{branch}.json`
- Each baseline includes metadata (commit hash, branch, timestamp)

### Regression Detection
- On pull requests, benchmarks are run and compared against the baseline
- Performance regressions exceeding the threshold (default: 20%) trigger failures
- Results are saved as artifacts for analysis

### CI Integration
- **Main branch**: Runs `./scripts/benchmark-regression.sh baseline` to create/update baselines
- **Pull requests**: Runs `./scripts/benchmark-regression.sh run` to detect regressions

## Usage

### Manual Usage

```bash
# Run benchmarks and compare against baseline (creates baseline if none exists)
./scripts/benchmark-regression.sh run

# Create/update baseline for current branch
./scripts/benchmark-regression.sh baseline

# Compare existing results only (requires current.json to exist)
./scripts/benchmark-regression.sh compare-only
```

### Configuration

The regression threshold can be configured by editing the `THRESHOLD_PERCENT` variable in `benchmark-regression.sh`:

```bash
THRESHOLD_PERCENT=20  # Alert if performance degrades by more than 20%
```

### Output Format

#### JSON Results
Benchmark results are stored in structured JSON format with the following fields:

```json
{
  "metadata": {
    "commit_hash": "abc123...",
    "branch": "feature-branch",
    "go_version": "1.24.x"
  },
  "benchmarks": [
    {
      "name": "MapLazyEvaluation",
      "package": "model",
      "iterations": 5000000,
      "ns_per_op": 234.5,
      "bytes_per_op": 24,
      "allocs_per_op": 2,
      "timestamp": "2024-01-01T12:00:00Z",
      "commit_hash": "abc123...",
      "branch": "feature-branch"
    }
  ],
  "generated_at": "2024-01-01T12:00:00Z"
}
```

#### Comparison Results
Regression detection results include detailed comparison information:

```json
{
  "total_benchmarks": 25,
  "regressions_detected": 2,
  "improvements_detected": 1,
  "threshold_percent": 20.0,
  "has_regressions": true,
  "results": [
    {
      "name": "MapComposition",
      "package": "model",
      "baseline_ns_per_op": 200.0,
      "current_ns_per_op": 250.0,
      "performance_change_percent": 25.0,
      "is_regression": true,
      "is_improvement": false
    }
  ]
}
```

## File Structure

```
benchmarks/
├── baselines/
│   ├── main.json           # Baseline for main branch
│   ├── feature-x.json      # Baseline for feature branch
│   └── ...
├── current.json            # Latest benchmark results
└── comparison.json         # Latest comparison results
```

## Integration with CI/CD

### GitHub Actions
The system is integrated with GitHub Actions workflows:

- **pull-request.yml**: Detects regressions on PRs
- **main-snapshot.yml**: Updates baselines on main branch

### Artifacts
Benchmark results are uploaded as GitHub Actions artifacts:
- `benchmark-results`: Current run results and comparisons (30 days retention)
- `benchmark-baseline-main`: Main branch baselines (90 days retention)

## Troubleshooting

### Common Issues

1. **No baseline found**
   - Solution: Run `./scripts/benchmark-regression.sh baseline` to create one

2. **Script not executable**
   - Solution: Run `chmod +x scripts/benchmark-regression.sh`

3. **Go modules not found**
   - Solution: Ensure you're in the project root and run `go mod download`

### Debugging

Enable verbose output by modifying the script to include:
```bash
set -x  # Enable debugging output
```

### Manual Benchmark Runs

Run benchmarks manually for debugging:
```bash
# Run all benchmarks with memory profiling
go test -bench=. -benchmem -benchtime=3s -count=5 ./...

# Run specific benchmark
go test -bench=BenchmarkMapLazyEvaluation -benchmem ./model

# Generate CPU profile
go test -bench=. -cpuprofile=cpu.prof ./...
```

## Best Practices

1. **Consistent Environment**: Run benchmarks in consistent environments for reliable comparisons
2. **Multiple Runs**: Use `-count=5` or similar for statistical significance  
3. **Appropriate Benchtime**: Use `-benchtime=3s` for stable results
4. **Monitor Trends**: Track performance over time, not just regressions
5. **Investigate Regressions**: Don't just ignore regression alerts - investigate root causes

## Limitations

1. **Environment Sensitivity**: Benchmark results can vary between different machines/environments
2. **Statistical Variance**: Single runs may show false positives due to natural variance
3. **Memory Pressure**: System memory pressure can affect results
4. **CPU Throttling**: Thermal throttling on CI systems can cause inconsistent results

Consider using multiple runs and statistical analysis for production-critical performance monitoring.