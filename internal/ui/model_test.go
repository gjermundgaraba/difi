package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"github.com/oug-t/difi/internal/config"
	diffparse "github.com/oug-t/difi/internal/diff"
	"github.com/oug-t/difi/internal/vcs"
)

const (
	testPathA = "a.go"
)

type fakeVCS struct {
	files        []string
	diffByPath   map[string]string
	currentRepo  string
	kind         string
	listErr      error
	statsAdded   int
	statsDeleted int
	statsByFile  map[string][2]int
}

func (f *fakeVCS) Kind() string {
	if f.kind != "" {
		return f.kind
	}
	return "git"
}
func (f *fakeVCS) CurrentLabel() string { return "feature" }
func (f *fakeVCS) RepoName() string {
	if f.currentRepo != "" {
		return f.currentRepo
	}
	return "repo"
}
func (f *fakeVCS) DefaultTarget() string { return "HEAD" }

func (f *fakeVCS) ListChangedFiles(target string) ([]string, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return append([]string{}, f.files...), nil
}

func (f *fakeVCS) DiffCmd(target, path string) tea.Cmd {
	diff := f.diffByPath[path]
	return func() tea.Msg {
		return vcs.DiffMsg{Content: diff, RawContent: diff}
	}
}

func (f *fakeVCS) OpenEditorCmd(path string, lineNumber int, target string, editor string) tea.Cmd {
	return nil
}

func (f *fakeVCS) DiffStats(target string) (int, int, error) {
	return f.statsAdded, f.statsDeleted, nil
}

func (f *fakeVCS) DiffStatsByFile(target string) (map[string][2]int, error) {
	result := make(map[string][2]int, len(f.statsByFile))
	for path, stats := range f.statsByFile {
		result[path] = stats
	}
	return result, nil
}

type fakeUndoVCS struct {
	*fakeVCS
	called     bool
	lastTarget string
	lastPath   string
	lastDiff   string
	lastCursor int
	result     vcs.UndoResultMsg
}

func (f *fakeUndoVCS) UndoSelectedChangeCmd(target, path, rawDiff string, cursorLine int) tea.Cmd {
	f.called = true
	f.lastTarget = target
	f.lastPath = path
	f.lastDiff = rawDiff
	f.lastCursor = cursorLine
	return func() tea.Msg {
		return f.result
	}
}

func TestUndoKeyRejectsNonHeadTarget(t *testing.T) {
	fake := &fakeVCS{
		files:      []string{testPathA},
		diffByPath: map[string]string{testPathA: "@@ -1 +1 @@\n-old\n+new\n"},
	}

	model := NewModel(config.Config{}, "main", "", fake)
	model.focus = FocusDiff
	model.rawDiffContent = fake.diffByPath[testPathA]
	model.rawDiffLines = []string{"@@ -1 +1 @@", "-old", "+new", ""}
	model.diffLines = append([]string{}, model.rawDiffLines...)
	model.diffCursor = 2

	updatedModel, cmd := model.Update(keyMsg("x"))
	updated := requireModel(t, updatedModel)

	require.Nil(t, cmd)
	require.Equal(t, "Undo only works when comparing against HEAD", updated.statusMessage)
}

func TestUndoKeyDispatchesToChangeUndoer(t *testing.T) {
	base := &fakeVCS{
		files:      []string{testPathA},
		diffByPath: map[string]string{testPathA: "@@ -1 +1 @@\n-old\n+new\n"},
	}
	fake := &fakeUndoVCS{
		fakeVCS: base,
		result:  vcs.UndoResultMsg{Changed: true, Message: "Hunk undone"},
	}

	model := NewModel(config.Config{}, "HEAD", "", fake)
	model.focus = FocusDiff
	model.rawDiffContent = base.diffByPath[testPathA]
	model.rawDiffLines = []string{"@@ -1 +1 @@", "-old", "+new", ""}
	model.diffLines = append([]string{}, model.rawDiffLines...)
	model.diffCursor = 2

	updatedModel, cmd := model.Update(keyMsg("x"))
	updated := requireModel(t, updatedModel)
	require.NotNil(t, cmd)
	require.True(t, fake.called)
	require.Equal(t, "HEAD", fake.lastTarget)
	require.Equal(t, testPathA, fake.lastPath)
	require.Equal(t, 2, fake.lastCursor)
	require.Equal(t, base.diffByPath[testPathA], fake.lastDiff)

	msg := cmd()
	undoMsg, ok := msg.(vcs.UndoResultMsg)
	require.Truef(t, ok, "cmd() returned %T, want vcs.UndoResultMsg", msg)
	require.Equal(t, "Hunk undone", undoMsg.Message)
	require.Empty(t, updated.statusMessage)
}

