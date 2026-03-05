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
		chromedp.Flag("user-data-dir", "/tmp/chrome-homes-profile"),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"),
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	var cardHTML string

	tasks := []chromedp.Action{
		chromedp.Navigate("https://www.homes.co.jp/chintai/aichi/nagoya/list/"),
		chromedp.Sleep(10 * time.Second),

		// Inject anti-detection
		chromedp.ActionFunc(func(ctx context.Context) error {
			script := `Object.defineProperty(navigator, 'webdriver', {get: () => undefined});`
			return chromedp.Evaluate(script, nil).Do(ctx)
		}),

		chromedp.Sleep(3 * time.Second),

		// Get the first property card directly using chromedp
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Try to get the first unitListBody element
			return chromedp.OuterHTML(`.unitListBody:first-of-type`, &cardHTML, chromedp.ByQuery).Do(ctx)
		}),
	}

	if err := chromedp.Run(taskCtx, tasks...); err != nil {
		log.Printf("Error getting card: %v", err)

		// Fallback: get all unitListBody count
		countTasks := []chromedp.Action{
			chromedp.Navigate("https://www.homes.co.jp/chintai/aichi/nagoya/list/"),
			chromedp.Sleep(10 * time.Second),
			chromedp.ActionFunc(func(ctx context.Context) error {
				script := `Object.defineProperty(navigator, 'webdriver', {get: () => undefined});`
				return chromedp.Evaluate(script, nil).Do(ctx)
			}),
			chromedp.Sleep(3 * time.Second),
			chromedp.ActionFunc(func(ctx context.Context) error {
				var result interface{}
				err := chromedp.Evaluate(`document.querySelectorAll('.unitListBody').length`, &result).Do(ctx)
				if err == nil {
					log.Printf("unitListBody count: %v", result)
				}
				return err
			}),
		}
		chromedp.Run(taskCtx, countTasks...)
		return
	}

	log.Printf("=== First Property Card HTML ===\n")
	log.Printf("%s\n", cardHTML)

	// Also try to get property info using JavaScript
	log.Printf("\n=== Using JavaScript to extract data ===\n")
	var jsResult string
	jsTasks := []chromedp.Action{
		chromedp.Navigate("https://www.homes.co.jp/chintai/aichi/nagoya/list/"),
		chromedp.Sleep(10 * time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			script := `
				Object.defineProperty(navigator, 'webdriver', {get: () => undefined});
			`
			return chromedp.Evaluate(script, nil).Do(ctx)
		}),
		chromedp.Sleep(3 * time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			script := `
				(function() {
					const cards = document.querySelectorAll('.unitListBody');
					if (cards.length === 0) return 'No cards found';

					const first = cards[0];

					// Try to find link
					const link = first.querySelector('a');
					const href = link ? link.getAttribute('href') : 'no link';

					// Try to find text content
					const text = first.textContent.substring(0, 200);

					return 'Link: ' + href + '\\nText: ' + text;
				})();
			`
			return chromedp.Evaluate(script, &jsResult).Do(ctx)
		}),
	}

	if err := chromedp.Run(taskCtx, jsTasks...); err != nil {
		log.Fatalf("JS error: %v", err)
	}

	log.Printf("%s\n", jsResult)
}
