# lazyworktree

git worktree を操作するための TUI ツール。Go + [Bubble Tea](https://github.com/charmbracelet/bubbletea) 製。

[English](README.md)

## Features

- TUI で git worktree の一覧・作成・削除・lock/unlock・prune
- GitHub の Issues / Pull Requests を閲覧専用で表示（`gh` CLI 経由）
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

## 使い方

git リポジトリ内（どの worktree でも可）で実行する。

```sh
lazyworktree
```

画面は Worktrees / Issues / Pull Requests の3タブ。`tab` で切り替える。

| キー | 動作 | タブ |
| --- | --- | --- |
| `↑`/`k`, `↓`/`j` | 選択移動 | 共通 |
| `tab` | Worktrees / Issues / Pull Requests を切り替え | 共通 |
| `r` | 現在のタブを再読み込み | 共通 |
| `/` | 一覧をフィルタ | 共通 |
| `q` / `ctrl+c` | 終了（何も選択しない） | 共通 |
| `enter` | worktree を選択して終了（パスを stdout に出力） | Worktrees |
| `n` | 新規 worktree を作成（既存/新規ブランチどちらも指定可） | Worktrees |
| `d` | 選択中の worktree を削除（確認あり、`f` で `--force` 切替） | Worktrees |
| `l` | 選択中の worktree の lock / unlock を切り替え | Worktrees |
| `p` | `git worktree prune` を実行 | Worktrees |
| `o` | 選択中の worktree を herdr のペインとして開く（既に開いていれば focus） | Worktrees |
| `R` | rename（挙動は起動モードによる。下記参照） | Worktrees |
| `enter` / `o` | 選択中の issue / PR をブラウザで開く（`gh ... --web`） | Issues / Pull Requests |
| `n` | 選択中の issue / PR を新規 worktree としてチェックアウト | Issues / Pull Requests |
| `s` | 設定した issue リポジトリを順番に切り替え（下記参照） | Issues |

Issues / Pull Requests タブの閲覧自体は `gh` CLI（既存の `gh auth login` 認証）を使った読み取り専用（GitHub 側に書き込みは一切しない）。`n` で worktree を作る操作だけはローカルに `git fetch` / `git worktree add` を行う（後述）。オープン中の worktree のブランチが、取得した PR の head ブランチと一致する場合、Worktrees タブ側に `[PR #123]` バッジが付く。

### issue / PR から worktree を作る（`n`）

- **Pull Requests タブ**: ブランチ名は `pr-<番号>` 固定。`refs/pull/<番号>/head:pr-<番号>` を直接 fetch してから worktree を作るので、fork から出された PR でも動く。同名ブランチの worktree が既にあればそれをそのまま使う（再 fetch はしない）。
- **Issues タブ**: ブランチ名は `issue-<番号>` 固定。リポジトリのデフォルトブランチを fetch し、そこから新規ブランチを切って worktree を作る。
- どちらも作成後、`herdr` が使えれば自動でそのペインを開く（既に開いていれば focus）。

### 新規 worktree の作成先

`n` で作成する際のデフォルトパスは、起動モードによって変わる（パスはどちらもフォームで編集可能）。

- **通常起動時**: リポジトリ直下の `.worktrees/<ブランチ名>`（ブランチ名の `/` はそのままネストしたディレクトリになる）。対象リポジトリの `.gitignore` に `.worktrees/` を追加しておくことを推奨する。
- **herdr のペイン内で起動時**: herdr 自身が使う場所と同じ `<herdr の worktrees.directory 設定>/<リポジトリ名>/<ブランチ名>`（デフォルトは `~/.herdr/worktrees/<repo>/<branch>`）。`~/.config/herdr/config.toml` の `[worktrees] directory` を変更していればそれに従う（herdr の CLI に設定値を問い合わせる手段が無いため、config.toml をこちらで直接読んで反映している）。

Issues / Pull Requests タブの `n`（worktree 作成）でも同じ使い分けをする。

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

`match` は現在のリポジトリの `owner/repo` に対するglobマッチ（`*` はスラッシュをまたがないので、`myorg/*` は `myorg` 配下の任意のリポジトリにマッチしてそれ以上ネストしない）。最初にマッチしたルールの `repos` が Issues タブの追加ソースになる — 今のリポジトリ自身の issue が隠れることはなく、常にアクセスできる。Issues タブで `s` を押すとソースを順番に切り替えられ、現在のソースはタブラベル（例: `Issues [myorg/specs]`）に表示される。`n`（issueからのworktree作成）は、issueがどのソースから来たものであっても、常に今 lazyworktree を実行しているリポジトリにブランチを作る。

### シェル連携（cd フック）

`enter` で worktree を選択すると、終了後にそのパスを標準出力に 1 行だけ出力する。
シェル関数でラップすると `cd` 込みで使える。

```sh
# ~/.zshrc など
lazyworktree() {
  local dir
  dir=$(command lazyworktree) && [ -n "$dir" ] && cd -- "$dir"
}
```

## herdr 連携

`internal/herdr` が `herdr worktree open` / `herdr worktree list` / `herdr worktree remove` / `herdr workspace rename` / `herdr workspace focus` を薄くラップしている（`herdr` が PATH 上にない場合は機能を無効化するだけで、エラーにはしない）。

- Worktrees タブで `o` を押すと、選択中の worktree を herdr のペインとして開く。`herdr worktree list` で既に開いているワークスペースがあれば `herdr worktree open` を呼ばず `herdr workspace focus` で切り替えるだけにする。
- `d` で削除する際、その worktree に開いている herdr ワークスペースがあれば `herdr worktree remove --workspace` でペインごと閉じてから削除する。開いていなければ通常の `git worktree remove` にフォールバックする。開いたままのペインが削除済みディレクトリを指し続ける事故を防ぐため。
- 新規作成フォーム（`n`）に「open in herdr after create」のトグルがあり（`herdr` が見つかっていればデフォルト ON）、作成後にそのまま herdr のペインとして開ける。
- Issues / Pull Requests タブの `n`（上記）でも同様に、作成後 herdr が使えれば自動でペインを開く。
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
