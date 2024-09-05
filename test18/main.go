package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io"
	"net/http"
	"net/url"
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

	// Collyのセットアップ
	c := colly.NewCollector()

	// HTMLの保存
	c.OnHTML("html", func(e *colly.HTMLElement) {
		htmlContent := e.DOM

		// 画像をBase64で埋め込む
		htmlContent.Find("img").Each(func(i int, s *goquery.Selection) {
			imgSrc, exists := s.Attr("src")
			if exists {
				// 相対パスを絶対パスに変換
				imgURL := resolveURL(e.Request.URL.String(), imgSrc)

				// 画像をBase64に変換
				base64Image, err := downloadAndConvertToBase64(imgURL)
				if err == nil {
					// <img>タグのsrcをBase64に置き換え
					s.SetAttr("src", base64Image)
				}
			}
		})

		// HTMLの中身をテキストとして取得
		finalHTML, _ := htmlContent.Html()

		// データベースに保存
		page := Page{Content: finalHTML}
		db.Create(&page)
		fmt.Println("HTML content with embedded images saved to database.")
	})

	// 指定のURLを訪問
	c.Visit("https://go.dev")
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
