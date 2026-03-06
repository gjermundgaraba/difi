package vcs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVCSInterfaceConsistency(t *testing.T) {
	implementations := []struct {
		name string
		vcs  VCS
	}{
		{"Git", GitVCS{}},
		{"Mercurial", HgVCS{}},
	}

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			vcs := impl.vcs

			t.Run("GetCurrentBranch", func(t *testing.T) {
				var branch string
				require.NotPanics(t, func() {
					branch = vcs.GetCurrentBranch()
				})
				if branch == "" {
					t.Logf("%s GetCurrentBranch() returned empty string", impl.name)
				}
			})

			t.Run("GetRepoName", func(t *testing.T) {
				var repoName string
				require.NotPanics(t, func() {
					repoName = vcs.GetRepoName()
				})
				if repoName == "" {
					t.Logf("%s GetRepoName() returned empty string", impl.name)
				}
			})

			t.Run("ListChangedFiles", func(t *testing.T) {
				testBranches := []string{"main", "master", "default", "HEAD"}
				require.NotPanics(t, func() {
					for _, branch := range testBranches {
						_, _ = vcs.ListChangedFiles(branch)
					}
				})
			})

			t.Run("DiffStats", func(t *testing.T) {
				require.NotPanics(t, func() {
					_, _, _ = vcs.DiffStats("main")
				})
			})

			t.Run("DiffStatsByFile", func(t *testing.T) {
				require.NotPanics(t, func() {
					_, _ = vcs.DiffStatsByFile("main")
				})
			})

			t.Run("ParseFilesFromDiff", func(t *testing.T) {
				files := vcs.ParseFilesFromDiff("")
				require.Empty(t, files)
				files = vcs.ParseFilesFromDiff("not a diff")
				require.Empty(t, files)
			})

			t.Run("ExtractFileDiff", func(t *testing.T) {
				require.Empty(t, vcs.ExtractFileDiff("", "file.txt"))
				require.Empty(t, vcs.ExtractFileDiff("some diff", ""))
			})

			t.Run("CalculateFileLine", func(t *testing.T) {
				line := vcs.CalculateFileLine("", 0)
				require.Truef(t, line == 1 || line == 0, "%s CalculateFileLine('', 0) returned %d", impl.name, line)
				line = vcs.CalculateFileLine("single line", 10)
				require.GreaterOrEqual(t, line, 0)
			})
		})
	}
}

func TestDiffMsgType(t *testing.T) {
	diffMsg := DiffMsg{Content: "test"}
	require.Equal(t, "test", diffMsg.Content)

	editorMsg := EditorFinishedMsg{Err: nil}
	require.NoError(t, editorMsg.Err)
}

func BenchmarkVCSDetection(b *testing.B) {
	tempDir := b.TempDir()
	b.Chdir(tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DetectVCS()
	}
}
