# Emoji Resizer (カスタム絵文字用 画像一括整形・リサイズツール)

長い方に合わせて正方形に整形，指定サイズにリサイズする．

## 要件

Windows 11以外では動作未確認

## ビルド

### npm (Recommended)
```bash
npm run build
```

### go コマンド
```bash
go build -o emoji-resizer.exe ./src
```
※ Go 1.18 以上．
※ この方法でビルドした場合，バージョン情報が `devel` になる

## 使い方

ビルドされた実行ファイル (`emoji-resizer.exe` または `emoji-resizer`) をbashで実行．

### 基本構文
対象の画像ファイルまたはディレクトリを指定．スペース区切り．
省略した場合は，カレントディレクトリ直下の画像を全て処理．

```bash
# 特定の画像をリサイズ
./emoji-resizer.exe image1.png image2.webp

# ディレクトリ直下の画像をすべてリサイズ
./emoji-resizer.exe path/to/images/
```

### オプション

| オプション | デフォルト値 | 説明 | 備考 |
| :--- | :--- | :--- | :--- |
| `-size` | `128` | 一辺のサイズ，または短辺のサイズ (`-rect`時) | 大きくても192程度で十分 |
| `-rect` | `false` | 短辺を `-size` に合わせてリサイズ．アスペクト比はそのまま維持 | |
| `-trim` | `false` | 四方の透過余白（アルファ領域）のみを自動トリミングする | |
| `-trim-opaque` | `false` | 四方の透過領域およびベタ塗り背景（左上隅の色と同色領域）を自動トリミングする | |
| `-auto-rect` | `false` | アスペクト比がしきい値を超える場合 `rect` モードとして扱う．引数は1を超えなければならない | 引数未指定時は`-auto-rect=2.5` |
| `-out` | `output/` | 出力ディレクトリのパス指定 | |
| `-suffix` | `なし` | 出力ファイル名に付与する接尾辞 | |
| `-name-prefix` | `なし` | 絵文字名の前に付与する接頭辞 | `-zip`時のみ意味があります |
| `-name-suffix` | `なし` | 絵文字名の後ろに付与する接尾辞 | `-zip`時のみ意味があります |
| `-recursive` | `false` | 指定したディレクトリ以下のディレクトリもすべて調べる | |
| `-grouping` | `false` | 入力ディレクトリからの相対パスをカテゴリに追加する | `-zip`時のみ意味があります．有効時は `-recursive=false` を指定しても無視 |
| `-category` | `なし` | 絵文字の基準カテゴリを指定する | 設定ファイルの `category` より優先．`-zip` 時のカテゴリ入力を省略可能 |
| `-no-resize` | `false` | 正方形にして終了．圧縮しない |  `-rect`指定時はなにもしない |
| `-no-resize-if-small` | `false` | size以下なら圧縮しない． | |
| `-zip` | `false` | Misskey一括インポート用ZIPアーカイブ (`emojis.zip`) を生成する | ファイル名をひらがなにしておくとエイリアスがある程度自動補完されます |
| `-skip` | `false` | すでにリサイズ先に出力ファイルと同じ名前のファイルが存在する場合，その画像の処理をスキップする |
| `-filename-option` | `false` | ファイル名にオプションをつける | 詳細は後述 |
| `-config` | `なし` | 設定ファイル（JSON）のパスを指定 | パスの指定を省略したらカレントディレクトリに`config.json`または`config`があるかをチェック |
| `-check` | `false` | 変換対象をすべてチェックして変換後の絵文字名に被りがないか検証する | 画像の変換処理は**行わない**．他のオプションを指定したときは，その変換を再現する |
| `-version` | `false` | バージョン情報を表示して終了する | |

### ディレクトリ別のカテゴリ

```bash
./emoji-resizer.exe -zip -category maincat -grouping /path/to/root
```

`root/foo.png` は `maincat`，`root/sub1/bar.png` は `maincat/sub1`，`root/sub2/baz.png` は `maincat/sub2` になります．深い階層も `/` で区切って追加します．基準カテゴリを省略した場合は，相対ディレクトリ名だけをカテゴリにし，ルート直下はカテゴリなしになります．

