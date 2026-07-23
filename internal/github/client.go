package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/auth"
)

const defaultRequestTimeout = 20 * time.Second
const defaultAuthTimeout = 5 * time.Second

// Client wraps the GitHub API client
type Client struct {
	httpClient *http.Client
	apiClient  *api.RESTClient
	ctx        context.Context
}

// NewClient creates a new GitHub API client
// It will use gh CLI authentication if available, or fall back to GITHUB_TOKEN env var
func NewClient(ctx context.Context) (*Client, error) {
	token, err := resolveAuthToken(ctx, "github.com")
	if err != nil {
		return nil, err
	}

	opts := api.ClientOptions{
		AuthToken: token,
		Host:      "github.com",
		Timeout:   defaultRequestTimeout,
	}

	// Create REST client (will use gh CLI auth or GITHUB_TOKEN)
	restClient, err := api.NewRESTClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Create HTTP client
	httpClient, err := api.NewHTTPClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &Client{
		httpClient: httpClient,
		apiClient:  restClient,
		ctx:        ctx,
	}, nil
}

func resolveAuthToken(ctx context.Context, host string) (string, error) {
	if token, _ := auth.TokenFromEnvOrConfig(host); token != "" {
		return token, nil
	}

	ghPath := os.Getenv("GH_PATH")
	if ghPath == "" {
		var err error
		ghPath, err = exec.LookPath("gh")
		if err != nil {
			return "", fmt.Errorf("authentication token not found; run gh auth login or set GH_TOKEN")
		}
	}

	authCtx, cancel := context.WithTimeout(ctx, defaultAuthTimeout)
	defer cancel()

	cmd := exec.CommandContext(authCtx, ghPath, "auth", "token", "--secure-storage", "--hostname", host) // #nosec G204 -- ghPath is resolved by LookPath/GH_PATH; arguments are fixed.
	output, err := cmd.Output()
	if authCtx.Err() != nil {
		return "", fmt.Errorf("timed out retrieving GitHub auth token from gh CLI after %s; run gh auth status or set GH_TOKEN", defaultAuthTimeout)
	}
	if err != nil {
		return "", fmt.Errorf("failed to retrieve GitHub auth token from gh CLI: %w", err)
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", fmt.Errorf("GitHub auth token was empty; run gh auth status or set GH_TOKEN")
	}

	return token, nil
}

// NewClientWithToken creates a new GitHub API client with an explicit token
func NewClientWithToken(ctx context.Context, token string) (*Client, error) {
	opts := api.ClientOptions{
		AuthToken: token,
		Host:      "github.com",
		Timeout:   defaultRequestTimeout,
	}

	restClient, err := api.NewRESTClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	httpClient, err := api.NewHTTPClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &Client{
		httpClient: httpClient,
		apiClient:  restClient,
		ctx:        ctx,
	}, nil
}

// Get performs a GET request to the GitHub API
func (c *Client) Get(path string, response interface{}) error {
	return c.apiClient.DoWithContext(c.ctx, http.MethodGet, path, nil, response)
}

// Post performs a POST request to the GitHub API
func (c *Client) Post(path string, body interface{}, response interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}
	return c.apiClient.DoWithContext(c.ctx, http.MethodPost, path, bytes.NewReader(jsonBody), response)
}

// Patch performs a PATCH request to the GitHub API
func (c *Client) Patch(path string, body interface{}, response interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}
	return c.apiClient.DoWithContext(c.ctx, http.MethodPatch, path, bytes.NewReader(jsonBody), response)
}

// Put performs a PUT request to the GitHub API
func (c *Client) Put(path string, body interface{}, response interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}
	return c.apiClient.DoWithContext(c.ctx, http.MethodPut, path, bytes.NewReader(jsonBody), response)
}

// Delete performs a DELETE request to the GitHub API
func (c *Client) Delete(path string, response interface{}) error {
	return c.apiClient.DoWithContext(c.ctx, http.MethodDelete, path, nil, response)
}

// Context returns the client's context
func (c *Client) Context() context.Context {
	return c.ctx
}
