package hosts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderPreservesUserEntries(t *testing.T) {
	original := "127.0.0.1 localhost\n::1 localhost\n"
	updated, err := render(original, true, []string{"reddit.com", "youtube.com"})
	if err != nil {
		t.Fatal(err)
	}

	want := original + "\n" + startMarker + "\n" +
		"127.0.0.1\treddit.com\twww.reddit.com\n" +
		"127.0.0.1\tyoutube.com\twww.youtube.com\n" + endMarker + "\n"
	if updated != want {
		t.Errorf("rendered hosts file:\n%s\nwant:\n%s", updated, want)
	}

	disabled, err := render(updated, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if disabled != original {
		t.Errorf("disabled hosts file = %q, want %q", disabled, original)
	}
}

func TestRenderRejectsMalformedBlock(t *testing.T) {
	_, err := render(startMarker+"\n127.0.0.1 youtube.com\n", true, []string{"youtube.com"})
	if err == nil {
		t.Fatal("render accepted an unterminated block")
	}
}

func TestSyncIsIdempotent(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "hosts")
	if err := os.WriteFile(path, []byte("127.0.0.1 localhost\n"), 0644); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(path)

	changed, err := manager.Sync(true, []string{"youtube.com"})
	if err != nil || !changed {
		t.Fatalf("first sync changed=%t, err=%v", changed, err)
	}
	changed, err = manager.Sync(true, []string{"youtube.com"})
	if err != nil || changed {
		t.Fatalf("second sync changed=%t, err=%v", changed, err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(contents), startMarker) != 1 {
		t.Errorf("expected one managed block, got:\n%s", contents)
	}
}
