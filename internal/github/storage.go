package github

import (
	"fmt"
	"strings"
	"time"
)

type StorageArtifact struct {
	ID          int64
	Name        string
	SizeBytes   int64
	Expired     bool
	CreatedAt   time.Time
	ExpiresAt   time.Time
	WorkflowRun int64
}

type StorageCache struct {
	ID             int64
	Key            string
	Ref            string
	Version        string
	SizeBytes      int64
	CreatedAt      time.Time
	LastAccessedAt time.Time
}

type StorageWorkflowRun struct {
	ID           int64
	Name         string
	Workflow     string
	Status       string
	Conclusion   string
	Branch       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DisplayTitle string
}

type StorageReleaseAsset struct {
	ID        int64
	Name      string
	SizeBytes int64
	CreatedAt time.Time
}

type StorageRelease struct {
	ID          int64
	TagName     string
	Name        string
	Draft       bool
	Prerelease  bool
	PublishedAt time.Time
	Assets      []StorageReleaseAsset
}

type StoragePackage struct {
	ID             int64
	Name           string
	PackageType    string
	Visibility     string
	UpdatedAt      time.Time
	LinkedRepo     string
	VersionCount   int
	InspectionNote string
}

type StorageInventory struct {
	Repository       string
	RepoGitSizeBytes int64
	Artifacts        []StorageArtifact
	Caches           []StorageCache
	Runs             []StorageWorkflowRun
	Releases         []StorageRelease
	Packages         []StoragePackage
	PackageError     string
}

type StorageSummary struct {
	ArtifactCount           int
	ArtifactBytes           int64
	CacheCount              int
	CacheBytes              int64
	RunCount                int
	FailedCancelledRunCount int
	ReleaseCount            int
	ReleaseAssetCount       int
	ReleaseAssetBytes       int64
	PackageCount            int
	RepoGitSizeBytes        int64
}

type artifactListResponse struct {
	TotalCount int `json:"total_count"`
	Artifacts  []struct {
		ID        int64     `json:"id"`
		Name      string    `json:"name"`
		Size      int64     `json:"size_in_bytes"`
		Expired   bool      `json:"expired"`
		CreatedAt time.Time `json:"created_at"`
		ExpiresAt time.Time `json:"expires_at"`
		Workflow  struct {
			ID int64 `json:"id"`
		} `json:"workflow_run"`
	} `json:"artifacts"`
}

type cacheListResponse struct {
	TotalCount int `json:"total_count"`
	Caches     []struct {
		ID             int64     `json:"id"`
		Key            string    `json:"key"`
		Ref            string    `json:"ref"`
		Version        string    `json:"version"`
		Size           int64     `json:"size_in_bytes"`
		CreatedAt      time.Time `json:"created_at"`
		LastAccessedAt time.Time `json:"last_accessed_at"`
	} `json:"actions_caches"`
}

