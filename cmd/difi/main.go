package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gjermundgaraba/difi/internal/config"
	"github.com/gjermundgaraba/difi/internal/ui"
	"github.com/gjermundgaraba/difi/internal/vcs"
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
	targetFlag := flags.String("target", "", "Review a specific target")

	if err := flags.Parse(normalizeInterspersedFlags(flags, args)); err != nil {
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
	if *targetFlag != "" {
		target = *targetFlag
	}

	if flags.NArg() > 1 {
		fmt.Fprintln(stderr, "Error: expected at most one path argument")
		return 2
	}

	pathScope := ""
	if flags.NArg() == 1 {
		pathScope = flags.Arg(0)
	}

	if *plain && pipedDiff == "" {
		files, err := vcsClient.ListChangedFiles(target, pathScope)
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

	p := tea.NewProgram(ui.NewModel(cfg, target, pathScope, pipedDiff, vcsClient), opts...)
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

type boolFlag interface {
	IsBoolFlag() bool
}

func normalizeInterspersedFlags(flags *flag.FlagSet, args []string) []string {
	takesValue := valueFlags(flags)
	flagArgs := make([]string, 0, len(args))
	positionalArgs := make([]string, 0, len(args))
	hasSeparator := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			hasSeparator = true
			positionalArgs = append(positionalArgs, args[i+1:]...)
			break
		}

		if isFlagArg(arg) {
			flagArgs = append(flagArgs, arg)
			if flagConsumesValue(takesValue, arg) && !strings.Contains(arg, "=") && i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
			continue
		}

		positionalArgs = append(positionalArgs, arg)
	}

	if hasSeparator {
		flagArgs = append(flagArgs, "--")
		return append(flagArgs, positionalArgs...)
	}

	return append(flagArgs, positionalArgs...)
}

func isFlagArg(arg string) bool {
	return strings.HasPrefix(arg, "-") && arg != "-"
}

func valueFlags(flags *flag.FlagSet) map[string]struct{} {
	takesValue := make(map[string]struct{})
	flags.VisitAll(func(f *flag.Flag) {
		if bf, ok := f.Value.(boolFlag); ok && bf.IsBoolFlag() {
			return
		}
		takesValue[f.Name] = struct{}{}
	})
	return takesValue
}

func flagConsumesValue(takesValue map[string]struct{}, arg string) bool {
	name := strings.TrimLeft(arg, "-")
	if eq := strings.IndexByte(name, '='); eq >= 0 {
		name = name[:eq]
	}

	_, ok := takesValue[name]
	return ok
}
