package protection

import (
	"context"
	"fmt"
	"strings"

	"github.com/andreamancuso/gh-sweep/internal/github"
	"github.com/andreamancuso/gh-sweep/internal/tui/components/reposelect"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the protection rules TUI state
type Model struct {
	repos     []string
	selector  reposelect.Model
	rules     map[string]*github.ProtectionRule
	baseline  string
	diffs     map[string][]string
	cursor    int
	width     int
	height    int
	loading   bool
	selecting bool
	err       error
}

// NewModel creates a new protection rules model
func NewModel(repos []string, baseline string) Model {
	return Model{
		repos:     repos,
		selector:  reposelect.New("Branch Protection: Select Repositories", repos),
		baseline:  baseline,
		rules:     make(map[string]*github.ProtectionRule),
		diffs:     make(map[string][]string),
		loading:   false,
		selecting: true,
	}
}

type rulesLoadedMsg struct {
	rules map[string]*github.ProtectionRule
	diffs map[string][]string
	err   error
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	if m.selecting {
		return nil
	}
	return m.loadRules
}

func (m Model) loadRules() tea.Msg {
	// Create GitHub client
	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		return rulesLoadedMsg{
			rules: make(map[string]*github.ProtectionRule),
			diffs: make(map[string][]string),
			err:   fmt.Errorf("failed to create GitHub client: %w", err),
		}
	}

	// Load protection rules for each repo
	rules := make(map[string]*github.ProtectionRule)
	for _, repoStr := range m.repos {
		parts := strings.Split(repoStr, "/")
		if len(parts) != 2 {
			continue
		}
		owner, repo := parts[0], parts[1]

		// Use default branch (main) for now
		branch := "main"
		rule, err := client.GetBranchProtection(owner, repo, branch)
		if err != nil {
			// Skip repos without protection or on error
			continue
		}

		rules[repoStr] = rule
	}

	// Compare rules if baseline is specified
	diffs := make(map[string][]string)
	if m.baseline != "" {
		baselineRule := rules[m.baseline]
		if baselineRule != nil {
			rulesSlice := make([]*github.ProtectionRule, 0, len(rules))
			for _, rule := range rules {
				rulesSlice = append(rulesSlice, rule)
			}
			diffs = github.CompareProtectionRules(rulesSlice)
		}
	}

	return rulesLoadedMsg{
		rules: rules,
		diffs: diffs,
		err:   nil,
	}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.selector = m.selector.SetSize(msg.Width, msg.Height)
		return m, nil

	case rulesLoadedMsg:
		m.loading = false
		m.selecting = false
		m.rules = msg.rules
		m.diffs = msg.diffs
		m.err = msg.err
		m.cursor = 0
		return m, nil

	case tea.KeyMsg:
		if m.selecting {
			return m.updateSelection(msg)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.repos)-1 {
				m.cursor++
			}
		}
	}

	return m, nil
}

func (m Model) updateSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var result reposelect.Result
	m.selector, result = m.selector.Update(msg)
	if result.Quit {
		return m, tea.Quit
	}
	if result.Confirmed {
		m.repos = includeRepo(result.Selected, m.baseline)
		m.loading = true
		m.selecting = false
		return m, m.loadRules
	}
	return m, nil
}

func includeRepo(repos []string, required string) []string {
	if required == "" {
		return repos
	}
	for _, repo := range repos {
		if repo == required {
			return repos
		}
	}
	return append([]string{required}, repos...)
}

// View renders the model
func (m Model) View() string {
	if m.selecting {
		return m.selector.View()
	}

	if m.loading {
		return "Loading protection rules...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF"))

	b.WriteString(titleStyle.Render("🛡️  Branch Protection Rules"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Repositories: %d\n\n", len(m.repos)))

	if m.baseline != "" {
		b.WriteString(fmt.Sprintf("Baseline: %s\n\n", m.baseline))
	}

	// Repository list with rules
	for i, repo := range m.repos {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		rule := m.rules[repo]
		if rule == nil {
			b.WriteString(fmt.Sprintf("%s %s: No protection\n", cursor, repo))
			continue
		}

		statusStyle := lipgloss.NewStyle()
		if m.cursor == i {
			statusStyle = statusStyle.Bold(true).Foreground(lipgloss.Color("#FFFF00"))
		}

		line := fmt.Sprintf("%s %s:\n", cursor, repo)
		line += fmt.Sprintf("   Reviews: %d | Code Owners: %v | Admins: %v\n",
			rule.RequiredReviews,
			rule.RequireCodeOwnerReviews,
			rule.EnforceAdmins,
		)
		line += fmt.Sprintf("   Status Checks: %s\n",
			strings.Join(rule.RequireStatusChecks, ", "))

		b.WriteString(statusStyle.Render(line))
		b.WriteString("\n")
	}

	// Differences
	if len(m.diffs) > 0 {
		b.WriteString("\n⚠️  Differences from baseline:\n\n")
		for field, differences := range m.diffs {
			b.WriteString(fmt.Sprintf("%s:\n", field))
			for _, diff := range differences {
				b.WriteString(fmt.Sprintf("  - %s\n", diff))
			}
		}
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
	b.WriteString(helpStyle.Render("↑/↓: navigate | q: quit"))

	return b.String()
}
