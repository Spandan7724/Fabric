package openai

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsGitHubCopilotURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "GitHub Copilot URL",
			url:      "https://api.githubcopilot.com",
			expected: true,
		},
		{
			name:     "GitHub Copilot URL with path",
			url:      "https://api.githubcopilot.com/v1/chat/completions",
			expected: true,
		},
		{
			name:     "GitHub Copilot URL case insensitive",
			url:      "https://API.GITHUBCOPILOT.COM",
			expected: true,
		},
		{
			name:     "OpenAI URL",
			url:      "https://api.openai.com/v1",
			expected: false,
		},
		{
			name:     "Azure OpenAI URL",
			url:      "https://myresource.openai.azure.com",
			expected: false,
		},
		{
			name:     "Local URL",
			url:      "http://localhost:11434",
			expected: false,
		},
		{
			name:     "Empty URL",
			url:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGitHubCopilotURL(tt.url)
			if result != tt.expected {
				t.Errorf("isGitHubCopilotURL(%q) = %v, expected %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestCopilotTransport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiVersion := r.Header.Get("X-GitHub-Api-Version")
		if apiVersion != "2023-05-01" {
			t.Errorf("Expected X-GitHub-Api-Version header to be '2023-05-01', got '%s'", apiVersion)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()
	transport := NewCopilotTransport(nil)
	client := &http.Client{Transport: transport}
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}
}

func TestCopilotTransportPreservesExistingHeaders(t *testing.T) {
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	transport := NewCopilotTransport(nil)
	client := &http.Client{Transport: transport}
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-agent")

	_, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if capturedHeaders.Get("X-GitHub-Api-Version") != "2023-05-01" {
		t.Errorf("Expected X-GitHub-Api-Version header to be '2023-05-01', got '%s'", 
			capturedHeaders.Get("X-GitHub-Api-Version"))
	}
	if capturedHeaders.Get("Authorization") != "Bearer test-token" {
		t.Errorf("Expected Authorization header to be preserved")
	}
	if capturedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type header to be preserved")
	}
	if capturedHeaders.Get("User-Agent") != "test-agent" {
		t.Errorf("Expected User-Agent header to be preserved")
	}
}

func TestNewCopilotTransport(t *testing.T) {
	transport1 := NewCopilotTransport(nil)
	if transport1 == nil {
		t.Fatal("Expected transport to be created, got nil")
	}
	if transport1.base != http.DefaultTransport {
		t.Error("Expected base transport to default to http.DefaultTransport when nil")
	}
	customTransport := &http.Transport{}
	transport2 := NewCopilotTransport(customTransport)
	if transport2 == nil {
		t.Fatal("Expected transport to be created, got nil")
	}
	if transport2.base != customTransport {
		t.Error("Expected base transport to be the provided custom transport")
	}
}