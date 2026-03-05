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

	log.Printf("=== Analyzing HOMES Website Structure ===\n")

	// Find unitList sections
	unitListPattern := regexp.MustCompile(`<div class="[^"]*unitList[^"]*"[^>]*>(.*?)</div>\s*<!-- /unitList -->`)
	unitListMatches := unitListPattern.FindAllStringSubmatch(html, 2)

	if len(unitListMatches) > 0 && len(unitListMatches[0]) > 1 {
		unitListHTML := unitListMatches[0][1]
		log.Printf("Found unitList section (length: %d)\n", len(unitListHTML))

		// Find first property card
		cardPattern := regexp.MustCompile(`<div class="[^"]*unitListBody[^"]*"[^>]*>(.*?)</div>\s*<!-- /unitListBody -->`)
		cardMatches := cardPattern.FindAllStringSubmatch(unitListHTML, 3)

		log.Printf("\nFound %d property cards\n", len(cardMatches))

		if len(cardMatches) > 0 && len(cardMatches[0]) > 1 {
			log.Printf("\n=== First property card HTML (first 2000 chars) ===\n")
			firstCard := cardMatches[0][1]
			if len(firstCard) > 2000 {
				log.Printf("%s...\n", firstCard[:2000])
			} else {
				log.Printf("%s\n", firstCard)
			}

			// Extract key information
			log.Printf("\n=== Extracting Information ===\n")

			// Try to find title
			titlePatterns := []string{
				`<h3[^>]*class="([^"]*)"[^>]*>([^<]+)</h3>`,
				`<h4[^>]*class="([^"]*)"[^>]*>([^<]+)</h4>`,
				`class="([^"]*title[^"]*)"[^>]*>([^<]+)<`,
			}
			for _, pattern := range titlePatterns {
				re := regexp.MustCompile(pattern)
				if matches := re.FindStringSubmatch(firstCard); len(matches) > 2 {
					log.Printf("Title found (pattern: %s): class='%s', text='%s'\n", pattern, matches[1], strings.TrimSpace(matches[2]))
				}
			}

			// Try to find price
			pricePatterns := []string{
				`<span[^>]*class="([^"]*price[^"]*)"[^>]*>([^<]+)</span>`,
				`<td[^>]*class="([^"]*price[^"]*)"[^>]*>([^<]+)</td>`,
				`class="([^"]*price[^"]*)"[^>]*>([^<]+)<`,
			}
			for _, pattern := range pricePatterns {
				re := regexp.MustCompile(pattern)
				if matches := re.FindStringSubmatch(firstCard); len(matches) > 2 {
					log.Printf("Price found (pattern: %s): class='%s', text='%s'\n", pattern, matches[1], strings.TrimSpace(matches[2]))
				}
			}

			// Try to find links
			linkPattern := regexp.MustCompile(`<a[^>]*href="(/chintai/[^"]+)"[^>]*>`)
			if matches := linkPattern.FindAllStringSubmatch(firstCard, 5); len(matches) > 0 {
				log.Printf("\nLinks found:\n")
				for i, match := range matches {
					if i >= 3 {
						break
					}
					log.Printf("  - %s\n", match[1])
				}
			}

			// Try to find images
			imgPattern := regexp.MustCompile(`<img[^>]*src="([^"]+)"[^>]*>`)
			if matches := imgPattern.FindAllStringSubmatch(firstCard, 3); len(matches) > 0 {
				log.Printf("\nImages found:\n")
				for _, match := range matches {
					log.Printf("  - %s\n", match[1])
				}
			}
		}
	} else {
		log.Printf("No unitList section found!\n")
	}

	// Look for property list container
	log.Printf("\n=== Looking for list containers ===\n")
	containerPatterns := []string{
		`<div class="([^"]*)"[^>]*>\s*<div class="[^"]*unitList`,
		`<div class="([^"]*unitList[^"]*)"[^>]*>`,
	}
	for _, pattern := range containerPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindAllStringSubmatch(html, 5); len(matches) > 0 {
			log.Printf("Pattern '%s' found:\n", pattern)
			for i, match := range matches {
				if i >= 3 {
					break
				}
				log.Printf("  - class='%s'\n", match[1])
			}
		}
	}
}
