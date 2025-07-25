package openai

import (
	"strings"
	"testing"

	"github.com/danielmiessler/fabric/internal/chat"
	"github.com/danielmiessler/fabric/internal/domain"
	"github.com/danielmiessler/fabric/internal/plugins"
	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"
	"github.com/stretchr/testify/assert"
)

func TestBuildResponseRequestWithMaxTokens(t *testing.T) {

	var msgs []*chat.ChatCompletionMessage

	for i := 0; i < 2; i++ {
		msgs = append(msgs, &chat.ChatCompletionMessage{
			Role:    "User",
			Content: "My msg",
		})
	}

	opts := &domain.ChatOptions{
		Temperature: 0.8,
		TopP:        0.9,
		Raw:         false,
		MaxTokens:   50,
	}

	var client = NewClient()
	request := client.buildResponseParams(msgs, opts)
	assert.Equal(t, shared.ResponsesModel(opts.Model), request.Model)
	assert.Equal(t, openai.Float(opts.Temperature), request.Temperature)
	assert.Equal(t, openai.Float(opts.TopP), request.TopP)
	assert.Equal(t, openai.Int(int64(opts.MaxTokens)), request.MaxOutputTokens)
}

func TestBuildResponseRequestNoMaxTokens(t *testing.T) {

	var msgs []*chat.ChatCompletionMessage

	for i := 0; i < 2; i++ {
		msgs = append(msgs, &chat.ChatCompletionMessage{
			Role:    "User",
			Content: "My msg",
		})
	}

	opts := &domain.ChatOptions{
		Temperature: 0.8,
		TopP:        0.9,
		Raw:         false,
	}

	var client = NewClient()
	request := client.buildResponseParams(msgs, opts)
	assert.Equal(t, shared.ResponsesModel(opts.Model), request.Model)
	assert.Equal(t, openai.Float(opts.Temperature), request.Temperature)
	assert.Equal(t, openai.Float(opts.TopP), request.TopP)
	assert.False(t, request.MaxOutputTokens.Valid())
}

func TestBuildResponseParams_WithoutSearch(t *testing.T) {
	client := NewClient()
	opts := &domain.ChatOptions{
		Model:       "gpt-4o",
		Temperature: 0.7,
		Search:      false,
	}

	msgs := []*chat.ChatCompletionMessage{
		{Role: "user", Content: "Hello"},
	}

	params := client.buildResponseParams(msgs, opts)

	assert.Nil(t, params.Tools, "Expected no tools when search is disabled")
	assert.Equal(t, shared.ResponsesModel(opts.Model), params.Model)
	assert.Equal(t, openai.Float(opts.Temperature), params.Temperature)
}

func TestBuildResponseParams_WithSearch(t *testing.T) {
	client := NewClient()
	opts := &domain.ChatOptions{
		Model:       "gpt-4o",
		Temperature: 0.7,
		Search:      true,
	}

	msgs := []*chat.ChatCompletionMessage{
		{Role: "user", Content: "What's the weather today?"},
	}

	params := client.buildResponseParams(msgs, opts)

	assert.NotNil(t, params.Tools, "Expected tools when search is enabled")
	assert.Len(t, params.Tools, 1, "Expected exactly one tool")

	tool := params.Tools[0]
	assert.NotNil(t, tool.OfWebSearchPreview, "Expected web search tool")
	assert.Equal(t, responses.WebSearchToolType("web_search_preview"), tool.OfWebSearchPreview.Type)
}

func TestBuildResponseParams_WithSearchAndLocation(t *testing.T) {
	client := NewClient()
	opts := &domain.ChatOptions{
		Model:          "gpt-4o",
		Temperature:    0.7,
		Search:         true,
		SearchLocation: "America/Los_Angeles",
	}

	msgs := []*chat.ChatCompletionMessage{
		{Role: "user", Content: "What's the weather in San Francisco?"},
	}

	params := client.buildResponseParams(msgs, opts)

	assert.NotNil(t, params.Tools, "Expected tools when search is enabled")
	tool := params.Tools[0]
	assert.NotNil(t, tool.OfWebSearchPreview, "Expected web search tool")

	userLocation := tool.OfWebSearchPreview.UserLocation
	assert.Equal(t, "approximate", string(userLocation.Type))
	assert.True(t, userLocation.Timezone.Valid(), "Expected timezone to be set")
	assert.Equal(t, opts.SearchLocation, userLocation.Timezone.Value)
}

