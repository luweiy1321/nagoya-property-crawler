package main

import (
	"context"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	log.Println("=== Testing Japanese Real Estate Websites ===")

	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("exclude-switches", "enable-automation"),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("user-data-dir", "/tmp/chrome-test-profile"),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"),
	}

	sites := []struct {
		name string
		url  string
	}{
		{"SUUMO", "https://suumo.jp/chintai/aichi/nc_nagoya/"},
		{"at-home", "https://www.at-home.co.jp/chintai/aichi/nagoya-city/list/"},
	}

	for _, site := range sites {
		log.Printf("\n=== Testing %s ===", site.name)

		allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
		defer cancel()

		taskCtx, cancel := chromedp.NewContext(allocCtx)
		defer cancel()

		var title string
		var html string

		tasks := []chromedp.Action{
			chromedp.Navigate(site.url),
			chromedp.Sleep(10 * time.Second),
			chromedp.ActionFunc(func(ctx context.Context) error {
				script := `Object.defineProperty(navigator, 'webdriver', {get: () => undefined});`
				return chromedp.Evaluate(script, nil).Do(ctx)
			}),
			chromedp.Sleep(5 * time.Second),
			chromedp.Title(&title),
			chromedp.OuterHTML("body", &html, chromedp.ByQuery),
		}

		if err := chromedp.Run(taskCtx, tasks...); err != nil {
			log.Printf("ERROR: %v", err)
			continue
		}

		log.Printf("Title: %s", title)
		log.Printf("HTML length: %d", len(html))

		// Check for errors
		if contains(title, "404") || contains(title, "Not Found") || contains(title, "見つかりません") {
			log.Printf("WARNING: 404 Not Found")
		} else if contains(title, "Verification") || contains(title, "Human") || contains(title, "チェック") {
			log.Printf("WARNING: Verification required")
		} else {
			log.Printf("SUCCESS: Page loaded")
		}
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
