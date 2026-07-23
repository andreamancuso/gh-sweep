package storage

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/andreamancuso/gh-sweep/internal/github"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	repo      string
	width     int
	height    int
	loading   bool
	err       error
	statusMsg string

	inventory *github.StorageInventory
	plan      cleanupPlan
	viewMode  viewMode
	sortDesc  bool
	cursor    int
	selected  map[string]bool

	confirming bool
	confirm    string
}

type viewMode int

const (
	viewArtifacts viewMode = iota
	viewCaches
	viewRuns
	viewReleases
	viewPackages
)

type cleanupPlan struct {
	Artifacts []github.StorageArtifact
	Caches    []github.StorageCache
	Runs      []github.StorageWorkflowRun
}

type inventoryLoadedMsg struct {
	inventory *github.StorageInventory
	err       error
}

type cleanupCompleteMsg struct {
	deletedArtifacts int
	deletedCaches    int
	deletedRuns      int
	failures         []string
}

func NewModel(repo string) Model {
	return Model{
		repo:     repo,
		loading:  true,
		sortDesc: true,
		selected: make(map[string]bool),
	}
}

func (m Model) Init() tea.Cmd {
	return m.load
}

func (m Model) load() tea.Msg {
	owner, repoName, err := splitRepo(m.repo)
	if err != nil {
		return inventoryLoadedMsg{err: err}
	}
	client, err := github.NewClient(context.Background())
	if err != nil {
		return inventoryLoadedMsg{err: err}
	}
	inventory, err := client.GetStorageInventory(owner, repoName)
	return inventoryLoadedMsg{inventory: inventory, err: err}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case inventoryLoadedMsg:
		m.loading = false
		m.inventory = msg.inventory
		m.err = msg.err
		m.statusMsg = ""
		m.plan = cleanupPlan{}
		m.cursor = 0
		m.selected = make(map[string]bool)
		if m.selected == nil {
			m.selected = make(map[string]bool)
		}
		return m, nil
	case cleanupCompleteMsg:
		m.confirming = false
		m.confirm = ""
		m.statusMsg = fmt.Sprintf("Deleted %d artifact(s), %d cache(s), %d run(s).",
			msg.deletedArtifacts, msg.deletedCaches, msg.deletedRuns)
		if len(msg.failures) > 0 {
			m.statusMsg += fmt.Sprintf(" %d failure(s); inspect CLI output for details.", len(msg.failures))
		}
		m.loading = true
		return m, m.load
	case tea.KeyMsg:
		if m.confirming {
			return m.handleConfirmKey(msg)
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.loading = true
			m.err = nil
			return m, m.load
		case "d":
			m.plan = m.selectedPlan()
			if m.plan.empty() {
				m.plan = m.recommendedPlan()
				m.statusMsg = "Recommended dry-run preview generated. Press x to execute after typed confirmation."
			} else {
				m.statusMsg = "Selected-item dry-run preview generated. Press x to execute after typed confirmation."
			}
			return m, nil
		case "x":
			if m.plan.empty() {
				m.plan = m.selectedPlan()
			}
			if m.plan.empty() {
				m.plan = m.recommendedPlan()
			}
			if m.plan.empty() {
				m.statusMsg = "Recommended cleanup found nothing to delete."
				return m, nil
			}
			m.confirming = true
			m.confirm = ""
			return m, nil
		case "1":
			m.viewMode = viewArtifacts
			m.cursor = 0
			return m, nil
		case "2":
			m.viewMode = viewCaches
			m.cursor = 0
			return m, nil
		case "3":
			m.viewMode = viewRuns
			m.cursor = 0
			return m, nil
		case "4":
			m.viewMode = viewReleases
			m.cursor = 0
			return m, nil
		case "5":
			m.viewMode = viewPackages
			m.cursor = 0
			return m, nil
		case "t":
			m.sortDesc = !m.sortDesc
			return m, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "j":
			if m.cursor < m.cleanableRowCount()-1 {
				m.cursor++
			}
			return m, nil
		case " ":
			m.toggleSelection()
			return m, nil
		case "n":
			m.selected = make(map[string]bool)
			m.plan = cleanupPlan{}
			m.statusMsg = "Selection cleared."
			return m, nil
		}
	}
	return m, nil
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.confirming = false
		m.confirm = ""
		m.statusMsg = "Cleanup cancelled."
	case "backspace":
		if len(m.confirm) > 0 {
			m.confirm = m.confirm[:len(m.confirm)-1]
		}
	case "enter":
		if m.inventory == nil || m.confirm != m.inventory.Repository {
			m.statusMsg = "Confirmation did not match repository name."
			m.confirming = false
			m.confirm = ""
			return m, nil
		}
		return m, m.executeCleanup
	default:
		if len(msg.String()) == 1 {
			m.confirm += msg.String()
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.loading {
		return "Loading storage inventory...\n"
	}
	if m.err != nil {
		return fmt.Sprintf("Storage error: %v\n\nPress r to retry or q to quit.\n", m.err)
	}
	if m.inventory == nil {
		return "No storage inventory loaded.\n"
	}

	var b strings.Builder
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FFFF"))
	b.WriteString(title.Render("GitHub Storage Cleanup"))
	b.WriteString("\n\n")
	b.WriteString(renderSummary(m.inventory))
	b.WriteString("\n")
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")
	b.WriteString(m.renderActiveView())

	if !m.plan.empty() {
		b.WriteString("\nRecommended cleanup preview:\n")
		b.WriteString(fmt.Sprintf("  Artifacts: %d (%s)\n", len(m.plan.Artifacts), formatBytes(totalArtifactBytes(m.plan.Artifacts))))
		b.WriteString(fmt.Sprintf("  Caches:    %d (%s)\n", len(m.plan.Caches), formatBytes(totalCacheBytes(m.plan.Caches))))
		b.WriteString(fmt.Sprintf("  Runs:      %d\n", len(m.plan.Runs)))
	}

	if m.confirming {
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("Type %s to permanently delete the previewed items:\n> %s\n", m.inventory.Repository, m.confirm))
	} else if m.statusMsg != "" {
		b.WriteString("\n")
		b.WriteString(m.statusMsg)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString("j/k: move | space: select | n: clear | 1-5: view | t: sort | d: dry-run | x: execute | r: refresh | esc: back | q: quit\n")
	return b.String()
}

