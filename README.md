# focus

`focus` is a macOS command-line tool for temporarily blocking chosen websites through `/etc/hosts`.

## Build

```bash
go build -o focus ./cmd/focus
```

Put the resulting binary somewhere on your `PATH`.

## Usage

```bash
focus add youtube.com
focus add https://www.reddit.com/r/golang
focus list
focus on
focus status
focus off
focus remove youtube.com
```

`add` stores a canonical domain and always manages its `www.` variant too. State is saved at `$XDG_CONFIG_HOME/focus/config.json`. Commands that change `/etc/hosts` request `sudo` only for that final synchronization step.

Focus owns only the section delimited by its markers and leaves every other `/etc/hosts` entry untouched. Each managed hostname is mapped to both `127.0.0.1` and `::1`, preventing IPv4 and IPv6 DNS resolution from bypassing the block. Before each change it creates a timestamped `/etc/hosts.focus-backup-*` backup. It flushes the macOS DNS cache after a successful hosts-file update.

## Limitations

A hosts-file block is a focus aid, not a security control. Browser secure DNS, VPNs, proxies, or DNS-over-HTTPS can bypass it. `/etc/hosts` does not support wildcard hostnames, so domains beyond the managed root and `www.` names must be added explicitly.

## Development

```bash
go test ./...
go vet ./...
```
