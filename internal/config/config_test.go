package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DIFI_CONFIG", filepath.Join(t.TempDir(), "missing.yaml"))
	t.Setenv("DIFI_EDITOR", "")
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")

	cfg := Load()

	require.Equal(t, "vi", cfg.Editor)
	require.Equal(t, "hybrid", cfg.UI.LineNumbers)
	require.Equal(t, "default", cfg.UI.Theme)
}

func TestLoadUsesDIFIConfigOverride(t *testing.T) {
	configPath := writeConfigFile(t, t.TempDir(), "custom.yaml", "editor: helix\nui:\n  line_numbers: absolute\n  theme: nord\n")
	t.Setenv("DIFI_CONFIG", configPath)

	cfg := Load()

	require.Equal(t, "helix", cfg.Editor)
	require.Equal(t, "absolute", cfg.UI.LineNumbers)
	require.Equal(t, "nord", cfg.UI.Theme)
}

func TestLoadUsesUserConfigDirWhenNoOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("DIFI_CONFIG", "")
	t.Setenv("DIFI_EDITOR", "")
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")

	configDir, err := os.UserConfigDir()
	require.NoError(t, err)

	writeConfigFile(t, filepath.Join(configDir, "difi"), "config.yaml", "editor: vim\n")

	cfg := Load()
	require.Equal(t, "vim", cfg.Editor)
}

func TestLoadEditorPrecedence(t *testing.T) {
	tests := []struct {
		name   string
		config string
		env    map[string]string
		want   string
	}{
		{
			name:   "config file wins",
			config: "editor: nvim\n",
			env: map[string]string{
				"DIFI_EDITOR": "helix",
				"EDITOR":      "vim",
				"VISUAL":      "nano",
			},
			want: "nvim",
		},
		{
			name: "DIFI_EDITOR beats EDITOR and VISUAL",
			env: map[string]string{
				"DIFI_EDITOR": "helix",
				"EDITOR":      "vim",
				"VISUAL":      "nano",
			},
			want: "helix",
		},
		{
			name: "EDITOR beats VISUAL",
			env: map[string]string{
				"DIFI_EDITOR": "",
				"EDITOR":      "vim",
				"VISUAL":      "nano",
			},
			want: "vim",
		},
		{
			name: "VISUAL used last",
			env: map[string]string{
				"DIFI_EDITOR": "",
				"EDITOR":      "",
				"VISUAL":      "nano",
			},
			want: "nano",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("DIFI_CONFIG", filepath.Join(dir, "config.yaml"))
			for key, value := range tt.env {
				t.Setenv(key, value)
			}
			if tt.config != "" {
				writeConfigFile(t, dir, "config.yaml", tt.config)
			}

			cfg := Load()
			require.Equal(t, tt.want, cfg.Editor)
		})
	}
}

func TestLoadPreservesDefaultsForPartialConfig(t *testing.T) {
	configPath := writeConfigFile(t, t.TempDir(), "config.yaml", "ui:\n  theme: nord\n")
	t.Setenv("DIFI_CONFIG", configPath)
	t.Setenv("DIFI_EDITOR", "")
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")

	cfg := Load()

	require.Equal(t, "vi", cfg.Editor)
	require.Equal(t, "hybrid", cfg.UI.LineNumbers)
	require.Equal(t, "nord", cfg.UI.Theme)
}

func writeConfigFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	require.NoError(t, os.MkdirAll(dir, 0o755))

	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	return path
}
