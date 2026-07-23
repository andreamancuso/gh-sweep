package branches

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModelStartsWithSingleRepoSelection(t *testing.T) {
	m := NewModel(
		[]string{"owner/repo1", "owner/repo2"},
		"owner/repo2",
		"main",
	)

	if cmd := m.Init(); cmd != nil {
		t.Fatal("expected no GitHub call before repository confirmation")
	}

	view := m.View()
	for _, want := range []string{
		"Branch Management: Select Repository",
		"owner/repo1",
		"owner/repo2",
		"Choose one",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected repository selection view to contain %q, got:\n%s", want, view)
		}
	}
}

func TestRepoSelectionStartsBranchLoadForChosenRepo(t *testing.T) {
	m := NewModel(
		[]string{"owner/repo1", "owner/repo2"},
		"owner/repo1",
		"main",
	)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("expected repository confirmation to start branch loading")
	}
	if m.selecting {
		t.Fatal("expected repository selection to finish")
	}
	if !m.loading {
		t.Fatal("expected branch loading state")
	}
	if m.repo != "owner/repo2" {
		t.Fatalf("expected selected repository, got %q", m.repo)
	}
	if !strings.Contains(m.View(), "Loading branches for owner/repo2") {
		t.Fatalf("expected selected repository in loading view, got:\n%s", m.View())
	}
}

func TestWithDefaultRepoPrependsMissingDefaultOnce(t *testing.T) {
	repos := withDefaultRepo(
		[]string{"owner/repo1", "owner/repo2", "owner/repo1"},
		"owner/default",
	)

	want := []string{"owner/default", "owner/repo1", "owner/repo2"}
	if len(repos) != len(want) {
		t.Fatalf("expected %v, got %v", want, repos)
	}
	for i := range want {
		if repos[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, repos)
		}
	}
}
