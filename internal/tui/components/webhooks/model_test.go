package webhooks

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModelStartsWithRepoSelection(t *testing.T) {
	m := NewModel([]string{"owner/repo1", "owner/repo2"})

	if !m.selecting {
		t.Fatal("expected selection mode")
	}
	if m.loading {
		t.Fatal("expected no loading before repo confirmation")
	}
	if cmd := m.Init(); cmd != nil {
		t.Fatal("expected no initial command before repo confirmation")
	}
	if !strings.Contains(m.View(), "Webhooks: Select Repositories") {
		t.Fatalf("expected selector view, got:\n%s", m.View())
	}
}

func TestSelectionConfirmStartsWebhookLoad(t *testing.T) {
	m := NewModel([]string{"owner/repo1", "owner/repo2"})

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("expected load command after selection confirmation")
	}
	if m.selecting {
		t.Fatal("expected selection mode to end")
	}
	if !m.loading {
		t.Fatal("expected loading after confirmation")
	}
}
