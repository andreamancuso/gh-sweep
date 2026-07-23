package watching

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/andreamancuso/gh-sweep/internal/github"
	"github.com/andreamancuso/gh-sweep/internal/tui/components/reposelect"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const watchStatusLoadTimeout = 15 * time.Second

var loadingFrames = []string{"|", "/", "-", "\\"}

type Model struct {
	username      string
	userRepos     []github.RepoBasic
	selector      reposelect.Model
	subscriptions map[string]*github.Subscription
	cursor        int
	width         int
	height        int
	loading       bool
	selecting     bool
	err           error
	viewMode      string
	selected      map[int]bool
	statusMsg     string
	loadingFrame  int
}

func NewModel(configRepos ...[]string) Model {
	repos := []string{}
	if len(configRepos) > 0 {
		repos = configRepos[0]
	}

	return Model{
		subscriptions: make(map[string]*github.Subscription),
		selected:      make(map[int]bool),
		loading:       false,
		selecting:     true,
		viewMode:      "unwatched",
		selector:      reposelect.New("Watch Status: Select Repositories", repos),
	}
}

type DataLoadedMsg struct {
	Username      string
	UserRepos     []github.RepoBasic
	Subscriptions map[string]*github.Subscription
	Warnings      []string
	Err           error
}

type watchResultMsg struct {
	repo string
	err  error
}

type unwatchResultMsg struct {
	repo string
	err  error
}

type loadingTickMsg time.Time

func loadingTick() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(now time.Time) tea.Msg {
		return loadingTickMsg(now)
	})
}

func (m Model) Init() tea.Cmd {
	if m.selecting {
		return nil
	}
	return m.loadData
}

func (m Model) loadData() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), watchStatusLoadTimeout)
	defer cancel()

	client, err := github.NewClient(ctx)
	if err != nil {
		return DataLoadedMsg{Err: fmt.Errorf("failed to create GitHub client: %w", err)}
	}

	username, err := client.GetAuthenticatedUser()
	if err != nil {
		return DataLoadedMsg{Err: fmt.Errorf("failed to get authenticated user: %w", err)}
	}

	repos, err := client.ListUserRepos()
	if err != nil {
		return DataLoadedMsg{Err: fmt.Errorf("failed to list user repos: %w", err)}
	}

	subscriptions, warnings, err := loadSubscriptions(client, repos)
	if err != nil {
		return DataLoadedMsg{Err: err}
	}

	return DataLoadedMsg{
		Username:      username,
		UserRepos:     repos,
		Subscriptions: subscriptions,
		Warnings:      warnings,
		Err:           nil,
	}
}

func loadSelectedData(repos []github.RepoBasic) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), watchStatusLoadTimeout)
		defer cancel()

		client, err := github.NewClient(ctx)
		if err != nil {
			return DataLoadedMsg{Err: fmt.Errorf("failed to create GitHub client: %w", err)}
		}

		subscriptions, warnings, err := loadSubscriptions(client, repos)
		if err != nil {
			return DataLoadedMsg{Err: err}
		}

		return DataLoadedMsg{
			Username:      "",
			UserRepos:     repos,
			Subscriptions: subscriptions,
			Warnings:      warnings,
			Err:           nil,
		}
	}
}

func (m Model) watchRepo(repo github.RepoBasic) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		client, err := github.NewClient(ctx)
		if err != nil {
			return watchResultMsg{repo: repo.FullName, err: err}
		}

		sub, err := client.SetRepoSubscription(repo.Owner, repo.Name, true, false)
		if err != nil {
			return watchResultMsg{repo: repo.FullName, err: err}
		}

		m.subscriptions[repo.FullName] = sub
		return watchResultMsg{repo: repo.FullName, err: nil}
	}
}

