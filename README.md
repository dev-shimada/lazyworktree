# lazyworktree

A TUI for managing git worktrees. Built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

[日本語](README.ja.md)

## Features

- Browse, create, delete, lock/unlock, and prune git worktrees from a terminal UI
- Read-only GitHub Issues / Pull Requests tabs (via the `gh` CLI), fuzzy-searchable, defaulting to items assigned to you
- Check out an issue or PR straight into a new worktree
- Optional integration with [herdr](https://github.com/herdrdev/herdr): hand a worktree off to a herdr pane, list worktrees by their live herdr workspace label, and create new worktrees under herdr's configured directory when launched from inside herdr

## Install

### Homebrew

```sh
brew tap dev-shimada/lazyworktree
brew install lazyworktree
```

### Go

```sh
go install github.com/dev-shimada/lazyworktree@latest
```

### As a herdr plugin

This repository is also a herdr plugin (`herdr-plugin.toml`). Install it, then bind a key to its `open` action:

```sh
herdr plugin install dev-shimada/lazyworktree
```

```toml
[[keys.command]]
key = "prefix+alt+w"
type = "plugin_action"
command = "dev-shimada.lazyworktree.open"
description = "lazyworktree: manage git worktrees"
```

The popup starts in the focused pane's directory. It runs the `lazyworktree` found on `PATH`, so install it with Homebrew or Go above first.

## Usage

Run it inside a git repository (any worktree):

```sh
lazyworktree
```

`lazyworktree --help` shows all flags. If you've wrapped it in the shell function below, make sure it forwards arguments — otherwise flags silently launch the TUI instead.

The screen has three tabs — Worktrees / Issues / Pull Requests — switched with `tab`.

| Key | Action | Tab |
| --- | --- | --- |
| `↑`/`k`, `↓`/`j` | Move selection | all |
| `tab` | Switch Worktrees / Issues / Pull Requests | all |
| `r` | Reload the current tab | all |
| `/` | Filter the Worktrees list locally; on Issues / Pull Requests, search all open items on GitHub | all |
| `q` / `esc` / `ctrl+c` | Quit without selecting (on Issues / Pull Requests, `esc` first clears an active search) | all |
| `c` | Select a worktree and exit (prints its path to stdout) | Worktrees |
| `n` | Create a new worktree (existing or new branch) | Worktrees |
| `d` | Delete the selected worktree (asks first; `f` toggles `--force`) | Worktrees |
| `l` | Toggle lock / unlock on the selected worktree | Worktrees |
| `p` | Run `git worktree prune` | Worktrees |
| `enter` | Open the selected worktree as a herdr pane (focuses it if already open) and exit | Worktrees |
| `o` | Open the worktree's PR / issue on GitHub in the browser — open, closed, or merged | Worktrees |
| `R` | Rename (behavior depends on how lazyworktree was launched — see below) | Worktrees |
| `o` | Open the selected issue / PR in the browser (`gh ... --web`) | Issues / Pull Requests |
| `enter` | Check out the selected issue / PR into a new worktree and exit | Issues / Pull Requests |
| `a` | Toggle between "assigned to me" (default) and "all" | Issues / Pull Requests |
| `s` | Cycle through configured issue repos (see below) | Issues |

Browsing the Issues / Pull Requests tabs is read-only, backed by the `gh` CLI (using your existing `gh auth login`) — nothing is ever written back to GitHub. Both default to "mine" (`a` toggles to everyone's): issues assigned to you, and PRs where you're either the assignee or a requested reviewer. `/` runs a GitHub search (the same syntax as the web UI, e.g. `label:bug auth`) across **all** open items — not just the first page that was loaded — within the current mine/all scope; an empty query clears it, and `esc` also clears an active search. The default list shows open items followed by the 20 most recently closed ones (issues) or closed/merged ones (PRs), tagged `[closed]` / `[merged]`; closed items are not searched. The `enter` action (below) does touch your local checkout: it runs `git fetch` / `git worktree add`.

The Worktrees tab tracks the issue or PR behind each worktree's branch whatever its state: `pr-N` / `issue-N` branches are looked up by number, and other branches are matched by head branch name against PRs in any state, preferring an open PR and otherwise the newest (fork PRs are ignored). Each gets a colored badge such as `[PR #123 open]`, `[PR #123 merged]`, `[PR #123 closed]` or `[issue #45 closed]`.

### Creating a worktree from an issue / PR (`enter`)

- **Pull Requests tab**: the branch is always named `pr-<number>`. It fetches `refs/pull/<number>/head:pr-<number>` directly, so this works for PRs from forks too. If a worktree for that branch already exists, it's reused (no re-fetch).
- **Issues tab**: the branch is always named `issue-<number>`. It fetches the repository's default branch and creates the new branch from there.
- Either way, if `herdr` is available, the new worktree is opened as a pane afterward (or focused, if already open).
- Since `pr-<number>`/`issue-<number>` alone doesn't say what a worktree is about, the Worktrees tab fetches and shows the actual GitHub title next to any branch matching that pattern.

### Where new worktrees are created

The default path used by `n` (and by `enter` on the Issues / Pull Requests tabs) depends on how lazyworktree was launched (either path is editable in the form before creating):

- **Standalone**: `.worktrees/<branch>` under the repository root (a branch containing `/` becomes nested directories). Add `.worktrees/` to the target repo's `.gitignore`.
- **Launched from inside a herdr pane**: the same location herdr itself would use — `<herdr's worktrees.directory>/<repo>/<branch>` (default `~/.herdr/worktrees/<repo>/<branch>`). If `[worktrees] directory` is customized in `~/.config/herdr/config.toml`, that's honored (lazyworktree reads the file directly, since herdr's CLI has no way to query the resolved setting).

The `enter` action on the Issues / Pull Requests tabs follows the same rule. All of this is anchored to the repo's **main** worktree, not whichever worktree you launched lazyworktree from — launching it from inside a linked worktree still groups new worktrees correctly and lets herdr open/create them (herdr's `worktree open`/`create` outright reject being pointed at a linked worktree's own path).

### Tracking issues in a different repository

Some projects track issues in a separate repo from the code (e.g. a shared spec/backlog repo used by several app repos). Configure this in `~/.config/lazyworktree/config.toml` — `lazyworktree --default-config` prints a commented-out starter file:

```sh
mkdir -p ~/.config/lazyworktree
lazyworktree --default-config > ~/.config/lazyworktree/config.toml
```

```toml
[[issues_repo]]
match = "myorg/*"
repos = ["myorg/specs"]
```

`match` is a glob against the current repo's `owner/repo` (`*` doesn't cross `/`, so `myorg/*` matches any repo under `myorg` but nothing nested further). The first matching rule's `repos` become the default Issues-tab source(s), shown before the current repo's own issues — which stay available too, they're never hidden, just last in the cycle. Press `s` on the Issues tab to cycle through them; the active source is shown in the tab label (e.g. `Issues [mine, myorg/specs]`). `enter` (checking out an issue into a worktree) always creates the branch in the repo you're running lazyworktree in, regardless of which source the issue came from.

