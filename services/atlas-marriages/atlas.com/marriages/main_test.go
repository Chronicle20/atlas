package main

import (
	"testing"
)

func TestGetServer(t *testing.T) {
	server := GetServer()
	
	expectedBaseURL := ""
	expectedPrefix := "/api/mas/"
	
	if server.GetBaseURL() != expectedBaseURL {
		t.Errorf("Expected baseURL '%s', got '%s'", expectedBaseURL, server.GetBaseURL())
	}
	
	if server.GetPrefix() != expectedPrefix {
		t.Errorf("Expected prefix '%s', got '%s'", expectedPrefix, server.GetPrefix())
	}
}

func TestServer_GetBaseURL(t *testing.T) {
	server := Server{baseUrl: "https://test.com", prefix: "/test/"}
	
	expected := "https://test.com"
	if server.GetBaseURL() != expected {
		t.Errorf("Expected baseURL '%s', got '%s'", expected, server.GetBaseURL())
	}
}

func TestServer_GetPrefix(t *testing.T) {
	server := Server{baseUrl: "https://test.com", prefix: "/test/"}
	
	expected := "/test/"
	if server.GetPrefix() != expected {
		t.Errorf("Expected prefix '%s', got '%s'", expected, server.GetPrefix())
	}
}

func TestServiceName(t *testing.T) {
	expected := "atlas-marriages"
	if serviceName != expected {
		t.Errorf("Expected service name '%s', got '%s'", expected, serviceName)
	}
}