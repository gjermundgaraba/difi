package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/oug-t/difi/internal/config"
	"github.com/oug-t/difi/internal/ui"
	"github.com/oug-t/difi/internal/vcs"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin *os.File, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("difi", flag.ContinueOnError)
	flags.SetOutput(stderr)

	showVersion := flags.Bool("version", false, "Show version")
	plain := flags.Bool("plain", false, "Print a plain summary")
	forceVCS := flags.String("vcs", "", "Force specific VCS (git or jj)")

	if err := flags.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintf(stdout, "difi version %s\n", version)
		return 0
	}

	pipedDiff := readPipedDiff(stdin)

	var vcsClient vcs.Backend
	if *forceVCS != "" {
		switch *forceVCS {
		case "git":
			vcsClient = vcs.GitBackend{}
		case "jj":
			vcsClient = vcs.JjBackend{}
		default:
			fmt.Fprintf(stderr, "Error: unsupported VCS '%s'. Supported values: git, jj\n", *forceVCS)
			return 1
		}
	} else {
		vcsClient = vcs.DetectBackend()
	}

	target := vcsClient.DefaultTarget()
	if flags.NArg() > 0 {
		target = flags.Arg(0)
	}

	if *plain && pipedDiff == "" {
		files, err := vcsClient.ListChangedFiles(target)
		if err != nil {
			fmt.Fprintf(stderr, "Error listing changed files: %v\n", err)
			return 1
		}
		for _, file := range files {
			fmt.Fprintln(stdout, file)
		}
		return 0
	}

	cfg := config.Load()

	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if pipedDiff != "" {
		if tty, err := os.Open("/dev/tty"); err == nil {
			defer tty.Close()
			opts = append(opts, tea.WithInput(tty))
		}
	}

	p := tea.NewProgram(ui.NewModel(cfg, target, pipedDiff, vcsClient), opts...)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	return 0
}

func readPipedDiff(stdin *os.File) string {
	if stdin == nil {
		return ""
	}

	stat, err := stdin.Stat()
	if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
		return ""
	}

	b, err := io.ReadAll(stdin)
	if err != nil {
		return ""
	}

	return string(b)
}
