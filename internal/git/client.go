package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	diffparse "github.com/oug-t/difi/internal/diff"
)

var unifiedHunkHeaderRe = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

type fileVersion struct {
	Exists  bool
	Content []byte
}

type diffHunk struct {
	OldStart     int
	OldLen       int
	NewStart     int
	NewLen       int
	DisplayStart int
	DisplayEnd   int
}

func gitCmd(args ...string) *exec.Cmd {
	fullArgs := append([]string{"--no-pager"}, args...)
	cmd := exec.Command("git", fullArgs...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	return cmd
}

func RepoRoot() string {
	out, err := gitCmd("rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func GetCurrentBranch() string {
	out, err := gitCmd("rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "HEAD"
	}
	return strings.TrimSpace(string(out))
}

func GetRepoName() string {
	path := RepoRoot()
	if path == "" {
		return "Repo"
	}
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "Repo"
}

func ListChangedFiles(targetBranch string) ([]string, error) {
	cmd := gitCmd("diff", "--name-only", targetBranch)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	files := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}, nil
	}
	return files, nil
}

func DiffCmd(targetBranch, path string) tea.Cmd {
	return func() tea.Msg {
		coloredOut, err := gitCmd("diff", "--color=always", targetBranch, "--", path).Output()
		if err != nil {
			msg := "Error fetching diff: " + err.Error()
			return DiffMsg{Content: msg, RawContent: msg}
		}

		rawOut, err := gitCmd("diff", "--no-color", targetBranch, "--", path).Output()
		if err != nil {
			return DiffMsg{Content: string(coloredOut), RawContent: diffparse.StripANSI(string(coloredOut))}
		}

		return DiffMsg{Content: string(coloredOut), RawContent: string(rawOut)}
	}
}

func OpenEditorCmd(path string, lineNumber int, targetBranch string, editor string) tea.Cmd {
	var args []string
	if lineNumber > 0 {
		args = append(args, fmt.Sprintf("+%d", lineNumber))
	}
	args = append(args, path)

	c := exec.Command(editor, args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	c.Env = append(os.Environ(), fmt.Sprintf("DIFI_TARGET=%s", targetBranch))

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{Err: err}
	})
}

func DiffStats(targetBranch string) (added int, deleted int, err error) {
	cmd := gitCmd("diff", "--numstat", targetBranch)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("git diff stats error: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		if parts[0] != "-" {
			if n, err := strconv.Atoi(parts[0]); err == nil {
				added += n
			}
		}
		if parts[1] != "-" {
			if n, err := strconv.Atoi(parts[1]); err == nil {
				deleted += n
			}
		}
	}
	return added, deleted, nil
}

func DiffStatsByFile(targetBranch string) (map[string][2]int, error) {
	cmd := gitCmd("diff", "--numstat", targetBranch)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff numstat error: %w", err)
	}

	result := make(map[string][2]int)
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		var a, d int
		if parts[0] != "-" {
			a, _ = strconv.Atoi(parts[0])
		}
		if parts[1] != "-" {
			d, _ = strconv.Atoi(parts[1])
		}
		filePath := diffparse.NormalizeStatPath(strings.Join(parts[2:], " "))
		result[filePath] = [2]int{a, d}
	}
	return result, nil
}

func UndoSelectedChangeCmd(targetBranch, path, rawDiff string, cursorLine int) tea.Cmd {
	return func() tea.Msg {
		return undoSelectedChange(targetBranch, path, rawDiff, cursorLine)
	}
}

func undoSelectedChange(targetBranch, path, rawDiff string, cursorLine int) UndoResultMsg {
	if targetBranch != "HEAD" {
		return UndoResultMsg{Err: fmt.Errorf("undo is only supported when comparing against HEAD")}
	}

	filteredLines := extractRenderableDiffLines(rawDiff)
	if cursorLine < 0 || cursorLine >= len(filteredLines) {
		return UndoResultMsg{Err: fmt.Errorf("invalid diff selection")}
	}

	selectedLine := filteredLines[cursorLine]
	if !strings.HasPrefix(selectedLine, "+") && !strings.HasPrefix(selectedLine, "-") {
		return UndoResultMsg{Err: fmt.Errorf("undo only works on changed lines")}
	}

	hunks, err := parseDiffHunks(rawDiff)
	if err != nil {
		return UndoResultMsg{Err: err}
	}

	hunk, err := findHunkForCursor(hunks, cursorLine)
	if err != nil {
		return UndoResultMsg{Err: err}
	}

	headVersion, err := readGitVersion("HEAD:" + path)
	if err != nil {
		return UndoResultMsg{Err: err}
	}
	indexVersion, err := readGitVersion(":" + path)
	if err != nil {
		return UndoResultMsg{Err: err}
	}
	worktreeVersion, err := readWorktreeVersion(path)
	if err != nil {
		return UndoResultMsg{Err: err}
	}

	headStart := diffRangeStart(hunk.OldStart)
	worktreeStart := diffRangeStart(hunk.NewStart)

	replacement, err := sliceLines(headVersion, headStart, hunk.OldLen)
	if err != nil {
		return UndoResultMsg{Err: err}
	}

	desiredWorktree, err := replaceVersionLines(worktreeVersion, worktreeStart, hunk.NewLen, replacement, !headVersion.Exists)
	if err != nil {
		return UndoResultMsg{Err: err}
	}

	stagedDiff, err := gitCmd("diff", "--cached", "--no-color", "--unified=0", "--", path).Output()
	if err != nil {
		return UndoResultMsg{Err: fmt.Errorf("failed to inspect staged diff: %w", err)}
	}
	stagedHunks, err := parseDiffHunks(string(stagedDiff))
	if err != nil {
		return UndoResultMsg{Err: err}
	}

	indexStart, indexEnd, err := mapHeadRangeToIndexRange(stagedHunks, headStart, headStart+hunk.OldLen)
	if err != nil {
		return UndoResultMsg{Err: err}
	}

	desiredIndex, err := replaceVersionLines(indexVersion, indexStart, indexEnd-indexStart, replacement, !headVersion.Exists)
	if err != nil {
		return UndoResultMsg{Err: err}
	}

	mode := resolveGitMode(path)
	if err := writeWorktreeVersion(path, desiredWorktree, mode); err != nil {
		return UndoResultMsg{Err: err}
	}
	if err := writeIndexVersion(path, desiredIndex, mode); err != nil {
		return UndoResultMsg{Err: err}
	}

	return UndoResultMsg{Changed: true, Message: "Hunk undone"}
}

