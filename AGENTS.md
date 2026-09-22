# AGENTS.md — focus CLI

## Project Overview

**focus** is a macOS command-line tool for temporarily blocking chosen websites by managing entries in `/etc/hosts`. It stores configuration via `os.UserConfigDir()` / `$XDG_CONFIG_HOME` (defaults to `~/Library/Application Support/focus/config.json` on macOS) and uses `sudo` only for the final `/etc/hosts` synchronization step.

## Key Commands

| Task | Command |
|------|---------|
| Build | `make build` or `go build -o bin/focus ./cmd/focus` |
| Install locally | `make install` (installs to `~/.local/bin`) |
| Test | `make test` or `go test ./... && go vet ./...` |
| Clean | `make clean` |
| Run binary | `./bin/focus <command>` |

## Architecture

```
cmd/focus/main.go          → Entry point, constructs Application
internal/app/app.go        → Command dispatch, business logic
internal/config/config.go  → JSON config storage ($XDG_CONFIG_HOME/focus/config.json or macOS Library default)
internal/domain/domain.go  → Domain normalization, validation, www variant handling
internal/hosts/hosts.go    → /etc/hosts manipulation with atomic writes, backups, markers
```

### Data Flow

1. User runs `focus add youtube.com`
2. `domain.Normalize` → extracts canonical domain (`youtube.com`), strips `www.`, validates
3. `config.Store.Load/Save` → persists to `$XDG_CONFIG_HOME/focus/config.json`
4. If enabled, `syncPrivileged` re-executes binary with `sudo focus __sync <config-path>`
5. `hosts.Manager.Sync` → reads `/etc/hosts`, renders managed block between markers, atomic write + backup
6. `flushDNSCache` → runs `dscacheutil -flushcache` and `killall -HUP mDNSResponder`

## Important Patterns

### Config Storage
- Atomic write via temp file + `os.Rename`
- Permissions `0600`, directory `0700`
- Domains normalized, deduplicated, sorted on every save/load

### Hosts File Manipulation
- Owns only the section between `# >>> focus managed block >>>` and `# <<< focus managed block <<<`
- Creates timestamped backup `/etc/hosts.focus-backup-<timestamp>` before each change
- Atomic write preserves ownership, mode via `syscall.Stat_t`
- Each domain maps to two entries: `domain` and `www.domain` → `127.0.0.1`

### Privilege Escalation
- Main binary never runs as root
- `syncPrivileged` re-executes self via `sudo focus __sync <config-path>`
- Internal `__sync` command reads config, calls `hosts.SystemManager().Sync()`, flushes DNS cache

## Code Conventions

- Go 1.25, standard library only (no external deps beyond `golang.org/x` in tests)
- Errors wrapped with `fmt.Errorf("%w", err)` for unwrapping
- Functions return `(T, error)`; avoid panics
- Tests in `*_test.go` alongside source
- `go vet ./...` must pass

## Key Files to Know

| File | Purpose |
|------|---------|
| `internal/domain/domain.go` | Normalization logic — handles URLs, bare domains, validation, `www.` stripping |
| `internal/hosts/hosts.go` | Hosts file rendering, backup, atomic write, marker parsing |
| `internal/config/config.go` | Config persistence, atomic JSON save/load |
| `internal/app/app.go` | All user-facing commands, privilege escalation |

## Testing

```bash
go test ./...           # Run all tests
go test -v ./internal/...  # Verbose
go test -run TestNormalize ./internal/domain  # Single test
```

Tests cover:
- Domain normalization (URLs, bare, www, invalid)
- Hosts block location, removal, rendering
- Config load/save, normalization, atomic write

## Common Tasks

### Add a new command
1. Add case in `Application.Run` (`internal/app/app.go`)
2. Implement handler method on `Application`
3. Add to `usage()` output
4. Update README.md

### Modify domain validation
Edit `domain.Normalize` and `domain.validHostname` in `internal/domain/domain.go`

### Change hosts file format
Edit `hosts.block`, `hosts.render`, `hosts.locateBlock` in `internal/hosts/hosts.go`

### Debugging
- Run without sudo: `focus add ...` (no-op if disabled), `focus list`, `focus status`
- Inspect config: `cat $XDG_CONFIG_HOME/focus/config.json` (or `~/Library/Application Support/focus/config.json`)
- Inspect hosts block: `grep -A 100 "focus managed block" /etc/hosts`
- Check backups: `ls -la /etc/hosts.focus-backup-*`

## Limitations (from README)

- Hosts-file block is a **focus aid, not a security control**
- Bypassed by: Browser secure DNS, VPNs, proxies, DNS-over-HTTPS
- No wildcard support — each domain and `www.` variant must be added explicitly
- macOS only (relies on `dscacheutil`, `mDNSResponder`)

## Environment

- Requires macOS (for DNS flush commands and `/etc/hosts` management)
- Go 1.25+ for building
- `sudo` access required for `focus on/off/add/remove` when enabled

## Environment Variables

| Variable | Purpose | Default | Used by |
|----------|---------|---------|---------|
| `XDG_CONFIG_HOME` | Config directory (replaces `os.UserConfigDir`) | (none) — falls back to platform default | `internal/config/config.go` |