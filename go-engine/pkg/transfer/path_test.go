package transfer

import (
	"path/filepath"
	"strings"
	"testing"
)

func toUnixPath(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

func toNativePath(p string) string {
	return filepath.FromSlash(p)
}

func TestToUnixPath_WindowsToUnix(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"simple windows path", `docs\report.pdf`, "docs/report.pdf"},
		{"nested windows path", `a\b\c\d.txt`, "a/b/c/d.txt"},
		{"already unix path", "a/b/c/d.txt", "a/b/c/d.txt"},
		{"single file", "readme.md", "readme.md"},
		{"deep nested", `project\src\components\ui\button.vue`, "project/src/components/ui/button.vue"},
		{"mixed with unicode", `数据\文件.txt`, "数据/文件.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toUnixPath(tt.input)
			if result != tt.expect {
				t.Errorf("toUnixPath(%q) = %q, want %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestToNativePath_UnixToNative(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"simple path", "docs/report.pdf", filepath.FromSlash("docs/report.pdf")},
		{"nested path", "a/b/c/d.txt", filepath.FromSlash("a/b/c/d.txt")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toNativePath(tt.input)
			if result != tt.expect {
				t.Errorf("toNativePath(%q) = %q, want %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestPathRoundTrip(t *testing.T) {
	paths := []string{
		"simple/file.txt",
		"deep/nested/path/file.txt",
		"unicode/文件名.txt",
		"spaces/in path/file name.txt",
		`windows\style\path.txt`,
	}

	for _, original := range paths {
		t.Run(original, func(t *testing.T) {
			unix := toUnixPath(original)
			t.Logf("  Original: %q", original)
			t.Logf("  Unix:     %q", unix)

			// Verify no backslashes remain in unix path
			if strings.Contains(unix, "\\") {
				t.Errorf("Unix path contains backslash: %q", unix)
			}
		})
	}
}

func TestFilepathToSlash_OnLinux(t *testing.T) {
	// filepath.ToSlash is a no-op on Linux since / is the separator.
	// This test documents the expected behavior.
	result := filepath.ToSlash("already/unix/path")
	if result != "already/unix/path" {
		t.Errorf("ToSlash changed path: %q", result)
	}

	// On Linux, backslashes are ordinary filename characters.
	result = filepath.ToSlash(`back\slash`)
	if result != `back\slash` {
		t.Logf("Note: on Linux, ToSlash does not convert \\: %q", result)
	}
}
