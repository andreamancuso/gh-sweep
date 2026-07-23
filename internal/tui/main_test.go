package tui

import (
	"strings"
	"testing"

	"github.com/andreamancuso/gh-sweep/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewMainModelUsesConfigRepositories(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultOrg = "andreamancuso"
	cfg.Repositories = []string{
		"andreamancuso/bt-browser",
		"andreamancuso/gh-sweep",
	}

	m := NewMainModel("", cfg)

	if m.repo != "andreamancuso/bt-browser" {
		t.Fatalf("expected first configured repo as default, got %q", m.repo)
	}
	if m.baseline != "andreamancuso/bt-browser" {
		t.Fatalf("expected baseline to use default repo, got %q", m.baseline)
	}
	if m.org != "andreamancuso" {
		t.Fatalf("expected default org from config, got %q", m.org)
	}
	if len(m.repos) != 2 {
		t.Fatalf("expected configured repositories, got %d", len(m.repos))
	}
}

func TestNewMainModelRepoFlagOverridesConfigDefault(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Repositories = []string{"andreamancuso/bt-browser"}

	m := NewMainModel("andreamancuso/gh-sweep", cfg)

	if m.repo != "andreamancuso/gh-sweep" {
		t.Fatalf("expected explicit repo to win, got %q", m.repo)
	}
}

func TestMainModelHomeShowsLoadedConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultOrg = "andreamancuso"
	cfg.Repositories = []string{"andreamancuso/bt-browser"}

	m := NewMainModel("", cfg)
	m.ready = true

	view := m.View()
	for _, want := range []string{
		"Default repo: andreamancuso/bt-browser",
		"Configured repos: 1",
		"Default org: andreamancuso",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected view to contain %q, got:\n%s", want, view)
		}
	}
}

func TestMainModelAppliesCurrentSizeWhenOpeningWatchStatus(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Repositories = []string{
		"andreamancuso/repo-01",
		"andreamancuso/repo-02",
		"andreamancuso/repo-03",
		"andreamancuso/repo-04",
		"andreamancuso/repo-05",
		"andreamancuso/repo-06",
		"andreamancuso/repo-07",
		"andreamancuso/repo-08",
	}

	model, _ := NewMainModel("", cfg).Update(tea.WindowSizeMsg{Width: 80, Height: 16})
	m := model.(MainModel)

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("0")})
	m = model.(MainModel)
	if cmd != nil {
		t.Fatal("expected watch status selection screen not to load before confirmation")
	}

	view := m.View()
	if !strings.Contains(view, "Showing repositories 1-5 of 8") {
		t.Fatalf("expected watch selector to use current terminal size, got:\n%s", view)
	}
}
