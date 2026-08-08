# lazyworktree

git worktree を操作するための TUI ツール。Go + [Bubble Tea](https://github.com/charmbracelet/bubbletea) 製。

## ビルド

```sh
go build -o lazyworktree .
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

Issues / Pull Requests タブの閲覧自体は `gh` CLI（既存の `gh auth login` 認証）を使った読み取り専用（GitHub 側に書き込みは一切しない）。`n` で worktree を作る操作だけはローカルに `git fetch` / `git worktree add` を行う（後述）。オープン中の worktree のブランチが、取得した PR の head ブランチと一致する場合、Worktrees タブ側に `[PR #123]` バッジが付く。

### issue / PR から worktree を作る（`n`）

- **Pull Requests タブ**: ブランチ名は `pr-<番号>` 固定。`refs/pull/<番号>/head:pr-<番号>` を直接 fetch してから worktree を作るので、fork から出された PR でも動く。同名ブランチの worktree が既にあればそれをそのまま使う（再 fetch はしない）。
- **Issues タブ**: ブランチ名は `issue-<番号>` 固定。リポジトリのデフォルトブランチを fetch し、そこから新規ブランチを切って worktree を作る。
- どちらも作成後、`herdr` が使えれば自動でそのペインを開く（既に開いていれば focus）。

### 新規 worktree の作成先

`n` で作成する際のデフォルトパスは、リポジトリ直下の `.worktrees/<ブランチ名>`（ブランチ名の `/` はそのままネストしたディレクトリになる）。パスはフォームで編集可能。

対象リポジトリの `.gitignore` に `.worktrees/` を追加しておくことを推奨する。

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

- Worktrees タブで `o` を押すと、選択中の worktree を herdr のペインとして開く。`herdr worktree list` で既に開いているワークスペースがあれば `herdr worktree open` を呼ばず `herdr workspace focus` で切り替えるだけにする（`hwtopen` 相当）。
- `d` で削除する際、その worktree に開いている herdr ワークスペースがあれば `herdr worktree remove --workspace` でペインごと閉じてから削除する。開いていなければ通常の `git worktree remove` にフォールバックする（`hwtremove` 相当）。開いたままのペインが削除済みディレクトリを指し続ける事故を防ぐため。
- 新規作成フォーム（`n`）に「open in herdr after create」のトグルがあり（`herdr` が見つかっていればデフォルト ON）、作成後にそのまま herdr のペインとして開ける。
- Issues / Pull Requests タブの `n`（上記）でも同様に、作成後 herdr が使えれば自動でペインを開く。

### rename の挙動

`R` の挙動は、`lazyworktree` が herdr のペイン内で起動されているか（`HERDR_ENV=1` 環境変数の有無で判定）によって変わる。どちらの場合も worktree のチェックアウトディレクトリ自体は変更しない。

- **通常起動時**: 選択中の worktree のブランチ名を `git branch -m` でリネームする。
- **herdr のペイン内で起動時**（popup 経由など）: 選択中の worktree に関係なく、そのペインをホストしている herdr workspace のラベルをリネームする。ブランチ名は変更しない。ホストしている workspace の ID は `$HERDR_WORKSPACE_ID` 環境変数がまず使われるが、popup コマンドのシェルにはこれが渡らないことがあるため、その場合は `herdr workspace list` で現在フォーカスされているワークスペースを探すフォールバックが入っている。

`lazyworktree` 自体を herdr の popup として呼び出す設定は `~/.config/herdr/config.toml` に追加済み:

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
