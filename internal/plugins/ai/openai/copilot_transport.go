package openai

import (
	"net/http"
	"strings"
)
type CopilotTransport struct {
	base http.RoundTripper
}

// creates a new transport that adds GitHub Copilot headers
func NewCopilotTransport(base http.RoundTripper) *CopilotTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &CopilotTransport{base: base}
}

// implements http.RoundTripper interface and adds GitHub-specific headers
func (t *CopilotTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq := req.Clone(req.Context())
	// Add the required GitHub API version header
	newReq.Header.Set("X-GitHub-Api-Version", "2023-05-01")
	return t.base.RoundTrip(newReq)
}

// checks if the given URL is a GitHub Copilot endpoint
func isGitHubCopilotURL(url string) bool {
	return strings.Contains(strings.ToLower(url), "api.githubcopilot.com")
}