package main

import (
	"context"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("exclude-switches", "enable-automation"),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("user-data-dir", "/tmp/chrome-homes-profile"),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"),
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	var html string

	tasks := []chromedp.Action{
		chromedp.Navigate("https://www.homes.co.jp/chintai/aichi/nagoya/list/"),
		chromedp.Sleep(10 * time.Second),

		// Inject anti-detection script
		chromedp.ActionFunc(func(ctx context.Context) error {
			script := `
				Object.defineProperty(navigator, 'webdriver', {get: () => undefined});
			`
			return chromedp.Evaluate(script, nil).Do(ctx)
		}),

		chromedp.Sleep(5 * time.Second),
		chromedp.OuterHTML("body", &html, chromedp.ByQuery),
	}

	if err := chromedp.Run(taskCtx, tasks...); err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("=== HOMES HTML Analysis ===\n")
	log.Printf("HTML length: %d\n\n", len(html))

	// Look for property-related patterns
	patterns := map[string]string{
		"unitListBody":       `<div class="[^"]*unitListBody`,
		"unitList":           `<div class="[^"]*unitList`,
		"property-unit":      `<div class="[^"]*property-unit`,
		"prg-unitListBody":   `<div class="[^"]*prg-unitListBody`,
		"データ内":           `データ内`,
		"賃料":               `賃料`,
	}

	log.Printf("=== Pattern Search ===\n")
	for name, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		count := len(re.FindAllString(html, -1))
		log.Printf("%s: %d matches\n", name, count)
	}

	// Find a sample property card
	log.Printf("\n=== Looking for property cards ===\n")

	// Try to find <a> tags with property links
	linkPattern := regexp.MustCompile(`<a[^>]*href="(/chintai/[^"]+)"[^>]*>([^<]+)</a>`)
	links := linkPattern.FindAllStringSubmatch(html, 10)

	log.Printf("Found %d property links\n", len(links))
	if len(links) > 0 {
		for i, link := range links {
			if i >= 3 {
				break
			}
			log.Printf("  - %s: %s\n", link[1], strings.TrimSpace(link[2]))
		}
	}

	// Look for price patterns
	log.Printf("\n=== Price patterns ===\n")
	pricePattern := regexp.MustCompile(`(\d{1,3}(?:,\d{3})*\s*万円)`)
	prices := pricePattern.FindAllString(html, 5)
	for i, price := range prices {
		log.Printf("  %d. %s\n", i+1, price)
	}

	// Find the main content area
	log.Printf("\n=== Looking for main content ===\n")
	contentPattern := regexp.MustCompile(`<div class="[^"]*(?:content|main|list)[^"]*"[^>]*>`)
	contentMatches := contentPattern.FindAllString(html, 5)
	for i, match := range contentMatches {
		if i >= 3 {
			break
		}
		log.Printf("  %s\n", match[:min(100, len(match))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
