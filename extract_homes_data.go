package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

type PropertyData struct {
	Title       string  `json:"title"`
	Price       string  `json:"price"`
	Area        string  `json:"area"`
	Layout      string  `json:"layout"`
	Floor       string  `json:"floor"`
	Link        string  `json:"link"`
	Image       string  `json:"image"`
	Station     string  `json:"station"`
	WalkMinutes string  `json:"walkMinutes"`
}

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

	var result string

	tasks := []chromedp.Action{
		chromedp.Navigate("https://www.homes.co.jp/chintai/aichi/nagoya/list/"),
		chromedp.Sleep(12 * time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			script := `Object.defineProperty(navigator, 'webdriver', {get: () => undefined});`
			return chromedp.Evaluate(script, nil).Do(ctx)
		}),
		chromedp.Sleep(3 * time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			script := `
				(function() {
					const properties = [];
					const cards = document.querySelectorAll('.unitListBody');

					console.log('Found cards:', cards.length);

					for (let i = 0; i < Math.min(cards.length, 5); i++) {
						const card = cards[i];

						// Find link
						const linkEl = card.querySelector('a');
						const link = linkEl ? linkEl.getAttribute('href') : '';

						// Find image
						const imgEl = card.querySelector('img');
						const img = imgEl ? imgEl.getAttribute('src') : '';

						// Get all text content
						const text = card.textContent;

						// Extract price (pattern: X万円)
						const priceMatch = text.match(/(\d{1,3}(?:,\d{3})*\s*万円)/);
						const price = priceMatch ? priceMatch[1] : '';

						// Extract area (pattern: XX.XX㎡)
						const areaMatch = text.match(/(\d+(?:\.\d+)?)\s*㎡/);
						const area = areaMatch ? areaMatch[1] : '';

						// Extract layout (pattern: 1LDK, 2DK, etc.)
						const layoutMatch = text.match(/[1-3][LDK][A-Z]?/);
						const layout = layoutMatch ? layoutMatch[0] : '';

						// Extract floor
						const floorMatch = text.match(/(\d+)階/);
						const floor = floorMatch ? floorMatch[1] : '';

						// Extract station info
						const stationMatch = text.match(/([^0-9]{2,})駅.*?(\d+)分/);
						const station = stationMatch ? stationMatch[1] + '駅' : '';
						const walkMinutes = stationMatch ? stationMatch[2] + '分' : '';

						properties.push({
							title: 'Property ' + (i + 1),
							price: price,
							area: area,
							layout: layout,
							floor: floor,
							link: link,
							image: img,
							station: station,
							walkMinutes: walkMinutes
						});
					}

					return JSON.stringify(properties);
				})();
			`
			return chromedp.Evaluate(script, &result).Do(ctx)
		}),
	}

	if err := chromedp.Run(taskCtx, tasks...); err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("=== Extracted Property Data ===\n")

	var properties []PropertyData
	if err := json.Unmarshal([]byte(result), &properties); err != nil {
		log.Printf("JSON parse error: %v\n", err)
		log.Printf("Raw result: %s\n", result)
		return
	}

	log.Printf("Found %d properties\n\n", len(properties))

	for i, p := range properties {
		log.Printf("=== Property %d ===\n", i+1)
		log.Printf("Price: %s\n", p.Price)
		log.Printf("Area: %s㎡\n", p.Area)
		log.Printf("Layout: %s\n", p.Layout)
		log.Printf("Floor: %s階\n", p.Floor)
		log.Printf("Station: %s (%s)\n", p.Station, p.WalkMinutes)
		log.Printf("Link: %s\n", p.Link)
		log.Printf("Image: %s\n\n", p.Image)
	}
}
