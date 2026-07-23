package github

import (
	"errors"
	"net/http"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

func TestIsMissingNotificationsScope(t *testing.T) {
	err := &api.HTTPError{
		StatusCode: http.StatusNotFound,
		Headers: http.Header{
			"X-Accepted-Oauth-Scopes": []string{"notifications"},
			"X-Oauth-Scopes":          []string{"repo, workflow"},
		},
	}

	if !isMissingNotificationsScope(err) {
		t.Fatal("expected missing notifications scope to be detected")
	}
}

func TestIsMissingNotificationsScopeAcceptsGrantedScope(t *testing.T) {
	err := &api.HTTPError{
		StatusCode: http.StatusNotFound,
		Headers: http.Header{
			"X-Accepted-Oauth-Scopes": []string{"notifications"},
			"X-Oauth-Scopes":          []string{"notifications, repo"},
		},
	}

	if isMissingNotificationsScope(err) {
		t.Fatal("did not expect granted notifications scope to be reported missing")
	}
}

func TestIsHTTPStatusUsesTypedGitHubError(t *testing.T) {
	err := &api.HTTPError{StatusCode: http.StatusNotFound}

	if !isHTTPStatus(err, http.StatusNotFound) {
		t.Fatal("expected typed HTTP status to match")
	}
	if isHTTPStatus(errors.New("repository 404-example exists"), http.StatusNotFound) {
		t.Fatal("did not expect unrelated text to match an HTTP status")
	}
}
