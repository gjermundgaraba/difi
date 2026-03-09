package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/stretchr/testify/require"

	"github.com/gjermundgaraba/difi/internal/config"
	"github.com/gjermundgaraba/difi/internal/vcs"
)

type modelGoldenCase struct {
	name   string
	width  int
	height int
	files  []string
	setup  func(t *testing.T, model *Model)
}

func TestModelGolden(t *testing.T) {
	cases := []modelGoldenCase{
		{
			name:   "Empty/Wide",
			width:  100,
			height: 28,
			files:  nil,
		},
		{
			name:   "Empty/Narrow",
			width:  70,
			height: 24,
			files:  nil,
		},
		{
			name:   "View/SelectedFile",
			width:  100,
			height: 28,
			files: []string{
				"docs/guide.md",
				"docs/sub/api.go",
				"src/app/main.go",
				"very/long/path/to/the/component/file_with_a_long_name.go",
			},
			setup: func(t *testing.T, model *Model) {
				t.Helper()
				model.selectedPath = "src/app/main.go"
				selectItemByPath(&model.fileList, model.fileList.Items(), model.selectedPath)
				applyStats(t, model, StatsMsg{
					Added:   3,
					Deleted: 1,
					ByFile: map[string][2]int{
						"src/app/main.go": {3, 1},
					},
				})
				applyDiff(t, model, strings.Join([]string{
					"diff --git a/src/app/main.go b/src/app/main.go",
					"index 1234567..89abcde 100644",
					"--- a/src/app/main.go",
					"+++ b/src/app/main.go",
					"@@ -1,2 +1,4 @@",
					" func main() {",
					"-\tprintln(\"old\")",
					"+\tprintln(\"new\")",
					"+\tprintln(\"more\")",
					" }",
					"",
				}, "\n"))
			},
		},
		{
			name:   "View/DirectorySelected",
			width:  100,
			height: 28,
			files: []string{
				"docs/guide.md",
				"docs/sub/api.go",
				"src/app/main.go",
			},
			setup: func(t *testing.T, model *Model) {
				t.Helper()
				applyStats(t, model, StatsMsg{
					Added:   4,
					Deleted: 2,
					ByFile: map[string][2]int{
						"docs/guide.md":   {1, 1},
						"docs/sub/api.go": {3, 1},
					},
				})
				model.fileList.Select(0)
			},
		},
		{
			name:   "View/HelpOpen",
			width:  100,
			height: 28,
			files: []string{
				"docs/guide.md",
				"docs/sub/api.go",
				"src/app/main.go",
			},
			setup: func(t *testing.T, model *Model) {
				t.Helper()
				model.selectedPath = "src/app/main.go"
				selectItemByPath(&model.fileList, model.fileList.Items(), model.selectedPath)
				applyDiff(t, model, strings.Join([]string{
					"diff --git a/src/app/main.go b/src/app/main.go",
					"@@ -1 +1 @@",
					"-old",
					"+new",
					"",
				}, "\n"))
				model.showHelp = true
				model.updateSizes()
			},
		},
		{
			name:   "View/TruncatedTopBar",
			width:  60,
			height: 22,
			files: []string{
				"very/long/path/to/the/component/file_with_a_long_name.go",
			},
			setup: func(t *testing.T, model *Model) {
				t.Helper()
				model.selectedPath = "very/long/path/to/the/component/file_with_a_long_name.go"
				selectItemByPath(&model.fileList, model.fileList.Items(), model.selectedPath)
				applyStats(t, model, StatsMsg{
					Added:   14,
					Deleted: 7,
					ByFile: map[string][2]int{
						"very/long/path/to/the/component/file_with_a_long_name.go": {14, 7},
					},
				})
				applyDiff(t, model, strings.Join([]string{
					"diff --git a/very/long/path/to/the/component/file_with_a_long_name.go b/very/long/path/to/the/component/file_with_a_long_name.go",
					"@@ -1 +1,2 @@",
					"-old",
					"+new",
					"+extra",
					"",
				}, "\n"))
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := newGoldenModel(tc.width, tc.height, tc.files)
			if tc.setup != nil {
				tc.setup(t, &model)
			}

			golden.RequireEqual(t, []byte(normalizeForGolden(model.View())))
		})
	}
}

func newGoldenModel(width, height int, files []string) Model {
	fake := &fakeVCS{
		files:       append([]string{}, files...),
		currentRepo: "repo-with-a-very-long-name",
		diffByPath: map[string]string{
			"src/app/main.go": "@@ -1 +1 @@\n-old\n+new\n",
			"very/long/path/to/the/component/file_with_a_long_name.go": "@@ -1 +1 @@\n-old\n+new\n",
		},
	}

	model := NewModel(config.Config{}, "HEAD", "", "", fake)
	model.width = width
	model.height = height
	model.updateSizes()
	return model
}

func applyDiff(t *testing.T, model *Model, diff string) {
	t.Helper()

	updatedModel, _ := model.Update(vcs.DiffMsg{Content: diff, RawContent: diff})
	updated, ok := updatedModel.(Model)
	require.True(t, ok)
	*model = updated
}

func applyStats(t *testing.T, model *Model, stats StatsMsg) {
	t.Helper()

	updatedModel, _ := model.Update(stats)
	updated, ok := updatedModel.(Model)
	require.True(t, ok)
	*model = updated
}

func normalizeForGolden(out string) string {
	out = strings.ReplaceAll(out, "\r\n", "\n")
	return ansi.Strip(out)
}
