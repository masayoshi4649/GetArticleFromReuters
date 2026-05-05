package main

import (
	"fmt"
	"os"
)

// main はアプリケーションのエントリーポイントです。
//
// 機能:
//   - runを呼び出してアプリケーション処理を開始する
//   - エラー発生時は標準エラーへ内容を出力する
//   - エラー発生時は終了コード1で異常終了する
//
// 引数:
//   - なし
//
// 返り値:
//   - なし
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