func (m Model) renderTabs() string {
	labels := []string{"[1] Artifacts", "[2] Caches", "[3] Runs", "[4] Releases", "[5] Packages"}
	var b strings.Builder
	for i, label := range labels {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
		if viewMode(i) == m.viewMode {
			style = style.Bold(true).Foreground(lipgloss.Color("#FFFF00"))
		}
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(style.Render(label))
	}
	if m.sortDesc {
		b.WriteString("  sort: desc")
	} else {
		b.WriteString("  sort: asc")
	}
	return b.String()
}

func (m Model) renderActiveView() string {
	switch m.viewMode {
	case viewCaches:
		return m.renderCaches()
	case viewRuns:
		return m.renderRuns()
	case viewReleases:
		return m.renderReleases()
	case viewPackages:
		return m.renderPackages()
	default:
		return m.renderArtifacts()
	}
}

func (m Model) renderArtifacts() string {
	artifacts := append([]github.StorageArtifact(nil), m.inventory.Artifacts...)
	sort.Slice(artifacts, func(i, j int) bool {
		if m.sortDesc {
			return artifacts[i].SizeBytes > artifacts[j].SizeBytes
		}
		return artifacts[i].SizeBytes < artifacts[j].SizeBytes
	})
	if len(artifacts) == 0 {
		return "Artifacts: none\n"
	}
	var b strings.Builder
	b.WriteString("Artifacts sorted by size:\n")
	for i, artifact := range artifacts {
		if i >= 20 {
			b.WriteString(fmt.Sprintf("... and %d more\n", len(artifacts)-20))
			break
		}
		cursor, mark := m.rowMarkers(i, selectionKey("artifact", artifact.ID))
		b.WriteString(fmt.Sprintf("%s%s %d  %10s  %s  %s\n",
			cursor, mark,
			artifact.ID, formatBytes(artifact.SizeBytes), artifact.CreatedAt.Format("2006-01-02"), artifact.Name))
	}
	return b.String()
}

