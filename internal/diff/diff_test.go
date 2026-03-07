package diff

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateFileLine(t *testing.T) {
	diffContent := `diff --git a/file.go b/file.go
index 1234567..89abcde 100644
--- a/file.go
+++ b/file.go
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
		want            int
	}{
		{name: "header line", visualLineIndex: 0, want: 1},
		{name: "hunk header", visualLineIndex: 4, want: 9},
		{name: "context line", visualLineIndex: 5, want: 10},
		{name: "deleted line", visualLineIndex: 6, want: 10},
		{name: "first added line", visualLineIndex: 7, want: 11},
		{name: "second added line", visualLineIndex: 8, want: 12},
		{name: "out of bounds", visualLineIndex: 99, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, CalculateFileLine(diffContent, tt.visualLineIndex))
		})
	}
}

func TestNormalizePathScope(t *testing.T) {
	require.Empty(t, NormalizePathScope(""))
	require.Empty(t, NormalizePathScope("."))
	require.Equal(t, "src/app", NormalizePathScope("./src/app"))
	require.Equal(t, "src/app", NormalizePathScope("src//app"))
}

func TestPathMatchesScope(t *testing.T) {
	require.True(t, PathMatchesScope("src/app/main.go", "src"))
	require.True(t, PathMatchesScope("src/app/main.go", "src/app/main.go"))
	require.False(t, PathMatchesScope("src-other/main.go", "src"))
	require.False(t, PathMatchesScope("src/app/main.go", "docs"))
}

func TestFilterPaths(t *testing.T) {
	paths := []string{"src/app/main.go", "src/lib/util.go", "docs/readme.md"}
	require.Equal(t, []string{"src/app/main.go", "src/lib/util.go"}, FilterPaths(paths, "src"))
	require.Equal(t, []string{"src/app/main.go"}, FilterPaths(paths, "src/app/main.go"))
	require.Empty(t, FilterPaths(paths, "test"))
}

func TestFilterStats(t *testing.T) {
	stats := map[string][2]int{
		"src/app/main.go": {3, 1},
		"src/lib/util.go": {2, 0},
		"docs/readme.md":  {1, 4},
	}

	added, deleted, filtered := FilterStats(stats, "src")
	require.Equal(t, 5, added)
	require.Equal(t, 1, deleted)
	require.Equal(t, map[string][2]int{
		"src/app/main.go": {3, 1},
		"src/lib/util.go": {2, 0},
	}, filtered)
}

func TestAppendPathScope(t *testing.T) {
	args := []string{"diff", "--name-only", "HEAD"}
	require.Equal(t, args, AppendPathScope(args, ""))
	require.Equal(t, []string{"diff", "--name-only", "HEAD", "--", "src/app"}, AppendPathScope(args, "./src//app"))
}
