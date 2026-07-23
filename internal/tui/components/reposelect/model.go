package reposelect

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Result struct {
	Confirmed bool
	Canceled  bool
	Quit      bool
	Selected  []string
}

type Model struct {
	title    string
	subtitle string
	repos    []string
	selected map[int]bool
	cursor   int
	status   string
}

type Option func(*Model)

func WithSubtitle(subtitle string) Option {
	return func(m *Model) {
		m.subtitle = subtitle
	}
}

func New(title string, repos []string, opts ...Option) Model {
	selected := make(map[int]bool, len(repos))
	for i := range repos {
		selected[i] = true
	}

	m := Model{
		title:    title,
		repos:    append([]string(nil), repos...),
		selected: selected,
	}
	for _, opt := range opts {
		opt(&m)
	}

	return m
}

func (m Model) Update(msg tea.KeyMsg) (Model, Result) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, Result{Quit: true}

	case "esc":
		return m, Result{Canceled: true}

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.repos)-1 {
			m.cursor++
		}

	case " ":
		if m.cursor < len(m.repos) {
			m.selected[m.cursor] = !m.selected[m.cursor]
		}

	case "a":
		for i := range m.repos {
			m.selected[i] = true
		}
		m.status = "Selected all repositories."

	case "n":
		m.selected = make(map[int]bool)
		m.status = "Cleared repository selection."

	case "enter":
		selected := m.Selected()
		if len(selected) == 0 {
			m.status = "Select at least one repository before loading."
			return m, Result{}
		}

		return m, Result{Confirmed: true, Selected: selected}
	}

	return m, Result{}
}

func (m Model) Selected() []string {
	repos := make([]string, 0, len(m.repos))
	for i, repo := range m.repos {
		if m.selected[i] {
			repos = append(repos, repo)
		}
	}
	return repos
}

func (m Model) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))

	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n\n")

	if m.subtitle != "" {
		b.WriteString(m.subtitle)
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("Configured repositories: %d | Selected: %d\n", len(m.repos), len(m.Selected())))
	b.WriteString("No GitHub calls run until you press Enter.\n\n")

	if len(m.repos) == 0 {
		b.WriteString("No configured repositories. Add repositories to .gh-sweep.yaml.\n")
	} else {
		for i, repo := range m.repos {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}

			selectMark := " "
			if m.selected[i] {
				selectMark = "x"
			}

			lineStyle := lipgloss.NewStyle()
			if m.cursor == i {
				lineStyle = lineStyle.Bold(true).Foreground(lipgloss.Color("#FFFF00"))
			}

			b.WriteString(lineStyle.Render(fmt.Sprintf("%s [%s] %s", cursor, selectMark, repo)))
			b.WriteString("\n")
		}
	}

	if m.status != "" {
		b.WriteString("\n")
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF"))
		b.WriteString(statusStyle.Render(m.status))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k: navigate | space: toggle | a: select all | n: select none | enter: load selected | esc: back | q: quit"))

	return b.String()
}
