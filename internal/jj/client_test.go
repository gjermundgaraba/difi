package jj

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func requireJujutsu(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj binary not available")
	}
}

func withTempRepo(t *testing.T, fn func(root string)) {
	t.Helper()
	requireJujutsu(t)

	base := t.TempDir()
	cmd := exec.Command("jj", "git", "init", "repo")
	cmd.Dir = base
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "jj git init failed\n%s", out)

	root := filepath.Join(base, "repo")
	t.Chdir(root)
	fn(root)
}

func TestRepoRootAndRepoName(t *testing.T) {
	withTempRepo(t, func(root string) {
		expectedRoot, err := filepath.EvalSymlinks(root)
		require.NoError(t, err)
		actualRoot, err := filepath.EvalSymlinks(RepoRoot())
		require.NoError(t, err)
		require.Equal(t, expectedRoot, actualRoot)
		require.Equal(t, "repo", GetRepoName())
	})
}

func TestListChangedFilesUsesTargetRevision(t *testing.T) {
	withTempRepo(t, func(string) {
		writeFile(t, "notes.txt", "one\n")

		files, err := ListChangedFiles("@")
		require.NoError(t, err)
		require.Equal(t, []string{"notes.txt"}, files)
	})
}

func TestDiffStatsHandlesInsertionOnlyAndDeletionOnly(t *testing.T) {
	t.Run("insertion only", func(t *testing.T) {
		withTempRepo(t, func(string) {
			writeFile(t, "notes.txt", "one\n")

			added, deleted, err := DiffStats("@")
			require.NoError(t, err)
			require.Equal(t, 1, added)
			require.Zero(t, deleted)
		})
	})

	t.Run("deletion only", func(t *testing.T) {
		withTempRepo(t, func(string) {
			writeFile(t, "notes.txt", "one\ntwo\n")
			runJj(t, "new", "-m", "base")
			writeFile(t, "notes.txt", "one\n")

			added, deleted, err := DiffStats("@")
			require.NoError(t, err)
			require.Zero(t, added)
			require.Equal(t, 1, deleted)
		})
	})
}

func TestDiffStatsByFileNormalizesRenamePaths(t *testing.T) {
	withTempRepo(t, func(string) {
		writeFile(t, "old.txt", "one\n")
		runJj(t, "new", "-m", "base")
		runJj(t, "bookmark", "create", "main", "-r", "@")
		require.NoError(t, os.Rename("old.txt", "new.txt"))

		stats, err := DiffStatsByFile("@")
		require.NoError(t, err)
		require.Contains(t, stats, "new.txt")
		require.Equal(t, [2]int{0, 0}, stats["new.txt"])
	})
}

func TestDiffCmdReturnsGitFormatRawContent(t *testing.T) {
	withTempRepo(t, func(string) {
		writeFile(t, "notes.txt", "one\n")

		msg := DiffCmd("@", "notes.txt")()
		diffMsg, ok := msg.(DiffMsg)
		require.True(t, ok)
		require.Contains(t, diffMsg.RawContent, "diff --git a/notes.txt b/notes.txt")
		require.Contains(t, diffMsg.RawContent, "@@")
	})
}

func TestRepoRootCacheTracksCurrentWorkingDirectory(t *testing.T) {
	requireJujutsu(t)
	clearRepoRootCache()

	base := t.TempDir()
	repoOne := initRepo(t, base, "one")
	repoTwo := initRepo(t, base, "two")

	t.Chdir(repoOne)
	rootOne := RepoRoot()
	require.Equal(t, repoOne, rootOne)

	t.Chdir(repoTwo)
	rootTwo := RepoRoot()
	require.Equal(t, repoTwo, rootTwo)
	require.NotEqual(t, rootOne, rootTwo)

	t.Chdir(repoOne)
	require.Equal(t, rootOne, RepoRoot())
}

func runJj(t *testing.T, args ...string) {
	t.Helper()

	cmd := exec.Command("jj", append([]string{"--no-pager"}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "jj %s failed\n%s", strings.Join(args, " "), out)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func initRepo(t *testing.T, base, name string) string {
	t.Helper()

	cmd := exec.Command("jj", "git", "init", name)
	cmd.Dir = base
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "jj git init failed\n%s", out)

	root := filepath.Join(base, name)
	resolvedRoot, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)
	return resolvedRoot
}

func clearRepoRootCache() {
	repoRootCache.Lock()
	defer repoRootCache.Unlock()

	repoRootCache.byDir = make(map[string]string)
}
