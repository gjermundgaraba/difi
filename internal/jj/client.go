package jj

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	diffparse "github.com/oug-t/difi/internal/diff"
)

var summaryRe = regexp.MustCompile(`(\d+) insertion[s]?\(\+\), (\d+) deletion[s]?\(\-\)`)

var repoRootCache = struct {
	sync.Mutex
	byDir map[string]string
}{
	byDir: make(map[string]string),
}

func RepoRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	repoRootCache.Lock()
	root, ok := repoRootCache.byDir[cwd]
	repoRootCache.Unlock()
	if ok {
		return root
	}

	out, err := exec.Command("jj", "--no-pager", "root").Output()
	root = ""
	if err == nil {
		root = strings.TrimSpace(string(out))
	}

	repoRootCache.Lock()
	repoRootCache.byDir[cwd] = root
	repoRootCache.Unlock()

	return root
}

func jjCmd(args ...string) *exec.Cmd {
	fullArgs := append([]string{"--no-pager"}, args...)
	cmd := exec.Command("jj", fullArgs...)
	if root := RepoRoot(); root != "" {
		cmd.Dir = root
	}
	return cmd
}

func GetRepoName() string {
	root := RepoRoot()
	if root == "" {
		return "Repo"
	}
	return filepath.Base(root)
}

func ListChangedFiles(target, path string) ([]string, error) {
	out, err := jjCmd(diffparse.AppendPathScope([]string{"diff", "-r", normalizeTarget(target), "--name-only"}, path)...).Output()
	if err != nil {
		return nil, err
	}

	files := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}, nil
	}
	return files, nil
}

func DiffCmd(target, path string) tea.Cmd {
	target = normalizeTarget(target)
	return func() tea.Msg {
		coloredOut, err := jjCmd("--color=always", "diff", "--git", "-r", target, "--", path).Output()
		if err != nil {
			msg := "Error fetching diff: " + err.Error()
			return DiffMsg{Content: msg, RawContent: msg}
		}

		rawOut, err := jjCmd("--color=never", "diff", "--git", "-r", target, "--", path).Output()
		if err != nil {
			return DiffMsg{Content: string(coloredOut), RawContent: diffparse.StripANSI(string(coloredOut))}
		}

		return DiffMsg{Content: string(coloredOut), RawContent: string(rawOut)}
	}
}

func OpenEditorCmd(path string, lineNumber int, target string, editor string) tea.Cmd {
	var args []string
	if lineNumber > 0 {
		args = append(args, fmt.Sprintf("+%d", lineNumber))
	}
	args = append(args, path)

	c := exec.Command(editor, args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if root := RepoRoot(); root != "" {
		c.Dir = root
	}
	c.Env = append(os.Environ(), fmt.Sprintf("DIFI_TARGET=%s", normalizeTarget(target)))

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{Err: err}
	})
}

func DiffStats(target, path string) (added int, deleted int, err error) {
	out, err := jjCmd(diffparse.AppendPathScope([]string{"diff", "-r", normalizeTarget(target), "--stat"}, path)...).Output()
	if err != nil {
		return 0, 0, fmt.Errorf("jj diff stats error: %w", err)
	}

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if !strings.Contains(line, "file changed") && !strings.Contains(line, "files changed") {
			continue
		}
		matches := summaryRe.FindStringSubmatch(line)
		if len(matches) == 3 {
			added, _ = strconv.Atoi(matches[1])
			deleted, _ = strconv.Atoi(matches[2])
		}
		return added, deleted, nil
	}

	return 0, 0, nil
}

func DiffStatsByFile(target, path string) (map[string][2]int, error) {
	out, err := jjCmd(diffparse.AppendPathScope([]string{"diff", "-r", normalizeTarget(target), "--stat"}, path)...).Output()
	if err != nil {
		return nil, fmt.Errorf("jj diff stat error: %w", err)
	}

	result := make(map[string][2]int)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.Contains(line, "file changed") || strings.Contains(line, "files changed") {
			continue
		}
		pipeIdx := strings.LastIndex(line, "|")
		if pipeIdx < 0 {
			continue
		}

		filePath := diffparse.NormalizeStatPath(strings.TrimSpace(line[:pipeIdx]))
		changesPart := strings.TrimSpace(line[pipeIdx+1:])
		var a, d int
		for _, ch := range changesPart {
			switch ch {
			case '+':
				a++
			case '-':
				d++
			}
		}
		if filePath != "" {
			result[filePath] = [2]int{a, d}
		}
	}

	return result, nil
}

func normalizeTarget(target string) string {
	if target == "" {
		return "@"
	}
	return target
}

type DiffMsg struct {
	Content    string
	RawContent string
}

type EditorFinishedMsg struct{ Err error }