func (m Model) renderCaches() string {
	caches := append([]github.StorageCache(nil), m.inventory.Caches...)
	sort.Slice(caches, func(i, j int) bool {
		left := caches[i].LastAccessedAt
		right := caches[j].LastAccessedAt
		if left.IsZero() {
			left = caches[i].CreatedAt
		}
		if right.IsZero() {
			right = caches[j].CreatedAt
		}
		if m.sortDesc {
			return left.After(right)
		}
		return left.Before(right)
	})
	if len(caches) == 0 {
		return "Caches: none\n"
	}
	var b strings.Builder
	b.WriteString("Caches sorted by last access:\n")
	for i, cache := range caches {
		if i >= 20 {
			b.WriteString(fmt.Sprintf("... and %d more\n", len(caches)-20))
			break
		}
		accessed := cache.LastAccessedAt
		if accessed.IsZero() {
			accessed = cache.CreatedAt
		}
		cursor, mark := m.rowMarkers(i, selectionKey("cache", cache.ID))
		b.WriteString(fmt.Sprintf("%s%s %d  %10s  %s  %s  %s\n",
			cursor, mark,
			cache.ID, formatBytes(cache.SizeBytes), accessed.Format("2006-01-02"), cache.Ref, cache.Key))
	}
	return b.String()
}

func (m Model) renderRuns() string {
	runs := append([]github.StorageWorkflowRun(nil), m.inventory.Runs...)
	sort.Slice(runs, func(i, j int) bool {
		if m.sortDesc {
			return runs[i].CreatedAt.After(runs[j].CreatedAt)
		}
		return runs[i].CreatedAt.Before(runs[j].CreatedAt)
	})
	if len(runs) == 0 {
		return "Workflow runs: none\n"
	}
	var b strings.Builder
	b.WriteString("Workflow runs sorted by created date:\n")
	for i, run := range runs {
		if i >= 20 {
			b.WriteString(fmt.Sprintf("... and %d more\n", len(runs)-20))
			break
		}
		title := run.DisplayTitle
		if title == "" {
			title = run.Name
		}
		cursor, mark := m.rowMarkers(i, selectionKey("run", run.ID))
		b.WriteString(fmt.Sprintf("%s%s %d  %-10s  %-12s  %s  %s\n",
			cursor, mark,
			run.ID, run.Conclusion, run.Workflow, run.CreatedAt.Format("2006-01-02"), title))
	}
	return b.String()
}

func (m Model) renderReleases() string {
	releases := append([]github.StorageRelease(nil), m.inventory.Releases...)
	sort.Slice(releases, func(i, j int) bool {
		left := releaseAssetTotal(releases[i])
		right := releaseAssetTotal(releases[j])
		if m.sortDesc {
			return left > right
		}
		return left < right
	})
	if len(releases) == 0 {
		return "Releases: none\n"
	}
	var b strings.Builder
	b.WriteString("Releases sorted by total asset size:\n")
	for _, release := range releases {
		b.WriteString(fmt.Sprintf("  %s  %10s  %d asset(s)\n",
			release.TagName, formatBytes(releaseAssetTotal(release)), len(release.Assets)))
		for _, asset := range release.Assets {
			b.WriteString(fmt.Sprintf("    - %10s  %s\n", formatBytes(asset.SizeBytes), asset.Name))
		}
	}
	return b.String()
}

func (m Model) renderPackages() string {
	if m.inventory.PackageError != "" {
		return "Packages: " + m.inventory.PackageError + "\n"
	}
	packages := append([]github.StoragePackage(nil), m.inventory.Packages...)
	sort.Slice(packages, func(i, j int) bool {
		if m.sortDesc {
			return packages[i].UpdatedAt.After(packages[j].UpdatedAt)
		}
		return packages[i].UpdatedAt.Before(packages[j].UpdatedAt)
	})
	if len(packages) == 0 {
		return "Packages: none\n"
	}
	var b strings.Builder
	b.WriteString("Packages sorted by updated date:\n")
	for _, pkg := range packages {
		b.WriteString(fmt.Sprintf("  %s/%s  %-8s  versions:%d  %s\n",
			pkg.PackageType, pkg.Name, pkg.Visibility, pkg.VersionCount, pkg.UpdatedAt.Format("2006-01-02")))
	}
	return b.String()
}

