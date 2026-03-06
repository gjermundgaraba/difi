package git

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDiffHunksTracksDisplayRanges(t *testing.T) {
	diff := `diff --git a/f.txt b/f.txt
index 1234567..89abcde 100644
--- a/f.txt
+++ b/f.txt
@@ -1,3 +1,3 @@
-one
+ONE
 two
 three
@@ -6 +6 @@
-six
+SIX
`

	hunks, err := parseDiffHunks(diff)
	require.NoError(t, err)
	require.Len(t, hunks, 2)
	require.Equal(t, 0, hunks[0].DisplayStart)
	require.Equal(t, 5, hunks[0].DisplayEnd)
	require.Equal(t, 5, hunks[1].DisplayStart)
	require.Equal(t, 9, hunks[1].DisplayEnd)
}

func TestMapHeadRangeToIndexRangeHandlesInsertionOnlyHunk(t *testing.T) {
	hunks := []diffHunk{
		{OldStart: 0, OldLen: 0, NewStart: 1, NewLen: 3},
	}

	start, end, err := mapHeadRangeToIndexRange(hunks, 0, 0)
	require.NoError(t, err)
	require.Equal(t, 0, start)
	require.Equal(t, 3, end)
}

func TestMapHeadRangeToIndexRangeRejectsPartialOverlap(t *testing.T) {
	hunks := []diffHunk{
		{OldStart: 4, OldLen: 2, NewStart: 4, NewLen: 3},
	}

	_, _, err := mapHeadRangeToIndexRange(hunks, 4, 5)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cuts through a staged change")
}

