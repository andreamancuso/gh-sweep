package github

import (
	"net/url"
	"testing"
)

func TestAPIPathEscapesSegments(t *testing.T) {
	got := apiPath("repos", "owner", "repo", "git", "refs", "heads", "feature/foo")
	want := "repos/owner/repo/git/refs/heads/feature%2Ffoo"
	if got != want {
		t.Fatalf("apiPath() = %q, want %q", got, want)
	}
}

func TestAPIPathWithQuery(t *testing.T) {
	values := url.Values{}
	values.Set("state", "all")
	values.Set("head", "owner:feature/foo")

	got := apiPathWithQuery(apiPath("repos", "owner", "repo", "pulls"), values)
	want := "repos/owner/repo/pulls?head=owner%3Afeature%2Ffoo&state=all"
	if got != want {
		t.Fatalf("apiPathWithQuery() = %q, want %q", got, want)
	}
}
