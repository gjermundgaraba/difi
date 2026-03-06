package tree

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/stretchr/testify/require"
)

func TestNewBuildsNormalizedSortedItems(t *testing.T) {
	tree := New([]string{
		"zeta.txt",
		"./docs/../docs/guide.md",
		"docs/sub/file.txt",
		"Alpha.go",
		"docs/guide.md",
	})

	items := treeItems(t, tree.Items())
	want := []TreeItem{
		{Name: "docs", FullPath: "docs", IsDir: true, Depth: 0, Expanded: true, Icon: ""},
		{Name: "sub", FullPath: "docs/sub", IsDir: true, Depth: 1, Expanded: true, Icon: ""},
		{Name: "file.txt", FullPath: "docs/sub/file.txt", IsDir: false, Depth: 2, Expanded: true, Icon: ""},
		{Name: "guide.md", FullPath: "docs/guide.md", IsDir: false, Depth: 1, Expanded: true, Icon: ""},
		{Name: "Alpha.go", FullPath: "Alpha.go", IsDir: false, Depth: 0, Expanded: true, Icon: ""},
		{Name: "zeta.txt", FullPath: "zeta.txt", IsDir: false, Depth: 0, Expanded: true, Icon: ""},
	}

	require.Equal(t, want, items)
}

func TestToggleExpandHidesAndRestoresChildren(t *testing.T) {
	tree := New([]string{
		"docs/guide.md",
		"docs/sub/file.txt",
		"root.txt",
	})

	tree.ToggleExpand("docs")
	collapsed := treeItems(t, tree.Items())
	require.Len(t, collapsed, 2)
	require.Equal(t, TreeItem{Name: "docs", FullPath: "docs", IsDir: true, Depth: 0, Icon: ""}, collapsed[0])
	require.Equal(t, "root.txt", collapsed[1].FullPath)

	tree.ToggleExpand("docs")
	expanded := treeItems(t, tree.Items())
	require.Len(t, expanded, 5)
	require.Equal(t, TreeItem{Name: "docs", FullPath: "docs", IsDir: true, Depth: 0, Expanded: true, Icon: ""}, expanded[0])
}

func TestToggleExpandIgnoresFiles(t *testing.T) {
	tree := New([]string{"docs/readme.md"})
	before := treeItems(t, tree.Items())

	tree.ToggleExpand("docs/readme.md")
	after := treeItems(t, tree.Items())

	require.Equal(t, before, after)
}

func TestGetIcon(t *testing.T) {
	tests := []struct {
		name  string
		file  string
		isDir bool
		want  string
	}{
		{name: "directory", file: "docs", isDir: true, want: ""},
		{name: "go", file: "main.go", want: ""},
		{name: "json", file: "package.json", want: ""},
		{name: "gitignore", file: ".gitignore", want: ""},
		{name: "default", file: "notes.txt", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, getIcon(tt.file, tt.isDir))
		})
	}
}

func treeItems(t *testing.T, items []list.Item) []TreeItem {
	t.Helper()

	result := make([]TreeItem, 0, len(items))
	for _, item := range items {
		treeItem, ok := item.(TreeItem)
		require.Truef(t, ok, "item has type %T, want TreeItem", item)
		result = append(result, treeItem)
	}
	return result
}
