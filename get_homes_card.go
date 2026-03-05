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

	// Find unitListBody cards directly
	cardPattern := regexp.MustCompile(`<div class="[^"]*unitListBody[^"]*"[^>]*>(.*?)</div>\s*<!-- /unitListBody -->`)
	cardMatches := cardPattern.FindAllStringSubmatch(html, -1)

	log.Printf("=== Found %d property cards ===\n", len(cardMatches))

	if len(cardMatches) > 0 && len(cardMatches[0]) > 1 {
		firstCard := cardMatches[0][1]

		log.Printf("\n=== First property card HTML ===\n")
		log.Printf("%s\n", firstCard)

		// Now extract information using patterns
		log.Printf("\n=== Extracted Information ===\n")

		// Title
		titlePattern := regexp.MustCompile(`<h3[^>]*class="([^"]*)"[^>]*>([^<]*)</h3>`)
		if matches := titlePattern.FindStringSubmatch(firstCard); len(matches) > 2 {
			log.Printf("Title: class='%s', text='%s'\n", matches[1], strings.TrimSpace(matches[2]))
		}

		// Price
		pricePattern := regexp.MustCompile(`(<td[^>]*class="[^"]*"[^>]*>[^<]*<span[^>]*class="([^"]*)"[^>]*>([^<]+)</span>[^<]*</td>)`)
		if matches := pricePattern.FindAllStringSubmatch(firstCard, -1); len(matches) > 0 {
			log.Printf("\nPrice sections found:\n")
			for i, match := range matches {
				if i >= 3 {
					break
				}
				log.Printf("  %s\n", match[1])
			}
		}

		// Links
		linkPattern := regexp.MustCompile(`<a[^>]*href="(/[^"]+)"[^>]*>`)
		if matches := linkPattern.FindAllStringSubmatch(firstCard, -1); len(matches) > 0 {
			log.Printf("\nLinks found:\n")
			for i, match := range matches {
				if i >= 5 {
					break
				}
				if strings.Contains(match[1], "chintai") {
					log.Printf("  PROPERTY: %s\n", match[1])
				}
			}
		}

		// Images
		imgPattern := regexp.MustCompile(`<img[^>]*src="([^"]+)"[^>]*>`)
		if matches := imgPattern.FindAllStringSubmatch(firstCard, 3); len(matches) > 0 {
			log.Printf("\nImages found:\n")
			for _, match := range matches {
				log.Printf("  %s\n", match[1])
			}
		}
	}

	// Also check second card
	if len(cardMatches) > 1 && len(cardMatches[1]) > 1 {
		log.Printf("\n=== Second property card (for comparison) ===\n")
		log.Printf("%s\n", cardMatches[1][1])
	}
}
