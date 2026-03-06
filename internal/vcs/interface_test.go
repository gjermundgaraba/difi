package vcs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackendInterfaceConsistency(t *testing.T) {
	implementations := []struct {
		name    string
		backend Backend
		setup   func(t *testing.T)
	}{
		{
			name:    "Git",
			backend: GitBackend{},
		},
		{
			name:    "JJ",
			backend: JjBackend{},
			setup: func(t *testing.T) {
				t.Helper()
				requireJujutsu(t)
			},
		},
	}

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			if impl.setup != nil {
				impl.setup(t)
			}

			backend := impl.backend
			require.NotEmpty(t, backend.Kind())
			require.NotEmpty(t, backend.CurrentLabel())
			require.NotEmpty(t, backend.RepoName())
			require.NotEmpty(t, backend.DefaultTarget())

			require.NotPanics(t, func() {
				_, _ = backend.ListChangedFiles(backend.DefaultTarget())
			})
			require.NotPanics(t, func() {
				_, _, _ = backend.DiffStats(backend.DefaultTarget())
			})
			require.NotPanics(t, func() {
				_, _ = backend.DiffStatsByFile(backend.DefaultTarget())
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

func BenchmarkBackendDetection(b *testing.B) {
	tempDir := b.TempDir()
	b.Chdir(tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DetectBackend()
	}
}
