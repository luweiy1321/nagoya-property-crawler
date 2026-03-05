package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("metrics-recording-only", true),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("safebrowsing-disable-auto-update", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	url := "https://www.homes.co.jp/chintai/aichi/nagoya/list/"

	var html string
	var foundElements []string

	tasks := []chromedp.Action{
		chromedp.Navigate(url),
		chromedp.Sleep(5 * time.Second),
		chromedp.OuterHTML("body", &html, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Check for various possible selectors
			selectors := []string{
				"prg-estateList",
				"estateList",
				"module",
				"property",
				"buken",
				"property-unit",
			}

			for _, sel := range selectors {
				count := countOccurrences(html, sel)
				if count > 0 {
					foundElements = append(foundElements, fmt.Sprintf("%s: %d", sel, count))
				}
			}
			return nil
		}),
		chromedp.Sleep(10 * time.Second),
	}

	if err := chromedp.Run(taskCtx, tasks...); err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("HTML length: %d", len(html))
	log.Printf("Found elements:")
	for _, elem := range foundElements {
		log.Printf("  - %s", elem)
	}

	// Show first 1000 chars
	start := 0
	if len(html) > 1000 {
		start = 1000
	}
	log.Printf("First %d chars: %s", start, html[:start])
}

func countOccurrences(s, substr string) int {
	count := 0
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			count++
		}
	}
	return count
}