func (m Model) unwatchRepo(repo github.RepoBasic) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		client, err := github.NewClient(ctx)
		if err != nil {
			return unwatchResultMsg{repo: repo.FullName, err: err}
		}

		if err := client.DeleteRepoSubscription(repo.Owner, repo.Name); err != nil {
			return unwatchResultMsg{repo: repo.FullName, err: err}
		}

		return unwatchResultMsg{repo: repo.FullName, err: nil}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.selector = m.selector.SetSize(msg.Width, msg.Height)
		return m, nil

	case DataLoadedMsg:
		m.loading = false
		m.selecting = false
		m.username = msg.Username
		m.userRepos = msg.UserRepos
		m.subscriptions = msg.Subscriptions
		m.err = msg.Err
		if len(msg.Warnings) > 0 {
			m.statusMsg = fmt.Sprintf("Loaded watch status for %d/%d repositories. %d failed or timed out.", len(msg.Subscriptions), len(msg.UserRepos), len(msg.Warnings))
		}
		m.cursor = 0
		m.selected = make(map[int]bool)
		return m, nil

	case loadingTickMsg:
		if !m.loading {
			return m, nil
		}
		m.loadingFrame = (m.loadingFrame + 1) % len(loadingFrames)
		return m, loadingTick()

	case watchResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Failed to watch %s: %v", msg.repo, msg.err)
		} else {
			m.statusMsg = fmt.Sprintf("Watching %s", msg.repo)
			if sub, ok := m.subscriptions[msg.repo]; ok {
				sub.Subscribed = true
				sub.Ignored = false
				sub.State = github.WatchStateSubscribed
			}
		}
		return m, nil

	case unwatchResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Failed to unwatch %s: %v", msg.repo, msg.err)
		} else {
			m.statusMsg = fmt.Sprintf("Unwatched %s", msg.repo)
			if sub, ok := m.subscriptions[msg.repo]; ok {
				sub.Subscribed = false
				sub.State = github.WatchStateNotWatching
			}
		}
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
			filtered := m.getFilteredRepos()
			if m.cursor < len(filtered)-1 {
				m.cursor++
			}

		case "1":
			m.viewMode = "unwatched"
			m.cursor = 0
			m.selected = make(map[int]bool)

		case "2":
			m.viewMode = "watched"
			m.cursor = 0
			m.selected = make(map[int]bool)

		case "3":
			m.viewMode = "all"
			m.cursor = 0
			m.selected = make(map[int]bool)

		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]

		case "w":
			return m.handleWatch()

		case "u":
			return m.handleUnwatch()
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
		repos := reposFromConfig(result.Selected)
		m.loading = true
		m.selecting = false
		m.userRepos = repos
		m.statusMsg = ""
		m.loadingFrame = 0
		return m, tea.Batch(loadSelectedData(repos), loadingTick())
	}

	return m, nil
}

func (m Model) getFilteredRepos() []github.RepoBasic {
	var filtered []github.RepoBasic
	for _, repo := range m.userRepos {
		sub := m.subscriptions[repo.FullName]
		switch m.viewMode {
		case "unwatched":
			if sub == nil || sub.State == github.WatchStateNotWatching {
				filtered = append(filtered, repo)
			}
		case "watched":
			if sub != nil && sub.State == github.WatchStateSubscribed {
				filtered = append(filtered, repo)
			}
		case "all":
			filtered = append(filtered, repo)
		}
	}
	return filtered
}

func (m Model) handleWatch() (tea.Model, tea.Cmd) {
	filtered := m.getFilteredRepos()
	var cmds []tea.Cmd

	hasSelection := false
	for idx := range m.selected {
		if m.selected[idx] && idx < len(filtered) {
			hasSelection = true
			cmds = append(cmds, m.watchRepo(filtered[idx]))
		}
	}

	if !hasSelection && m.cursor < len(filtered) {
		cmds = append(cmds, m.watchRepo(filtered[m.cursor]))
	}

	m.selected = make(map[int]bool)
	return m, tea.Batch(cmds...)
}

