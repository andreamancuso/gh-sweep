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
	width    int
	height   int
	status   string
	single   bool
	initial  string
}

type Option func(*Model)

const defaultVisibleRepoCount = 12

func WithSubtitle(subtitle string) Option {
	return func(m *Model) {
		m.subtitle = subtitle
	}
}

func WithSingleSelection(initial string) Option {
	return func(m *Model) {
		m.single = true
		m.initial = initial
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
	if m.single {
		m.cursor = indexOfRepo(m.repos, m.initial)
		m.selected = make(map[int]bool)
	}

	return m
}

func (m Model) SetSize(width, height int) Model {
	m.width = width
	m.height = height
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

	case "pgup", "pageup":
		m.cursor -= m.visibleRepoCount()
		if m.cursor < 0 {
			m.cursor = 0
		}

	case "pgdown", "pagedown":
		m.cursor += m.visibleRepoCount()
		if m.cursor >= len(m.repos) {
			m.cursor = len(m.repos) - 1
		}

	case "home":
		m.cursor = 0

	case "end":
		if len(m.repos) > 0 {
			m.cursor = len(m.repos) - 1
		}

	case " ":
		if !m.single && m.cursor < len(m.repos) {
			m.selected[m.cursor] = !m.selected[m.cursor]
		}

	case "a":
		if m.single {
			break
		}
		for i := range m.repos {
			m.selected[i] = true
		}
		m.status = "Selected all repositories."

	case "n":
		if m.single {
			break
		}
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
	if m.single {
		if m.cursor >= 0 && m.cursor < len(m.repos) {
			return []string{m.repos[m.cursor]}
		}
		return nil
	}

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

	if m.single {
		b.WriteString(fmt.Sprintf("Configured repositories: %d | Choose one\n", len(m.repos)))
	} else {
		b.WriteString(fmt.Sprintf("Configured repositories: %d | Selected: %d\n", len(m.repos), len(m.Selected())))
	}
	b.WriteString("No GitHub calls run until you press Enter.\n\n")

	if len(m.repos) == 0 {
		b.WriteString("No configured repositories. Add repositories to .gh-sweep.yaml.\n")
	} else {
		start, end := m.visibleRange()
		b.WriteString(fmt.Sprintf("Showing repositories %d-%d of %d\n", start+1, end, len(m.repos)))
		if start > 0 {
			b.WriteString(fmt.Sprintf("  ... %d repositories above\n", start))
		}

		for i := start; i < end; i++ {
			repo := m.repos[i]
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}

			lineStyle := lipgloss.NewStyle()
			if m.cursor == i {
				lineStyle = lineStyle.Bold(true).Foreground(lipgloss.Color("#FFFF00"))
			}

			if m.single {
				b.WriteString(lineStyle.Render(fmt.Sprintf("%s %s", cursor, repo)))
			} else {
				selectMark := " "
				if m.selected[i] {
					selectMark = "x"
				}
				b.WriteString(lineStyle.Render(fmt.Sprintf("%s [%s] %s", cursor, selectMark, repo)))
			}
			b.WriteString("\n")
		}

		if end < len(m.repos) {
			b.WriteString(fmt.Sprintf("  ... %d repositories below\n", len(m.repos)-end))
		}
	}

	if m.status != "" {
		b.WriteString("\n")
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF"))
		b.WriteString(statusStyle.Render(m.status))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.single {
		b.WriteString(helpStyle.Render("j/k: navigate | pgup/pgdown: page | enter: choose repository | esc: back | q: quit"))
	} else {
		b.WriteString(helpStyle.Render("j/k: navigate | pgup/pgdown: page | space: toggle | a: select all | n: select none | enter: load selected | esc: back | q: quit"))
	}

	return b.String()
}

func indexOfRepo(repos []string, target string) int {
	for i, repo := range repos {
		if repo == target {
			return i
		}
	}
	return 0
}

func (m Model) visibleRange() (int, int) {
	total := len(m.repos)
	if total == 0 {
		return 0, 0
	}

	visible := m.visibleRepoCount()
	if visible >= total {
		return 0, total
	}

	start := m.cursor - visible/2
	if start < 0 {
		start = 0
	}

	end := start + visible
	if end > total {
		end = total
		start = end - visible
	}

	return start, end
}

func (m Model) visibleRepoCount() int {
	if m.height <= 0 {
		return defaultVisibleRepoCount
	}

	const reservedRows = 11
	visible := m.height - reservedRows
	if visible < 5 {
		return 5
	}
	return visible
}
