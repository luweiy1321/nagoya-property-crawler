package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "host=localhost port=5432 user=lw dbname=nagoya_properties sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check all sources
	fmt.Println("数据源统计:")
	rows, err := db.Query("SELECT source, COUNT(*) as count FROM properties GROUP BY source")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var source string
		var count int
		rows.Scan(&source, &count)
		fmt.Printf("  %s: %d\n", source, count)
	}

	// Check homes properties with complete data
	var complete int
	err = db.QueryRow("SELECT COUNT(*) FROM properties WHERE source = 'homes' AND title != '' AND address != '' AND area > 0").Scan(&complete)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nhomes完整数据房源（有标题、地址、面积）: %d\n", complete)

	// Show sample homes properties
	fmt.Println("\nhomes示例房源:")
	rows2, err := db.Query("SELECT property_id, title, address, area, price_display FROM properties WHERE source = 'homes' AND title != '' AND address != '' LIMIT 10")
	if err != nil {
		log.Fatal(err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var id, title, address, price string
		var area float64
		rows2.Scan(&id, &title, &address, &area, &price)
		fmt.Printf("  %s: %s | %s | %.2f㎡ | %s\n", id, title, address, area, price)
	}
}
