package protection

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModelStartsWithRepoSelection(t *testing.T) {
	m := NewModel([]string{"owner/baseline", "owner/repo2"}, "owner/baseline")

	if !m.selecting {
		t.Fatal("expected selection mode")
	}
	if m.loading {
		t.Fatal("expected no loading before repo confirmation")
	}
	if cmd := m.Init(); cmd != nil {
		t.Fatal("expected no initial command before repo confirmation")
	}
	if !strings.Contains(m.View(), "Branch Protection: Select Repositories") {
		t.Fatalf("expected selector view, got:\n%s", m.View())
	}
}

func TestSelectionConfirmIncludesBaseline(t *testing.T) {
	m := NewModel([]string{"owner/baseline", "owner/repo2"}, "owner/baseline")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	m = updated.(Model)
	if cmd != nil {
		t.Fatal("expected toggle to avoid starting a command")
	}

	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("expected load command after selection confirmation")
	}
	if m.repos[0] != "owner/baseline" {
		t.Fatalf("expected baseline to be included first, got %v", m.repos)
	}
}
