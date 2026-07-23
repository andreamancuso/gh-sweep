package github

import (
	"testing"
	"time"
)

func TestSummarizeStorage(t *testing.T) {
	inventory := &StorageInventory{
		RepoGitSizeBytes: 7 * 1024 * 1024,
		Artifacts: []StorageArtifact{
			{SizeBytes: 100},
			{SizeBytes: 250},
		},
		Caches: []StorageCache{
			{SizeBytes: 10},
		},
		Runs: []StorageWorkflowRun{
			{Conclusion: "success"},
			{Conclusion: "failure"},
			{Conclusion: "cancelled"},
		},
		Releases: []StorageRelease{
			{Assets: []StorageReleaseAsset{{SizeBytes: 1024}, {SizeBytes: 2048}}},
		},
		Packages: []StoragePackage{{Name: "pkg"}},
	}

	got := SummarizeStorage(inventory)
	if got.ArtifactCount != 2 || got.ArtifactBytes != 350 {
		t.Fatalf("artifact summary = %d/%d", got.ArtifactCount, got.ArtifactBytes)
	}
	if got.FailedCancelledRunCount != 2 {
		t.Fatalf("failed/cancelled runs = %d, want 2", got.FailedCancelledRunCount)
	}
	if got.ReleaseAssetCount != 2 || got.ReleaseAssetBytes != 3072 {
		t.Fatalf("release asset summary = %d/%d", got.ReleaseAssetCount, got.ReleaseAssetBytes)
	}
}

func TestSelectArtifactsForCleanup(t *testing.T) {
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	artifacts := []StorageArtifact{
		{ID: 1, CreatedAt: now.Add(-4 * 24 * time.Hour)},
		{ID: 2, CreatedAt: now.Add(-2 * 24 * time.Hour)},
	}

	got := SelectArtifactsForCleanup(artifacts, 3*24*time.Hour, now, false)
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("selected artifacts = %#v, want ID 1 only", got)
	}
}

func TestSelectRunsForCleanup(t *testing.T) {
	runs := []StorageWorkflowRun{
		{ID: 1, Conclusion: "success"},
		{ID: 2, Conclusion: "failure"},
		{ID: 3, Conclusion: "cancelled"},
	}

	got := SelectRunsForCleanup(runs, ParseConclusionSet("failure,cancelled"), 0, time.Now())
	if len(got) != 2 || got[0].ID != 2 || got[1].ID != 3 {
		t.Fatalf("selected runs = %#v, want failed/cancelled", got)
	}
}
