package main

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"--version"}, nil, &stdout, &stderr)

	require.Zero(t, exitCode)
	require.Equal(t, "difi version "+version+"\n", stdout.String())
	require.Empty(t, stderr.String())
}

func TestRunRejectsInvalidVCS(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"--vcs", "svn"}, nil, &stdout, &stderr)

	require.Equal(t, 1, exitCode)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "unsupported VCS 'svn'")
}

func TestReadPipedDiff(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stdin.diff")
	want := "diff --git a/a.txt b/a.txt\n+hello\n"
	require.NoError(t, os.WriteFile(path, []byte(want), 0o644))

	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()

	require.Equal(t, want, readPipedDiff(file))
}

func TestRunPlainForcedGitListsChangedFiles(t *testing.T) {
	withTempRepo(t, func() {
		writeNotesFile(t, "one\n")
		runGit(t, "add", "notes.txt")
		runGit(t, "commit", "-q", "-m", "init")

		writeNotesFile(t, "one\ntwo\n")

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		exitCode := run([]string{"--vcs", "git", "--plain"}, nil, &stdout, &stderr)

		require.Zero(t, exitCode, "stderr = %q", stderr.String())
		require.Equal(t, "notes.txt\n", stdout.String())
		require.Empty(t, stderr.String())
	})
}

func TestRunPlainUsesTargetFlag(t *testing.T) {
	withTempRepo(t, func() {
		writeNotesFile(t, "one\n")
		runGit(t, "add", "notes.txt")
		runGit(t, "commit", "-q", "-m", "init")

		writeNotesFile(t, "one\ntwo\n")
		runGit(t, "add", "notes.txt")
		runGit(t, "commit", "-q", "-m", "second")

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		exitCode := run([]string{"--vcs", "git", "--plain", "--target", "HEAD~1"}, nil, &stdout, &stderr)

		require.Zero(t, exitCode, "stderr = %q", stderr.String())
		require.Equal(t, "notes.txt\n", stdout.String())
	})
}

func TestRunPlainUsesPositionalPath(t *testing.T) {
	withTempRepo(t, func() {
		writeNotesFile(t, "one\n")
		require.NoError(t, os.MkdirAll("docs", 0o755))
		require.NoError(t, os.WriteFile("docs/readme.md", []byte("base\n"), 0o644))
		runGit(t, "add", "notes.txt", "docs/readme.md")
		runGit(t, "commit", "-q", "-m", "init")

		writeNotesFile(t, "one\ntwo\n")
		require.NoError(t, os.WriteFile("docs/readme.md", []byte("base\nnext\n"), 0o644))

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		exitCode := run([]string{"--vcs", "git", "--plain", "docs"}, nil, &stdout, &stderr)

		require.Zero(t, exitCode, "stderr = %q", stderr.String())
		require.Equal(t, "docs/readme.md\n", stdout.String())
	})
}

func TestRunPlainAcceptsTargetAfterPath(t *testing.T) {
	withTempRepo(t, func() {
		writeNotesFile(t, "one\n")
		runGit(t, "add", "notes.txt")
		runGit(t, "commit", "-q", "-m", "init")

		writeNotesFile(t, "one\ntwo\n")
		runGit(t, "add", "notes.txt")
		runGit(t, "commit", "-q", "-m", "second")

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		exitCode := run([]string{"notes.txt", "--vcs", "git", "--plain", "--target", "HEAD~1"}, nil, &stdout, &stderr)

		require.Zero(t, exitCode, "stderr = %q", stderr.String())
		require.Equal(t, "notes.txt\n", stdout.String())
	})
}

func TestRunPlainAllowsDashedPathAfterSeparator(t *testing.T) {
	withTempRepo(t, func() {
		require.NoError(t, os.MkdirAll("--target", 0o755))
		require.NoError(t, os.WriteFile("--target/readme.md", []byte("base\n"), 0o644))
		runGit(t, "add", "--", "--target/readme.md")
		runGit(t, "commit", "-q", "-m", "init")

		require.NoError(t, os.WriteFile("--target/readme.md", []byte("base\nnext\n"), 0o644))

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		exitCode := run([]string{"--vcs", "git", "--plain", "--", "--target"}, nil, &stdout, &stderr)

		require.Zero(t, exitCode, "stderr = %q", stderr.String())
		require.Equal(t, "--target/readme.md\n", stdout.String())
	})
}

func TestRunRejectsMultiplePositionalPaths(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"a", "b"}, nil, &stdout, &stderr)

	require.Equal(t, 2, exitCode)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "expected at most one path argument")
}

func TestRunPlainForcedJjListsChangedFiles(t *testing.T) {
	withTempJjRepo(t, func() {
		writeNotesFile(t, "one\n")

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		exitCode := run([]string{"--vcs", "jj", "--plain"}, nil, &stdout, &stderr)

		require.Zero(t, exitCode, "stderr = %q", stderr.String())
		require.Equal(t, "notes.txt\n", stdout.String())
		require.Empty(t, stderr.String())
	})
}

func TestRunPlainAutoDetectsJjRepo(t *testing.T) {
	withTempJjRepo(t, func() {
		writeNotesFile(t, "one\n")

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		exitCode := run([]string{"--plain"}, nil, &stdout, &stderr)

		require.Zero(t, exitCode, "stderr = %q", stderr.String())
		require.Equal(t, "notes.txt\n", stdout.String())
		require.Empty(t, stderr.String())
	})
}

func TestNormalizeInterspersedFlags(t *testing.T) {
	args := []string{"src", "--plain", "--target", "HEAD~1", "--vcs=git"}
	require.Equal(t, []string{"--plain", "--target", "HEAD~1", "--vcs=git", "src"}, normalizeInterspersedFlags(testFlags(), args))
}

func TestNormalizeInterspersedFlagsPreservesSeparator(t *testing.T) {
	args := []string{"src", "--plain", "--", "--target", "HEAD~1"}
	require.Equal(t, []string{"--plain", "--", "src", "--target", "HEAD~1"}, normalizeInterspersedFlags(testFlags(), args))
}

func TestNormalizeInterspersedFlagsKeepsEqualsFormSelfContained(t *testing.T) {
	args := []string{"src", "--target=HEAD~1", "--vcs", "git"}
	require.Equal(t, []string{"--target=HEAD~1", "--vcs", "git", "src"}, normalizeInterspersedFlags(testFlags(), args))
}

func testFlags() *flag.FlagSet {
	flags := flag.NewFlagSet("difi", flag.ContinueOnError)
	flags.Bool("version", false, "Show version")
	flags.Bool("plain", false, "Print a plain summary")
	flags.String("vcs", "", "Force specific VCS (git or jj)")
	flags.String("target", "", "Review a specific target")
	return flags
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

func withTempJjRepo(t *testing.T, fn func()) {
	t.Helper()

	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj binary not available")
	}

	base := t.TempDir()
	cmd := exec.Command("jj", "git", "init", "repo")
	cmd.Dir = base
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "jj git init failed\n%s", out)

	repo := filepath.Join(base, "repo")
	t.Chdir(repo)
	fn()
}

func runGit(t *testing.T, args ...string) {
	t.Helper()

	cmd := exec.Command("git", append([]string{"--no-pager"}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %s failed\n%s", strings.Join(args, " "), out)
}

func writeNotesFile(t *testing.T, content string) {
	t.Helper()

	path := "notes.txt"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		require.NoError(t, err)
	}
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
