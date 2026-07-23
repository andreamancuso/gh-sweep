package github

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

type WatchState string

const (
	WatchStateSubscribed  WatchState = "subscribed"
	WatchStateIgnored     WatchState = "ignored"
	WatchStateNotWatching WatchState = ""
)

var ErrNotificationsScopeRequired = errors.New(
	"GitHub authentication is missing the notifications scope; run `gh auth refresh -h github.com -s notifications` and try again",
)

type Subscription struct {
	Repository string
	Subscribed bool
	Ignored    bool
	Reason     string
	CreatedAt  time.Time
	State      WatchState
}

type RepoBasic struct {
	Name     string
	FullName string
	Owner    string
	Private  bool
}

type userResponse struct {
	Login string `json:"login"`
}

type repoListResponse struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
	Private bool `json:"private"`
}

type subscriptionResponse struct {
	Subscribed bool      `json:"subscribed"`
	Ignored    bool      `json:"ignored"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
}

func (c *Client) GetAuthenticatedUser() (string, error) {
	var response userResponse
	if err := c.Get("user", &response); err != nil {
		return "", fmt.Errorf("failed to get authenticated user: %w", err)
	}
	return response.Login, nil
}

func (c *Client) ListUserRepos() ([]RepoBasic, error) {
	var allRepos []RepoBasic
	page := 1
	perPage := 100

	for {
		var response []repoListResponse
		path := apiPathWithQuery("user/repos", query(map[string]string{
			"affiliation": "owner",
			"per_page":    fmt.Sprintf("%d", perPage),
			"page":        fmt.Sprintf("%d", page),
		}))

		if err := c.Get(path, &response); err != nil {
			return nil, fmt.Errorf("failed to list user repos: %w", err)
		}

		if len(response) == 0 {
			break
		}

		for _, repo := range response {
			allRepos = append(allRepos, RepoBasic{
				Name:     repo.Name,
				FullName: repo.FullName,
				Owner:    repo.Owner.Login,
				Private:  repo.Private,
			})
		}

		if len(response) < perPage {
			break
		}
		page++
	}

	return allRepos, nil
}

func (c *Client) GetRepoSubscription(owner, repo string) (*Subscription, error) {
	var response subscriptionResponse
	path := apiPath("repos", owner, repo, "subscription")

	if err := c.Get(path, &response); err != nil {
		if isMissingNotificationsScope(err) {
			return nil, ErrNotificationsScopeRequired
		}
		if isHTTPStatus(err, http.StatusNotFound) {
			return &Subscription{
				Repository: repoFullName(owner, repo),
				Subscribed: false,
				Ignored:    false,
				State:      WatchStateNotWatching,
			}, nil
		}
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	state := WatchStateSubscribed
	if response.Ignored {
		state = WatchStateIgnored
	} else if !response.Subscribed {
		state = WatchStateNotWatching
	}

	return &Subscription{
		Repository: repoFullName(owner, repo),
		Subscribed: response.Subscribed,
		Ignored:    response.Ignored,
		Reason:     response.Reason,
		CreatedAt:  response.CreatedAt,
		State:      state,
	}, nil
}

func isMissingNotificationsScope(err error) bool {
	var httpErr *api.HTTPError
	if !errors.As(err, &httpErr) {
		return false
	}

	accepted := splitScopes(httpErr.Headers.Get("X-Accepted-OAuth-Scopes"))
	granted := splitScopes(httpErr.Headers.Get("X-OAuth-Scopes"))
	return accepted["notifications"] && len(granted) > 0 && !granted["notifications"]
}

func isHTTPStatus(err error, status int) bool {
	var httpErr *api.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == status
	}

	return strings.Contains(err.Error(), fmt.Sprintf("HTTP %d", status))
}

func splitScopes(value string) map[string]bool {
	scopes := make(map[string]bool)
	for scope := range strings.SplitSeq(value, ",") {
		scope = strings.TrimSpace(scope)
		if scope != "" {
			scopes[scope] = true
		}
	}
	return scopes
}

func (c *Client) SetRepoSubscription(owner, repo string, subscribed, ignored bool) (*Subscription, error) {
	path := apiPath("repos", owner, repo, "subscription")
	body := map[string]bool{
		"subscribed": subscribed,
		"ignored":    ignored,
	}

	var response subscriptionResponse
	if err := c.Put(path, body, &response); err != nil {
		return nil, fmt.Errorf("failed to set subscription: %w", err)
	}

	state := WatchStateSubscribed
	if response.Ignored {
		state = WatchStateIgnored
	} else if !response.Subscribed {
		state = WatchStateNotWatching
	}

	return &Subscription{
		Repository: repoFullName(owner, repo),
		Subscribed: response.Subscribed,
		Ignored:    response.Ignored,
		Reason:     response.Reason,
		CreatedAt:  response.CreatedAt,
		State:      state,
	}, nil
}

func (c *Client) DeleteRepoSubscription(owner, repo string) error {
	path := apiPath("repos", owner, repo, "subscription")
	if err := c.Delete(path, nil); err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}
	return nil
}
