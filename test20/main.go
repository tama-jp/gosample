package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/chromedp/chromedp"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Page struct {
	ID      uint   `gorm:"primaryKey"`
	Content string `gorm:"type:text"` // テキストとしてHTMLを保存
}

func main() {
	// データベース接続のセットアップ
	db, err := gorm.Open(sqlite.Open("example.db"), &gorm.Config{})
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		return
	}
	db.AutoMigrate(&Page{})

	// Chromeのコンテキストを作成
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// ターゲットのURL
	url := "https://go.dev"

	// 結果を保存する変数
	var res string

	// タスクを実行してHTMLを取得
	err = chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),     // ページが表示されるまで待つ
		chromedp.OuterHTML("html", &res, chromedp.ByQuery), // HTMLを取得
	)
	if err != nil {
		fmt.Println("Failed to fetch HTML:", err)
		return
	}

	// GoQueryで画像を処理
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(res))
	if err != nil {
		fmt.Println("Failed to parse HTML:", err)
		return
	}

	// 画像をBase64で埋め込む
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		imgSrc, exists := s.Attr("src")
		if exists {
			imgURL := resolveURL(url, imgSrc)
			base64Image, err := downloadAndConvertToBase64(imgURL)
			if err == nil {
				s.SetAttr("src", base64Image)
			}
		}
	})

	// HTMLの中身をテキストとして取得
	finalHTML, _ := doc.Html()

	// データベースに保存
	page := Page{Content: finalHTML}
	db.Create(&page)

	fmt.Println("HTML content with embedded images saved to database.")
}

// URLを解決して絶対パスに変換
func resolveURL(baseURL, relativePath string) string {
	u, err := url.Parse(relativePath)
	if err != nil || u.IsAbs() {
		return relativePath
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return relativePath
	}
	return base.ResolveReference(u).String()
}

// 画像をBase64に変換する
func downloadAndConvertToBase64(imgSrc string) (string, error) {
	resp, err := http.Get(imgSrc)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var imgBuf bytes.Buffer
	_, err = io.Copy(&imgBuf, resp.Body)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(imgBuf.Bytes())
	contentType := http.DetectContentType(imgBuf.Bytes())

	return fmt.Sprintf("data:%s;base64,%s", contentType, encoded), nil
}
