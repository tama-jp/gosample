package main

import (
	"encoding/base64"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"net/http"
	"net/url"
)

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

func main() {
	pageURL := "https://golang.org"

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

	// 最終的なHTMLをファイルに保存
	html, err := doc.Html()
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile("full_page_with_embeds.html", []byte(html), 0644)
	if err != nil {
		panic(err)
	}

	fmt.Println("ページが 'full_page_with_embeds.html' に保存されました。")
}
