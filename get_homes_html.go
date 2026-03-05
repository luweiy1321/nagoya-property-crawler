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
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	url := "https://www.homes.co.jp/chintai/aichi/nagoya/list/"

	var html string

	tasks := []chromedp.Action{
		chromedp.Navigate(url),
		chromedp.Sleep(8 * time.Second),
		chromedp.OuterHTML("body", &html, chromedp.ByQuery),
	}

	if err := chromedp.Run(taskCtx, tasks...); err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("=== Analyzing HOMES ===\n")

	// Look for unitListBody without requiring closing comment
	cardPattern := regexp.MustCompile(`<div class="[^"]*unitListBody[^"]*"[^>]*>(?P<content>.*?)(?=<div class="[^"]*unitListBody|<!-- /unitListBody|<div class="[^"]*unitList[^"]*|</div>\s*$)`)
	cardMatches := cardPattern.FindAllStringSubmatch(html, 2)

	if len(cardMatches) > 0 {
		log.Printf("Found %d cards with pattern 1\n", len(cardMatches))
	}

	// Try simpler pattern - just get everything between unitListBody tags
	simplePattern := regexp.MustCompile(`<div class="[^"]*unitListBody[^"]*"[^>]*>`)
	matches := simplePattern.FindAllStringIndex(html, 3)

	log.Printf("Found %d unitListBody div starts\n", len(matches))

	if len(matches) > 0 {
		for i, match := range matches {
			if i >= 2 {
				break
			}
			start := match[1]
			// Find the end of this div (look for closing divs at same level)
			end := findMatchingEnd(html, start)
			if end > start && end < start+5000 {
				content := html[start:end]
				log.Printf("\n=== Card %d HTML (length: %d) ===\n", i+1, len(content))
				log.Printf("%s\n", content[:min(2000, len(content))])
			}
		}
	}

	// Also save full HTML for inspection
	log.Printf("\n=== Full HTML length: %d ===\n", len(html))

	// Look for specific patterns
	log.Printf("\n=== Looking for price patterns ===\n")
	pricePatterns := []string{
		`万円`,
		`円`,
		`賃料`,
		`価格`,
	}
	for _, pattern := range pricePatterns {
		count := strings.Count(html, pattern)
		log.Printf("'%s': %d occurrences\n", pattern, count)
	}
}

func findMatchingEnd(html string, start int) int {
	// Simple heuristic: find the closing div for unitListBody
	depth := 1
	for i := start; i < len(html) && i < start+5000; i++ {
		if strings.HasPrefix(html[i:], "<div") {
			depth++
		} else if strings.HasPrefix(html[i:], "</div>") {
			depth--
			if depth == 0 {
				return i + 6
			}
		}
	}
	return start + 2000 // fallback
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