func ParseFilesFromDiff(diffText string) []string {
	return diffparse.ParseFiles(diffText)
}

func ExtractFileDiff(diffText, targetPath string) string {
	return diffparse.ExtractFile(diffText, targetPath)
}

func parseDiffHunks(diff string) ([]diffHunk, error) {
	lines := extractRenderableDiffLines(diff)
	var hunks []diffHunk

	for idx, line := range lines {
		matches := unifiedHunkHeaderRe.FindStringSubmatch(line)
		if len(matches) == 0 {
			continue
		}

		oldStart, oldLen, err := parseRange(matches[1], matches[2])
		if err != nil {
			return nil, err
		}
		newStart, newLen, err := parseRange(matches[3], matches[4])
		if err != nil {
			return nil, err
		}

		if len(hunks) > 0 && hunks[len(hunks)-1].DisplayEnd == 0 {
			hunks[len(hunks)-1].DisplayEnd = idx
		}

		hunks = append(hunks, diffHunk{
			OldStart:     oldStart,
			OldLen:       oldLen,
			NewStart:     newStart,
			NewLen:       newLen,
			DisplayStart: idx,
		})
	}

	if len(hunks) > 0 && hunks[len(hunks)-1].DisplayEnd == 0 {
		hunks[len(hunks)-1].DisplayEnd = len(lines)
	}

	return hunks, nil
}

func parseRange(startText, lenText string) (int, int, error) {
	start, err := strconv.Atoi(startText)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid hunk start %q: %w", startText, err)
	}

	length := 1
	if lenText != "" {
		length, err = strconv.Atoi(lenText)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid hunk length %q: %w", lenText, err)
		}
	}

	return start, length, nil
}

func findHunkForCursor(hunks []diffHunk, cursorLine int) (diffHunk, error) {
	for _, hunk := range hunks {
		if cursorLine >= hunk.DisplayStart && cursorLine < hunk.DisplayEnd {
			return hunk, nil
		}
	}
	return diffHunk{}, fmt.Errorf("no diff hunk found at cursor")
}

func mapHeadRangeToIndexRange(hunks []diffHunk, start, end int) (int, int, error) {
	indexStart, err := mapHeadBoundaryToIndex(hunks, start, true)
	if err != nil {
		return 0, 0, err
	}
	indexEnd, err := mapHeadBoundaryToIndex(hunks, end, false)
	if err != nil {
		return 0, 0, err
	}
	if indexEnd < indexStart {
		return 0, 0, fmt.Errorf("invalid index range after mapping")
	}
	return indexStart, indexEnd, nil
}

func mapHeadBoundaryToIndex(hunks []diffHunk, boundary int, isStart bool) (int, error) {
	delta := 0

	for _, hunk := range hunks {
		oldStart := diffRangeStart(hunk.OldStart)
		oldEnd := oldStart + hunk.OldLen
		newStart := diffRangeStart(hunk.NewStart)

		if boundary < oldStart {
			return boundary + delta, nil
		}
		if boundary == oldStart {
			if hunk.OldLen == 0 && !isStart {
				return newStart + hunk.NewLen, nil
			}
			return newStart, nil
		}
		if boundary > oldStart && boundary < oldEnd {
			return 0, fmt.Errorf("selected range cuts through a staged change")
		}
		if boundary == oldEnd {
			return newStart + hunk.NewLen, nil
		}

		delta += hunk.NewLen - hunk.OldLen
	}

	return boundary + delta, nil
}

func readGitVersion(spec string) (fileVersion, error) {
	cmd := gitCmd("show", spec)
	out, err := cmd.Output()
	if err != nil {
		if isMissingObjectError(err) {
			return fileVersion{}, nil
		}
		return fileVersion{}, fmt.Errorf("failed to read %s: %w", spec, err)
	}
	return fileVersion{Exists: true, Content: out}, nil
}

