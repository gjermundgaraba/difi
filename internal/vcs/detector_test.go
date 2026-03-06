package vcs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectVCS_GitPriority(t *testing.T) {
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	hgDir := filepath.Join(tempDir, ".hg")

	require.NoError(t, os.Mkdir(gitDir, 0o755))
	require.NoError(t, os.Mkdir(hgDir, 0o755))
	t.Chdir(tempDir)

	require.IsType(t, GitVCS{}, DetectVCS())
}

func TestDetectVCS_GitOnly(t *testing.T) {
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	require.NoError(t, os.Mkdir(gitDir, 0o755))
	t.Chdir(tempDir)

	require.IsType(t, GitVCS{}, DetectVCS())
}

func TestDetectVCS_HgOnly(t *testing.T) {
	tempDir := t.TempDir()
	hgDir := filepath.Join(tempDir, ".hg")
	require.NoError(t, os.Mkdir(hgDir, 0o755))
	t.Chdir(tempDir)

	require.IsType(t, HgVCS{}, DetectVCS())
}

func TestDetectVCS_NoVCS(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	require.IsType(t, GitVCS{}, DetectVCS())
}

func TestDetectVCS_NestedDirectories(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir", "nested")

	require.NoError(t, os.MkdirAll(subDir, 0o755))
	gitDir := filepath.Join(tempDir, ".git")
	require.NoError(t, os.Mkdir(gitDir, 0o755))
	t.Chdir(subDir)

	require.IsType(t, GitVCS{}, DetectVCS())
}

func TestDetectVCS_GitInParentHgInChild(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")

	require.NoError(t, os.MkdirAll(subDir, 0o755))
	gitDir := filepath.Join(tempDir, ".git")
	require.NoError(t, os.Mkdir(gitDir, 0o755))
	hgDir := filepath.Join(subDir, ".hg")
	require.NoError(t, os.Mkdir(hgDir, 0o755))
	t.Chdir(subDir)

	require.IsType(t, GitVCS{}, DetectVCS())
}

func TestVCSInterface_GitVCS(t *testing.T) {
	var vcs VCS = GitVCS{}

	_ = vcs.GetCurrentBranch()
	_ = vcs.GetRepoName()
	_, _ = vcs.ListChangedFiles("main")
}

func TestVCSInterface_HgVCS(t *testing.T) {
	var vcs VCS = HgVCS{}

	_ = vcs.GetCurrentBranch()
	_ = vcs.GetRepoName()
	_, _ = vcs.ListChangedFiles("default")
}

func TestDetectVCS_ErrorHandling(t *testing.T) {
	vcs := DetectVCS()
	require.NotNil(t, vcs)
	if _, ok := vcs.(GitVCS); !ok {
		if _, ok := vcs.(HgVCS); !ok {
			require.Failf(t, "unexpected VCS type", "DetectVCS() returned %T", vcs)
		}
	}
}
