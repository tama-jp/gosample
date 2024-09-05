package main

import (
	"fmt"
	"log"
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
	Items         []RssItem `gorm:"foreignKey:FeedID"`
}

// RSSアイテム（詳細）の構造
type RssItem struct {
	ID      uint `gorm:"primaryKey"`
	FeedID  uint
	Title   string
	Link    string
	Content string
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
		rssItem := RssItem{
			FeedID:  feedID,
			Title:   item.Title,
			Link:    item.Link,
			Content: item.Description,
		}
		if err := db.Create(&rssItem).Error; err != nil {
			return err
		}
	}

	return nil
}

// フィードの保存または更新処理
func upsertFeed(db *gorm.DB, feed *gofeed.Feed) error {
	// フィードの更新日時を取得
	updatedTime := time.Now()
	if feed.UpdatedParsed != nil {
		updatedTime = *feed.UpdatedParsed
	}

	// RSSの方式とバージョン番号を判別
	feedType, versionNumber := getRSSTypeAndVersion(feed)

	// 既存フィードの確認
	var existingFeed RssFeed
	result := db.Where("link = ? AND type = ? AND version_number = ?", feed.Link, feedType, versionNumber).First(&existingFeed)

	if result.Error == nil {
		// 更新が必要かチェック
		if !isFeedUpdated(existingFeed, feed, updatedTime) {
			fmt.Println("フィードの更新なし。スキップします。")
			return nil
		}

		// フィードの更新
		existingFeed.Title = feed.Title
		existingFeed.Description = feed.Description
		existingFeed.Updated = updatedTime

		// データベースにフィードを更新
		if err := db.Save(&existingFeed).Error; err != nil {
			return err
		}

		// アイテムの更新
		return updateFeedItems(db, existingFeed.ID, feed)

	} else if result.Error == gorm.ErrRecordNotFound {
		// フィードが存在しない場合、新規作成
		newFeed := RssFeed{
			Title:         feed.Title,
			Description:   feed.Description,
			Link:          feed.Link,
			Type:          feedType,
			VersionNumber: versionNumber,
			Updated:       updatedTime,
		}

		// フィードを保存
		if err := db.Create(&newFeed).Error; err != nil {
			return err
		}

		// アイテムの挿入
		return updateFeedItems(db, newFeed.ID, feed)
	}

	return result.Error
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
		"https://xkcd.com/atom.xml",
	}

	// 各URLのフィードを取得して処理
	for _, url := range urls {
		fmt.Printf("Fetching feed from: %s\n", url)
		feed, err := fp.ParseURL(url)
		if err != nil {
			fmt.Printf("Failed to fetch feed: %s\n", err)
			continue
		}

		// フィードの挿入または更新
		err = upsertFeed(db, feed)
		if err != nil {
			fmt.Printf("Failed to upsert feed: %s\n", err)
		}
	}

	fmt.Println("すべてのRSSフィードのデータベース操作が完了しました。")
}
