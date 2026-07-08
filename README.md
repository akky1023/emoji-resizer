# Emoji Resizer (カスタム絵文字用 画像一括整形・リサイズツール)

長い方に合わせて正方形に整形、指定サイズにリサイズする。

## 要件

Windows 11以外では動作未確認

## ビルド

### npm (Recommended)
```bash
npm run build
```

### go コマンド
```bash
go build -o emoji-resizer.exe main.go
```
※ Go 1.18 以上。
※ この方法でビルドした場合、バージョン情報が `devel` になる

## 使い方

ビルドされた実行ファイル (`emoji-resizer.exe` または `emoji-resizer`) をbashで実行。

### 基本構文
対象の画像ファイルまたはディレクトリを指定。スペース区切り。
省略した場合は、カレントディレクトリ直下の画像を全て処理。

```bash
# 特定の画像をリサイズ
./emoji-resizer.exe image1.png image2.webp

# ディレクトリ直下の画像をすべてリサイズ
./emoji-resizer.exe path/to/images/
```

### オプション

* `-size`: リサイズ後の一辺、px単位 (デフォルト: 128)
* `-out`: 出力ディレクトリ指定 (デフォルト: output/)
* `-suffix`: 出力ファイル名に付与する接尾辞 (デフォルト: なし)
* `-r`: 再帰的に画像取得 
* `-no-resize`: リサイズしない (正方形にするだけモード)
* `-version`: バージョン情報を表示して終了

例
```bash
./emoji-resizer.exe -size 256 -suffix _resized sample.png
```

## ライセンス

[MIT License](LICENSE)