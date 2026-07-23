package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func TestParseAgeDuration(t *testing.T) {
	got, err := parseAgeDuration("3d")
	if err != nil {
		t.Fatalf("parseAgeDuration failed: %v", err)
	}
	if got != 72*time.Hour {
		t.Fatalf("parseAgeDuration = %v, want 72h", got)
	}
}

func TestBuildStorageCleanupPlanRecommendedShape(t *testing.T) {
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	inventory := &github.StorageInventory{
		Artifacts: []github.StorageArtifact{
			{ID: 1, CreatedAt: now.Add(-4 * 24 * time.Hour), SizeBytes: 100},
			{ID: 2, CreatedAt: now.Add(-1 * 24 * time.Hour), SizeBytes: 100},
		},
		Caches: []github.StorageCache{
			{ID: 3, LastAccessedAt: now.Add(-4 * 24 * time.Hour), SizeBytes: 100},
		},
		Runs: []github.StorageWorkflowRun{
			{ID: 4, Conclusion: "success"},
			{ID: 5, Conclusion: "failure"},
			{ID: 6, Conclusion: "cancelled"},
		},
	}

	plan := buildStorageCleanupPlan(inventory, storageCleanupOptions{
		DeleteArtifacts: true,
		DeleteCaches:    true,
		DeleteRuns:      true,
		OlderThan:       3 * 24 * time.Hour,
		RunOlderThan:    0,
		Conclusions:     github.ParseConclusionSet("failure,cancelled"),
		Now:             now,
	})

	if len(plan.Artifacts) != 1 || plan.Artifacts[0].ID != 1 {
		t.Fatalf("artifact plan = %#v, want old artifact only", plan.Artifacts)
	}
	if len(plan.Caches) != 1 || plan.Caches[0].ID != 3 {
		t.Fatalf("cache plan = %#v, want old cache", plan.Caches)
	}
	if len(plan.Runs) != 2 {
		t.Fatalf("run plan = %#v, want failed/cancelled", plan.Runs)
	}
}

func TestConfirmStorageCleanupRequiresRepoName(t *testing.T) {
	plan := storageCleanupPlan{
		Artifacts: []github.StorageArtifact{{ID: 1}},
	}
	var output bytes.Buffer
	if confirmStorageCleanup("owner/repo", plan, strings.NewReader("owner\n"), &output) {
		t.Fatal("confirmStorageCleanup accepted incomplete repo name")
	}
	if !confirmStorageCleanup("owner/repo", plan, strings.NewReader("owner/repo\n"), &output) {
		t.Fatal("confirmStorageCleanup rejected exact repo name")
	}
}

func TestPlanHasSuccessfulRuns(t *testing.T) {
	if !planHasSuccessfulRuns(storageCleanupPlan{
		Runs: []github.StorageWorkflowRun{{Conclusion: "success"}},
	}) {
		t.Fatal("planHasSuccessfulRuns returned false for a successful run")
	}
	if planHasSuccessfulRuns(storageCleanupPlan{
		Runs: []github.StorageWorkflowRun{{Conclusion: "failure"}},
	}) {
		t.Fatal("planHasSuccessfulRuns returned true without a successful run")
	}
}
