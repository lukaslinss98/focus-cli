package hosts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/lukas/focus/internal/domain"
)

const (
	startMarker = "# >>> focus managed block >>>"
	endMarker   = "# <<< focus managed block <<<"
)

type Manager struct {
	path string
}

func NewManager(path string) *Manager {
	return &Manager{path: path}
}

func SystemManager() *Manager {
	return NewManager("/etc/hosts")
}

func (m *Manager) HasManagedBlock() (bool, error) {
	contents, err := os.ReadFile(m.path)
	if err != nil {
		return false, fmt.Errorf("read hosts file: %w", err)
	}
	start, end, err := locateBlock(string(contents))
	if err != nil {
		return false, err
	}
	return start >= 0 && end >= 0, nil
}

func (m *Manager) Sync(enabled bool, domains []string) (bool, error) {
	contents, err := os.ReadFile(m.path)
	if err != nil {
		return false, fmt.Errorf("read hosts file: %w", err)
	}

	updated, err := render(string(contents), enabled, domains)
	if err != nil {
		return false, err
	}
	if updated == string(contents) {
		return false, nil
	}
	if err := m.backup(contents); err != nil {
		return false, err
	}
	if err := atomicWrite(m.path, []byte(updated)); err != nil {
		return false, err
	}
	return true, nil
}

func render(contents string, enabled bool, domains []string) (string, error) {
	start, _, err := locateBlock(contents)
	if err != nil {
		return "", err
	}
	withoutBlock := contents
	if start >= 0 {
		withoutBlock = removeBlock(contents)
	}
	if !enabled || len(domains) == 0 {
		return withoutBlock, nil
	}

	trimmed := strings.TrimRight(withoutBlock, "\n")
	if trimmed == "" {
		return block(domains) + "\n", nil
	}
	return trimmed + "\n\n" + block(domains) + "\n", nil
}

func locateBlock(contents string) (int, int, error) {
	start, end := -1, -1
	for index, line := range strings.Split(contents, "\n") {
		switch line {
		case startMarker:
			if start >= 0 || end >= 0 {
				return -1, -1, errors.New("hosts file has malformed focus block")
			}
			start = index
		case endMarker:
			if start < 0 || end >= 0 {
				return -1, -1, errors.New("hosts file has malformed focus block")
			}
			end = index
		}
	}
	if start >= 0 && end < 0 {
		return -1, -1, errors.New("hosts file has unterminated focus block")
	}
	return start, end, nil
}

func removeBlock(contents string) string {
	lines := strings.Split(contents, "\n")
	start, end, _ := locateBlock(contents)
	remaining := append(lines[:start:start], lines[end+1:]...)
	result := strings.Join(remaining, "\n")
	return strings.TrimRight(result, "\n") + "\n"
}

func block(domains []string) string {
	lines := []string{startMarker}
	for _, name := range domains {
		variants := strings.Join(domain.Variants(name), "\t")
		lines = append(lines, "127.0.0.1\t"+variants, "::1\t"+variants)
	}
	lines = append(lines, endMarker)
	return strings.Join(lines, "\n")
}

func (m *Manager) backup(contents []byte) error {
	backupPath := fmt.Sprintf("%s.focus-backup-%s", m.path, time.Now().UTC().Format("20060102T150405.000000000Z"))
	if err := os.WriteFile(backupPath, contents, 0644); err != nil {
		return fmt.Errorf("back up hosts file: %w", err)
	}
	return nil
}

func atomicWrite(path string, contents []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat hosts file: %w", err)
	}

	temporary, err := os.CreateTemp(filepath.Dir(path), ".focus-hosts-*")
	if err != nil {
		return fmt.Errorf("create temporary hosts file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(info.Mode()); err != nil {
		temporary.Close()
		return fmt.Errorf("set hosts file permissions: %w", err)
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		temporary.Close()
		return errors.New("read hosts file ownership")
	}
	if err := temporary.Chown(int(owner.Uid), int(owner.Gid)); err != nil {
		temporary.Close()
		return fmt.Errorf("set hosts file ownership: %w", err)
	}
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary hosts file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary hosts file: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace hosts file: %w", err)
	}
	return nil
}