func (m Model) recommendedPlan() cleanupPlan {
	if m.inventory == nil {
		return cleanupPlan{}
	}
	now := time.Now()
	olderThan := 3 * 24 * time.Hour
	return cleanupPlan{
		Artifacts: github.SelectArtifactsForCleanup(m.inventory.Artifacts, olderThan, now, false),
		Caches:    github.SelectCachesForCleanup(m.inventory.Caches, olderThan, now, false),
		Runs:      github.SelectRunsForCleanup(m.inventory.Runs, github.ParseConclusionSet("failure,cancelled"), 0, now),
	}
}

func (m Model) selectedPlan() cleanupPlan {
	if m.inventory == nil {
		return cleanupPlan{}
	}
	plan := cleanupPlan{}
	for _, artifact := range m.inventory.Artifacts {
		if m.selected[selectionKey("artifact", artifact.ID)] {
			plan.Artifacts = append(plan.Artifacts, artifact)
		}
	}
	for _, cache := range m.inventory.Caches {
		if m.selected[selectionKey("cache", cache.ID)] {
			plan.Caches = append(plan.Caches, cache)
		}
	}
	for _, run := range m.inventory.Runs {
		if m.selected[selectionKey("run", run.ID)] {
			plan.Runs = append(plan.Runs, run)
		}
	}
	return plan
}

func (m Model) cleanableRowCount() int {
	if m.inventory == nil {
		return 0
	}
	switch m.viewMode {
	case viewArtifacts:
		return min(len(m.inventory.Artifacts), 20)
	case viewCaches:
		return min(len(m.inventory.Caches), 20)
	case viewRuns:
		return min(len(m.inventory.Runs), 20)
	default:
		return 0
	}
}

func (m *Model) toggleSelection() {
	if m.inventory == nil || m.cursor < 0 {
		return
	}
	key := ""
	switch m.viewMode {
	case viewArtifacts:
		rows := m.sortedArtifacts()
		if m.cursor < len(rows) {
			key = selectionKey("artifact", rows[m.cursor].ID)
		}
	case viewCaches:
		rows := m.sortedCaches()
		if m.cursor < len(rows) {
			key = selectionKey("cache", rows[m.cursor].ID)
		}
	case viewRuns:
		rows := m.sortedRuns()
		if m.cursor < len(rows) {
			key = selectionKey("run", rows[m.cursor].ID)
		}
	}
	if key == "" {
		return
	}
	if m.selected == nil {
		m.selected = make(map[string]bool)
	}
	if m.selected[key] {
		delete(m.selected, key)
	} else {
		m.selected[key] = true
	}
}

func (m Model) rowMarkers(index int, key string) (string, string) {
	cursor := " "
	if index == m.cursor {
		cursor = ">"
	}
	mark := " "
	if m.selected[key] {
		mark = "*"
	}
	return cursor, mark
}

func (m Model) sortedArtifacts() []github.StorageArtifact {
	artifacts := append([]github.StorageArtifact(nil), m.inventory.Artifacts...)
	sort.Slice(artifacts, func(i, j int) bool {
		if m.sortDesc {
			return artifacts[i].SizeBytes > artifacts[j].SizeBytes
		}
		return artifacts[i].SizeBytes < artifacts[j].SizeBytes
	})
	if len(artifacts) > 20 {
		return artifacts[:20]
	}
	return artifacts
}

func (m Model) sortedCaches() []github.StorageCache {
	caches := append([]github.StorageCache(nil), m.inventory.Caches...)
	sort.Slice(caches, func(i, j int) bool {
		left := caches[i].LastAccessedAt
		right := caches[j].LastAccessedAt
		if left.IsZero() {
			left = caches[i].CreatedAt
		}
		if right.IsZero() {
			right = caches[j].CreatedAt
		}
		if m.sortDesc {
			return left.After(right)
		}
		return left.Before(right)
	})
	if len(caches) > 20 {
		return caches[:20]
	}
	return caches
}

