package tui

import (
	"fmt"

	"github.com/andreamancuso/gh-sweep/internal/config"
	"github.com/andreamancuso/gh-sweep/internal/orphans"
	"github.com/andreamancuso/gh-sweep/internal/tui/components/analytics"
	"github.com/andreamancuso/gh-sweep/internal/tui/components/branches"
	"github.com/andreamancuso/gh-sweep/internal/tui/components/collaborators"
	"github.com/andreamancuso/gh-sweep/internal/tui/components/comments"
	"github.com/andreamancuso/gh-sweep/internal/tui/components/ghaperf"
	orphanstui "github.com/andreamancuso/gh-sweep/internal/tui/components/orphans"
	"github.com/andreamancuso/gh-sweep/internal/tui/components/protection"
	"github.com/andreamancuso/gh-sweep/internal/tui/components/releases"
	"github.com/andreamancuso/gh-sweep/internal/tui/components/secrets"
	"github.com/andreamancuso/gh-sweep/internal/tui/components/settings"
	"github.com/andreamancuso/gh-sweep/internal/tui/components/storage"
	"github.com/andreamancuso/gh-sweep/internal/tui/components/watching"
	"github.com/andreamancuso/gh-sweep/internal/tui/components/webhooks"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ViewMode represents different TUI views
type ViewMode int

const (
	ViewHome ViewMode = iota
	ViewBranches
	ViewProtection
	ViewComments
	ViewAnalytics
	ViewGHAPerf
	ViewSettings
	ViewWatching
	ViewWebhooks
	ViewCollaborators
	ViewSecrets
	ViewReleases
	ViewOrphans
	ViewStorage
)

// MainModel represents the main TUI application state with navigation
type MainModel struct {
	width  int
	height int
	ready  bool
	mode   ViewMode

	// Sub-models for each view
	analyticsModel     analytics.Model
	branchesModel      branches.Model
	collaboratorsModel collaborators.Model
	commentsModel      comments.Model
	ghaPerfModel       ghaperf.Model
	orphansModel       orphanstui.Model
	protectionModel    protection.Model
	releasesModel      releases.Model
	secretsModel       secrets.Model
	settingsModel      settings.Model
	storageModel       storage.Model
	watchingModel      watching.Model
	webhooksModel      webhooks.Model

	// Configuration
	repo     string
	repos    []string
	baseline string
	org      string
}

// NewMainModel creates a new main TUI model
func NewMainModel(repo string, cfg *config.Config) MainModel {
	repos := []string{}
	org := ""
	if cfg != nil {
		repos = cfg.Repositories
		org = cfg.DefaultOrg
	}
	if repo == "" && len(repos) > 0 {
		repo = repos[0]
	}
	baseline := repo
	if baseline == "" && len(repos) > 0 {
		baseline = repos[0]
	}

	return MainModel{
		ready:    false,
		mode:     ViewHome,
		repo:     repo,
		repos:    repos,
		baseline: baseline,
		org:      org,
	}
}

// Init initializes the model
func (m MainModel) Init() tea.Cmd {
	return nil
}

func (m MainModel) applyCurrentSize(model tea.Model) tea.Model {
	if !m.ready {
		return model
	}

	resized, _ := model.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	return resized
}

// Update handles messages and updates the model
func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Forward to sub-models
		var newModel tea.Model
		newModel, _ = m.branchesModel.Update(msg)
		m.branchesModel = newModel.(branches.Model)
		newModel, _ = m.protectionModel.Update(msg)
		m.protectionModel = newModel.(protection.Model)
		newModel, _ = m.commentsModel.Update(msg)
		m.commentsModel = newModel.(comments.Model)
		newModel, _ = m.analyticsModel.Update(msg)
		m.analyticsModel = newModel.(analytics.Model)
		newModel, _ = m.ghaPerfModel.Update(msg)
		m.ghaPerfModel = newModel.(ghaperf.Model)
		newModel, _ = m.settingsModel.Update(msg)
		m.settingsModel = newModel.(settings.Model)
		newModel, _ = m.storageModel.Update(msg)
		m.storageModel = newModel.(storage.Model)
		newModel, _ = m.webhooksModel.Update(msg)
		m.webhooksModel = newModel.(webhooks.Model)
		newModel, _ = m.collaboratorsModel.Update(msg)
		m.collaboratorsModel = newModel.(collaborators.Model)
		newModel, _ = m.secretsModel.Update(msg)
		m.secretsModel = newModel.(secrets.Model)
		newModel, _ = m.releasesModel.Update(msg)
		m.releasesModel = newModel.(releases.Model)
		newModel, _ = m.watchingModel.Update(msg)
		m.watchingModel = newModel.(watching.Model)
		newModel, _ = m.orphansModel.Update(msg)
		m.orphansModel = newModel.(orphanstui.Model)

		return m, nil

	case tea.KeyMsg:
		// Handle navigation in home view
		if m.mode == ViewHome {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit

			case "0":
				m.mode = ViewWatching
				m.watchingModel = watching.NewModel(m.repos)
				m.watchingModel = m.applyCurrentSize(m.watchingModel).(watching.Model)
				return m, m.watchingModel.Init()

			case "1":
				m.mode = ViewBranches
				if m.repo != "" {
					m.branchesModel = branches.NewModel(m.repo, "main")
					m.branchesModel = m.applyCurrentSize(m.branchesModel).(branches.Model)
					return m, m.branchesModel.Init()
				}

			case "2":
				m.mode = ViewProtection
				if len(m.repos) > 0 {
					m.protectionModel = protection.NewModel(m.repos, m.baseline)
					m.protectionModel = m.applyCurrentSize(m.protectionModel).(protection.Model)
					return m, m.protectionModel.Init()
				}

			case "3":
				m.mode = ViewComments
				if m.repo != "" {
					m.commentsModel = comments.NewModel(m.repo)
					m.commentsModel = m.applyCurrentSize(m.commentsModel).(comments.Model)
					return m, m.commentsModel.Init()
				}

			case "4":
				m.mode = ViewAnalytics
				if m.repo != "" {
					m.analyticsModel = analytics.NewModel(m.repo)
					m.analyticsModel = m.applyCurrentSize(m.analyticsModel).(analytics.Model)
					return m, m.analyticsModel.Init()
				}

			case "p":
				m.mode = ViewGHAPerf
				if m.repo != "" {
					m.ghaPerfModel = ghaperf.NewModel(m.repo)
					m.ghaPerfModel = m.applyCurrentSize(m.ghaPerfModel).(ghaperf.Model)
					return m, m.ghaPerfModel.Init()
				}

			case "5":
				m.mode = ViewSettings
				if len(m.repos) > 0 {
					m.settingsModel = settings.NewModel(m.repos, m.baseline)
					m.settingsModel = m.applyCurrentSize(m.settingsModel).(settings.Model)
					return m, m.settingsModel.Init()
				}

			case "6":
				m.mode = ViewWebhooks
				if len(m.repos) > 0 {
					m.webhooksModel = webhooks.NewModel(m.repos)
					m.webhooksModel = m.applyCurrentSize(m.webhooksModel).(webhooks.Model)
					return m, m.webhooksModel.Init()
				}

			case "7":
				m.mode = ViewCollaborators
				if len(m.repos) > 0 {
					m.collaboratorsModel = collaborators.NewModel(m.repos)
					m.collaboratorsModel = m.applyCurrentSize(m.collaboratorsModel).(collaborators.Model)
					return m, m.collaboratorsModel.Init()
				}

			case "8":
				m.mode = ViewSecrets
				if m.org != "" && len(m.repos) > 0 {
					m.secretsModel = secrets.NewModel(m.org, m.repos)
					m.secretsModel = m.applyCurrentSize(m.secretsModel).(secrets.Model)
					return m, m.secretsModel.Init()
				}

			case "9":
				m.mode = ViewReleases
				if len(m.repos) > 0 {
					m.releasesModel = releases.NewModel(m.repos)
					m.releasesModel = m.applyCurrentSize(m.releasesModel).(releases.Model)
					return m, m.releasesModel.Init()
				}

			case "o":
				m.mode = ViewOrphans
				namespace := m.org
				if namespace == "" {
					namespace = ""
				}
				m.orphansModel = orphanstui.NewModel(namespace, orphans.DefaultScanOptions())
				m.orphansModel = m.applyCurrentSize(m.orphansModel).(orphanstui.Model)
				return m, m.orphansModel.Init()

			case "s":
				m.mode = ViewStorage
				if m.repo != "" {
					m.storageModel = storage.NewModel(m.repo)
					m.storageModel = m.applyCurrentSize(m.storageModel).(storage.Model)
					return m, m.storageModel.Init()
				}
			}
		} else {
			// Handle back navigation
			if msg.String() == "esc" {
				m.mode = ViewHome
				return m, nil
			}

			// Forward to active sub-model
			var cmd tea.Cmd
			switch m.mode {
			case ViewBranches:
				var newModel tea.Model
				newModel, cmd = m.branchesModel.Update(msg)
				m.branchesModel = newModel.(branches.Model)

			case ViewProtection:
				var newModel tea.Model
				newModel, cmd = m.protectionModel.Update(msg)
				m.protectionModel = newModel.(protection.Model)

			case ViewComments:
				var newModel tea.Model
				newModel, cmd = m.commentsModel.Update(msg)
				m.commentsModel = newModel.(comments.Model)

			case ViewAnalytics:
				var newModel tea.Model
				newModel, cmd = m.analyticsModel.Update(msg)
				m.analyticsModel = newModel.(analytics.Model)

			case ViewGHAPerf:
				var newModel tea.Model
				newModel, cmd = m.ghaPerfModel.Update(msg)
				m.ghaPerfModel = newModel.(ghaperf.Model)

			case ViewSettings:
				var newModel tea.Model
				newModel, cmd = m.settingsModel.Update(msg)
				m.settingsModel = newModel.(settings.Model)

			case ViewWebhooks:
				var newModel tea.Model
				newModel, cmd = m.webhooksModel.Update(msg)
				m.webhooksModel = newModel.(webhooks.Model)

			case ViewCollaborators:
				var newModel tea.Model
				newModel, cmd = m.collaboratorsModel.Update(msg)
				m.collaboratorsModel = newModel.(collaborators.Model)

			case ViewSecrets:
				var newModel tea.Model
				newModel, cmd = m.secretsModel.Update(msg)
				m.secretsModel = newModel.(secrets.Model)

			case ViewReleases:
				var newModel tea.Model
				newModel, cmd = m.releasesModel.Update(msg)
				m.releasesModel = newModel.(releases.Model)

			case ViewWatching:
				var newModel tea.Model
				newModel, cmd = m.watchingModel.Update(msg)
				m.watchingModel = newModel.(watching.Model)

			case ViewOrphans:
				var newModel tea.Model
				newModel, cmd = m.orphansModel.Update(msg)
				m.orphansModel = newModel.(orphanstui.Model)

			case ViewStorage:
				var newModel tea.Model
				newModel, cmd = m.storageModel.Update(msg)
				m.storageModel = newModel.(storage.Model)
			}

			return m, cmd
		}
	}

	return m, nil
}

