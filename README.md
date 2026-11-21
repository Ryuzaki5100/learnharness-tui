# learnharness-tui (`lhtui`)

A terminal companion for [LearnHarness](https://github.com/amosblomqvist/learnharness)
learning records. It reads teaching and practice sessions, the knowledge graph,
and opencode study notes from the same data directory the desktop app and the
`learnharness` skill use — and it can start a study session through
`opencode run`, saving the transcript into the synced data repository.

It is deliberately **read-only over app-owned records**: it never rewrites
`chats/` or `graph/`. The only files it writes are its own `study/*.md` notes.

## Install

```bash
nix profile install github:Ryuzaki5100/learnharness-tui
# or run without installing
nix run github:Ryuzaki5100/learnharness-tui -- --data-dir ~/learning/learnharness-data
```

Home Manager:

```nix
inputs.learnharness-tui.url = "github:Ryuzaki5100/learnharness-tui";
# ...
home.packages = [ inputs.learnharness-tui.packages.${pkgs.system}.default ];
```

Requires `opencode` on `PATH` for study mode. Session browsing works without it.

## Data directory

Resolution order:

1. `--data-dir <path>`
2. `LH_DATA_DIR`
3. OS default: `~/Library/Application Support/LearnHarness/data` (macOS),
   `$XDG_CONFIG_HOME/LearnHarness/data` or `~/.config/LearnHarness/data` (Linux),
   `%APPDATA%\LearnHarness\data` (Windows)

## Keys

| Scope | Keys | Action |
| --- | --- | --- |
| Move | `j` `k` / `↓` `↑` | next / previous row |
| Jump | `gg` / `G` | top / bottom |
| Page | `ctrl+d` / `ctrl+u`, `ctrl+f` / `ctrl+b` | half / full page |
| Counts | `4j` … `9j` | repeat a movement (`1`–`3` switch tabs) |
| Panes | `h` `l` / `←` `→` | focus left / right pane |
| Tabs | `1` / `2` / `3`, `tab` / `shift+tab` | sessions / graph / study |
| Search | `/` then text, `enter`, `n` / `N` | filter, next / previous match |
| Open | `enter` | open the selected session or study note |
| Study | `i` | type a topic and start `opencode run` |
| Study | `O` | run interactive `opencode` in the terminal |
| Cancel | `esc` | leave input mode / cancel a running study session |
| Sync | `s` | `lh-sync cycle` (pull, then push) |
| Refresh | `r` | reload the current tab |
| Help | `?` | full keymap |
| Quit | `q`, `ctrl+c` | exit |

## Study mode

`i` asks for a topic, then runs:

```bash
opencode run --format json "<study prompt with the learnharness skill>"
```

The streamed output is shown live and saved as
`study/YYYY-MM-DD-HHMM-<slug>.md` with frontmatter (topic, command, start/end).
These are opencode session notes; LearnHarness's native teaching records are
still written only by the app's own server.

## Sync

When the dotfiles-provided `lh-sync` helper is installed, `lhtui` pulls before
starting and pushes on a clean exit. `s` runs a full cycle. Conflicts are
reported, never auto-resolved. Use `--no-sync` to opt out for a run.

## Development

```bash
devenv shell             # Go, gopls, golangci-lint, just
just test                # go vet + go test
just nix-build           # buildGoModule with vendored dependencies
```

`testdata/golden/` is copied from the LearnHarness test fixtures so the readers
are exercised against the real stored format. Do not edit those files to make a
test pass.

## License

AGPL-3.0-only, matching LearnHarness.
