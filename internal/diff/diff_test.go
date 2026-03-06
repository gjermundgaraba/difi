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