func TestInitBatchesDiffAndStatsCommands(t *testing.T) {
	fake := &fakeVCS{
		files:        []string{testPathA},
		diffByPath:   map[string]string{testPathA: "@@ -1 +1 @@\n-old\n+new\n"},
		statsAdded:   2,
		statsDeleted: 1,
		statsByFile:  map[string][2]int{testPathA: {2, 1}},
	}

	model := NewModel(config.Config{}, "HEAD", "", fake)
	cmd := model.Init()
	require.NotNil(t, cmd)

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	require.Truef(t, ok, "cmd() returned %T, want tea.BatchMsg", msg)
	require.Len(t, batch, 2)

	var foundDiff bool
	var foundStats bool
	for _, batchCmd := range batch {
		if batchCmd == nil {
			continue
		}
		switch batchMsg := batchCmd().(type) {
		case vcs.DiffMsg:
			foundDiff = true
			require.Equal(t, fake.diffByPath[testPathA], batchMsg.RawContent)
		case StatsMsg:
			foundStats = true
			require.Equal(t, 2, batchMsg.Added)
			require.Equal(t, 1, batchMsg.Deleted)
		default:
			require.Failf(t, "unexpected batch message type", "%T", batchMsg)
		}
	}

	require.True(t, foundDiff)
	require.True(t, foundStats)
}

func TestApplyFileListSelectsNextFileWhenCurrentDisappears(t *testing.T) {
	fake := &fakeVCS{
		files: []string{testPathA, "b.go"},
		diffByPath: map[string]string{
			testPathA: "@@ -1 +1 @@\n-old\n+new\n",
			"c.go":    "@@ -1 +1 @@\n-old\n+replacement\n",
		},
	}

	model := NewModel(config.Config{}, "HEAD", "", fake)
	model.selectedPath = "b.go"
	selectItemByPath(&model.fileList, model.fileList.Items(), "b.go")

	cmd := model.applyFileList([]string{testPathA, "c.go"})
	require.Equal(t, "c.go", model.selectedPath)
	require.NotNil(t, cmd)

	msg := cmd()
	diffMsg, ok := msg.(vcs.DiffMsg)
	require.Truef(t, ok, "cmd() returned %T, want vcs.DiffMsg", msg)
	require.Equal(t, fake.diffByPath["c.go"], diffMsg.RawContent)
}

func TestDiffMsgUsesProvidedRawContent(t *testing.T) {
	fake := &fakeVCS{files: []string{testPathA}}
	model := NewModel(config.Config{}, "HEAD", "", fake)

	coloredDiff := strings.Join([]string{
		"diff --git a/a.go b/a.go",
		"index 1234567..89abcde 100644",
		"--- a/a.go",
		"+++ b/a.go",
		"@@ -1 +1,2 @@",
		"\x1b[31m-old\x1b[0m",
		"\x1b[32m+new\x1b[0m",
		"\x1b[32m+extra\x1b[0m",
		"",
	}, "\n")

	rawDiff := diffparse.StripANSI(coloredDiff)
	updatedModel, _ := model.Update(vcs.DiffMsg{Content: coloredDiff, RawContent: rawDiff})
	updated := requireModel(t, updatedModel)
	require.Equal(t, rawDiff, updated.rawDiffContent)

	wantRawLines := []string{"@@ -1 +1,2 @@", "-old", "+new", "+extra", ""}
	require.Equal(t, wantRawLines, updated.rawDiffLines)
	require.Equal(t, 2, updated.currentFileAdded)
	require.Equal(t, 1, updated.currentFileDeleted)
}

