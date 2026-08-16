package main

import (
	"os"
	"strings"
	"testing"
)

func TestMainWiresTheEnvironmentRegistry(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(src), "service.WithEnvironmentRegistry(serviceName)") {
		t.Fatal("main.go does not pass service.WithEnvironmentRegistry to Bootstrap")
	}
}
