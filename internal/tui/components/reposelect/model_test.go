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

func TestViewWindowsLongRepoListToTerminalHeight(t *testing.T) {
	repos := []string{
		"owner/repo-01",
		"owner/repo-02",
		"owner/repo-03",
		"owner/repo-04",
		"owner/repo-05",
		"owner/repo-06",
		"owner/repo-07",
		"owner/repo-08",
		"owner/repo-09",
		"owner/repo-10",
		"owner/repo-11",
		"owner/repo-12",
	}
	m := New("Select repos", repos).SetSize(80, 16)

	view := m.View()
	if strings.Contains(view, "owner/repo-12") {
		t.Fatalf("expected short viewport to hide lower repos, got:\n%s", view)
	}
	if !strings.Contains(view, "repositories below") {
		t.Fatalf("expected below overflow indicator, got:\n%s", view)
	}

	for range 11 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	view = m.View()
	if !strings.Contains(view, "repositories above") {
		t.Fatalf("expected above overflow indicator, got:\n%s", view)
	}
	if !strings.Contains(view, "owner/repo-12") {
		t.Fatalf("expected cursor window to include last repo, got:\n%s", view)
	}
}

func TestViewWindowsLongRepoListWithoutTerminalSize(t *testing.T) {
	repos := []string{
		"owner/repo-01",
		"owner/repo-02",
		"owner/repo-03",
		"owner/repo-04",
		"owner/repo-05",
		"owner/repo-06",
		"owner/repo-07",
		"owner/repo-08",
		"owner/repo-09",
		"owner/repo-10",
		"owner/repo-11",
		"owner/repo-12",
		"owner/repo-13",
		"owner/repo-14",
	}
	m := New("Select repos", repos)

	view := m.View()
	if strings.Contains(view, "owner/repo-14") {
		t.Fatalf("expected fallback viewport to hide lower repos, got:\n%s", view)
	}
	if !strings.Contains(view, "Showing repositories 1-12 of 14") {
		t.Fatalf("expected fallback viewport status, got:\n%s", view)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	view = m.View()
	if !strings.Contains(view, "owner/repo-14") {
		t.Fatalf("expected page-down to show lower repos, got:\n%s", view)
	}
}

func TestSingleSelectionChoosesOneRepository(t *testing.T) {
	m := New(
		"Choose repo",
		[]string{"owner/repo1", "owner/repo2", "owner/repo3"},
		WithSingleSelection("owner/repo2"),
	)

	if selected := m.Selected(); len(selected) != 1 || selected[0] != "owner/repo2" {
		t.Fatalf("expected configured default repo, got %v", selected)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	_, result := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !result.Confirmed {
		t.Fatal("expected single repository confirmation")
	}
	if len(result.Selected) != 1 || result.Selected[0] != "owner/repo3" {
		t.Fatalf("expected cursor repository, got %v", result.Selected)
	}
}

func TestSingleSelectionViewUsesSingleRepoHelp(t *testing.T) {
	m := New(
		"Choose repo",
		[]string{"owner/repo1", "owner/repo2"},
		WithSingleSelection("owner/repo1"),
	)

	view := m.View()
	for _, want := range []string{"Choose one", "enter: choose repository"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected single-selection view to contain %q, got:\n%s", want, view)
		}
	}
	for _, unwanted := range []string{"a: select all", "n: select none", "space: toggle"} {
		if strings.Contains(view, unwanted) {
			t.Fatalf("did not expect single-selection view to contain %q, got:\n%s", unwanted, view)
		}
	}
}
