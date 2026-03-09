# difi

> A personal fork of [oug-t/difi](https://github.com/oug-t/difi) — a terminal diff viewer built with Go and Bubble Tea.

The original tool is excellent. This fork exists to add [Jujutsu (jj)](https://github.com/martinvonz/jj) support and personal functionality that fits my workflow.

## What's different from upstream

- Jujutsu (`jj`) support — review the working-copy commit or any revset
- Personal tweaks and additions over time

## Installation

```bash
go install github.com/gjermundgaraba/difi/cmd/difi@latest
```

Or clone and build:

```bash
git clone https://github.com/gjermundgaraba/difi
cd difi
go build ./cmd/difi
```

## Usage

Run in any Git or Jujutsu repository:

```bash
difi
```

Scope to a path:

```bash
difi src
```

Review a specific Jujutsu revset:

```bash
difi --vcs jj --target @-
```

Pipe a diff directly:

```bash
jj diff --git | difi
git diff | difi
cat changes.patch | difi
```

## Controls

| Key           | Action                                       |
| ------------- | -------------------------------------------- |
| `Tab`         | Toggle focus between File Tree and Diff View |
| `j / k`       | Move cursor down / up                        |
| `h / l`       | Focus Left (Tree) / Focus Right (Diff)       |
| `e` / `Enter` | Edit file (opens editor at selected line)    |
| `x`           | Undo the current Git hunk in Diff View       |
| `?`           | Toggle help drawer                           |
| `q`           | Quit                                         |

`x` is currently available only for live Git diffs against `HEAD`. Piped diffs, Jujutsu reviews, and non-`HEAD` Git targets are read-only.

## Development

```bash
go test ./...
go run cmd/difi/main.go
```

To regenerate golden files after intentional UI changes:

```bash
go test ./... -update
```
