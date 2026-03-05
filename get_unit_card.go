package main

import (
	"context"
	"log"
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
		chromedp.Sleep(8 * time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			script := `Object.defineProperty(navigator, 'webdriver', {get: () => undefined});`
			return chromedp.Evaluate(script, nil).Do(ctx)
		}),
		chromedp.Sleep(5 * time.Second),
		chromedp.OuterHTML("body", &html, chromedp.ByQuery),
	}

	if err := chromedp.Run(taskCtx, tasks...); err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Find the first unitListBody
	searchStr := `<div class="unitListBody`
	idx := strings.Index(html, searchStr)

	if idx != -1 {
		// Find the end of this div (simplified)
		endIdx := idx + len(searchStr)
		depth := 1
		i := endIdx

		for i < len(html) && i < idx+4000 {
			if strings.HasPrefix(html[i:], `<div`) {
				depth++
				i += 4
			} else if strings.HasPrefix(html[i:], `</div>`) {
				depth--
				if depth == 0 {
					endIdx = i + 6
					break
				}
				i += 6
			} else {
				i++
			}
		}

		if endIdx > idx {
			cardHTML := html[idx:min(endIdx, idx+3500)]
			log.Printf("=== First Property Card (unitListBody) ===\n")
			log.Printf("%s\n", cardHTML)
		}
	} else {
		log.Printf("unitListBody not found!\n")
	}

	// Also look for property link patterns
	log.Printf("\n=== Looking for property detail links ===\n")
	linksStart := 0
	linkCount := 0

	for linkCount < 5 {
		// Find /chintai/.../.../ pattern (property detail links)
		linkIdx := strings.Index(html[linksStart:], `/chintai/`)
		if linkIdx == -1 {
			break
		}
		absIdx := linksStart + linkIdx

		// Extract full URL
		endIdx := strings.Index(html[absIdx:], `"`)
		if endIdx != -1 {
			url := html[absIdx:absIdx+endIdx]
			if strings.Contains(url, `/aichi/`) && !strings.Contains(url, `/list/`) {
				linkCount++
				log.Printf("%d. %s\n", linkCount, url)
			}
		}
		linksStart = absIdx + 10
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