func TestCitationFormatting(t *testing.T) {
	// Test the citation formatting logic by simulating the citation extraction
	var textParts []string
	var citations []string
	citationMap := make(map[string]bool)

	// Simulate text content
	textParts = append(textParts, "Based on recent research, artificial intelligence is advancing rapidly.")

	// Simulate citations (as they would be extracted from OpenAI response)
	mockCitations := []struct {
		URL   string
		Title string
	}{
		{"https://example.com/ai-research", "AI Research Advances 2025"},
		{"https://another-source.com/tech-news", "Technology News Today"},
		{"https://example.com/ai-research", "AI Research Advances 2025"}, // Duplicate to test deduplication
	}

	for _, citation := range mockCitations {
		citationKey := citation.URL + "|" + citation.Title
		if !citationMap[citationKey] {
			citationMap[citationKey] = true
			citationText := "- [" + citation.Title + "](" + citation.URL + ")"
			citations = append(citations, citationText)
		}
	}

	result := strings.Join(textParts, "")
	if len(citations) > 0 {
		result += "\n\n## Sources\n\n" + strings.Join(citations, "\n")
	}

	// Verify the result contains the expected text
	expectedText := "Based on recent research, artificial intelligence is advancing rapidly."
	assert.Contains(t, result, expectedText, "Expected result to contain original text")

	// Verify citations are included
	assert.Contains(t, result, "## Sources", "Expected result to contain Sources section")
	assert.Contains(t, result, "[AI Research Advances 2025](https://example.com/ai-research)", "Expected result to contain first citation")
	assert.Contains(t, result, "[Technology News Today](https://another-source.com/tech-news)", "Expected result to contain second citation")

	// Verify deduplication - should only have 2 unique citations, not 3
	citationCount := strings.Count(result, "- [")
	assert.Equal(t, 2, citationCount, "Expected 2 unique citations")
}

func TestGitHubCopilotConfiguration(t *testing.T) {
	tests := []struct {
		name                string
		apiBaseURL          string
		expectCopilotHeader bool
	}{
		{
			name:                "GitHub Copilot URL",
			apiBaseURL:          "https://api.githubcopilot.com",
			expectCopilotHeader: true,
		},
		{
			name:                "GitHub Copilot URL with path",
			apiBaseURL:          "https://api.githubcopilot.com/v1/chat/completions",
			expectCopilotHeader: true,
		},
		{
			name:                "Regular OpenAI URL",
			apiBaseURL:          "https://api.openai.com/v1",
			expectCopilotHeader: false,
		},
		{
			name:                "Azure OpenAI URL",
			apiBaseURL:          "https://myresource.openai.azure.com",
			expectCopilotHeader: false,
		},
		{
			name:                "Empty URL (default OpenAI)",
			apiBaseURL:          "",
			expectCopilotHeader: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a client with the test URL
			client := NewClient()
			client.ApiKey = &plugins.SetupQuestion{
				Setting: &plugins.Setting{Value: "test-api-key"},
			}
			client.ApiBaseURL = &plugins.SetupQuestion{
				Setting: &plugins.Setting{Value: tt.apiBaseURL},
			}
			err := client.configure()
			assert.NoError(t, err, "Configure should not return an error")

			// Check if the client was created successfully
			assert.NotNil(t, client.ApiClient, "API client should be initialized")
			actualDetection := isGitHubCopilotURL(tt.apiBaseURL)
			assert.Equal(t, tt.expectCopilotHeader, actualDetection, 
				"isGitHubCopilotURL should correctly detect GitHub Copilot URLs")
		})
	}
}

func TestOpenAIClientWithGitHubCopilotEnvironment(t *testing.T) {
	client := NewClient()
	
	// Simulate environment variables being set
	client.ApiKey = &plugins.SetupQuestion{
		Setting: &plugins.Setting{Value: "ghs_test_github_oauth_token"},
	}
	client.ApiBaseURL = &plugins.SetupQuestion{
		Setting: &plugins.Setting{Value: "https://api.githubcopilot.com"},
	}
	err := client.configure()
	assert.NoError(t, err, "Configure should work with GitHub Copilot settings")
	assert.NotNil(t, client.ApiClient, "API client should be initialized")
	
	// Verify that the client detects this as a GitHub Copilot endpoint
	assert.True(t, isGitHubCopilotURL("https://api.githubcopilot.com"), 
		"Should detect GitHub Copilot URL correctly")
}
