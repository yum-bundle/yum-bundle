package yumfile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yum-bundle/yum-bundle/internal/yumfile"
)

func TestParse(t *testing.T) {
	t.Run("parses all directive types", func(t *testing.T) {
		content := `# comment line

yum vim
yum curl
key https://download.docker.com/linux/centos/gpg
repo https://download.docker.com/linux/centos/docker-ce.repo
baseurl https://packages.example.com/stable/x86_64/
copr atim/lazygit
epel
module nodejs:18
rpm https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm
group Development Tools
`
		path := writeTempYumfile(t, content)
		entries, err := yumfile.Parse(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 10 {
			t.Fatalf("expected 10 entries, got %d", len(entries))
		}
		assertEntry(t, entries[0], yumfile.EntryTypeYum, "vim")
		assertEntry(t, entries[1], yumfile.EntryTypeYum, "curl")
		assertEntry(t, entries[2], yumfile.EntryTypeKey, "https://download.docker.com/linux/centos/gpg")
		assertEntry(t, entries[3], yumfile.EntryTypeRepo, "https://download.docker.com/linux/centos/docker-ce.repo")
		assertEntry(t, entries[4], yumfile.EntryTypeBaseurl, "https://packages.example.com/stable/x86_64/")
		assertEntry(t, entries[5], yumfile.EntryTypeCopr, "atim/lazygit")
		assertEntry(t, entries[6], yumfile.EntryTypeEPEL, "")
		assertEntry(t, entries[7], yumfile.EntryTypeModule, "nodejs:18")
		assertEntry(t, entries[8], yumfile.EntryTypeRPM, "https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm")
		assertEntry(t, entries[9], yumfile.EntryTypeGroup, "Development Tools")
	})

	t.Run("version pinning formats", func(t *testing.T) {
		content := `yum "nodejs = 18.0.0"
yum "curl-7.76.1"
yum vim
`
		path := writeTempYumfile(t, content)
		entries, err := yumfile.Parse(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(entries))
		}
		assertEntry(t, entries[0], yumfile.EntryTypeYum, "nodejs = 18.0.0")
		assertEntry(t, entries[1], yumfile.EntryTypeYum, "curl-7.76.1")
		assertEntry(t, entries[2], yumfile.EntryTypeYum, "vim")
	})

	t.Run("skips blank lines and comments", func(t *testing.T) {
		content := `# this is a comment
yum vim

# another comment
yum curl
`
		path := writeTempYumfile(t, content)
		entries, err := yumfile.Parse(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(entries))
		}
	})

	t.Run("inline comments are stripped", func(t *testing.T) {
		content := `yum bat          # cat with syntax highlighting
yum ripgrep      # fast grep alternative (binary: rg)
key https://example.com/gpg.key  # import signing key
repo https://example.com/my.repo # nightly builds
baseurl https://example.com/el9/ # custom repo
copr atim/lazygit                # terminal UI for git
epel                             # enable EPEL
module nodejs:18                 # LTS stream
rpm https://example.com/pkg.rpm  # bootstrap rpm
group "Development Tools"        # compiler toolchain
`
		path := writeTempYumfile(t, content)
		entries, err := yumfile.Parse(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 10 {
			t.Fatalf("expected 10 entries, got %d", len(entries))
		}
		assertEntry(t, entries[0], yumfile.EntryTypeYum, "bat")
		assertEntry(t, entries[1], yumfile.EntryTypeYum, "ripgrep")
		assertEntry(t, entries[2], yumfile.EntryTypeKey, "https://example.com/gpg.key")
		assertEntry(t, entries[3], yumfile.EntryTypeRepo, "https://example.com/my.repo")
		assertEntry(t, entries[4], yumfile.EntryTypeBaseurl, "https://example.com/el9/")
		assertEntry(t, entries[5], yumfile.EntryTypeCopr, "atim/lazygit")
		assertEntry(t, entries[6], yumfile.EntryTypeEPEL, "")
		assertEntry(t, entries[7], yumfile.EntryTypeModule, "nodejs:18")
		assertEntry(t, entries[8], yumfile.EntryTypeRPM, "https://example.com/pkg.rpm")
		assertEntry(t, entries[9], yumfile.EntryTypeGroup, "Development Tools")
	})

	t.Run("returns error on unknown directive", func(t *testing.T) {
		content := "unknown foo\n"
		path := writeTempYumfile(t, content)
		_, err := yumfile.Parse(path)
		if err == nil {
			t.Fatal("expected error for unknown directive")
		}
	})

	t.Run("returns error on missing argument", func(t *testing.T) {
		content := "yum\n"
		path := writeTempYumfile(t, content)
		_, err := yumfile.Parse(path)
		if err == nil {
			t.Fatal("expected error for missing argument")
		}
	})

	t.Run("returns error on missing file", func(t *testing.T) {
		_, err := yumfile.Parse("/nonexistent/Yumfile")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("quoted values are unquoted", func(t *testing.T) {
		content := `yum "nodejs = 18.0.0"
repo 'https://example.com/my.repo'
`
		path := writeTempYumfile(t, content)
		entries, err := yumfile.Parse(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertEntry(t, entries[0], yumfile.EntryTypeYum, "nodejs = 18.0.0")
		assertEntry(t, entries[1], yumfile.EntryTypeRepo, "https://example.com/my.repo")
	})

	t.Run("preserves original line text", func(t *testing.T) {
		content := "  yum vim  \n"
		path := writeTempYumfile(t, content)
		entries, err := yumfile.Parse(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if entries[0].Original != "  yum vim  " {
			t.Errorf("expected original line preserved, got %q", entries[0].Original)
		}
	})
}

func TestExtractPkgName(t *testing.T) {
	tests := []struct {
		spec string
		want string
	}{
		{"vim", "vim"},
		{"curl-7.76.1", "curl"},
		{"nodejs = 18.0.0", "nodejs"},
		{"nodejs=18.0.0", "nodejs"},
		{"python3-pip", "python3-pip"},      // no version part
		{"python3-pip-21.0", "python3-pip"}, // version is numeric
		{"docker-ce", "docker-ce"},          // no version
		{"docker-ce-20.10.0", "docker-ce"},  // version is numeric
	}
	for _, tt := range tests {
		got := yumfile.ExtractPkgName(tt.spec)
		if got != tt.want {
			t.Errorf("ExtractPkgName(%q) = %q, want %q", tt.spec, got, tt.want)
		}
	}
}

func assertEntry(t *testing.T, e yumfile.Entry, typ yumfile.EntryType, value string) {
	t.Helper()
	if e.Type != typ {
		t.Errorf("expected type %s, got %s", typ, e.Type)
	}
	if e.Value != value {
		t.Errorf("expected value %q, got %q", value, e.Value)
	}
}

func writeTempYumfile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "Yumfile")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write temp Yumfile: %v", err)
	}
	return path
}
