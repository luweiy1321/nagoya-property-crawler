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
		chromedp.Sleep(10 * time.Second),
		chromedp.OuterHTML("body", &html, chromedp.ByQuery),
	}

	if err := chromedp.Run(taskCtx, tasks...); err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("=== Analyzing HOMES ===\n")
	log.Printf("HTML length: %d\n\n", len(html))

	// Find unitListBody positions
	searchStr := `<div class="unitListBody`
	pos := 0
	foundCount := 0

	for {
		idx := strings.Index(html[pos:], searchStr)
		if idx == -1 {
			break
		}
		foundCount++
		absPos := pos + idx

		if foundCount <= 2 {
			// Find the end of this div
			endPos := findEndOfDiv(html, absPos)
			content := html[absPos:min(endPos, absPos+3000)]

			log.Printf("=== Card %d (position: %d, length: %d) ===\n", foundCount, absPos, len(content))
			log.Printf("%s\n\n", content)
		}

		pos = absPos + len(searchStr)
		if foundCount >= 3 {
			break
		}
	}

	log.Printf("Total unitListBody found: %d\n", foundCount)
}

func findEndOfDiv(html string, start int) int {
	// Find the end tag for this unitListBody div
	depth := 1
	i := start + len(`<div class="unitListBody`)
	for i < len(html) && i < start+5000 {
		if strings.HasPrefix(html[i:], `<div`) {
			depth++
			i += 4
		} else if strings.HasPrefix(html[i:], `</div>`) {
			depth--
			if depth == 0 {
				return i + 6
			}
			i += 6
		} else {
			i++
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