func readWorktreeVersion(path string) (fileVersion, error) {
	root := RepoRoot()
	if root == "" {
		return fileVersion{}, fmt.Errorf("failed to locate git repository root")
	}

	fullPath := filepath.Join(root, filepath.FromSlash(path))
	content, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fileVersion{}, nil
		}
		return fileVersion{}, fmt.Errorf("failed to read %s: %w", path, err)
	}

	return fileVersion{Exists: true, Content: content}, nil
}

func replaceVersionLines(version fileVersion, start, length int, replacement []string, removeWhenEmpty bool) (fileVersion, error) {
	lines := splitLines(version.Content)
	if !version.Exists && (start != 0 || length != 0) {
		return fileVersion{}, fmt.Errorf("cannot replace lines in a missing file")
	}
	if start < 0 || length < 0 || start > len(lines) || start+length > len(lines) {
		return fileVersion{}, fmt.Errorf("line range %d:%d is out of bounds", start, length)
	}

	newLines := append([]string{}, lines[:start]...)
	newLines = append(newLines, replacement...)
	newLines = append(newLines, lines[start+length:]...)

	if removeWhenEmpty && len(newLines) == 0 {
		return fileVersion{}, nil
	}

	return fileVersion{Exists: true, Content: joinLines(newLines)}, nil
}

func sliceLines(version fileVersion, start, length int) ([]string, error) {
	lines := splitLines(version.Content)
	if !version.Exists && length == 0 {
		return nil, nil
	}
	if !version.Exists {
		return nil, fmt.Errorf("cannot slice lines from a missing file")
	}
	if start < 0 || length < 0 || start > len(lines) || start+length > len(lines) {
		return nil, fmt.Errorf("line range %d:%d is out of bounds", start, length)
	}
	return append([]string{}, lines[start:start+length]...), nil
}

func splitLines(content []byte) []string {
	if len(content) == 0 {
		return nil
	}
	lines := strings.SplitAfter(string(content), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func joinLines(lines []string) []byte {
	return []byte(strings.Join(lines, ""))
}

func writeWorktreeVersion(path string, version fileVersion, mode string) error {
	root := RepoRoot()
	if root == "" {
		return fmt.Errorf("failed to locate git repository root")
	}

	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if !version.Exists {
		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", path, err)
	}

	if err := os.WriteFile(fullPath, version.Content, gitModeToFileMode(mode)); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

func writeIndexVersion(path string, version fileVersion, mode string) error {
	if !version.Exists {
		if getIndexMode(path) == "" {
			return nil
		}
		if err := gitCmd("update-index", "--remove", "--", path).Run(); err != nil {
			return fmt.Errorf("failed to remove %s from index: %w", path, err)
		}
		return nil
	}

	sha, err := hashBlob(version.Content)
	if err != nil {
		return err
	}

	arg := fmt.Sprintf("%s,%s,%s", mode, sha, path)
	if err := gitCmd("update-index", "--add", "--cacheinfo", arg).Run(); err != nil {
		return fmt.Errorf("failed to update %s in index: %w", path, err)
	}
	return nil
}

func hashBlob(content []byte) (string, error) {
	cmd := gitCmd("hash-object", "-w", "--stdin")
	cmd.Stdin = bytes.NewReader(content)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to store git blob: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func resolveGitMode(path string) string {
	if mode := getIndexMode(path); mode != "" {
		return mode
	}
	if mode := getHeadMode(path); mode != "" {
		return mode
	}

	root := RepoRoot()
	if root != "" {
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err == nil && info.Mode()&0o111 != 0 {
			return "100755"
		}
	}

	return "100644"
}

func getIndexMode(path string) string {
	out, err := gitCmd("ls-files", "-s", "--", path).Output()
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func getHeadMode(path string) string {
	out, err := gitCmd("ls-tree", "HEAD", "--", path).Output()
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func gitModeToFileMode(mode string) os.FileMode {
	if mode == "100755" {
		return 0o755
	}
	return 0o644
}

func isMissingObjectError(err error) bool {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}
	stderr := string(exitErr.Stderr)
	return strings.Contains(stderr, "does not exist in") ||
		strings.Contains(stderr, "exists on disk, but not in") ||
		strings.Contains(stderr, "not in the index")
}

func diffRangeStart(start int) int {
	if start <= 0 {
		return 0
	}
	return start - 1
}

func extractRenderableDiffLines(diff string) []string {
	lines := strings.Split(diff, "\n")
	var cleanLines []string
	foundHunk := false

	for _, line := range lines {
		if unifiedHunkHeaderRe.MatchString(line) {
			foundHunk = true
		}
		if !foundHunk {
			continue
		}
		cleanLines = append(cleanLines, line)
	}

	return cleanLines
}

type DiffMsg struct {
	Content    string
	RawContent string
}

type EditorFinishedMsg struct{ Err error }

type UndoResultMsg struct {
	Err     error
	Changed bool
	Message string
}
