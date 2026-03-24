# Implementation Tasks — kubectl-co

> Reverse-engineered from the existing codebase. All tasks below reflect
> features that are **already implemented and tested**.

---

## Project Setup

- [x] **[Setup]** Initialise Go module and directory structure
  _Done when:_ `go.mod` exists, `internal/` package compiles, `main.go` builds.

- [x] **[Setup]** Configure golangci-lint
  _Done when:_ `.golangci-lint.yaml` is present and `golangci-lint run` passes.

- [x] **[Setup]** Configure GoReleaser with Homebrew tap
  _Done when:_ `.goreleaser.yaml` builds for linux/darwin and publishes to `homebrew-kubectl-co`.

- [x] **[Setup]** Add Renovate config with Go version and GitHub Actions managers
  _Done when:_ `renovate.json` tracks Go deps, go:generate tools, and workflow Go versions.

---

## Core Logic (`internal/co.go`)

- [x] **[Core]** Implement `NewCO` constructor (init kube home, CO home, read symlinks)
  _Done when:_ `NewCO(home)` returns a populated `CO` struct; directories are created if absent.

- [x] **[Core]** Implement `AddConfig` (copy existing or create empty config file)
  _Done when:_ `AddConfig("")` creates empty file; `AddConfig(path)` copies content; permissions are 0700.

- [x] **[Core]** Implement `LinkKubeConfig` (symlink switching with previous tracking)
  _Done when:_ `~/.kube/config` symlinks to the target; `~/.kube/co/previous` records the old target.

- [x] **[Core]** Implement `DeleteConfig` (remove config and re-link)
  _Done when:_ Config file is removed; kubeconfig re-links to previous; error if config missing.

- [x] **[Core]** Implement `ListConfigs` (read directory, exclude `previous`)
  _Done when:_ `Configs` field populated with all entries except the `previous` symlink.

- [x] **[Core]** Implement cleanup helper (remove old symlinks safely)
  _Done when:_ Both `~/.kube/config` and `~/.kube/co/previous` are removed; missing files are tolerated.

---

## CLI Layer (`main.go`)

- [x] **[CLI]** Define flags with pflag and bind via viper
  _Done when:_ `--add`, `--delete`, `--previous`, `--current`, `--debug`, `--help`, `--version` are registered.

- [x] **[CLI]** Implement flag validation (mutual exclusivity, argument count)
  _Done when:_ Conflicting flags produce a clear error; wrong argument counts are rejected.

- [x] **[CLI]** Implement `execute` dispatch (route to add/delete/link/list)
  _Done when:_ Each flag combination dispatches to the correct `CO` method.

- [x] **[CLI]** Implement `printConfigs` with colour highlighting
  _Done when:_ Active config is printed in red; others in default colour.

---

## Shell Completion (`completion.go`)

- [x] **[Completion]** Implement bash/zsh completion output
  _Done when:_ `kubectl-co completion bash` and `zsh` output valid completion scripts.

- [x] **[Completion]** Implement dynamic completion via `COMP_LINE`/`COMP_POINT`
  _Done when:_ Flag names and config names are completed based on cursor position.

---

## CI/CD (`.github/workflows/`)

- [x] **[CI]** GitHub Actions workflow for `go test` on push/PR
  _Done when:_ `go-test.yml` runs tests with coverage on every push and PR to main.

- [x] **[CI]** GitHub Actions workflow for CodeQL analysis
  _Done when:_ `codeql-analysis.yml` scans Go code on push/PR and weekly schedule.

- [x] **[CD]** GitHub Actions workflow for semantic release + GoReleaser
  _Done when:_ `release.yml` runs monthly, creates a GitHub release, and publishes Homebrew formula.

---

## Testing (`internal/co_test.go`)

- [x] **[Test]** Tests for `NewCO`, `initKubeHome`, `initCOHome`
  _Done when:_ Constructor and directory-init paths are covered including error cases.

- [x] **[Test]** Tests for `AddConfig` (create, copy, source missing, bad base path)
  _Done when:_ All four scenarios pass with testify assertions.

- [x] **[Test]** Tests for `LinkKubeConfig` (previous, no input, missing target, existing target)
  _Done when:_ Table-driven test covers all four cases.

- [x] **[Test]** Tests for `DeleteConfig` (success, non-existing, link error, remove failure)
  _Done when:_ Four subtests validate delete behaviour and error paths.

- [x] **[Test]** Tests for `ListConfigs` (success, error)
  _Done when:_ Both happy-path and bad-directory subtests pass.

- [x] **[Test]** Tests for `cleanup`
  _Done when:_ Cleanup of existing files passes; missing files are tolerated.
