package reposelect

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewSelectsAllReposByDefault(t *testing.T) {
	m := New("Select repos", []string{"owner/repo1", "owner/repo2"})

	selected := m.Selected()
	if len(selected) != 2 {
		t.Fatalf("expected 2 selected repos, got %d", len(selected))
	}
}

func TestUpdateToggleSelectAllSelectNoneAndConfirm(t *testing.T) {
	m := New("Select repos", []string{"owner/repo1", "owner/repo2"})

	var result Result
	m, result = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	if result.Confirmed {
		t.Fatal("toggle should not confirm")
	}
	if len(m.Selected()) != 1 {
		t.Fatalf("expected one selected repo after toggle, got %d", len(m.Selected()))
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if len(m.Selected()) != 0 {
		t.Fatalf("expected no selected repos after select none, got %d", len(m.Selected()))
	}

	_, result = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if result.Confirmed {
		t.Fatal("empty selection must not confirm")
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	_, result = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !result.Confirmed {
		t.Fatal("expected confirmation with selected repos")
	}
	if len(result.Selected) != 2 {
		t.Fatalf("expected 2 confirmed repos, got %d", len(result.Selected))
	}
}

func TestViewShowsPreflightWarningAndHelpers(t *testing.T) {
	m := New("Watch Status: Select Repositories", []string{"owner/repo"})

	view := m.View()
	for _, want := range []string{
		"Watch Status: Select Repositories",
		"No GitHub calls run until you press Enter",
		"a: select all",
		"n: select none",
		"enter: load selected",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected view to contain %q, got:\n%s", want, view)
		}
	}
}

func TestUpdateEscCancels(t *testing.T) {
	m := New("Select repos", []string{"owner/repo"})

	_, result := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !result.Canceled {
		t.Fatal("expected esc to cancel selection")
	}
}
