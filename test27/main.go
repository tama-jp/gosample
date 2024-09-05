package main

import (
	"encoding/base64"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/mmcdole/gofeed"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// RSSフィードのトップレベル構造
type RssFeed struct {
	ID            uint `gorm:"primaryKey"`
	Title         string
	Description   string
	Link          string    `gorm:"uniqueIndex:idx_feed_link_version"` // リンクはユニークだが方式とバージョン番号ごとに区別
	Type          string    // RSSの方式（RSS, Atomなど）
	VersionNumber string    // バージョン番号（1.0, 2.0など）
	Updated       time.Time // フィードの更新日時
	URL           string    `gorm:"unique"` // フィードのURLを保持する
	LastFetched   time.Time // 最後に取得した時間
	Items         []RssItem `gorm:"foreignKey:FeedID"`
}

// RSSアイテム（詳細）の構造
type RssItem struct {
	ID       uint `gorm:"primaryKey"`
	FeedID   uint
	Title    string
	Link     string
	Content  string
	Content2 string `gorm:"type:text"`
}

// RSS方式（Type）とバージョン番号を判別する関数
func getRSSTypeAndVersion(feed *gofeed.Feed) (string, string) {
	switch feed.FeedType {
	case "rss":
		// RSS 1.0かRSS 2.0を判別
		if feed.FeedVersion == "1.0" {
			return "RSS", "1.0"
		}
		return "RSS", "2.0"
	case "atom":
		return "Atom", feed.FeedVersion
	default:
		return "Unknown", ""
	}
}

// フィードの更新が必要かどうかをチェック
func isFeedUpdated(existingFeed RssFeed, feed *gofeed.Feed, updatedTime time.Time) bool {
	return updatedTime.After(existingFeed.Updated) ||
		existingFeed.Title != feed.Title ||
		existingFeed.Description != feed.Description
}

// アイテムを保存または更新する処理
func updateFeedItems(db *gorm.DB, feedID uint, feed *gofeed.Feed) error {
	// 既存のアイテムを一度削除して新しいものを挿入
	err := db.Where("feed_id = ?", feedID).Delete(&RssItem{}).Error
	if err != nil {
		return err
	}

	// 新しいアイテムを挿入
	for _, item := range feed.Items {

		html, err := getHtml(item.Link)

		if err != nil {
			continue
		}

		rssItem := RssItem{
			FeedID:   feedID,
			Title:    item.Title,
			Link:     item.Link,
			Content:  item.Description,
			Content2: html,
		}
		if err := db.Create(&rssItem).Error; err != nil {
			return err
		}
	}

	return nil
}

// フィードを更新する処理
func updateFeed(db *gorm.DB, existingFeed *RssFeed, feed *gofeed.Feed, updatedTime time.Time) error {
	existingFeed.Title = feed.Title
	existingFeed.Description = feed.Description
	existingFeed.Updated = updatedTime
	existingFeed.LastFetched = time.Now()

	// データベースにフィードを更新
	if err := db.Save(existingFeed).Error; err != nil {
		return err
	}

	// アイテムの更新
	return updateFeedItems(db, existingFeed.ID, feed)
}

// フィードを新規作成する処理
func createFeed(db *gorm.DB, feed *gofeed.Feed, url string, updatedTime time.Time) error {
	// RSS方式とバージョン番号を取得
	feedType, versionNumber := getRSSTypeAndVersion(feed)

	// 新規フィードの作成
	newFeed := RssFeed{
		Title:         feed.Title,
		Description:   feed.Description,
		Link:          feed.Link,
		Type:          feedType,
		VersionNumber: versionNumber,
		Updated:       updatedTime,
		URL:           url,
		LastFetched:   time.Now(),
	}

	// フィードを保存
	if err := db.Create(&newFeed).Error; err != nil {
		return err
	}

	// アイテムの挿入
	return updateFeedItems(db, newFeed.ID, feed)
}

// フィードの挿入または更新処理
func upsertFeedByURL(db *gorm.DB, feed *gofeed.Feed, url string) error {
	// フィードの更新日時を取得
	updatedTime := time.Now()
	if feed.UpdatedParsed != nil {
		updatedTime = *feed.UpdatedParsed
	}

	// URLを基準にして既存フィードを検索
	var existingFeed RssFeed
	result := db.Where("url = ?", url).First(&existingFeed)

	if result.Error == nil {
		// フィードが存在する場合、更新が必要かチェック
		if !isFeedUpdated(existingFeed, feed, updatedTime) {
			fmt.Println("フィードの更新なし。スキップします。")
			return nil
		}

		// フィードの更新処理
		return updateFeed(db, &existingFeed, feed, updatedTime)
	} else if result.Error == gorm.ErrRecordNotFound {
		// 新規フィードの作成処理
		return createFeed(db, feed, url, updatedTime)
	}

	return result.Error
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

// RssItemのContent2をファイルに出力する関数
func exportContentToHTML(db *gorm.DB) error {
	// RssItemの全件取得
	var items []RssItem
	if err := db.Find(&items).Error; err != nil {
		return err
	}

	// 各アイテムのContent2をHTMLファイルに出力
	for _, item := range items {
		// ファイル名をID.htmlとして設定
		fileName := fmt.Sprintf("%d.html", item.ID)

		// ファイルを作成
		file, err := os.Create(fileName)
		if err != nil {
			return fmt.Errorf("failed to create file: %v", err)
		}

		// Content2をファイルに書き込む
		_, err = file.WriteString(item.Content2)
		if err != nil {
			file.Close() // ファイルを閉じる
			return fmt.Errorf("failed to write content to file: %v", err)
		}

		// ファイルを閉じる
		file.Close()
		fmt.Printf("Exported %s\n", fileName)
	}

	return nil
}

func main() {
	// SQLiteデータベース接続
	db, err := gorm.Open(sqlite.Open("rss.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}

	// テーブルのマイグレーション（自動生成）
	err = db.AutoMigrate(&RssFeed{}, &RssItem{})
	if err != nil {
		log.Fatal("failed to migrate database schema")
	}

	// フィードパーサーの初期化
	fp := gofeed.NewParser()

	// 複数のRSSフィードURL
	urls := []string{
		"https://rss.nytimes.com/services/xml/rss/nyt/World.xml",
		"https://feeds.bbci.co.uk/news/world/rss.xml",
		"https://www.npr.org/rss/rss.php?id=1001",
		"http://rss.cnn.com/rss/cnn_topstories.rss",
		"http://feeds.reuters.com/reuters/topNews",
		"https://www.theguardian.com/world/rss",
		"https://www.aljazeera.com/xml/rss/all.xml",
		"https://hnrss.org/frontpage",
		"http://feeds.feedburner.com/TechCrunch/",
		"https://xkcd.com/atom.xml", // 特定のフィード
	}

	// 各URLのフィードを取得して処理
	for _, url := range urls {
		//fmt.Printf("Fetching feed from: %s\n", url)
		feed, err := fp.ParseURL(url)
		if err != nil {
			fmt.Printf("Failed to fetch feed: %s\n", err)
			continue
		}

		// フィードの挿入または更新（URLを基準に）
		err = upsertFeedByURL(db, feed, url)
		if err != nil {
			fmt.Printf("Failed to upsert feed: %s\n", err)
		}
	}

	//// データベースのRssItemをHTMLにエクスポート
	//err = exportContentToHTML(db)
	//if err != nil {
	//	log.Fatal("failed to export content to HTML: ", err)
	//}

	fmt.Println("すべてのRSSフィードURLとデータベース操作が完了しました。")
}
