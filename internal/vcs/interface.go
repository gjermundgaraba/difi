package vcs

import tea "github.com/charmbracelet/bubbletea"

type Backend interface {
	Kind() string
	CurrentLabel() string
	RepoName() string
	DefaultTarget() string
	ListChangedFiles(target string) ([]string, error)
	DiffCmd(target, path string) tea.Cmd
	OpenEditorCmd(path string, lineNumber int, target string, editor string) tea.Cmd
	DiffStats(target string) (added int, deleted int, err error)
	DiffStatsByFile(target string) (map[string][2]int, error)
}

type ChangeUndoer interface {
	UndoSelectedChangeCmd(target, path, rawDiff string, cursorLine int) tea.Cmd
}

type DiffMsg struct {
	Content    string
	RawContent string
}
type (
	EditorFinishedMsg struct{ Err error }
	UndoResultMsg     struct {
		Err     error
		Changed bool
		Message string
	}
)