func (m Model) sortedRuns() []github.StorageWorkflowRun {
	runs := append([]github.StorageWorkflowRun(nil), m.inventory.Runs...)
	sort.Slice(runs, func(i, j int) bool {
		if m.sortDesc {
			return runs[i].CreatedAt.After(runs[j].CreatedAt)
		}
		return runs[i].CreatedAt.Before(runs[j].CreatedAt)
	})
	if len(runs) > 20 {
		return runs[:20]
	}
	return runs
}

func (m Model) executeCleanup() tea.Msg {
	owner, repoName, err := splitRepo(m.repo)
	if err != nil {
		return cleanupCompleteMsg{failures: []string{err.Error()}}
	}
	client, err := github.NewClient(context.Background())
	if err != nil {
		return cleanupCompleteMsg{failures: []string{err.Error()}}
	}

	result := cleanupCompleteMsg{}
	for _, artifact := range m.plan.Artifacts {
		if err := client.DeleteStorageArtifact(owner, repoName, artifact.ID); err != nil {
			result.failures = append(result.failures, err.Error())
			continue
		}
		result.deletedArtifacts++
	}
	for _, cache := range m.plan.Caches {
		if err := client.DeleteStorageCache(owner, repoName, cache.ID); err != nil {
			result.failures = append(result.failures, err.Error())
			continue
		}
		result.deletedCaches++
	}
	for _, run := range m.plan.Runs {
		if err := client.DeleteStorageWorkflowRun(owner, repoName, run.ID); err != nil {
			result.failures = append(result.failures, err.Error())
			continue
		}
		result.deletedRuns++
	}
	return result
}

func (p cleanupPlan) empty() bool {
	return len(p.Artifacts) == 0 && len(p.Caches) == 0 && len(p.Runs) == 0
}

func renderSummary(inventory *github.StorageInventory) string {
	summary := github.SummarizeStorage(inventory)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s\n\n", inventory.Repository))
	b.WriteString(fmt.Sprintf("Artifacts      %5d  %10s\n", summary.ArtifactCount, formatBytes(summary.ArtifactBytes)))
	b.WriteString(fmt.Sprintf("Caches         %5d  %10s\n", summary.CacheCount, formatBytes(summary.CacheBytes)))
	b.WriteString(fmt.Sprintf("Runs           %5d  %10s retained logs (%d failed/cancelled)\n", summary.RunCount, "-", summary.FailedCancelledRunCount))
	b.WriteString(fmt.Sprintf("Releases       %5d  %10s in %d assets\n", summary.ReleaseCount, formatBytes(summary.ReleaseAssetBytes), summary.ReleaseAssetCount))
	if inventory.PackageError != "" {
		b.WriteString(fmt.Sprintf("Packages       %5d  %10s (%s)\n", summary.PackageCount, "unknown", inventory.PackageError))
	} else {
		b.WriteString(fmt.Sprintf("Packages       %5d  %10s\n", summary.PackageCount, "unknown"))
	}
	b.WriteString(fmt.Sprintf("Repo Git       %5s  %10s\n", "", formatBytes(summary.RepoGitSizeBytes)))
	return b.String()
}

func splitRepo(repo string) (string, string, error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repo format, expected owner/repo")
	}
	return parts[0], parts[1], nil
}

func selectionKey(kind string, id int64) string {
	return fmt.Sprintf("%s:%d", kind, id)
}

func formatBytes(value int64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	div, exp := int64(unit), 0
	for n := value / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(value)/float64(div), "KMGTPE"[exp])
}

func totalArtifactBytes(artifacts []github.StorageArtifact) int64 {
	var total int64
	for _, artifact := range artifacts {
		total += artifact.SizeBytes
	}
	return total
}

func totalCacheBytes(caches []github.StorageCache) int64 {
	var total int64
	for _, cache := range caches {
		total += cache.SizeBytes
	}
	return total
}

func releaseAssetTotal(release github.StorageRelease) int64 {
	var total int64
	for _, asset := range release.Assets {
		total += asset.SizeBytes
	}
	return total
}
