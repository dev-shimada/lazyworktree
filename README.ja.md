# lazyworktree

<p align="center">
  <a href="https://github.com/dev-shimada/lazyworktree/actions/workflows/ci.yml"><img alt="CI" src="https://img.shields.io/github/actions/workflow/status/dev-shimada/lazyworktree/ci.yml?branch=main&style=flat-square&logo=githubactions&logoColor=white&label=CI"></a>
  <a href="https://codecov.io/gh/dev-shimada/lazyworktree"><img alt="Coverage" src="https://codecov.io/gh/dev-shimada/lazyworktree/branch/main/graph/badge.svg?style=flat-square"></a>
  <a href="https://github.com/dev-shimada/lazyworktree/blob/main/go.mod"><img alt="Go Version" src="https://img.shields.io/github/go-mod/go-version/dev-shimada/lazyworktree?style=flat-square&logo=go&logoColor=white"></a>
  <a href="https://github.com/dev-shimada/lazyworktree/blob/main/LICENSE"><img alt="MIT License" src="https://img.shields.io/github/license/dev-shimada/lazyworktree?style=flat-square"></a>
</p>

<p align="center">
  <a href="https://github.com/dev-shimada/lazyworktree/releases/latest"><img alt="Release" src="https://img.shields.io/github/v/release/dev-shimada/lazyworktree?style=flat-square&logo=github&logoColor=white"></a>
  <a href="#install"><img alt="Homebrew" src="https://img.shields.io/badge/homebrew-dev--shimada%2Flazyworktree-FBB040?style=flat-square&logo=homebrew&logoColor=white"></a>
</p>