// View renders the model
func (m MainModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Render active view
	switch m.mode {
	case ViewBranches:
		return m.branchesModel.View()
	case ViewProtection:
		return m.protectionModel.View()
	case ViewComments:
		return m.commentsModel.View()
	case ViewAnalytics:
		return m.analyticsModel.View()
	case ViewGHAPerf:
		return m.ghaPerfModel.View()
	case ViewSettings:
		return m.settingsModel.View()
	case ViewWebhooks:
		return m.webhooksModel.View()
	case ViewCollaborators:
		return m.collaboratorsModel.View()
	case ViewSecrets:
		return m.secretsModel.View()
	case ViewReleases:
		return m.releasesModel.View()
	case ViewWatching:
		return m.watchingModel.View()
	case ViewOrphans:
		return m.orphansModel.View()
	case ViewStorage:
		return m.storageModel.View()
	default:
		return m.renderHome()
	}
}

func (m MainModel) renderHome() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF")).
		Padding(1, 0)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFF00")).
		Padding(0, 0)

	menuItemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 2)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#777777"))

	content := titleStyle.Render("🧹 gh-sweep") + "\n"
	content += titleStyle.Render("GitHub Repository Management TUI") + "\n\n"

	// Namespace Audit
	content += sectionStyle.Render("Namespace Audit") + "\n"
	content += menuItemStyle.Render("[0] 👁️  Watch Status")
	content += " - Audit and manage repo watching\n"
	content += menuItemStyle.Render("[o] 🌿 Orphan Branches")
	content += " - Detect and clean up orphaned branches\n\n"
	content += menuItemStyle.Render("[s] 🧹 Storage Cleanup")
	content += " - Inspect and clean up Actions storage\n\n"

	// Phase 1: Core Management
	content += sectionStyle.Render("Phase 1: Core Management") + "\n"
	content += menuItemStyle.Render("[1] 🌳 Branch Management")
	content += " - Interactive branch operations\n"
	content += menuItemStyle.Render("[2] 🛡️  Branch Protection")
	content += " - Compare and sync protection rules\n"
	content += menuItemStyle.Render("[3] 💬 PR Comments")
	content += " - Review unresolved comments\n"
	content += menuItemStyle.Render("[4] 📊 Analytics")
	content += " - CI/CD and repository statistics\n"
	content += menuItemStyle.Render("[p] ⏱️  GHA Performance")
	content += " - Workflow timing analysis\n\n"

	// Phase 2: Analytics & Settings
	content += sectionStyle.Render("Phase 2: Analytics & Settings") + "\n"
	content += menuItemStyle.Render("[5] ⚙️  Settings Comparison")
	content += " - Cross-repo settings diff\n"
	content += menuItemStyle.Render("[6] 🔔 Webhooks")
	content += " - Webhook health monitoring\n\n"

	// Phase 3: Access & Releases
	content += sectionStyle.Render("Phase 3: Access & Releases") + "\n"
	content += menuItemStyle.Render("[7] 👥 Collaborators")
	content += " - Manage repository access\n"
	content += menuItemStyle.Render("[8] 🔐 Secrets Audit")
	content += " - Review secrets usage (read-only)\n"
	content += menuItemStyle.Render("[9] 📦 Releases")
	content += " - Release version overview\n\n"

	if m.repo == "" && len(m.repos) == 0 {
		content += helpStyle.Render("💡 Configure with --repo flag or .gh-sweep.yaml\n\n")
	} else {
		if m.repo != "" {
			content += helpStyle.Render("Default repo: "+m.repo) + "\n"
		}
		if len(m.repos) > 0 {
			content += helpStyle.Render("Configured repos: "+fmt.Sprintf("%d", len(m.repos))) + "\n"
		}
		if m.org != "" {
			content += helpStyle.Render("Default org: "+m.org) + "\n"
		}
		content += "\n"
	}

	content += helpStyle.Render("Press 0-9/o/p/s to select a view | q to quit")

	return content
}
