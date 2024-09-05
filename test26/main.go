package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Page データベースのモデル
type HTMLContent struct {
	ID      uint   `gorm:"primaryKey"`
	URL     string `gorm:"unique"`
	Content string `gorm:"type:text"`
}

func main() {
	// GORMのセットアップ
	db, err := gorm.Open(sqlite.Open("html_contents.db"), &gorm.Config{})
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		return
	}

	// すべてのページを取得
	var pages []HTMLContent
	db.Find(&pages)

	// ページごとにファイルを作成して保存
	for _, page := range pages {
		filename := fmt.Sprintf("%d.html", page.ID)
		filePath := filepath.Join(".", filename)

		// ファイルに書き込む
		err := os.WriteFile(filePath, []byte(page.Content), 0644)
		if err != nil {
			fmt.Printf("Failed to write file %s: %v\n", filename, err)
			continue
		}
		fmt.Printf("Content saved to file: %s\n", filename)
	}
}
