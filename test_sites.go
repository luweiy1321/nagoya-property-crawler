package main

import (
	"context"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func testURL(name, url string) {
	log.Printf("=== Testing %s ===", name)
	log.Printf("URL: %s", url)

	// Configure Chrome options
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", true),
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

	// Set timeout
	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, 30*time.Second)
	defer cancelTimeout()

	var title string
	var html string

	tasks := []chromedp.Action{
		chromedp.Navigate(url),
		chromedp.Sleep(5 * time.Second),
		chromedp.Title(&title),
		chromedp.OuterHTML("body", &html, chromedp.ByQuery),
	}

	if err := chromedp.Run(taskCtx, tasks...); err != nil {
		log.Printf("ERROR: %v", err)
		return
	}

	log.Printf("Page title: %s", title)
	log.Printf("HTML length: %d", len(html))

	// Check for error indicators
	errorIndicators := []string{"404", "Not Found", "見つかりません", "Error", "エラー"}
	hasError := false
	for _, indicator := range errorIndicators {
		if contains(html, indicator) {
			log.Printf("WARNING: Found error indicator: %s", indicator)
			hasError = true
		}
	}

	if !hasError {
		log.Printf("SUCCESS: Page loaded successfully")
	}

	log.Printf("")
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

func main() {
	log.Println("Testing Japanese Real Estate Websites...")

	// Test HOMES
	testURL("HOMES Rent", "https://www.homes.co.jp/chintai/aichi/nagoya/list/")
	testURL("HOMES Sale", "https://www.homes.co.jp/chuko/aichi/nagoya/list/")

	// Test at-home
	testURL("at-home Rent", "https://www.at-home.co.jp/chintai/aichi/nagoya-city/list/")
	testURL("at-home Sale", "https://www.at-home.co.jp/chuko/aichi/nagoya-city/list/")
}
