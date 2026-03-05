package main

import (
	"context"
	"log"
	"time"

	"nagoya-property-crawler/internal/crawler"
)

func main() {
	log.Println("=== Testing HOMES V3 Crawler (JavaScript-based) ===")

	// Create crawler with visible browser (headless=false) for testing
	homes := crawler.NewHOMESCrawlerV3(
		false, // headless - set to false to see browser
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		180*time.Second, // Increased timeout
	)

	ctx := context.Background()

	// Try to scrape first page of rental listings
	log.Println("Scraping HOMES rental listings page 1...")
	properties, err := homes.ScrapeRentListings(ctx, 1)
	if err != nil {
		log.Fatalf("Failed to scrape: %v", err)
	}

	log.Printf("Successfully scraped %d properties!", len(properties))

	if len(properties) > 0 {
		for i, p := range properties {
			if i >= 3 {
				break
			}
			log.Printf("\n=== Property %d ===", i+1)
			log.Printf("Title: %s", p.Title)
			log.Printf("Price: %s (%d yen)", p.PriceDisplay, p.Price)
			log.Printf("Area: %.2f㎡", p.Area)
			log.Printf("Layout: %s", p.Layout)
			log.Printf("Floor: %s", p.Floor)
			log.Printf("Detail URL: %s", p.DetailURL)
		}
	}
}
