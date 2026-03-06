package vcs

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/oug-t/difi/internal/git"
	"github.com/oug-t/difi/internal/jj"
)

type (
	GitBackend struct{}
	JjBackend  struct{}
)

func (g GitBackend) Kind() string         { return "git" }
func (g GitBackend) CurrentLabel() string { return git.GetCurrentBranch() }
func (g GitBackend) RepoName() string     { return git.GetRepoName() }
func (g GitBackend) DefaultTarget() string {
	return "HEAD"
}

func (g GitBackend) ListChangedFiles(target string) ([]string, error) {
	return git.ListChangedFiles(target)
}

func (g GitBackend) DiffCmd(target, path string) tea.Cmd {
	gitCmd := git.DiffCmd(target, path)
	return func() tea.Msg {
		msg := gitCmd()
		if gitMsg, ok := msg.(git.DiffMsg); ok {
			return DiffMsg{Content: gitMsg.Content, RawContent: gitMsg.RawContent}
		}
		return msg
	}
}

func (g GitBackend) OpenEditorCmd(path string, lineNumber int, target string, editor string) tea.Cmd {
	gitCmd := git.OpenEditorCmd(path, lineNumber, target, editor)
	return func() tea.Msg {
		msg := gitCmd()
		if gitMsg, ok := msg.(git.EditorFinishedMsg); ok {
			return EditorFinishedMsg{Err: gitMsg.Err}
		}
		return msg
	}
}

func (g GitBackend) DiffStats(target string) (added int, deleted int, err error) {
	return git.DiffStats(target)
}

func (g GitBackend) DiffStatsByFile(target string) (map[string][2]int, error) {
	return git.DiffStatsByFile(target)
}

func (g GitBackend) UndoSelectedChangeCmd(target, path, rawDiff string, cursorLine int) tea.Cmd {
	gitCmd := git.UndoSelectedChangeCmd(target, path, rawDiff, cursorLine)
	return func() tea.Msg {
		msg := gitCmd()
		if gitMsg, ok := msg.(git.UndoResultMsg); ok {
			return UndoResultMsg{Err: gitMsg.Err, Changed: gitMsg.Changed, Message: gitMsg.Message}
		}
		return msg
	}
}

func (j JjBackend) Kind() string         { return "jj" }
func (j JjBackend) CurrentLabel() string { return "@" }
func (j JjBackend) RepoName() string     { return jj.GetRepoName() }
func (j JjBackend) DefaultTarget() string {
	return "@"
}

func (j JjBackend) ListChangedFiles(target string) ([]string, error) {
	return jj.ListChangedFiles(target)
}

func (j JjBackend) DiffCmd(target, path string) tea.Cmd {
	jjCmd := jj.DiffCmd(target, path)
	return func() tea.Msg {
		msg := jjCmd()
		if jjMsg, ok := msg.(jj.DiffMsg); ok {
			return DiffMsg{Content: jjMsg.Content, RawContent: jjMsg.RawContent}
		}
		return msg
	}
}

func (j JjBackend) OpenEditorCmd(path string, lineNumber int, target string, editor string) tea.Cmd {
	jjCmd := jj.OpenEditorCmd(path, lineNumber, target, editor)
	return func() tea.Msg {
		msg := jjCmd()
		if jjMsg, ok := msg.(jj.EditorFinishedMsg); ok {
			return EditorFinishedMsg{Err: jjMsg.Err}
		}
		return msg
	}
}

func (j JjBackend) DiffStats(target string) (added int, deleted int, err error) {
	return jj.DiffStats(target)
}

func (j JjBackend) DiffStatsByFile(target string) (map[string][2]int, error) {
	return jj.DiffStatsByFile(target)
}

func DetectBackend() Backend {
	if jj.RepoRoot() != "" {
		return JjBackend{}
	}
	if git.RepoRoot() != "" {
		return GitBackend{}
	}
	return GitBackend{}
}
