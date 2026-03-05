package main

import (
	"context"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	// Configure Chrome options
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

	var title string
	var html string
	var cassetteCount int

	url := "https://suumo.jp/chintai/aichi/nc_nagoya/"

	tasks := []chromedp.Action{
		chromedp.Navigate(url),
		chromedp.Sleep(5 * time.Second),
		chromedp.Title(&title),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("Page title: %s", title)
			return nil
		}),
		chromedp.OuterHTML("body", &html, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Check if cassetteitem exists
			if len(html) > 0 {
				log.Printf("HTML length: %d", len(html))
				// Simple count
				count := 0
				for i := 0; i < len(html)-12; i++ {
					if html[i:i+12] == "cassetteitem" {
						count++
					}
				}
				cassetteCount = count
				log.Printf("Found 'cassetteitem' %d times", count)
			}
			return nil
		}),
		chromedp.Sleep(10 * time.Second),
	}

	if err := chromedp.Run(taskCtx, tasks...); err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Test completed. Found %d cassetteitem occurrences", cassetteCount)
	log.Printf("First 500 chars of HTML: %s", html[:min(500, len(html))])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
