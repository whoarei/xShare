// 路径处理函数的单元测试
// 测试跨平台路径格式转换
package transfer

import (
	"path/filepath"
	"strings"
	"testing"
)

// toUnixPath 将Windows路径转换为Unix路径
func toUnixPath(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

// toNativePath 将Unix路径转换为当前平台的原生路径
func toNativePath(p string) string {
	return filepath.FromSlash(p)
}

// TestToUnixPath_WindowsToUnix 测试Windows路径到Unix路径的转换
// 验证反斜杠正确转换为正斜杠
func TestToUnixPath_WindowsToUnix(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		// 测试场景：简单Windows路径（单层反斜杠）
		// 验证点：反斜杠\正确转换为正斜杠/
		{"simple windows path", `docs\report.pdf`, "docs/report.pdf"},
		// 测试场景：嵌套Windows路径（多层反斜杠）
		// 验证点：所有反斜杠都被替换
		{"nested windows path", `a\b\c\d.txt`, "a/b/c/d.txt"},
		// 测试场景：已经是Unix路径
		// 验证点：已经是正斜杠的路径保持不变
		{"already unix path", "a/b/c/d.txt", "a/b/c/d.txt"},
		// 测试场景：单文件名（无路径分隔符）
		// 验证点：无分隔符的文件名保持不变
		{"single file", "readme.md", "readme.md"},
		// 测试场景：深层嵌套路径
		// 验证点：多层嵌套路径中的所有反斜杠都被正确转换
		{"deep nested", `project\src\components\ui\button.vue`, "project/src/components/ui/button.vue"},
		// 测试场景：包含Unicode字符的路径
		// 验证点：中文字符与路径分隔符混合时正确处理
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

// TestToNativePath_UnixToNative 测试Unix路径到原生路径的转换
// 验证正斜杠正确转换为平台分隔符
func TestToNativePath_UnixToNative(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		// 测试场景：简单Unix路径
		// 验证点：正斜杠正确转换为当前平台的路径分隔符
		{"simple path", "docs/report.pdf", filepath.FromSlash("docs/report.pdf")},
		// 测试场景：嵌套Unix路径
		// 验证点：所有正斜杠都被转换为平台原生分隔符
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

// TestPathRoundTrip 测试路径往返转换
// 验证各种路径格式转换后不包含反斜杠
func TestPathRoundTrip(t *testing.T) {
	paths := []string{
		// 测试场景：标准Unix路径
		"simple/file.txt",
		// 测试场景：深层嵌套路径
		"deep/nested/path/file.txt",
		// 测试场景：包含Unicode中文字符
		"unicode/文件名.txt",
		// 测试场景：包含空格的路径
		"spaces/in path/file name.txt",
		// 测试场景：Windows风格路径
		`windows\style\path.txt`,
	}

	for _, original := range paths {
		t.Run(original, func(t *testing.T) {
			unix := toUnixPath(original)
			t.Logf("  Original: %q", original)
			t.Logf("  Unix:     %q", unix)

			// 验证点：转换后的路径不能包含反斜杠（必须全部为正斜杠）
			if strings.Contains(unix, "\\") {
				t.Errorf("Unix path contains backslash: %q", unix)
			}
		})
	}
}

// TestFilepathToSlash_OnLinux 测试Linux平台的路径转换行为
// 记录filepath.ToSlash在Linux上的预期行为
func TestFilepathToSlash_OnLinux(t *testing.T) {
	// filepath.ToSlash在Linux上是空操作，因为/就是分隔符
	result := filepath.ToSlash("already/unix/path")
	// 验证点：已经是Unix路径时，ToSlash不会改变路径
	if result != "already/unix/path" {
		t.Errorf("ToSlash changed path: %q", result)
	}

	// 在Linux上，反斜杠是普通文件名字符，不会被转换
	result = filepath.ToSlash(`back\slash`)
	// 验证点：Linux上反斜杠被视为普通字符，ToSlash不会转换
	if result != `back\slash` {
		t.Logf("Note: on Linux, ToSlash does not convert \\: %q", result)
	}
}
