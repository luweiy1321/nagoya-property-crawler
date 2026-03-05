package main

import (
	"context"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("exclude-switches", "enable-automation"),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("user-data-dir", "/tmp/chrome-test-profile"),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"),
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Use a real HOMES detail URL
	detailURL := "https://www.homes.co.jp/chintai/room/c3c023313ac42d4bd80e8d60ccc4ee8c00929c9f/"

	var title string
	var html string

	tasks := []chromedp.Action{
		chromedp.Navigate(detailURL),
		chromedp.Sleep(8 * time.Second),

		chromedp.ActionFunc(func(ctx context.Context) error {
			script := `Object.defineProperty(navigator, 'webdriver', {get: () => undefined});`
			return chromedp.Evaluate(script, nil).Do(ctx)
		}),

		chromedp.Sleep(5 * time.Second),

		chromedp.Title(&title),
		chromedp.OuterHTML("body", &html, chromedp.ByQuery),
	}

	if err := chromedp.Run(taskCtx, tasks...); err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("=== HOMES Detail Page ===\n")
	log.Printf("URL: %s\n", detailURL)
	log.Printf("Title: %s\n", title)
	log.Printf("HTML length: %d\n", len(html))

	// Check for verification
	if contains(title, "Verif") || contains(title, "Human") || contains(title, "チェック") {
		log.Printf("WARNING: Verification required!\n")
	}

	// Look for data patterns
	patterns := []string{
		"所在地",
		"賃料",
		"間取り",
		"築",
		"駅",
	}

	for _, pattern := range patterns {
		count := countOccurrences(html, pattern)
		log.Printf("'%s': %d occurrences", pattern, count)
	}

	// Find and show context around "所在地"
	idx := findIndex(html, "所在地")
	if idx >= 0 {
		start := idx - 50
		if start < 0 { start = 0 }
		end := idx + 300
		if end > len(html) { end = len(html) }
		log.Printf("\n=== Context around '所在地' ===")
		log.Printf("%s", html[start:end])
	}

	// Find and show context around "streetAddress"
	idx2 := findIndex(html, "streetAddress")
	if idx2 >= 0 {
		start := idx2 - 50
		if start < 0 { start = 0 }
		end := idx2 + 300
		if end > len(html) { end = len(html) }
		log.Printf("\n=== Context around 'streetAddress' ===")
		log.Printf("%s", html[start:end])
	}

	// Show first 1000 chars
	log.Printf("\n=== First 1000 chars of HTML ===\n")
	if len(html) > 1000 {
		log.Printf("%s", html[:1000])
	} else {
		log.Printf("%s", html)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstr(s, substr)
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func findIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
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