### ファイル名オプション (個別指定)

`ファイル名@エイリアス(1)@エイリアス(2)@ … @エイリアス(n).オプション.拡張子` のようなファイル名に対応．

エイリアスとファイル名は，どちらもひらがなだったらある程度訓令式とヘボン式で自動補完される．

| オプション | 説明 | 補足 |
| :--- | :--- | :--- |
| `r` | 強制的に `-rect` として扱う | `-auto-rect` を無視 |
| `s` | 強制的に正方形 (`-rect`未指定時) として扱う | `-auto-rect`, `-rect` を無視 |
| `t` | 強制的に `-trim` (透過余白のみトリミング) を有効化して処理する | `r`や`s`と併用可能 (`.tr.png`など) |
| `o` | 強制的に `-trim-opaque` (透過＋ベタ塗り背景トリミング) を有効化して処理する | `r`や`s`と併用可能 (`.ro.png`など) |

例
```
ねこ@ぬこ.r.png
// ねこ, ぬこ, neko, nuko をエイリアスとする長方形の絵文字になる

ほげほげ@ふがふが@ぴよぴよ.s.png
// ほげほげ, ふがふが, ぴよぴよ, hogehoge, hugahuga, fugafuga, piyopiyo をエイリアスとする長方形の絵文字になる

いぬ.t.png
// 透過領域の余白のみを削ってから正方形化・リサイズされる絵文字になる

いぬ.o.png
// 透過領域およびベタ塗り背景を削ってから正方形化・リサイズされる絵文字になる
```

### 設定ファイル (config.json)

オプションをファイルで指定することができます．コマンドライン引数で同じオプションが指定された場合は，コマンドライン引数の値が優先．

`config.json` の記述例:
```json
{
  "size": 128,
  "out": "output",
  "suffix": "_resized",
  "name_prefix": "pref_",
  "name_suffix": "_suf",
  "recursive": true,
  "grouping": false,
  "no_resize": false,
  "no_resise_if_small": false,
  "rect": false,
  "trim": false,
  "trim_opaque": false,
  "zip": true,
  "auto_rect": 1.618,
  "skip": false,
  "filename_option": true,
  "category": "イラスト",
  "license": "CC-BY-4.0"
}
```

※ `category` および `license` を設定ファイルに記述しておくと，`-zip` 指定時の対話式プロンプト入力をスキップできます．

例
```bash
./emoji-resizer.exe -size 96 -suffix _resized -r sample.png
./emoji-resizer.exe -config path/to/config.json sample.png sample2.png
```
(なんかオプションは手前に付けないと怒られます．)

## リリース

`main` への変更から Release Please がバージョン更新PRを自動作成します．そのPRをマージすると，タグとGitHub Releaseを作成し，GoReleaserが各OSの実行ファイル・アーカイブ・チェックサムを添付します．`package.json` のバージョンと `CHANGELOG.md` もPR内で更新されます．

- コミット名（Squash mergeの場合はPRタイトル）を `fix: 修正内容` にするとパッチ，`feat: 機能内容` にするとマイナー，`feat!: 互換性のない変更内容` にするとメジャーバージョンが上がります．
- `docs:` や `chore:` だけではリリースPRは作成されません．
- 手動リリース済みの `v1.11.0` とそのコミットを起点に移行します．既存のタグ・リリースはそのまま残ります．以後，通常のリリースで手動のバージョン更新やタグ作成は不要です．
- 初回導入時は，GitHubの **Settings → Actions → General → Workflow permissions → Allow GitHub Actions to create and approve pull requests** を有効にします．
- 標準の `GITHUB_TOKEN` を使うため追加のシークレットは不要です．このトークンで作成されたリリースPRでは通常の `pull_request` CIは自動起動しません．公開用ビルドの前にはテストを実行します．リリースPRにも必須CIを課す場合は，GitHub AppまたはPATのトークンを別途設定してください．
- 添付ファイルの公開に失敗した場合は，Actionsの **Release → Run workflow** で対象の既存タグを指定して再実行できます．手動の `v*` タグpushによる公開も引き続き利用できます．

## ライセンス

[MIT License](LICENSE)
