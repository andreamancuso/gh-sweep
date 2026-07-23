package storage

import (
	"strings"
	"testing"
	"time"

	"github.com/andreamancuso/gh-sweep/internal/github"
)

func TestRecommendedPlan(t *testing.T) {
	now := time.Now()
	model := Model{
		inventory: &github.StorageInventory{
			Artifacts: []github.StorageArtifact{
				{ID: 1, CreatedAt: now.Add(-4 * 24 * time.Hour)},
				{ID: 2, CreatedAt: now},
			},
			Runs: []github.StorageWorkflowRun{
				{ID: 3, Conclusion: "success"},
				{ID: 4, Conclusion: "failure"},
			},
		},
	}

	plan := model.recommendedPlan()
	if len(plan.Artifacts) != 1 || plan.Artifacts[0].ID != 1 {
		t.Fatalf("recommended artifacts = %#v, want old artifact", plan.Artifacts)
	}
	if len(plan.Runs) != 1 || plan.Runs[0].ID != 4 {
		t.Fatalf("recommended runs = %#v, want failed run", plan.Runs)
	}
}

func TestRenderSummaryMentionsPackageScope(t *testing.T) {
	rendered := renderSummary(&github.StorageInventory{
		Repository:   "owner/repo",
		PackageError: "gh auth refresh -s read:packages",
	})
	if !strings.Contains(rendered, "read:packages") {
		t.Fatalf("summary = %q, want package scope hint", rendered)
	}
}

func TestRenderActiveViewArtifacts(t *testing.T) {
	model := Model{
		viewMode: viewArtifacts,
		sortDesc: true,
		inventory: &github.StorageInventory{
			Artifacts: []github.StorageArtifact{
				{ID: 1, Name: "small", SizeBytes: 10},
				{ID: 2, Name: "large", SizeBytes: 100},
			},
		},
	}

	rendered := model.renderActiveView()
	if !strings.Contains(rendered, "Artifacts sorted by size") {
		t.Fatalf("artifact view = %q, want artifact heading", rendered)
	}
	if strings.Index(rendered, "large") > strings.Index(rendered, "small") {
		t.Fatalf("artifact view = %q, want largest artifact first", rendered)
	}
}

func TestRenderTabsShowsActiveView(t *testing.T) {
	model := Model{viewMode: viewRuns, sortDesc: false}
	rendered := model.renderTabs()
	if !strings.Contains(rendered, "[3] Runs") || !strings.Contains(rendered, "sort: asc") {
		t.Fatalf("tabs = %q, want runs tab and sort state", rendered)
	}
}

func TestSelectedPlanUsesSelectedRows(t *testing.T) {
	model := Model{
		selected: map[string]bool{
			selectionKey("artifact", 1): true,
			selectionKey("cache", 2):    true,
			selectionKey("run", 3):      true,
		},
		inventory: &github.StorageInventory{
			Artifacts: []github.StorageArtifact{{ID: 1}, {ID: 10}},
			Caches:    []github.StorageCache{{ID: 2}, {ID: 20}},
			Runs:      []github.StorageWorkflowRun{{ID: 3}, {ID: 30}},
		},
	}

	plan := model.selectedPlan()
	if len(plan.Artifacts) != 1 || plan.Artifacts[0].ID != 1 {
		t.Fatalf("selected artifacts = %#v", plan.Artifacts)
	}
	if len(plan.Caches) != 1 || plan.Caches[0].ID != 2 {
		t.Fatalf("selected caches = %#v", plan.Caches)
	}
	if len(plan.Runs) != 1 || plan.Runs[0].ID != 3 {
		t.Fatalf("selected runs = %#v", plan.Runs)
	}
}

func TestToggleSelectionUsesSortedRows(t *testing.T) {
	model := Model{
		viewMode: viewArtifacts,
		sortDesc: true,
		cursor:   0,
		selected: make(map[string]bool),
		inventory: &github.StorageInventory{
			Artifacts: []github.StorageArtifact{
				{ID: 1, SizeBytes: 10},
				{ID: 2, SizeBytes: 100},
			},
		},
	}

	model.toggleSelection()
	if !model.selected[selectionKey("artifact", 2)] {
		t.Fatalf("selection = %#v, want largest sorted artifact selected", model.selected)
	}
}
