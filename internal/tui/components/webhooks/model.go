package webhooks

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/andreamancuso/gh-sweep/internal/github"
	"github.com/andreamancuso/gh-sweep/internal/tui/components/reposelect"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the webhook management TUI state
type Model struct {
	repos     []string
	selector  reposelect.Model
	webhooks  map[string][]github.Webhook             // repo -> webhooks
	health    map[string]map[int]github.WebhookHealth // repo -> webhook ID -> health
	cursor    int
	width     int
	height    int
	loading   bool
	selecting bool
	err       error
}

// NewModel creates a new webhook management model
func NewModel(repos []string) Model {
	return Model{
		repos:     repos,
		selector:  reposelect.New("Webhooks: Select Repositories", repos),
		webhooks:  make(map[string][]github.Webhook),
		health:    make(map[string]map[int]github.WebhookHealth),
		loading:   false,
		selecting: true,
	}
}

type webhooksLoadedMsg struct {
	webhooks map[string][]github.Webhook
	health   map[string]map[int]github.WebhookHealth
	err      error
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	if m.selecting {
		return nil
	}
	return m.loadWebhooks
}

func (m Model) loadWebhooks() tea.Msg {
	// Create GitHub client
	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		return webhooksLoadedMsg{
			webhooks: make(map[string][]github.Webhook),
			health:   make(map[string]map[int]github.WebhookHealth),
			err:      fmt.Errorf("failed to create GitHub client: %w", err),
		}
	}

	webhooks, health := loadWebhookData(client, m.repos)

	return webhooksLoadedMsg{
		webhooks: webhooks,
		health:   health,
		err:      nil,
	}
}

func loadWebhookData(client *github.Client, repos []string) (map[string][]github.Webhook, map[string]map[int]github.WebhookHealth) {
	const concurrency = 4

	webhooks := make(map[string][]github.Webhook)
	health := make(map[string]map[int]github.WebhookHealth)
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	for _, repoStr := range repos {
		parts := strings.Split(repoStr, "/")
		if len(parts) != 2 {
			continue
		}
		owner, repo := parts[0], parts[1]

		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			repoWebhooks, err := client.ListWebhooks(owner, repo)
			if err != nil {
				return
			}

			repoHealth := make(map[int]github.WebhookHealth)
			for _, webhook := range repoWebhooks {
				deliveries, err := client.ListWebhookDeliveries(owner, repo, webhook.ID)
				if err != nil {
					continue
				}
				repoHealth[webhook.ID] = github.AnalyzeWebhookHealth(deliveries)
			}

			mu.Lock()
			webhooks[repoStr] = repoWebhooks
			health[repoStr] = repoHealth
			mu.Unlock()
		}()
	}

	wg.Wait()
	return webhooks, health
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.selector = m.selector.SetSize(msg.Width, msg.Height)
		return m, nil

	case webhooksLoadedMsg:
		m.loading = false
		m.selecting = false
		m.webhooks = msg.webhooks
		m.health = msg.health
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
		m.repos = result.Selected
		m.loading = true
		m.selecting = false
		return m, m.loadWebhooks
	}
	return m, nil
}

// View renders the model
func (m Model) View() string {
	if m.selecting {
		return m.selector.View()
	}

	if m.loading {
		return "Loading webhooks...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF"))

	b.WriteString(titleStyle.Render("🔔 Webhooks"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Repositories: %d\n\n", len(m.repos)))

	// Webhook list by repository
	if len(m.webhooks) == 0 {
		b.WriteString("No webhooks found.\n")
	} else {
		for i, repo := range m.repos {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}

			repoStyle := lipgloss.NewStyle()
			if m.cursor == i {
				repoStyle = repoStyle.Bold(true).Foreground(lipgloss.Color("#FFFF00"))
			}

			webhooks := m.webhooks[repo]
			line := fmt.Sprintf("%s %s (%d webhooks):\n", cursor, repo, len(webhooks))

			// Show first few webhooks
			for j, webhook := range webhooks {
				if j >= 3 {
					line += fmt.Sprintf("   ... and %d more\n", len(webhooks)-3)
					break
				}

				line += fmt.Sprintf("   ID: %d | %s\n", webhook.ID, webhook.URL)
				line += fmt.Sprintf("   Events: %s\n", strings.Join(webhook.Events, ", "))

				// Add health metrics if available
				if repoHealth, ok := m.health[repo]; ok {
					if health, ok := repoHealth[webhook.ID]; ok {
						statusColor := "#00FF00" // green for healthy
						if health.SuccessRate < 80 {
							statusColor = "#FF0000" // red for unhealthy
						} else if health.SuccessRate < 95 {
							statusColor = "#FFFF00" // yellow for warning
						}

						healthStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor))
						healthLine := fmt.Sprintf("   Health: %.1f%% success | Avg: %dms | Total: %d\n",
							health.SuccessRate,
							health.AvgDuration,
							health.TotalDeliveries)
						line += healthStyle.Render(healthLine)
					}
				}
			}

			b.WriteString(repoStyle.Render(line))
			b.WriteString("\n")
		}
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
	b.WriteString(helpStyle.Render("↑/↓: navigate | q: quit"))

	return b.String()
}