func TestUndoSelectedChange(t *testing.T) {
	t.Run("unstaged hunk", func(t *testing.T) {
		withTempRepo(t, func() {
			writeFile(t, "f.txt", "one\ntwo\nthree\n")
			runGit(t, "add", "f.txt")
			runGit(t, "commit", "-q", "-m", "init")

			writeFile(t, "f.txt", "one\ntwo changed\nthree\n")

			rawDiff := runGit(t, "diff", "--no-color", "HEAD", "--", "f.txt")
			cursor := mustFindRenderedLine(t, rawDiff, "+two changed")

			msg := undoSelectedChange("HEAD", "f.txt", rawDiff, cursor)
			require.NoError(t, msg.Err)
			require.True(t, msg.Changed)

			assertNoGitDiff(t, "f.txt")
			assertFileContent(t, "f.txt", "one\ntwo\nthree\n")
		})
	})

	t.Run("staged hunk", func(t *testing.T) {
		withTempRepo(t, func() {
			writeFile(t, "f.txt", "one\ntwo\nthree\n")
			runGit(t, "add", "f.txt")
			runGit(t, "commit", "-q", "-m", "init")

			writeFile(t, "f.txt", "one\ntwo staged\nthree\n")
			runGit(t, "add", "f.txt")

			rawDiff := runGit(t, "diff", "--no-color", "HEAD", "--", "f.txt")
			cursor := mustFindRenderedLine(t, rawDiff, "+two staged")

			msg := undoSelectedChange("HEAD", "f.txt", rawDiff, cursor)
			require.NoError(t, msg.Err)

			assertNoGitDiff(t, "f.txt")
			assertFileContent(t, "f.txt", "one\ntwo\nthree\n")
		})
	})

	t.Run("combined staged and unstaged hunk", func(t *testing.T) {
		withTempRepo(t, func() {
			writeFile(t, "f.txt", "1\n2\n3\n4\n5\n6\n7\n8\n")
			runGit(t, "add", "f.txt")
			runGit(t, "commit", "-q", "-m", "init")

			writeFile(t, "f.txt", "1\n2 staged\n3\n4\n5\n6\n7\n8\n")
			runGit(t, "add", "f.txt")
			writeFile(t, "f.txt", "1\n2 staged\n3\n4\n5\n6\n7 unstaged\n8\n")

			rawDiff := runGit(t, "diff", "--no-color", "HEAD", "--", "f.txt")
			cursor := mustFindRenderedLine(t, rawDiff, "+2 staged")

			msg := undoSelectedChange("HEAD", "f.txt", rawDiff, cursor)
			require.NoError(t, msg.Err)

			assertNoGitDiff(t, "f.txt")
			assertFileContent(t, "f.txt", "1\n2\n3\n4\n5\n6\n7\n8\n")
		})
	})

	t.Run("staged new file removes file", func(t *testing.T) {
		withTempRepo(t, func() {
			writeFile(t, "tracked.txt", "base\n")
			runGit(t, "add", "tracked.txt")
			runGit(t, "commit", "-q", "-m", "init")

			writeFile(t, "new.txt", "hello\nworld\n")
			runGit(t, "add", "new.txt")

			rawDiff := runGit(t, "diff", "--no-color", "HEAD", "--", "new.txt")
			cursor := mustFindRenderedLine(t, rawDiff, "+hello")

			msg := undoSelectedChange("HEAD", "new.txt", rawDiff, cursor)
			require.NoError(t, msg.Err)

			assertNoGitDiff(t, "new.txt")
			_, err := os.Stat("new.txt")
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})

	t.Run("deleted file restores file", func(t *testing.T) {
		withTempRepo(t, func() {
			writeFile(t, "gone.txt", "restore me\n")
			runGit(t, "add", "gone.txt")
			runGit(t, "commit", "-q", "-m", "init")

			require.NoError(t, os.Remove("gone.txt"))

			rawDiff := runGit(t, "diff", "--no-color", "HEAD", "--", "gone.txt")
			cursor := mustFindRenderedLine(t, rawDiff, "-restore me")

			msg := undoSelectedChange("HEAD", "gone.txt", rawDiff, cursor)
			require.NoError(t, msg.Err)

			assertNoGitDiff(t, "gone.txt")
			assertFileContent(t, "gone.txt", "restore me\n")
		})
	})

	t.Run("executable file keeps mode", func(t *testing.T) {
		withTempRepo(t, func() {
			writeFile(t, "script.sh", "#!/bin/sh\necho one\n")
			require.NoError(t, os.Chmod("script.sh", 0o755))
			runGit(t, "add", "script.sh")
			runGit(t, "commit", "-q", "-m", "init")

			writeFile(t, "script.sh", "#!/bin/sh\necho two\n")
			require.NoError(t, os.Chmod("script.sh", 0o755))

			rawDiff := runGit(t, "diff", "--no-color", "HEAD", "--", "script.sh")
			cursor := mustFindRenderedLine(t, rawDiff, "+echo two")

			msg := undoSelectedChange("HEAD", "script.sh", rawDiff, cursor)
			require.NoError(t, msg.Err)

			info, err := os.Stat("script.sh")
			require.NoError(t, err)
			require.NotZero(t, info.Mode()&0o111)
		})
	})
}

func TestListChangedFiles(t *testing.T) {
	withTempRepo(t, func() {
		writeFile(t, "alpha.txt", "one\n")
		writeFile(t, "dir/file name.txt", "two\n")
		runGit(t, "add", "alpha.txt", "dir/file name.txt")
		runGit(t, "commit", "-q", "-m", "init")

		files, err := ListChangedFiles("HEAD")
		require.NoError(t, err)
		require.Empty(t, files)

		writeFile(t, "alpha.txt", "one\ntwo\n")
		writeFile(t, "dir/file name.txt", "two\nthree\n")

		files, err = ListChangedFiles("HEAD")
		require.NoError(t, err)

		sort.Strings(files)
		want := []string{"alpha.txt", "dir/file name.txt"}
		require.Equal(t, want, files)
	})
}

func TestDiffStatsIgnoresBinaryNumstatRows(t *testing.T) {
	withTempRepo(t, func() {
		writeFile(t, "text.txt", "one\ntwo\nthree\n")
		writeBinaryFile(t, []byte{0x00, 0x01, 0x02, 0x03})
		runGit(t, "add", "text.txt", "image.bin")
		runGit(t, "commit", "-q", "-m", "init")

		writeFile(t, "text.txt", "zero\none\nthree\nfour\n")
		writeBinaryFile(t, []byte{0x00, 0x09, 0x02, 0x03, 0x04})

		added, deleted, err := DiffStats("HEAD")
		require.NoError(t, err)
		require.Equal(t, 2, added)
		require.Equal(t, 1, deleted)
	})
}

func TestDiffStatsByFileHandlesRenamesAndBinaryFiles(t *testing.T) {
	withTempRepo(t, func() {
		writeFile(t, "docs/old name.txt", "one\ntwo\n")
		writeBinaryFile(t, []byte{0x00, 0x01, 0x02})
		runGit(t, "add", "docs/old name.txt", "image.bin")
		runGit(t, "commit", "-q", "-m", "init")

		runGit(t, "mv", "docs/old name.txt", "docs/new name.txt")
		writeFile(t, "docs/new name.txt", "one\nTWO\n")
		writeBinaryFile(t, []byte{0x00, 0x09, 0x02, 0x03})

		stats, err := DiffStatsByFile("HEAD")
		require.NoError(t, err)

		renamed, ok := stats["docs/new name.txt"]
		require.Truef(t, ok, "DiffStatsByFile() missing renamed file entry, got %#v", stats)
		require.Equal(t, [2]int{1, 1}, renamed)
		require.Equal(t, [2]int{0, 0}, stats["image.bin"])
	})
}

func TestParseFilesFromDiff(t *testing.T) {
	diffText := `diff --git a/file1.go b/file1.go
index 1234567..89abcde 100644
--- a/file1.go
+++ b/file1.go
@@ -1 +1 @@
-old
+new
diff --git a/dir/file two.txt b/dir/file two.txt
index 7654321..fedcba9 100644
--- a/dir/file two.txt
+++ b/dir/file two.txt
@@ -1 +1 @@
-left
+right
diff --git a/file1.go b/file1.go
index 1234567..89abcde 100644
`

	files := ParseFilesFromDiff(diffText)
	want := []string{"file1.go", "dir/file two.txt"}
	require.Equal(t, want, files)
}

func TestExtractFileDiff(t *testing.T) {
	diffText := `diff --git a/file1.go b/file1.go
index 1234567..89abcde 100644
--- a/file1.go
+++ b/file1.go
@@ -1 +1 @@
-old
+new
diff --git a/dir/file two.txt b/dir/file two.txt
index 7654321..fedcba9 100644
--- a/dir/file two.txt
+++ b/dir/file two.txt
@@ -1 +1 @@
-left
+right
`

	got := strings.TrimSpace(ExtractFileDiff(diffText, "dir/file two.txt"))
	want := strings.TrimSpace(`diff --git a/dir/file two.txt b/dir/file two.txt
index 7654321..fedcba9 100644
--- a/dir/file two.txt
+++ b/dir/file two.txt
@@ -1 +1 @@
-left
+right`)

	require.Equal(t, want, got)
	require.Empty(t, strings.TrimSpace(ExtractFileDiff(diffText, "missing.txt")))
}

func withTempRepo(t *testing.T, fn func()) {
	t.Helper()

	repo := t.TempDir()
	t.Chdir(repo)

	runGit(t, "init", "-q")
	runGit(t, "config", "user.email", "test@example.com")
	runGit(t, "config", "user.name", "Test User")

	fn()
}

func runGit(t *testing.T, args ...string) string {
	t.Helper()

	cmd := gitCmd(args...)
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %s failed\n%s", strings.Join(args, " "), out)
	return string(out)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func writeBinaryFile(t *testing.T, content []byte) {
	t.Helper()

	const path = "image.bin"
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, content, 0o644))
}

func mustFindRenderedLine(t *testing.T, rawDiff, line string) int {
	t.Helper()

	lines := extractRenderableDiffLines(rawDiff)
	for idx, candidate := range lines {
		if candidate == line {
			return idx
		}
	}
	require.FailNowf(t, "line not found", "line %q not found in rendered diff:\n%s", line, rawDiff)
	return 0
}

func assertNoGitDiff(t *testing.T, path string) {
	t.Helper()

	require.Emptyf(t, strings.TrimSpace(runGit(t, "diff", "--no-color", "HEAD", "--", path)), "expected no git diff for %s", path)
	require.Emptyf(t, strings.TrimSpace(runGit(t, "diff", "--cached", "--no-color", "--", path)), "expected no staged diff for %s", path)
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, want, string(got))
}