### Shell integration (cd hook)

Selecting a worktree with `c` prints its path to stdout as a single line on exit. Wrap it in a shell function to also `cd`. Forward arguments through to the real binary — otherwise flags like `--default-config` or `--help` silently launch the interactive TUI instead:

```sh
# e.g. in ~/.zshrc
lazyworktree() {
  if [ "$#" -gt 0 ]; then
    command lazyworktree "$@"
    return
  fi
  local dir
  dir=$(command lazyworktree) && [ -n "$dir" ] && cd -- "$dir"
}
```

## herdr integration

`internal/herdr` thinly wraps `herdr worktree open` / `herdr worktree list` / `herdr worktree remove` / `herdr workspace rename` / `herdr workspace focus`. If `herdr` isn't on PATH, these features just quietly disable themselves — nothing errors.

- `enter` on the Worktrees tab opens the selected worktree as a herdr pane and exits. If `herdr worktree list` shows it already has an open workspace, it calls `herdr workspace focus` instead of opening it again.
- On delete (`d`), if the worktree has an open herdr workspace, `herdr worktree remove --workspace` closes the pane and removes the checkout in one step; otherwise it falls back to a plain `git worktree remove`. This avoids leaving a pane pointed at a deleted directory.
- The create form (`n`) has an "open in herdr after create" toggle (on by default when `herdr` is found), so a fresh worktree opens straight into a pane.
- The `enter` action on the Issues / Pull Requests tabs does the same after checking out.
- **When launched from inside a herdr pane**, the Worktrees tab shows each worktree's live herdr workspace label instead of its branch name (`herdr worktree list`'s own `label` field is a static default that doesn't track renames, so this cross-references `herdr workspace list` instead). The branch name stays visible as `[branch]` in the description line. Worktrees with no open workspace still show their branch name as usual.

### Rename behavior

`R` behaves differently depending on whether lazyworktree is running inside a herdr pane (detected via the `HERDR_ENV=1` environment variable). Neither mode ever touches the worktree's checkout directory.

- **Standalone**: renames the selected worktree's branch with `git branch -m`.
- **Inside a herdr pane** (e.g. launched as a popup): renames the label of the herdr workspace hosting the pane — regardless of which worktree is selected. The branch name is left untouched. The workspace ID is normally read from `$HERDR_WORKSPACE_ID`, but popup commands don't always get that env var, so this falls back to asking `herdr workspace list` for the currently focused workspace.

Calling lazyworktree as a herdr popup, e.g. in `~/.config/herdr/config.toml`:

```toml
[[keys.command]]
key = "prefix+alt+w"
type = "popup"
command = "lazyworktree"
description = "lazyworktree: manage git worktrees"
width = "80%"
height = "70%"
```

## GitHub integration

`internal/github` wraps the `gh` CLI (reusing your existing `gh auth login`) to show a read-only Issues / Pull Requests list. Writing back to GitHub — creating, commenting, merging — is out of scope. If `gh` isn't on PATH, selecting either tab just shows an error; nothing else is affected.

## Development

```sh
go vet ./...
go test ./...
```

## License

[MIT](LICENSE)