func (m Model) handleUnwatch() (tea.Model, tea.Cmd) {
	filtered := m.getFilteredRepos()
	var cmds []tea.Cmd

	hasSelection := false
	for idx := range m.selected {
		if m.selected[idx] && idx < len(filtered) {
			hasSelection = true
			cmds = append(cmds, m.unwatchRepo(filtered[idx]))
		}
	}

	if !hasSelection && m.cursor < len(filtered) {
		cmds = append(cmds, m.unwatchRepo(filtered[m.cursor]))
	}

	m.selected = make(map[int]bool)
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.selecting {
		return m.renderRepoSelection()
	}

	if m.loading {
		return fmt.Sprintf(
			"%s Loading watch status for %d selected repositories... (maximum %s)\n\nEsc: back\n",
			loadingFrames[m.loadingFrame],
			len(m.userRepos),
			watchStatusLoadTimeout,
		)
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF"))

	b.WriteString(titleStyle.Render("Watch Status Audit"))
	b.WriteString("\n")
	if m.username != "" {
		b.WriteString(fmt.Sprintf("User: %s\n", m.username))
	}
	b.WriteString(fmt.Sprintf("Repositories: %d\n\n", len(m.userRepos)))

	activeTab := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFF00"))

	inactiveTab := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#777777"))

	if m.viewMode == "unwatched" {
		b.WriteString(activeTab.Render("[1] Unwatched"))
	} else {
		b.WriteString(inactiveTab.Render("[1] Unwatched"))
	}
	b.WriteString("  ")
	if m.viewMode == "watched" {
		b.WriteString(activeTab.Render("[2] Watched"))
	} else {
		b.WriteString(inactiveTab.Render("[2] Watched"))
	}
	b.WriteString("  ")
	if m.viewMode == "all" {
		b.WriteString(activeTab.Render("[3] All"))
	} else {
		b.WriteString(inactiveTab.Render("[3] All"))
	}
	b.WriteString("\n\n")

	filtered := m.getFilteredRepos()

	if len(filtered) == 0 {
		b.WriteString("No repositories in this view.\n")
	} else {
		for i, repo := range filtered {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}

			selectMark := " "
			if m.selected[i] {
				selectMark = "*"
			}

			sub := m.subscriptions[repo.FullName]
			status := "not watching"
			statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
			if sub != nil {
				switch sub.State {
				case github.WatchStateSubscribed:
					status = "watching"
					statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
				case github.WatchStateIgnored:
					status = "ignored"
					statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00"))
				}
			}

			lineStyle := lipgloss.NewStyle()
			if m.cursor == i {
				lineStyle = lineStyle.Bold(true).Foreground(lipgloss.Color("#FFFF00"))
			}

			line := fmt.Sprintf("%s%s %s ", cursor, selectMark, repo.FullName)
			b.WriteString(lineStyle.Render(line))
			b.WriteString(statusStyle.Render(fmt.Sprintf("[%s]", status)))
			b.WriteString("\n")
		}
	}

	if m.statusMsg != "" {
		b.WriteString("\n")
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF"))
		b.WriteString(statusStyle.Render(m.statusMsg))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
	b.WriteString(helpStyle.Render("j/k: navigate | space: select | w: watch | u: unwatch | 1/2/3: view mode | esc: back"))

	return b.String()
}

func (m Model) renderRepoSelection() string {
	return m.selector.View()
}

func reposFromConfig(configRepos []string) []github.RepoBasic {
	repos := make([]github.RepoBasic, 0, len(configRepos))
	for _, repo := range configRepos {
		repo = strings.TrimSpace(repo)
		owner, name, ok := strings.Cut(repo, "/")
		if !ok || owner == "" || name == "" {
			continue
		}

		repos = append(repos, github.RepoBasic{
			Name:     name,
			FullName: repo,
			Owner:    owner,
		})
	}
	return repos
}

func loadSubscriptions(client *github.Client, repos []github.RepoBasic) (map[string]*github.Subscription, []string, error) {
	const concurrency = 8

	subscriptions := make(map[string]*github.Subscription)
	var warnings []string
	var fatalErr error
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	for _, repo := range repos {
		repo := repo
		wg.Add(1)
		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			sub, err := client.GetRepoSubscription(repo.Owner, repo.Name)
			if err != nil {
				mu.Lock()
				warnings = append(warnings, fmt.Sprintf("%s: %v", repo.FullName, err))
				if errors.Is(err, github.ErrNotificationsScopeRequired) {
					fatalErr = err
				}
				mu.Unlock()
				return
			}

			mu.Lock()
			subscriptions[repo.FullName] = sub
			mu.Unlock()
		}()
	}

	wg.Wait()
	return subscriptions, warnings, fatalErr
}
