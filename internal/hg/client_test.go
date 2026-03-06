package hg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStripAnsi(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no ansi codes",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "colored text",
			input:    "\033[31mred text\033[0m",
			expected: "red text",
		},
		{
			name:     "multiple colors",
			input:    "\033[32m+added\033[0m \033[31m-deleted\033[0m",
			expected: "+added -deleted",
		},
		{
			name:     "complex ansi",
			input:    "\033[1;32m+\033[0m\033[32mline\033[0m",
			expected: "+line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, stripAnsi(tt.input))
		})
	}
}

func TestCalculateFileLine(t *testing.T) {
	diffContent := `diff -r 123456 file.go
--- a/file.go	Tue Jan 01 00:00:00 2024 +0000
+++ b/file.go	Tue Jan 01 00:00:01 2024 +0000
@@ -10,7 +10,8 @@
 func main() {
-	fmt.Println("old")
+	fmt.Println("new")
+	fmt.Println("added")
 }
 `

	tests := []struct {
		name            string
		visualLineIndex int
		expectedLineNo  int
	}{
		{
			name:            "header line",
			visualLineIndex: 0,
			expectedLineNo:  1,
		},
		{
			name:            "hunk header",
			visualLineIndex: 3,
			expectedLineNo:  9,
		},
		{
			name:            "context line",
			visualLineIndex: 4,
			expectedLineNo:  10,
		},
		{
			name:            "deleted line",
			visualLineIndex: 5,
			expectedLineNo:  10,
		},
		{
			name:            "added line 1",
			visualLineIndex: 6,
			expectedLineNo:  11,
		},
		{
			name:            "added line 2",
			visualLineIndex: 7,
			expectedLineNo:  12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expectedLineNo, CalculateFileLine(diffContent, tt.visualLineIndex))
		})
	}
}

func TestParseFilesFromDiff(t *testing.T) {
	diffText := `diff -r 123456 file1.go
--- a/file1.go	Tue Jan 01 00:00:00 2024 +0000
+++ b/file1.go	Tue Jan 01 00:00:01 2024 +0000
@@ -1,3 +1,3 @@
 package main
-import "fmt"
+import "log"
 func main() {}
diff -r 123456 file2.py
--- a/file2.py	Tue Jan 01 00:00:00 2024 +0000
+++ b/file2.py	Tue Jan 01 00:00:01 2024 +0000
@@ -1,2 +1,2 @@
-print("hello")
+print("world")
`

	expected := []string{"file1.go", "file2.py"}
	result := ParseFilesFromDiff(diffText)

	require.Equal(t, expected, result)
}

func TestParseFilesFromDiffEmpty(t *testing.T) {
	require.Empty(t, ParseFilesFromDiff(""))
}

func TestParseFilesFromDiffNoDuplicates(t *testing.T) {
	diffText := `diff -r 123456 file1.go
--- a/file1.go	Tue Jan 01 00:00:00 2024 +0000
+++ b/file1.go	Tue Jan 01 00:00:01 2024 +0000
@@ -1,3 +1,3 @@
 package main
diff -r 123456 file1.go
--- a/file1.go	Tue Jan 01 00:00:00 2024 +0000
+++ b/file1.go	Tue Jan 01 00:00:01 2024 +0000
@@ -5,3 +5,3 @@
 func main() {}
`

	result := ParseFilesFromDiff(diffText)
	require.Equal(t, []string{"file1.go"}, result)
}

func TestExtractFileDiff(t *testing.T) {
	diffText := `diff -r 123456 file1.go
--- a/file1.go	Tue Jan 01 00:00:00 2024 +0000
+++ b/file1.go	Tue Jan 01 00:00:01 2024 +0000
@@ -1,3 +1,3 @@
 package main
-import "fmt"
+import "log"
 func main() {}
diff -r 123456 file2.py
--- a/file2.py	Tue Jan 01 00:00:00 2024 +0000
+++ b/file2.py	Tue Jan 01 00:00:01 2024 +0000
@@ -1,2 +1,2 @@
-print("hello")
+print("world")
`

	expectedFile1 := `diff -r 123456 file1.go
--- a/file1.go	Tue Jan 01 00:00:00 2024 +0000
+++ b/file1.go	Tue Jan 01 00:00:01 2024 +0000
@@ -1,3 +1,3 @@
 package main
-import "fmt"
+import "log"
 func main() {}`

	result := ExtractFileDiff(diffText, "file1.go")
	result = strings.TrimSpace(result)
	expectedFile1 = strings.TrimSpace(expectedFile1)

	require.Equal(t, expectedFile1, result)
}

func TestExtractFileDiffNotFound(t *testing.T) {
	diffText := `diff -r 123456 file1.go
--- a/file1.go	Tue Jan 01 00:00:00 2024 +0000
+++ b/file1.go	Tue Jan 01 00:00:01 2024 +0000
@@ -1,3 +1,3 @@
 package main
`

	require.Empty(t, strings.TrimSpace(ExtractFileDiff(diffText, "nonexistent.go")))
}
