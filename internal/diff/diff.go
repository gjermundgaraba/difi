package diff

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	ansiRe       = regexp.MustCompile(`[\x1b\x9b][[\]()#;?]*(?:(?:(?:[a-zA-Z\d]*(?:;[a-zA-Z\d]*)*)?\x07)|(?:(?:\d{1,4}(?:;\d{0,4})*)?[\dA-PRZcf-ntqry=><~]))`)
	hunkHeaderRe = regexp.MustCompile(`^.*?@@ \-\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)
)

func ParseFiles(diffText string) []string {
	var files []string
	seen := make(map[string]bool)

	for _, line := range strings.Split(diffText, "\n") {
		if !strings.HasPrefix(line, "diff --git a/") {
			continue
		}
		parts := strings.SplitN(line, " b/", 2)
		if len(parts) != 2 {
			continue
		}
		file := strings.TrimPrefix(parts[0], "diff --git a/")
		if seen[file] {
			continue
		}
		seen[file] = true
		files = append(files, file)
	}

	return files
}

func ExtractFile(diffText, targetPath string) string {
	lines := strings.Split(diffText, "\n")
	var out []string
	inTarget := false
	targetHeader := fmt.Sprintf("diff --git a/%s b/%s", targetPath, targetPath)

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			inTarget = strings.HasPrefix(line, targetHeader)
		}
		if inTarget {
			out = append(out, line)
		}
	}

	return strings.Join(out, "\n")
}

func CalculateFileLine(diffContent string, visualLineIndex int) int {
	lines := strings.Split(diffContent, "\n")
	if visualLineIndex >= len(lines) {
		return 0
	}

	currentLineNo := 0
	lastWasHunk := false
	inHeader := true

	for i := 0; i <= visualLineIndex; i++ {
		line := lines[i]
		matches := hunkHeaderRe.FindStringSubmatch(line)
		if len(matches) > 1 {
			startLine, _ := strconv.Atoi(matches[1])
			currentLineNo = startLine
			lastWasHunk = true
			inHeader = false
			continue
		}

		lastWasHunk = false
		cleanLine := StripANSI(line)

		if inHeader {
			continue
		}

		if strings.HasPrefix(cleanLine, " ") || strings.HasPrefix(cleanLine, "+") {
			currentLineNo++
		}
	}

	if currentLineNo == 0 {
		return 1
	}
	if lastWasHunk {
		return currentLineNo - 1
	}
	return currentLineNo - 1
}

func Stats(diffText string) (added int, deleted int, byFile map[string][2]int) {
	byFile = make(map[string][2]int)
	currentFile := ""

	for _, line := range strings.Split(diffText, "\n") {
		clean := StripANSI(line)
		if strings.HasPrefix(clean, "diff --git ") {
			parts := strings.Fields(clean)
			if len(parts) >= 4 {
				currentFile = strings.TrimPrefix(parts[3], "b/")
			} else {
				currentFile = ""
			}
			continue
		}
		if currentFile == "" {
			continue
		}
		if strings.HasPrefix(clean, "+") && !strings.HasPrefix(clean, "+++") {
			stats := byFile[currentFile]
			stats[0]++
			byFile[currentFile] = stats
			added++
			continue
		}
		if strings.HasPrefix(clean, "-") && !strings.HasPrefix(clean, "---") {
			stats := byFile[currentFile]
			stats[1]++
			byFile[currentFile] = stats
			deleted++
		}
	}

	return added, deleted, byFile
}

func NormalizePathScope(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	path = filepath.ToSlash(filepath.Clean(path))
	if path == "." {
		return ""
	}

	return strings.TrimPrefix(path, "./")
}

func PathMatchesScope(path, scope string) bool {
	scope = NormalizePathScope(scope)
	if scope == "" {
		return true
	}

	path = NormalizePathScope(path)
	return path == scope || strings.HasPrefix(path, scope+"/")
}

func FilterPaths(paths []string, scope string) []string {
	scope = NormalizePathScope(scope)
	if scope == "" {
		return append([]string{}, paths...)
	}

	filtered := make([]string, 0, len(paths))
	for _, path := range paths {
		if PathMatchesScope(path, scope) {
			filtered = append(filtered, path)
		}
	}
	return filtered
}

func FilterStats(byFile map[string][2]int, scope string) (added int, deleted int, filtered map[string][2]int) {
	scope = NormalizePathScope(scope)
	filtered = make(map[string][2]int)

	for path, stats := range byFile {
		if !PathMatchesScope(path, scope) {
			continue
		}
		filtered[path] = stats
		added += stats[0]
		deleted += stats[1]
	}

	return added, deleted, filtered
}

func AppendPathScope(args []string, scope string) []string {
	scope = NormalizePathScope(scope)
	if scope == "" {
		return args
	}
	return append(args, "--", scope)
}

func NormalizeStatPath(path string) string {
	if !strings.Contains(path, " => ") {
		return path
	}

	if open := strings.Index(path, "{"); open != -1 {
		if end := strings.Index(path[open:], "}"); end != -1 {
			end += open
			rename := path[open+1 : end]
			parts := strings.SplitN(rename, " => ", 2)
			if len(parts) == 2 {
				return path[:open] + parts[1] + path[end+1:]
			}
		}
	}

	parts := strings.SplitN(path, " => ", 2)
	if len(parts) == 2 {
		return parts[1]
	}

	return path
}

func StripANSI(str string) string {
	return ansiRe.ReplaceAllString(str, "")
}