func TestComputePipedStatsCmdCountsGitDiff(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/a.go b/a.go",
		"@@ -1 +1,2 @@",
		"\x1b[31m-old\x1b[0m",
		"\x1b[32m+new\x1b[0m",
		"\x1b[32m+extra\x1b[0m",
		"diff --git a/b.go b/b.go",
		"@@ -2 +2 @@",
		"-before",
		"+after",
		"",
	}, "\n")

	model := NewModel(config.Config{}, "HEAD", diff, &fakeVCS{})
	msg := model.computePipedStatsCmd()()

	statsMsg, ok := msg.(StatsMsg)
	require.Truef(t, ok, "cmd() returned %T, want StatsMsg", msg)
	require.Equal(t, 3, statsMsg.Added)
	require.Equal(t, 2, statsMsg.Deleted)
	require.Equal(t, [2]int{2, 1}, statsMsg.ByFile[testPathA])
	require.Equal(t, [2]int{1, 1}, statsMsg.ByFile["b.go"])
}

func TestUndoKeyDispatchesToAnyBackendImplementingChangeUndoer(t *testing.T) {
	base := &fakeVCS{
		files:      []string{testPathA},
		diffByPath: map[string]string{testPathA: "@@ -1 +1 @@\n-old\n+new\n"},
		kind:       "jj",
	}
	fake := &fakeUndoVCS{
		fakeVCS: base,
		result:  vcs.UndoResultMsg{Changed: true, Message: "Hunk undone"},
	}

	model := NewModel(config.Config{}, "HEAD", "", fake)
	model.focus = FocusDiff
	model.rawDiffContent = base.diffByPath[testPathA]
	model.rawDiffLines = []string{"@@ -1 +1 @@", "-old", "+new", ""}
	model.diffLines = append([]string{}, model.rawDiffLines...)
	model.diffCursor = 2

	updatedModel, cmd := model.Update(keyMsg("x"))
	updated := requireModel(t, updatedModel)
	require.NotNil(t, cmd)
	require.True(t, fake.called)
	require.Equal(t, "HEAD", fake.lastTarget)
	require.Equal(t, testPathA, fake.lastPath)
	require.Equal(t, 2, fake.lastCursor)
	require.Equal(t, base.diffByPath[testPathA], fake.lastDiff)
	require.Empty(t, updated.statusMessage)
}

func TestApplyFileListClearsSelectionAndDiffWhenFilesDisappear(t *testing.T) {
	fake := &fakeVCS{files: []string{testPathA}}
	model := NewModel(config.Config{}, "HEAD", "", fake)
	model.selectedPath = testPathA
	model.diffContent = "@@ -1 +1 @@\n-old\n+new\n"
	model.diffLines = []string{"@@ -1 +1 @@", "-old", "+new", ""}
	model.rawDiffContent = model.diffContent
	model.rawDiffLines = append([]string{}, model.diffLines...)
	model.currentFileAdded = 1
	model.currentFileDeleted = 1
	model.diffCursor = 2
	model.diffViewport.SetContent(model.diffContent)

	cmd := model.applyFileList(nil)
	require.Nil(t, cmd)
	require.Empty(t, model.selectedPath)
	require.Empty(t, model.diffContent)
	require.Empty(t, model.rawDiffContent)
	require.Nil(t, model.diffLines)
	require.Nil(t, model.rawDiffLines)
	require.Zero(t, model.currentFileAdded)
	require.Zero(t, model.currentFileDeleted)
	require.Empty(t, model.fileList.Items())
}

func TestLoadSelectedDiffCmdUsesExtractedPipedDiff(t *testing.T) {
	fake := &fakeVCS{}

	fullDiff := strings.Join([]string{
		"diff --git a/a.go b/a.go",
		"@@ -1 +1 @@",
		"-old",
		"+new",
		"diff --git a/b.go b/b.go",
		"@@ -2 +2 @@",
		"-old",
		"+new",
		"",
	}, "\n")
	model := NewModel(config.Config{}, "HEAD", fullDiff, fake)
	model.selectedPath = "b.go"

	cmd := model.loadSelectedDiffCmd()
	require.NotNil(t, cmd)

	msg := cmd()
	diffMsg, ok := msg.(vcs.DiffMsg)
	require.Truef(t, ok, "cmd() returned %T, want vcs.DiffMsg", msg)
	require.Equal(t, "diff --git a/b.go b/b.go\n@@ -2 +2 @@\n-old\n+new", strings.TrimSpace(diffMsg.Content))
	require.Equal(t, strings.TrimSpace(diffMsg.Content), strings.TrimSpace(diffMsg.RawContent))
}

