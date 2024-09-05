package main

import (
	"fmt"
	"log"

	"github.com/mmcdole/gofeed"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// RSSフィードのトップレベル構造
type RssFeed struct {
	ID          uint `gorm:"primaryKey"`
	Title       string
	Description string
	Link        string
	Version     string
	Items       []RssItem `gorm:"foreignKey:FeedID"`
}

// RSSアイテム（詳細）の構造
type RssItem struct {
	ID      uint `gorm:"primaryKey"`
	FeedID  uint
	Title   string
	Link    string
	Content string
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

	// RSSフィードのURL（例）
	url := "https://rss.nytimes.com/services/xml/rss/nyt/World.xml"

	// RSSフィードの取得と解析
	feed, err := fp.ParseURL(url)
	if err != nil {
		log.Fatal(err)
	}

	// RSSフィードの情報を構造体に格納
	rssFeed := RssFeed{
		Title:       feed.Title,
		Description: feed.Description,
		Link:        feed.Link,
		Version:     feed.FeedType, // バージョン情報
	}

	// アイテム情報の追加
	for _, item := range feed.Items {
		rssItem := RssItem{
			Title:   item.Title,
			Link:    item.Link,
			Content: item.Description,
		}
		rssFeed.Items = append(rssFeed.Items, rssItem)
	}

	// データベースにフィードとアイテムを保存
	result := db.Create(&rssFeed)
	if result.Error != nil {
		log.Fatal(result.Error)
	}

	fmt.Println("RSSフィードとアイテムがデータベースに保存されました。")
}
