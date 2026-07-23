package cache

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGHAPerfCacheUsesPrivatePermissions(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	manager, err := NewGHAPerfCacheManager(cacheDir)
	if err != nil {
		t.Fatalf("NewGHAPerfCacheManager failed: %v", err)
	}

	dirInfo, err := os.Stat(cacheDir)
	if err != nil {
		t.Fatalf("cache dir stat failed: %v", err)
	}
	if runtime.GOOS != "windows" && dirInfo.Mode().Perm() != 0700 {
		t.Fatalf("cache dir permissions = %v, want 0700", dirInfo.Mode().Perm())
	}

	if err := manager.Save("owner", "repo", &GHAPerfCache{}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	fileInfo, err := os.Stat(filepath.Join(cacheDir, "owner_repo.json"))
	if err != nil {
		t.Fatalf("cache file stat failed: %v", err)
	}
	if runtime.GOOS != "windows" && fileInfo.Mode().Perm() != 0600 {
		t.Fatalf("cache file permissions = %v, want 0600", fileInfo.Mode().Perm())
	}
}
