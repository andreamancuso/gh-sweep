package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
	storagetui "github.com/KyleKing/gh-sweep/internal/tui/components/storage"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var storageCmd = &cobra.Command{
	Use:   "storage",
	Short: "Inspect and clean up GitHub Actions storage",
	Long: `Inspect GitHub repository storage and clean up Actions artifacts,
caches, and workflow run logs from your local terminal.

Destructive operations are preview-first. Use --dry-run to force preview-only
behavior, or --yes to execute without an interactive confirmation prompt.`,
	Run: runStorage,
}

func init() {
	rootCmd.AddCommand(storageCmd)

	storageCmd.Flags().String("repo", "", "Repository (owner/repo)")
	storageCmd.Flags().Bool("list", false, "Print storage inventory and exit")
	storageCmd.Flags().Bool("recommended", false, "Delete artifacts older than 3 days, old caches, and failed/cancelled runs")
	storageCmd.Flags().Bool("delete-artifacts", false, "Delete artifacts older than --older-than")
	storageCmd.Flags().Bool("delete-all-artifacts", false, "Delete all artifacts")
	storageCmd.Flags().Bool("delete-caches", false, "Delete all caches or caches older than --older-than")
	storageCmd.Flags().Bool("delete-runs", false, "Delete workflow runs matching --conclusion and --older-than")
	storageCmd.Flags().String("conclusion", "", "Comma-separated workflow run conclusions to delete (default for --delete-runs: failure,cancelled)")
	storageCmd.Flags().String("older-than", "", "Age threshold such as 3d, 12h, or 30m")
	storageCmd.Flags().Bool("inspect-releases", false, "Print release asset details")
	storageCmd.Flags().Bool("inspect-packages", false, "Print package details")
	storageCmd.Flags().Bool("dry-run", false, "Preview cleanup without deleting")
	storageCmd.Flags().Bool("yes", false, "Execute destructive cleanup without prompting except for successful workflow-run deletion")
}

type storageCleanupPlan struct {
	Artifacts []github.StorageArtifact
	Caches    []github.StorageCache
	Runs      []github.StorageWorkflowRun
}

func runStorage(cmd *cobra.Command, args []string) {
	repo, _ := cmd.Flags().GetString("repo")
	if repo == "" {
		repo, _ = rootCmd.Flags().GetString("repo")
	}
	owner, repoName, err := parseRepo(repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	listOnly, _ := cmd.Flags().GetBool("list")
	recommended, _ := cmd.Flags().GetBool("recommended")
	deleteArtifacts, _ := cmd.Flags().GetBool("delete-artifacts")
	deleteAllArtifacts, _ := cmd.Flags().GetBool("delete-all-artifacts")
	deleteCaches, _ := cmd.Flags().GetBool("delete-caches")
	deleteRuns, _ := cmd.Flags().GetBool("delete-runs")
	inspectReleases, _ := cmd.Flags().GetBool("inspect-releases")
	inspectPackages, _ := cmd.Flags().GetBool("inspect-packages")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	yes, _ := cmd.Flags().GetBool("yes")
	olderThanText, _ := cmd.Flags().GetString("older-than")
	conclusionsText, _ := cmd.Flags().GetString("conclusion")

	destructive := recommended || deleteArtifacts || deleteAllArtifacts || deleteCaches || deleteRuns
	if !destructive && !listOnly && !inspectReleases && !inspectPackages {
		if err := launchStorageTUI(repo); err != nil {
			fmt.Fprintf(os.Stderr, "Error running storage TUI: %v\n", err)
			os.Exit(1)
		}
		return
	}

	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create GitHub client: %v\n", err)
		os.Exit(1)
	}

	inventory, err := client.GetStorageInventory(owner, repoName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get storage inventory: %v\n", err)
		os.Exit(1)
	}

	if !destructive || listOnly {
		printStorageInventory(os.Stdout, inventory, inspectReleases, inspectPackages)
		if !destructive {
			return
		}
	}

	olderThan, err := parseAgeDuration(olderThanText)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if recommended {
		deleteArtifacts = true
		deleteCaches = true
		deleteRuns = true
		if olderThan == 0 {
			olderThan = 3 * 24 * time.Hour
		}
		if conclusionsText == "" {
			conclusionsText = "failure,cancelled"
		}
	}

	if deleteRuns && conclusionsText == "" {
		conclusionsText = "failure,cancelled"
	}
	runOlderThan := olderThan
	if recommended {
		runOlderThan = 0
	}

	plan := buildStorageCleanupPlan(inventory, storageCleanupOptions{
		DeleteArtifacts:    deleteArtifacts,
		DeleteAllArtifacts: deleteAllArtifacts,
		DeleteCaches:       deleteCaches,
		DeleteRuns:         deleteRuns,
		OlderThan:          olderThan,
		RunOlderThan:       runOlderThan,
		Conclusions:        github.ParseConclusionSet(conclusionsText),
		Now:                time.Now(),
	})

	printStorageCleanupPlan(os.Stdout, inventory.Repository, plan)
	if dryRun || plan.empty() {
		return
	}

	if (!yes || planHasSuccessfulRuns(plan)) && !confirmStorageCleanup(inventory.Repository, plan, os.Stdin, os.Stdout) {
		fmt.Fprintln(os.Stdout, "Cleanup cancelled.")
		return
	}

	result := executeStorageCleanup(client, owner, repoName, plan)
	printStorageCleanupResult(os.Stdout, result)

	refreshed, err := client.GetStorageInventory(owner, repoName)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Warning: cleanup completed but refresh failed: %v\n", err)
		return
	}
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "After cleanup:")
	printStorageSummary(os.Stdout, refreshed)
}