type storageRunsResponse struct {
	WorkflowRuns []struct {
		ID           int64     `json:"id"`
		Name         string    `json:"name"`
		WorkflowName string    `json:"workflow_name"`
		Status       string    `json:"status"`
		Conclusion   string    `json:"conclusion"`
		HeadBranch   string    `json:"head_branch"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		DisplayTitle string    `json:"display_title"`
	} `json:"workflow_runs"`
}

type storageRepoResponse struct {
	SizeKB int64 `json:"size"`
}

type storageReleaseResponse struct {
	ID          int64     `json:"id"`
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []struct {
		ID        int64     `json:"id"`
		Name      string    `json:"name"`
		Size      int64     `json:"size"`
		CreatedAt time.Time `json:"created_at"`
	} `json:"assets"`
}

type storagePackageResponse struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	PackageType string    `json:"package_type"`
	Visibility  string    `json:"visibility"`
	UpdatedAt   time.Time `json:"updated_at"`
	Repository  struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

type packageVersionResponse struct {
	ID int64 `json:"id"`
}

func (c *Client) GetStorageInventory(owner, repo string) (*StorageInventory, error) {
	repository := repoFullName(owner, repo)

	artifacts, err := c.ListStorageArtifacts(owner, repo)
	if err != nil {
		return nil, err
	}
	caches, err := c.ListStorageCaches(owner, repo)
	if err != nil {
		return nil, err
	}
	runs, err := c.ListStorageWorkflowRuns(owner, repo)
	if err != nil {
		return nil, err
	}
	releases, err := c.ListStorageReleases(owner, repo)
	if err != nil {
		return nil, err
	}
	repoSizeBytes, err := c.GetRepoGitSizeBytes(owner, repo)
	if err != nil {
		return nil, err
	}

	packages, packageErr := c.ListStoragePackagesForRepository(owner, repo)
	inventory := &StorageInventory{
		Repository:       repository,
		RepoGitSizeBytes: repoSizeBytes,
		Artifacts:        artifacts,
		Caches:           caches,
		Runs:             runs,
		Releases:         releases,
		Packages:         packages,
	}
	if packageErr != nil {
		inventory.PackageError = packageErr.Error()
	}

	return inventory, nil
}

func (c *Client) ListStorageArtifacts(owner, repo string) ([]StorageArtifact, error) {
	var artifacts []StorageArtifact
	page := 1
	perPage := 100
	for {
		var response artifactListResponse
		path := apiPathWithQuery(apiPath("repos", owner, repo, "actions", "artifacts"), query(map[string]string{
			"per_page": fmt.Sprintf("%d", perPage),
			"page":     fmt.Sprintf("%d", page),
		}))
		if err := c.Get(path, &response); err != nil {
			return nil, fmt.Errorf("failed to list artifacts: %w", err)
		}
		for _, artifact := range response.Artifacts {
			artifacts = append(artifacts, StorageArtifact{
				ID:          artifact.ID,
				Name:        artifact.Name,
				SizeBytes:   artifact.Size,
				Expired:     artifact.Expired,
				CreatedAt:   artifact.CreatedAt,
				ExpiresAt:   artifact.ExpiresAt,
				WorkflowRun: artifact.Workflow.ID,
			})
		}
		if len(response.Artifacts) < perPage {
			break
		}
		page++
	}
	return artifacts, nil
}

func (c *Client) DeleteStorageArtifact(owner, repo string, artifactID int64) error {
	path := apiPath("repos", owner, repo, "actions", "artifacts", fmt.Sprintf("%d", artifactID))
	if err := c.Delete(path, nil); err != nil {
		return fmt.Errorf("failed to delete artifact %d: %w", artifactID, err)
	}
	return nil
}

func (c *Client) ListStorageCaches(owner, repo string) ([]StorageCache, error) {
	var caches []StorageCache
	page := 1
	perPage := 100
	for {
		var response cacheListResponse
		path := apiPathWithQuery(apiPath("repos", owner, repo, "actions", "caches"), query(map[string]string{
			"per_page": fmt.Sprintf("%d", perPage),
			"page":     fmt.Sprintf("%d", page),
		}))
		if err := c.Get(path, &response); err != nil {
			return nil, fmt.Errorf("failed to list caches: %w", err)
		}
		for _, cache := range response.Caches {
			caches = append(caches, StorageCache{
				ID:             cache.ID,
				Key:            cache.Key,
				Ref:            cache.Ref,
				Version:        cache.Version,
				SizeBytes:      cache.Size,
				CreatedAt:      cache.CreatedAt,
				LastAccessedAt: cache.LastAccessedAt,
			})
		}
		if len(response.Caches) < perPage {
			break
		}
		page++
	}
	return caches, nil
}

func (c *Client) DeleteStorageCache(owner, repo string, cacheID int64) error {
	path := apiPath("repos", owner, repo, "actions", "caches", fmt.Sprintf("%d", cacheID))
	if err := c.Delete(path, nil); err != nil {
		return fmt.Errorf("failed to delete cache %d: %w", cacheID, err)
	}
	return nil
}

func (c *Client) ListStorageWorkflowRuns(owner, repo string) ([]StorageWorkflowRun, error) {
	var runs []StorageWorkflowRun
	page := 1
	perPage := 100
	for {
		var response storageRunsResponse
		path := apiPathWithQuery(apiPath("repos", owner, repo, "actions", "runs"), query(map[string]string{
			"per_page": fmt.Sprintf("%d", perPage),
			"page":     fmt.Sprintf("%d", page),
		}))
		if err := c.Get(path, &response); err != nil {
			return nil, fmt.Errorf("failed to list workflow runs: %w", err)
		}
		for _, run := range response.WorkflowRuns {
			workflow := run.WorkflowName
			if workflow == "" {
				workflow = run.Name
			}
			runs = append(runs, StorageWorkflowRun{
				ID:           run.ID,
				Name:         run.Name,
				Workflow:     workflow,
				Status:       run.Status,
				Conclusion:   run.Conclusion,
				Branch:       run.HeadBranch,
				CreatedAt:    run.CreatedAt,
				UpdatedAt:    run.UpdatedAt,
				DisplayTitle: run.DisplayTitle,
			})
		}
		if len(response.WorkflowRuns) < perPage {
			break
		}
		page++
	}
	return runs, nil
}

func (c *Client) DeleteStorageWorkflowRun(owner, repo string, runID int64) error {
	path := apiPath("repos", owner, repo, "actions", "runs", fmt.Sprintf("%d", runID))
	if err := c.Delete(path, nil); err != nil {
		return fmt.Errorf("failed to delete workflow run %d: %w", runID, err)
	}
	return nil
}

func (c *Client) ListStorageReleases(owner, repo string) ([]StorageRelease, error) {
	var releases []StorageRelease
	page := 1
	perPage := 100
	for {
		var response []storageReleaseResponse
		path := apiPathWithQuery(apiPath("repos", owner, repo, "releases"), query(map[string]string{
			"per_page": fmt.Sprintf("%d", perPage),
			"page":     fmt.Sprintf("%d", page),
		}))
		if err := c.Get(path, &response); err != nil {
			return nil, fmt.Errorf("failed to list releases: %w", err)
		}
		for _, release := range response {
			assets := make([]StorageReleaseAsset, 0, len(release.Assets))
			for _, asset := range release.Assets {
				assets = append(assets, StorageReleaseAsset{
					ID:        asset.ID,
					Name:      asset.Name,
					SizeBytes: asset.Size,
					CreatedAt: asset.CreatedAt,
				})
			}
			releases = append(releases, StorageRelease{
				ID:          release.ID,
				TagName:     release.TagName,
				Name:        release.Name,
				Draft:       release.Draft,
				Prerelease:  release.Prerelease,
				PublishedAt: release.PublishedAt,
				Assets:      assets,
			})
		}
		if len(response) < perPage {
			break
		}
		page++
	}
	return releases, nil
}

func (c *Client) GetRepoGitSizeBytes(owner, repo string) (int64, error) {
	var response storageRepoResponse
	if err := c.Get(apiPath("repos", owner, repo), &response); err != nil {
		return 0, fmt.Errorf("failed to get repository size: %w", err)
	}
	return response.SizeKB * 1024, nil
}

func (c *Client) ListStoragePackagesForRepository(owner, repo string) ([]StoragePackage, error) {
	repository := repoFullName(owner, repo)
	packageTypes := []string{"container", "nuget", "npm", "maven", "rubygems"}
	var packages []StoragePackage
	var errors []string

	for _, packageType := range packageTypes {
		userPackages, err := c.listStoragePackagesForNamespace("users", owner, packageType)
		if err == nil {
			packages = append(packages, filterPackagesForRepository(userPackages, repository)...)
			continue
		}
		errors = append(errors, err.Error())

		orgPackages, orgErr := c.listStoragePackagesForNamespace("orgs", owner, packageType)
		if orgErr == nil {
			packages = append(packages, filterPackagesForRepository(orgPackages, repository)...)
			continue
		}
		errors = append(errors, orgErr.Error())
	}

	if len(packages) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("failed to list packages; ensure gh auth has read:packages scope (gh auth refresh -s read:packages -s delete:packages)")
	}
	return packages, nil
}

func (c *Client) listStoragePackagesForNamespace(namespaceKind, namespace, packageType string) ([]StoragePackage, error) {
	var response []storagePackageResponse
	path := apiPathWithQuery(apiPath(namespaceKind, namespace, "packages"), query(map[string]string{
		"package_type": packageType,
		"per_page":     "100",
	}))
	if err := c.Get(path, &response); err != nil {
		return nil, err
	}

	packages := make([]StoragePackage, 0, len(response))
	for _, pkg := range response {
		versionCount, _ := c.countPackageVersions(namespaceKind, namespace, packageType, pkg.Name)
		note := ""
		if versionCount == 0 {
			note = "version count unavailable or empty"
		}
		packages = append(packages, StoragePackage{
			ID:             pkg.ID,
			Name:           pkg.Name,
			PackageType:    pkg.PackageType,
			Visibility:     pkg.Visibility,
			UpdatedAt:      pkg.UpdatedAt,
			LinkedRepo:     pkg.Repository.FullName,
			VersionCount:   versionCount,
			InspectionNote: note,
		})
	}
	return packages, nil
}

func (c *Client) countPackageVersions(namespaceKind, namespace, packageType, packageName string) (int, error) {
	var response []packageVersionResponse
	path := apiPathWithQuery(apiPath(namespaceKind, namespace, "packages", packageType, packageName, "versions"), query(map[string]string{
		"per_page": "100",
	}))
	if err := c.Get(path, &response); err != nil {
		return 0, err
	}
	return len(response), nil
}

func filterPackagesForRepository(packages []StoragePackage, repository string) []StoragePackage {
	filtered := make([]StoragePackage, 0, len(packages))
	for _, pkg := range packages {
		if pkg.LinkedRepo == repository {
			filtered = append(filtered, pkg)
		}
	}
	return filtered
}

func SummarizeStorage(inventory *StorageInventory) StorageSummary {
	summary := StorageSummary{
		ArtifactCount:    len(inventory.Artifacts),
		CacheCount:       len(inventory.Caches),
		RunCount:         len(inventory.Runs),
		ReleaseCount:     len(inventory.Releases),
		PackageCount:     len(inventory.Packages),
		RepoGitSizeBytes: inventory.RepoGitSizeBytes,
	}
	for _, artifact := range inventory.Artifacts {
		summary.ArtifactBytes += artifact.SizeBytes
	}
	for _, cache := range inventory.Caches {
		summary.CacheBytes += cache.SizeBytes
	}
	for _, run := range inventory.Runs {
		if run.Conclusion == "failure" || run.Conclusion == "cancelled" {
			summary.FailedCancelledRunCount++
		}
	}
	for _, release := range inventory.Releases {
		summary.ReleaseAssetCount += len(release.Assets)
		for _, asset := range release.Assets {
			summary.ReleaseAssetBytes += asset.SizeBytes
		}
	}
	return summary
}

func SelectArtifactsForCleanup(artifacts []StorageArtifact, olderThan time.Duration, now time.Time, includeAll bool) []StorageArtifact {
	if includeAll {
		return artifacts
	}
	if olderThan <= 0 {
		return nil
	}
	var selected []StorageArtifact
	cutoff := now.Add(-olderThan)
	for _, artifact := range artifacts {
		if artifact.CreatedAt.Before(cutoff) {
			selected = append(selected, artifact)
		}
	}
	return selected
}

func SelectCachesForCleanup(caches []StorageCache, olderThan time.Duration, now time.Time, includeAll bool) []StorageCache {
	if includeAll {
		return caches
	}
	if olderThan <= 0 {
		return nil
	}
	var selected []StorageCache
	cutoff := now.Add(-olderThan)
	for _, cache := range caches {
		timestamp := cache.LastAccessedAt
		if timestamp.IsZero() {
			timestamp = cache.CreatedAt
		}
		if timestamp.Before(cutoff) {
			selected = append(selected, cache)
		}
	}
	return selected
}

func SelectRunsForCleanup(runs []StorageWorkflowRun, conclusions map[string]bool, olderThan time.Duration, now time.Time) []StorageWorkflowRun {
	var selected []StorageWorkflowRun
	cutoff := now.Add(-olderThan)
	for _, run := range runs {
		if len(conclusions) > 0 && !conclusions[run.Conclusion] {
			continue
		}
		if olderThan > 0 && !run.CreatedAt.Before(cutoff) {
			continue
		}
		selected = append(selected, run)
	}
	return selected
}

func ParseConclusionSet(value string) map[string]bool {
	result := map[string]bool{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			result[part] = true
		}
	}
	return result
}