func TestFileListMsgErrorSetsStatus(t *testing.T) {
	fake := &fakeVCS{files: []string{testPathA}}
	model := NewModel(config.Config{}, "HEAD", "", fake)

	updatedModel, cmd := model.Update(FileListMsg{Err: errors.New("refresh failed")})
	updated := requireModel(t, updatedModel)

	require.Nil(t, cmd)
	require.Equal(t, "Failed to refresh changed files: refresh failed", updated.statusMessage)
	require.Equal(t, testPathA, updated.selectedPath)
}

func TestChooseRefreshedPath(t *testing.T) {
	tests := []struct {
		name     string
		oldPaths []string
		newPaths []string
		selected string
		want     string
	}{
		{
			name:     "preserves current selection when present",
			oldPaths: []string{"a.go", "b.go"},
			newPaths: []string{"a.go", "b.go", "c.go"},
			selected: "b.go",
			want:     "b.go",
		},
		{
			name:     "picks same index when selection disappears",
			oldPaths: []string{"a.go", "b.go", "c.go"},
			newPaths: []string{"a.go", "c.go"},
			selected: "b.go",
			want:     "c.go",
		},
		{
			name:     "falls back to last item when index is too large",
			oldPaths: []string{"a.go", "b.go", "c.go"},
			newPaths: []string{"a.go"},
			selected: "c.go",
			want:     "a.go",
		},
		{
			name:     "returns empty when no files remain",
			oldPaths: []string{"a.go"},
			newPaths: nil,
			selected: "a.go",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, chooseRefreshedPath(tt.oldPaths, tt.newPaths, tt.selected))
		})
	}
}

func TestUpdateSizes(t *testing.T) {
	model := NewModel(config.Config{}, "HEAD", "", &fakeVCS{files: []string{testPathA}})
	model.width = 100
	model.height = 40

	model.updateSizes()
	require.Equal(t, 16, model.fileList.Width())
	require.Equal(t, 36, model.fileList.Height())
	require.Equal(t, 80, model.diffViewport.Width)
	require.Equal(t, 36, model.diffViewport.Height)

	model.showHelp = true
	model.updateSizes()
	require.Equal(t, 30, model.fileList.Height())
	require.Equal(t, 30, model.diffViewport.Height)
}

func TestGetRepeatCountConsumesInputBuffer(t *testing.T) {
	model := Model{inputBuffer: "12"}

	require.Equal(t, 12, model.getRepeatCount())
	require.Empty(t, model.inputBuffer)
}

func TestHelpToggleUpdatesLayout(t *testing.T) {
	model := NewModel(config.Config{}, "HEAD", "", &fakeVCS{files: []string{testPathA}})
	model.width = 100
	model.height = 40
	model.updateSizes()

	updatedModel, cmd := model.Update(keyMsg("?"))
	updated := requireModel(t, updatedModel)

	require.Nil(t, cmd)
	require.True(t, updated.showHelp)
	require.Equal(t, 30, updated.diffViewport.Height)
}

func TestFocusKeysSwitchPanels(t *testing.T) {
	model := NewModel(config.Config{}, "HEAD", "", &fakeVCS{files: []string{testPathA}})

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated := requireModel(t, updatedModel)
	require.Equal(t, FocusDiff, updated.focus)

	updatedModel, _ = updated.Update(keyMsg("h"))
	updated = requireModel(t, updatedModel)
	require.Equal(t, FocusTree, updated.focus)
}

func TestUndoKeyRejectsPipedDiff(t *testing.T) {
	fake := &fakeVCS{
		files:      []string{testPathA},
		diffByPath: map[string]string{testPathA: "@@ -1 +1 @@\n-old\n+new\n"},
	}

	diff := strings.Join([]string{
		"diff --git a/a.go b/a.go",
		"@@ -1 +1 @@",
		"-old",
		"+new",
		"",
	}, "\n")
	model := NewModel(config.Config{}, "HEAD", diff, fake)
	model.focus = FocusDiff
	model.selectedPath = testPathA

	updatedModel, cmd := model.Update(keyMsg("x"))
	updated := requireModel(t, updatedModel)

	require.Nil(t, cmd)
	require.Equal(t, "Undo is unavailable for piped diffs", updated.statusMessage)
}

func keyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

func requireModel(t *testing.T, model tea.Model) Model {
	t.Helper()

	updated, ok := model.(Model)
	require.Truef(t, ok, "expected ui.Model, got %T", model)
	return updated
}
