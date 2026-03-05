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

	log.Printf("HTML length: %d", len(html))

	// Find all class attributes in the HTML
	classPattern := regexp.MustCompile(`class="([^"]+)"`)
	classes := make(map[string]int)

	matches := classPattern.FindAllStringSubmatch(html, -1)
	for _, match := range matches {
		if len(match) > 1 {
			// Split by space to get individual classes
			for _, cls := range strings.Fields(match[1]) {
				// Focus on classes that might be property-related
				if strings.Contains(cls, "estate") ||
				   strings.Contains(cls, "property") ||
				   strings.Contains(cls, "buken") ||
				   strings.Contains(cls, "item") ||
				   strings.Contains(cls, "unit") ||
				   strings.Contains(cls, "list") {
					classes[cls]++
				}
			}
		}
	}

	log.Printf("\n=== Property-related classes found ===")
	for cls, count := range classes {
		if count > 5 {
			log.Printf("  %s: %d occurrences", cls, count)
		}
	}

	// Look for div patterns
	log.Printf("\n=== Looking for property container patterns ===")
	patterns := []string{
		`<div class="[^"]*prg-estateListItem`,
		`<div class="[^"]*estateListItem`,
		`<div class="[^"]*estate`,
		`<div class="[^"]*property`,
		`<div class="[^"]*buken`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		count := len(re.FindAllString(html, -1))
		if count > 0 {
			log.Printf("  %s: %d matches", pattern, count)
		}
	}
}