type storageCleanupOptions struct {
	DeleteArtifacts    bool
	DeleteAllArtifacts bool
	DeleteCaches       bool
	DeleteRuns         bool
	OlderThan          time.Duration
	RunOlderThan       time.Duration
	Conclusions        map[string]bool
	Now                time.Time
}

func buildStorageCleanupPlan(inventory *github.StorageInventory, opts storageCleanupOptions) storageCleanupPlan {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	plan := storageCleanupPlan{}
	if opts.DeleteArtifacts || opts.DeleteAllArtifacts {
		plan.Artifacts = github.SelectArtifactsForCleanup(inventory.Artifacts, opts.OlderThan, opts.Now, opts.DeleteAllArtifacts)
	}
	if opts.DeleteCaches {
		plan.Caches = github.SelectCachesForCleanup(inventory.Caches, opts.OlderThan, opts.Now, opts.OlderThan == 0)
	}
	if opts.DeleteRuns {
		plan.Runs = github.SelectRunsForCleanup(inventory.Runs, opts.Conclusions, opts.RunOlderThan, opts.Now)
	}
	return plan
}

func (p storageCleanupPlan) empty() bool {
	return len(p.Artifacts) == 0 && len(p.Caches) == 0 && len(p.Runs) == 0
}

func planHasSuccessfulRuns(plan storageCleanupPlan) bool {
	for _, run := range plan.Runs {
		if run.Conclusion == "success" {
			return true
		}
	}
	return false
}

func printStorageInventory(w io.Writer, inventory *github.StorageInventory, inspectReleases, inspectPackages bool) {
	printStorageSummary(w, inventory)

	sort.Slice(inventory.Artifacts, func(i, j int) bool {
		return inventory.Artifacts[i].SizeBytes > inventory.Artifacts[j].SizeBytes
	})
	if len(inventory.Artifacts) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Largest artifacts:")
		for i, artifact := range inventory.Artifacts {
			if i >= 10 {
				break
			}
			fmt.Fprintf(w, "  %d  %s  %s  created %s\n",
				artifact.ID, formatBytes(artifact.SizeBytes), artifact.Name, artifact.CreatedAt.Format("2006-01-02"))
		}
	}

	if inspectReleases {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Releases:")
		for _, release := range inventory.Releases {
			var total int64
			for _, asset := range release.Assets {
				total += asset.SizeBytes
			}
			fmt.Fprintf(w, "  %s  %s  %d assets\n", release.TagName, formatBytes(total), len(release.Assets))
			for _, asset := range release.Assets {
				fmt.Fprintf(w, "    - %s  %s\n", asset.Name, formatBytes(asset.SizeBytes))
			}
		}
	}

	if inspectPackages {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Packages:")
		if inventory.PackageError != "" {
			fmt.Fprintf(w, "  Warning: %s\n", inventory.PackageError)
		}
		if len(inventory.Packages) == 0 {
			fmt.Fprintln(w, "  none")
		}
		for _, pkg := range inventory.Packages {
			fmt.Fprintf(w, "  %s/%s  %s  versions:%d\n", pkg.PackageType, pkg.Name, pkg.Visibility, pkg.VersionCount)
		}
	}
}

