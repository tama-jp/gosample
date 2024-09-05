package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"golang.org/x/net/html"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io"
	"net/http"
	"net/url"
)

type Page struct {
	ID      uint   `gorm:"primaryKey"`
	Content []byte `gorm:"type:blob"` // BLOB型でHTMLの内容を保存
}

func main() {
	// 対象のURL
	urlStr := "https://golang.org"

	// URLのHTMLを取得
	resp, err := http.Get(urlStr)
	if err != nil {
		fmt.Println("Failed to fetch the URL:", err)
		return
	}
	defer resp.Body.Close()

	// HTMLデータをパース
	doc, err := html.Parse(resp.Body)
	if err != nil {
		fmt.Println("Failed to parse HTML:", err)
		return
	}

	// 画像をBase64に埋め込み、<script>タグを除去し、相対パスを絶対パスに変換
	baseURL, _ := url.Parse(urlStr)
	var buf bytes.Buffer
	cleanHTML(&buf, doc, baseURL)

	// 埋め込み済みHTMLをデータベースに直接保存
	saveToDatabaseWithGORM(buf.Bytes())
}

// 画像をBase64に変換して埋め込む、<script>タグを除去、相対パスを絶対パスに変換
func cleanHTML(w io.Writer, node *html.Node, baseURL *url.URL) {
	if node.Type == html.ElementNode {
		// <script>タグを除去
		if node.Data == "script" {
			return
		}
		// 相対URLを絶対URLに変換
		for i, attr := range node.Attr {
			if attr.Key == "src" || attr.Key == "href" {
				attrURL, err := url.Parse(attr.Val)
				if err == nil && !attrURL.IsAbs() {
					attr.Val = baseURL.ResolveReference(attrURL).String()
					node.Attr[i] = attr
				}
			}
		}
	}

	// 子ノードを再帰的に処理
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		cleanHTML(w, c, baseURL)
	}

	// 結果のHTMLを生成
	html.Render(w, node)
}

// GORMを使ってファイルの内容をデータベースに保存
func saveToDatabaseWithGORM(content []byte) {
	// データベース接続
	db, err := gorm.Open(sqlite.Open("example.db"), &gorm.Config{})
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		return
	}

	// テーブルが存在しない場合、作成
	err = db.AutoMigrate(&Page{})
	if err != nil {
		fmt.Println("Failed to migrate database:", err)
		return
	}

	// ファイルの内容をPageとして保存
	page := Page{Content: content}
	result := db.Create(&page)
	if result.Error != nil {
		fmt.Println("Failed to save content to database:", result.Error)
		return
	}

	fmt.Println("File content saved to database successfully!")
}

// 画像をダウンロードしてBase64にエンコードする
func downloadAndEncodeImage(imageURL string) (string, error) {
	resp, err := http.Get(imageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 画像データを読み込み
	var imgBuf bytes.Buffer
	_, err = io.Copy(&imgBuf, resp.Body)
	if err != nil {
		return "", err
	}

	// Base64エンコード
	encoded := base64.StdEncoding.EncodeToString(imgBuf.Bytes())

	// 画像のContent-Typeを推測
	contentType := http.DetectContentType(imgBuf.Bytes())

	// Data URLとして返す
	return fmt.Sprintf("data:%s;base64,%s", contentType, encoded), nil
}
