package main

import (
	"encoding/base64"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io/ioutil"
	"net/http"
	"net/url"
)

// HTMLContent モデル
type HTMLContent struct {
	ID      uint   `gorm:"primaryKey"`
	URL     string `gorm:"unique"`
	Content string `gorm:"type:text"`
}

// URLから画像を取得し、Base64に変換する関数
func imageToBase64(imageURL string) (string, error) {
	resp, err := http.Get(imageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	contentType := resp.Header.Get("Content-Type")
	base64Data := base64.StdEncoding.EncodeToString(data)

	return fmt.Sprintf("data:%s;base64,%s", contentType, base64Data), nil
}

// 外部CSSファイルをインライン化する関数
func inlineCSS(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	css, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("<style>%s</style>", string(css)), nil
}

// 外部JavaScriptファイルをインライン化する関数
func inlineJS(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	js, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("<script>%s</script>", string(js)), nil
}

// 相対URLを絶対URLに変換する関数
func toAbsoluteURL(baseURL, relativeURL string) string {
	u, err := url.Parse(relativeURL)
	if err != nil || u.IsAbs() {
		return relativeURL
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return relativeURL
	}

	return base.ResolveReference(u).String()
}

func getHtml(pageURL string) (string, error) {
	// URLからHTMLを取得
	resp, err := http.Get(pageURL)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// goqueryでHTMLを解析
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		panic(err)
	}

	// CSSリンクをインライン化
	doc.Find("link[rel='stylesheet']").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			absoluteHref := toAbsoluteURL(pageURL, href)
			css, err := inlineCSS(absoluteHref)
			if err == nil {
				s.ReplaceWithHtml(css)
			}
		}
	})

	// JavaScriptリンクをインライン化
	doc.Find("script[src]").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if exists {
			absoluteSrc := toAbsoluteURL(pageURL, src)
			js, err := inlineJS(absoluteSrc)
			if err == nil {
				s.ReplaceWithHtml(js)
			}
		}
	})

	// 画像の埋め込み
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if exists {
			absoluteSrc := toAbsoluteURL(pageURL, src)
			base64Data, err := imageToBase64(absoluteSrc)
			if err == nil {
				s.SetAttr("src", base64Data)
			}
		}
	})

	// HTMLコンテンツを取得
	html, err := doc.Html()

	return html, err
}

func main() {
	pageURL := "https://golang.org"

	// データベース接続
	db, err := gorm.Open(sqlite.Open("html_contents.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database")
	}

	// テーブルのマイグレーション
	db.AutoMigrate(&HTMLContent{})

	// HTMLコンテンツを取得
	html, err := getHtml(pageURL)
	if err != nil {
		panic(err)
	}

	// HTMLコンテンツをデータベースに保存
	content := HTMLContent{
		URL:     pageURL,
		Content: html,
	}

	// データベースに保存
	db.Create(&content)

	fmt.Println("HTMLコンテンツがデータベースに保存されました。")
}
