# gh-sweep 🧹

This project is an MIT-licensed fork of [Kyle King's original gh-sweep project](https://github.com/KyleKing/gh-sweep).

`gh-sweep` is a local Go TUI/CLI for inspecting and cleaning up GitHub repository maintenance clutter, with a strong focus on GitHub Actions storage: artifacts, caches, workflow runs/logs, releases/assets, packages, and repo git size.

It runs from your machine. It does not rely on GitHub Actions being available, which matters when Actions are blocked because storage quota is exhausted.

## Install

```bash
go install github.com/andreamancuso/gh-sweep@latest
```

Or build from source:

```bash
git clone https://github.com/andreamancuso/gh-sweep.git
cd gh-sweep
go build -o gh-sweep
```

## Authentication

Use GitHub CLI auth:

```bash
gh auth login
```

For package inspection/deletion, refresh package scopes:

```bash
gh auth refresh -s read:packages -s delete:packages
```

For Watch Status, add the notifications scope:

```bash
gh auth refresh -h github.com -s notifications
```

Automation can use `GH_TOKEN` or `GITHUB_TOKEN`. `gh-sweep` does not persist GitHub tokens in `.gh-sweep.yaml`.

## Common commands

Inspect storage:

```bash
gh-sweep storage --repo owner/repo --list
```

Inspect releases and packages too:

```bash
gh-sweep storage --repo owner/repo --list --inspect-releases --inspect-packages
```

Preview recommended cleanup:

```bash
gh-sweep storage --repo owner/repo --recommended --dry-run
```

Run recommended cleanup:

```bash
gh-sweep storage --repo owner/repo --recommended --yes
```

Recommended cleanup deletes:

- Actions artifacts older than 3 days
- Actions caches older than 3 days
- failed/cancelled workflow runs

It preserves:

- releases and release assets
- packages
- successful release runs
- source branches, tags, and git history

Targeted cleanup:

```bash
gh-sweep storage --repo owner/repo --delete-artifacts --older-than 3d
gh-sweep storage --repo owner/repo --delete-all-artifacts
gh-sweep storage --repo owner/repo --delete-caches --older-than 7d
gh-sweep storage --repo owner/repo --delete-runs --conclusion failure,cancelled
```

Open the storage TUI:

```bash
gh-sweep storage --repo owner/repo
```

Open the full TUI:

```bash
gh-sweep --repo owner/repo
```

Other available commands:

```bash
gh-sweep branches --repo owner/repo
gh-sweep comments --repo owner/repo
gh-sweep protection --repos owner/repo1,owner/repo2
gh-sweep gha-perf --repo owner/repo
gh-sweep orphans --namespace owner --list
gh-sweep watching
```

## Safety model

Destructive storage operations are preview-first.

- `--dry-run` never deletes.
- `--yes` skips prompts for normal cleanup.
- Successful workflow-run deletion still requires typed repository-name confirmation.
- Release and package deletion are not part of recommended cleanup.
- Deletions are irreversible.

GitHub billing/storage pages can lag after deletion. The repository API may show cleanup before billing dashboards refresh.

## Optional config

`gh-sweep` looks for `.gh-sweep.yaml` in:

1. the current working directory
2. the executable directory
3. your home directory
4. `~/.config/gh-sweep/config.yaml`

When the full TUI is launched without `--repo`, the first configured repository becomes the default single-repo target, and `repositories` powers the multi-repo views.

Multi-repo views show a repository-selection screen before calling GitHub. Use `space` to toggle one repo, `a` to select all, `n` to select none, and `enter` to load the selected repos.

Example:

```yaml
default_org: your-org
repositories:
  - owner/repo1
  - owner/repo2

cache:
  ttl: 1h
  path: ~/.cache/gh-sweep

filters:
  exclude_users:
    - dependabot
    - renovate
```

## Development

For local setup, tests, security checks, and release-oriented validation, see [docs/development.md](docs/development.md).

## License

MIT License. See [LICENSE](LICENSE).
