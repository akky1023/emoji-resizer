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
| `-auto-rect` | `false` | アスペクト比がしきい値を超える場合 `rect` モードとして扱う．引数は1を超えなければならない | 引数未指定時は`-auto-rect=2.5` |
| `-out` | `output/` | 出力ディレクトリのパス指定 | |
| `-suffix` | `なし` | 出力ファイル名に付与する接尾辞 | |
| `-name-prefix` | `なし` | 絵文字名の前に付与する接頭辞 | `-zip`時のみ意味があります |
| `-name-suffix` | `なし` | 絵文字名の後ろに付与する接尾辞 | `-zip`時のみ意味があります |
| `-r` | `false` | 指定したディレクトリ以下のディレクトリもすべて調べる | |
| `-no-resize` | `false` | 正方形にして終了．圧縮しない |  `-rect`指定時はなにもしない |
| `-no-resize-if-small` | `false` | size以下なら圧縮しない． | |
| `-zip` | `false` | Misskey一括インポート用ZIPアーカイブ (`emojis.zip`) を生成する | ファイル名をひらがなにしておくとエイリアスがある程度自動補完されます |
| `-skip` | `false` | すでにリサイズ先に出力ファイルと同じ名前のファイルが存在する場合，その画像の処理をスキップする |
| `-filename-option` | `false` | ファイル名にオプションをつける | 詳細は後述 |
| `-config` | `なし` | 設定ファイル（JSON）のパスを指定 | パスの指定を省略したらカレントディレクトリに`config.json`または`config`があるかをチェック |
| `-check` | `false` | 変換対象をすべてチェックして変換後の絵文字名に被りがないか検証する | 画像の変換処理は**行わない**．他のオプションを指定したときは，その変換を再現する |
| `-version` | `false` | バージョン情報を表示して終了する | |

### ファイル名オプション (個別指定)

`ファイル名@エイリアス(1)@エイリアス(2)@ … @エイリアス(n).オプション.拡張子` のようなファイル名に対応します．

エイリアスとファイル名は，どちらもひらがなだったらある程度訓令式とヘボン式で自動補完されます．

| オプション | 説明 | 補足 |
| :--- | :--- | :--- |
| `r` | 強制的に `-rect` として扱う | `-auto-rect` を無視 |
| `s` | 強制的に正方形 (`-rect`未指定時) として扱う | `-auto-rect`, `-rect` を無視 |

例
```
ねこ@ぬこ.r.png
ほげほげ@ふがふが@ぴよぴよ.r.png
```

### 設定ファイル (config.json)

オプションをファイルで指定することができます．コマンドライン引数で同じオプションが指定された場合は，コマンドライン引数の値が優先されます．

`config.json` の記述例:
```json
{
  "size": 128,
  "out": "output",
  "suffix": "_resized",
  "name_prefix": "pref_",
  "name_suffix": "_suf",
  "r": true,
  "no_resize": false,
  "no_resise_if_small": false,
  "rect": false,
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

## ライセンス

[MIT License](LICENSE)