func printStorageSummary(w io.Writer, inventory *github.StorageInventory) {
	summary := github.SummarizeStorage(inventory)
	fmt.Fprintf(w, "Storage inventory for %s\n\n", inventory.Repository)
	fmt.Fprintf(w, "Artifacts      %5d  %10s\n", summary.ArtifactCount, formatBytes(summary.ArtifactBytes))
	fmt.Fprintf(w, "Caches         %5d  %10s\n", summary.CacheCount, formatBytes(summary.CacheBytes))
	fmt.Fprintf(w, "Runs           %5d  %10s retained logs (%d failed/cancelled)\n", summary.RunCount, "-", summary.FailedCancelledRunCount)
	fmt.Fprintf(w, "Releases       %5d  %10s in %d assets\n", summary.ReleaseCount, formatBytes(summary.ReleaseAssetBytes), summary.ReleaseAssetCount)
	if inventory.PackageError != "" {
		fmt.Fprintf(w, "Packages       %5d  %10s (%s)\n", summary.PackageCount, "unknown", inventory.PackageError)
	} else {
		fmt.Fprintf(w, "Packages       %5d  %10s\n", summary.PackageCount, "unknown")
	}
	fmt.Fprintf(w, "Repo Git       %5s  %10s\n", "", formatBytes(summary.RepoGitSizeBytes))
}

func printStorageCleanupPlan(w io.Writer, repo string, plan storageCleanupPlan) {
	fmt.Fprintf(w, "Cleanup preview for %s\n\n", repo)
	fmt.Fprintf(w, "Artifacts to delete:      %d (%s)\n", len(plan.Artifacts), formatBytes(totalArtifactBytes(plan.Artifacts)))
	fmt.Fprintf(w, "Caches to delete:         %d (%s)\n", len(plan.Caches), formatBytes(totalCacheBytes(plan.Caches)))
	fmt.Fprintf(w, "Workflow runs to delete:  %d\n", len(plan.Runs))
	if plan.empty() {
		fmt.Fprintln(w, "\nNothing selected for deletion.")
		return
	}
	fmt.Fprintln(w, "\nUse --dry-run to keep this preview-only, --yes to execute without prompting, or type the repository name when prompted.")
}

func confirmStorageCleanup(repo string, plan storageCleanupPlan, input io.Reader, output io.Writer) bool {
	fmt.Fprintf(output, "\nThis will permanently delete %d artifact(s), %d cache(s), and %d workflow run(s) from %s.\n",
		len(plan.Artifacts), len(plan.Caches), len(plan.Runs), repo)
	fmt.Fprintf(output, "To confirm, type: %s\n> ", repo)
	scanner := bufio.NewScanner(input)
	if !scanner.Scan() {
		return false
	}
	return scanner.Text() == repo
}

type storageCleanupResult struct {
	DeletedArtifacts int
	DeletedCaches    int
	DeletedRuns      int
	Failures         []string
}

func executeStorageCleanup(client *github.Client, owner, repo string, plan storageCleanupPlan) storageCleanupResult {
	result := storageCleanupResult{}
	for _, artifact := range plan.Artifacts {
		if err := client.DeleteStorageArtifact(owner, repo, artifact.ID); err != nil {
			result.Failures = append(result.Failures, err.Error())
			continue
		}
		result.DeletedArtifacts++
	}
	for _, cache := range plan.Caches {
		if err := client.DeleteStorageCache(owner, repo, cache.ID); err != nil {
			result.Failures = append(result.Failures, err.Error())
			continue
		}
		result.DeletedCaches++
	}
	for _, run := range plan.Runs {
		if err := client.DeleteStorageWorkflowRun(owner, repo, run.ID); err != nil {
			result.Failures = append(result.Failures, err.Error())
			continue
		}
		result.DeletedRuns++
	}
	return result
}

func printStorageCleanupResult(w io.Writer, result storageCleanupResult) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Cleanup result:")
	fmt.Fprintf(w, "  Deleted artifacts:     %d\n", result.DeletedArtifacts)
	fmt.Fprintf(w, "  Deleted caches:        %d\n", result.DeletedCaches)
	fmt.Fprintf(w, "  Deleted workflow runs: %d\n", result.DeletedRuns)
	if len(result.Failures) > 0 {
		fmt.Fprintln(w, "  Failures:")
		for _, failure := range result.Failures {
			fmt.Fprintf(w, "    - %s\n", failure)
		}
	}
}

func parseRepo(repo string) (string, string, error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repo must be in format owner/repo")
	}
	return parts[0], parts[1], nil
}

func parseAgeDuration(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	unit := value[len(value)-1]
	number := value[:len(value)-1]
	amount, err := strconv.Atoi(number)
	if err != nil || amount < 0 {
		return 0, fmt.Errorf("older-than must be a positive duration like 3d, 12h, or 30m")
	}
	switch unit {
	case 'd':
		return time.Duration(amount) * 24 * time.Hour, nil
	case 'h':
		return time.Duration(amount) * time.Hour, nil
	case 'm':
		return time.Duration(amount) * time.Minute, nil
	default:
		return 0, fmt.Errorf("older-than must use d, h, or m units")
	}
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

func launchStorageTUI(repo string) error {
	m := storagetui.NewModel(repo)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
