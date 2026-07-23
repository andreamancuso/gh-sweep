package watching

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModelStoresConfiguredRepos(t *testing.T) {
	m := NewModel([]string{"andreamancuso/bt-browser", "andreamancuso/gh-sweep"})

	if !m.selecting {
		t.Fatal("expected configured repos to start in selection mode")
	}
	if m.loading {
		t.Fatal("expected configured repos not to load before confirmation")
	}
	if len(m.selector.Selected()) != 2 {
		t.Fatalf("expected all configured repos selected by default, got %d", len(m.selector.Selected()))
	}
	if cmd := m.Init(); cmd != nil {
		t.Fatal("expected no initial GitHub load command while selecting configured repos")
	}
}

func TestReposFromConfig(t *testing.T) {
	repos := reposFromConfig([]string{
		"andreamancuso/bt-browser",
		"invalid",
		"/missing-owner",
		"missing-name/",
		"andreamancuso/gh-sweep",
	})

	if len(repos) != 2 {
		t.Fatalf("expected 2 valid repos, got %d", len(repos))
	}

	if repos[0].Owner != "andreamancuso" || repos[0].Name != "bt-browser" || repos[0].FullName != "andreamancuso/bt-browser" {
		t.Fatalf("unexpected first repo: %+v", repos[0])
	}
}

func TestSelectionViewSupportsSelectNoneGuard(t *testing.T) {
	m := NewModel([]string{"andreamancuso/bt-browser"})

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = updated.(Model)
	if cmd != nil {
		t.Fatal("expected select-none to avoid starting a command")
	}
	if len(m.selector.Selected()) != 0 {
		t.Fatalf("expected no selected repos, got %d", len(m.selector.Selected()))
	}

	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd != nil {
		t.Fatal("expected enter with no selection to avoid starting a command")
	}
	if !strings.Contains(m.View(), "Select at least one repository") {
		t.Fatalf("expected selection warning, got:\n%s", m.View())
	}
}

func TestSelectionViewShowsBulkHelpers(t *testing.T) {
	m := NewModel([]string{"andreamancuso/bt-browser"})

	view := m.View()
	for _, want := range []string{
		"Watch Status: Select Repositories",
		"a: select all",
		"n: select none",
		"enter: load selected",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected view to contain %q, got:\n%s", want, view)
		}
	}
}