git worktree を操作するための TUI ツール。Go + [Bubble Tea](https://github.com/charmbracelet/bubbletea) 製。

[English](README.md)

## Features

- TUI で git worktree の一覧・作成・削除・lock/unlock・prune
- GitHub の Issues / Pull Requests を閲覧専用で表示（`gh` CLI 経由）。ファジー検索対応、デフォルトは自分にアサインされたものだけ
- issue / PR からそのまま新規 worktree をチェックアウト
- [herdr](https://github.com/herdrdev/herdr) との連携（任意）: worktree を herdr のペインとして開く、herdr ワークスペースの現在のラベルで一覧表示する、herdr 内から起動した場合は herdr の設定通りの場所に worktree を作成する

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

### herdr plugin として使う

このリポジトリは herdr plugin(`herdr-plugin.toml`)としても利用できます。インストール後、`open` アクションにキーを割り当ててください。

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

popup はフォーカス中のペインのディレクトリで起動します。`PATH` 上の `lazyworktree` を実行するので、先に上記の Homebrew または Go で本体をインストールしてください。

## 使い方

git リポジトリ内（どの worktree でも可）で実行する。

```sh
lazyworktree
```

`lazyworktree --help` で全フラグを確認できる。下記のシェル関数でラップしている場合、引数をちゃんと転送する実装にしておくこと（転送しないとフラグを渡しても素のTUIが起動してしまう）。

画面は Worktrees / Issues / Pull Requests の3タブ。`tab` で切り替える。

| キー | 動作 | タブ |
| --- | --- | --- |
| `↑`/`k`, `↓`/`j` | 選択移動 | 共通 |
| `tab` | Worktrees / Issues / Pull Requests を切り替え | 共通 |
| `r` | 現在のタブを再読み込み | 共通 |
| `/` | Worktrees は一覧のローカルフィルタ。Issues / Pull Requests は GitHub 上のオープンな全件を検索 | 共通 |
| `q` / `esc` / `ctrl+c` | 終了（何も選択しない）。Issues / Pull Requests で検索中の `esc` は先に検索を解除 | 共通 |
| `c` | worktree を選択して終了（パスを stdout に出力） | Worktrees |
| `n` | 新規 worktree を作成（既存/新規ブランチどちらも指定可） | Worktrees |
| `d` | 選択中の worktree を削除（確認あり、`f` で `--force` 切替） | Worktrees |
| `l` | 選択中の worktree の lock / unlock を切り替え | Worktrees |
| `p` | `git worktree prune` を実行 | Worktrees |
| `enter` | 選択中の worktree を herdr のペインとして開いて終了（既に開いていれば focus） | Worktrees |
| `o` | worktree に紐づく PR / issue を GitHub でブラウザに開く（open / closed / merged 問わず） | Worktrees |
| `R` | rename（挙動は起動モードによる。下記参照） | Worktrees |
| `o` | 選択中の issue / PR をブラウザで開く（`gh ... --web`） | Issues / Pull Requests |
| `enter` | 選択中の issue / PR を新規 worktree としてチェックアウトして終了 | Issues / Pull Requests |
| `a` | 「自分にアサイン」（デフォルト）⇔「全体」を切り替え | Issues / Pull Requests |
| `s` | 設定した issue リポジトリを順番に切り替え（下記参照） | Issues |

Issues / Pull Requests タブの閲覧自体は `gh` CLI（既存の `gh auth login` 認証）を使った読み取り専用（GitHub 側に書き込みは一切しない）。デフォルトは「自分のもの」（`a` で全体表示に切り替え）: issueは自分にアサインされたもの、PRは自分がassigneeまたはreview-requestedになっているもの。`/` は GitHub の検索構文（Web UI と同じ。例: `label:bug auth`）で、最初に読み込んだ分だけでなくオープンな**全件**を、現在の mine/all の範囲で検索する。空のクエリか `esc` で検索を解除できる。通常表示ではオープン分の後ろに、直近でクローズ（PR はマージ含む）された20件が `[closed]` / `[merged]` 付きで並ぶ。クローズ済みは検索対象外。`enter` で worktree を作る操作だけはローカルに `git fetch` / `git worktree add` を行う（後述）。

Worktrees タブは各 worktree のブランチに紐づく issue / PR を状態に関わらず追跡する。`pr-N` / `issue-N` ブランチは番号で引き、それ以外のブランチは head ブランチ名が一致する PR（状態不問、open があればそれを優先、無ければ最新のもの）に対応づける（fork 由来の PR は除外）。`[PR #123 open]` / `[PR #123 merged]` / `[PR #123 closed]` / `[issue #45 closed]` のように色付きバッジで状態を表示する。

### issue / PR から worktree を作る（`enter`）

- **Pull Requests タブ**: ブランチ名は `pr-<番号>` 固定。`refs/pull/<番号>/head:pr-<番号>` を直接 fetch してから worktree を作るので、fork から出された PR でも動く。同名ブランチの worktree が既にあればそれをそのまま使う（再 fetch はしない）。
- **Issues タブ**: ブランチ名は `issue-<番号>` 固定。リポジトリのデフォルトブランチを fetch し、そこから新規ブランチを切って worktree を作る。
- どちらも作成後、`herdr` が使えれば自動でそのペインを開く（既に開いていれば focus）。
- `pr-<番号>`/`issue-<番号>` だけでは何のworktreeか分からないので、Worktrees タブではそのパターンに一致するブランチについて実際のGitHubタイトルを取得して表示する。

### 新規 worktree の作成先

`n`（および Issues / Pull Requests タブの `enter`）で作成する際のデフォルトパスは、起動モードによって変わる（パスはどちらもフォームで編集可能）。

- **通常起動時**: リポジトリ直下の `.worktrees/<ブランチ名>`（ブランチ名の `/` はそのままネストしたディレクトリになる）。対象リポジトリの `.gitignore` に `.worktrees/` を追加しておくことを推奨する。
- **herdr のペイン内で起動時**: herdr 自身が使う場所と同じ `<herdr の worktrees.directory 設定>/<リポジトリ名>/<ブランチ名>`（デフォルトは `~/.herdr/worktrees/<repo>/<branch>`）。`~/.config/herdr/config.toml` の `[worktrees] directory` を変更していればそれに従う（herdr の CLI に設定値を問い合わせる手段が無いため、config.toml をこちらで直接読んで反映している）。

Issues / Pull Requests タブの `enter`（worktree 作成）でも同じ使い分けをする。これらは全て、lazyworktree をどの worktree から起動したかに関わらず、リポジトリの**メイン worktree** を基準にする — linked worktree（サブworktree）の中から起動しても、新規 worktree のグルーピング先を誤らず、herdr でも正しく開ける（herdr の `worktree open`/`create` は linked worktree 自身のパスを渡すと拒否するため）。

### issue を別リポジトリで管理している場合

コードとは別のリポジトリで issue を管理しているプロジェクト向け（複数のアプリリポジトリで1つの spec/backlog リポジトリを共有している場合など）に、`~/.config/lazyworktree/config.toml` で設定できる。`lazyworktree --default-config` でコメントアウト済みの雛形が出力される:

```sh
mkdir -p ~/.config/lazyworktree
lazyworktree --default-config > ~/.config/lazyworktree/config.toml
```

```toml
[[issues_repo]]
match = "myorg/*"
repos = ["myorg/specs"]
```

`match` は現在のリポジトリの `owner/repo` に対するglobマッチ（`*` はスラッシュをまたがないので、`myorg/*` は `myorg` 配下の任意のリポジトリにマッチしてそれ以上ネストしない）。最初にマッチしたルールの `repos` が Issues タブの**デフォルト**ソースになり、今のリポジトリ自身の issue はその後（隠れることはなく、切り替えの最後）に表示される。Issues タブで `s` を押すとソースを順番に切り替えられ、現在のソースはタブラベル（例: `Issues [mine, myorg/specs]`）に表示される。`enter`（issueからのworktree作成）は、issueがどのソースから来たものであっても、常に今 lazyworktree を実行しているリポジトリにブランチを作る。

### シェル連携（cd フック）

`c` で worktree を選択すると、終了後にそのパスを標準出力に 1 行だけ出力する。
シェル関数でラップすると `cd` 込みで使える。引数は必ず実体のバイナリへ転送すること — 転送しないと `--default-config` や `--help` を渡しても素の対話TUIが起動してしまう:

```sh
# ~/.zshrc など
lazyworktree() {
  if [ "$#" -gt 0 ]; then
    command lazyworktree "$@"
    return
  fi
  local dir
  dir=$(command lazyworktree) && [ -n "$dir" ] && cd -- "$dir"
}
```

## herdr 連携

`internal/herdr` が `herdr worktree open` / `herdr worktree list` / `herdr worktree remove` / `herdr workspace rename` / `herdr workspace focus` を薄くラップしている（`herdr` が PATH 上にない場合は機能を無効化するだけで、エラーにはしない）。

- Worktrees タブで `enter` を押すと、選択中の worktree を herdr のペインとして開いて終了する。`herdr worktree list` で既に開いているワークスペースがあれば `herdr worktree open` を呼ばず `herdr workspace focus` で切り替えるだけにする。
- `d` で削除する際、その worktree に開いている herdr ワークスペースがあれば `herdr worktree remove --workspace` でペインごと閉じてから削除する。開いていなければ通常の `git worktree remove` にフォールバックする。開いたままのペインが削除済みディレクトリを指し続ける事故を防ぐため。
- 新規作成フォーム（`n`）に「open in herdr after create」のトグルがあり（`herdr` が見つかっていればデフォルト ON）、作成後にそのまま herdr のペインとして開ける。
- Issues / Pull Requests タブの `enter`（上記）でも同様に、作成後 herdr が使えれば自動でペインを開く。
- **herdr のペイン内で起動時**、Worktrees タブの一覧はブランチ名の代わりに、その worktree に開いている herdr ワークスペースの現在のラベル（`R` でリネームした名前を含む）を表示する（`herdr worktree list` の `label` フィールドはリネームを反映しない静的な値なので、`herdr workspace list` を別途引いて解決している）。ブランチ名は一覧の説明行に `[branch]` として残る。開いているワークスペースが無い worktree は従来通りブランチ名で表示される。

### rename の挙動

`R` の挙動は、`lazyworktree` が herdr のペイン内で起動されているか（`HERDR_ENV=1` 環境変数の有無で判定）によって変わる。どちらの場合も worktree のチェックアウトディレクトリ自体は変更しない。

- **通常起動時**: 選択中の worktree のブランチ名を `git branch -m` でリネームする。
- **herdr のペイン内で起動時**（popup 経由など）: 選択中の worktree に関係なく、そのペインをホストしている herdr workspace のラベルをリネームする。ブランチ名は変更しない。ホストしている workspace の ID は `$HERDR_WORKSPACE_ID` 環境変数がまず使われるが、popup コマンドのシェルにはこれが渡らないことがあるため、その場合は `herdr workspace list` で現在フォーカスされているワークスペースを探すフォールバックが入っている。

`lazyworktree` 自体を herdr の popup として呼び出す設定例（`~/.config/herdr/config.toml`）:

```toml
[[keys.command]]
key = "prefix+alt+w"
type = "popup"
command = "lazyworktree"
description = "lazyworktree: manage git worktrees"
width = "80%"
height = "70%"
```

## GitHub 連携

`internal/github` が `gh` CLI（既存の `gh auth login` 認証をそのまま利用）をラップし、Issues / Pull Requests タブに閲覧専用の一覧を表示する。作成・コメント・マージなどの書き込み操作は対象外。`gh` が PATH 上にない場合はタブ選択時にエラーメッセージを表示するだけで、他の機能には影響しない。

## 開発

```sh
go vet ./...
go test ./...
```

## License

[MIT](LICENSE)
