package vcs

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

func withTempGitRepo(t *testing.T, fn func()) {
	t.Helper()

	repo := t.TempDir()
	t.Chdir(repo)

	runGit(t, "init", "-q")
	runGit(t, "config", "user.email", "test@example.com")
	runGit(t, "config", "user.name", "Test User")

	fn()
}

func withTempJjRepo(t *testing.T, fn func(root string)) {
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

func runGit(t *testing.T, args ...string) {
	t.Helper()

	cmd := exec.Command("git", append([]string{"--no-pager"}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %s failed\n%s", strings.Join(args, " "), out)
}

func TestDetectBackend_GitRepo(t *testing.T) {
	withTempGitRepo(t, func() {
		require.IsType(t, GitBackend{}, DetectBackend())
	})
}

func TestDetectBackend_GitNestedDirectory(t *testing.T) {
	withTempGitRepo(t, func() {
		subDir := filepath.Join(".", "subdir", "nested")
		require.NoError(t, os.MkdirAll(subDir, 0o755))
		t.Chdir(subDir)

		require.IsType(t, GitBackend{}, DetectBackend())
	})
}

func TestDetectBackend_JjRepo(t *testing.T) {
	withTempJjRepo(t, func(string) {
		require.IsType(t, JjBackend{}, DetectBackend())
	})
}

func TestDetectBackend_JjNestedDirectory(t *testing.T) {
	withTempJjRepo(t, func(root string) {
		subDir := filepath.Join(root, "subdir", "nested")
		require.NoError(t, os.MkdirAll(subDir, 0o755))
		t.Chdir(subDir)

		require.IsType(t, JjBackend{}, DetectBackend())
	})
}

func TestDetectBackend_NoVCSDefaultsToGit(t *testing.T) {
	t.Chdir(t.TempDir())

	require.IsType(t, GitBackend{}, DetectBackend())
}
