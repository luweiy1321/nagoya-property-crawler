package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"nagoya-property-crawler/internal/models"
)

// HOMESCrawlerV3 implements JavaScript-based extraction for HOMES
type HOMESCrawlerV3 struct {
	baseURL   string
	rentURL   string
	saleURL   string
	headless  bool
	userAgent string
	timeout   time.Duration
}

// NewHOMESCrawlerV3 creates a new JS-based HOMES crawler
func NewHOMESCrawlerV3(headless bool, userAgent string, timeout time.Duration) *HOMESCrawlerV3 {
	return &HOMESCrawlerV3{
		baseURL:   "https://www.homes.co.jp",
		rentURL:   "https://www.homes.co.jp/chintai/aichi/nagoya/list/",
		saleURL:   "https://www.homes.co.jp/chuko/aichi/nagoya/list/",
		headless:  headless,
		userAgent: userAgent,
		timeout:   timeout,
	}
}

// ScrapeRentListings scrapes rental listings from HOMES
func (h *HOMESCrawlerV3) ScrapeRentListings(ctx context.Context, page int) ([]*models.Property, error) {
	url := fmt.Sprintf("%s?page=%d", h.rentURL, page)
	return h.scrapeListings(ctx, url, models.ListingTypeRent)
}

// ScrapeSaleListings scrapes sale listings from HOMES
func (h *HOMESCrawlerV3) ScrapeSaleListings(ctx context.Context, page int) ([]*models.Property, error) {
	url := fmt.Sprintf("%s?page=%d", h.saleURL, page)
	return h.scrapeListings(ctx, url, models.ListingTypeSale)
}

// scrapeListings is the main scraping function with JavaScript extraction
func (h *HOMESCrawlerV3) scrapeListings(ctx context.Context, url string, listingType models.ListingType) ([]*models.Property, error) {
	// Configure chromedp with anti-detection
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", h.headless),
		chromedp.Flag("disable-gpu", false),
		chromedp.Flag("start-maximized", true),
		chromedp.Flag("window-size", "1920,1080"),
		chromedp.UserAgent(h.userAgent),
		chromedp.Flag("lang", "ja-JP"),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("exclude-switches", "enable-automation"),
		chromedp.Flag("disable-infobars", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("user-data-dir", "/tmp/chrome-homes-profile"),
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, h.timeout)
	defer cancelTimeout()

	var propertiesJSON string
	var pageTitle string

	tasks := []chromedp.Action{
		chromedp.Navigate(url),
		chromedp.Sleep(3 * time.Second),

		// Inject anti-detection script
		chromedp.ActionFunc(func(ctx context.Context) error {
			script := `
				Object.defineProperty(navigator, 'webdriver', {get: () => undefined});
				Object.defineProperty(navigator, 'plugins', {get: () => [1, 2, 3, 4, 5]});
				Object.defineProperty(navigator, 'languages', {get: () => ['ja-JP', 'ja', 'en']});
				window.chrome = {runtime: {}};
			`
			return chromedp.Evaluate(script, nil).Do(ctx)
		}),

		chromedp.Sleep(2 * time.Second),

		// Check for verification page
		chromedp.ActionFunc(func(ctx context.Context) error {
			if err := chromedp.Title(&pageTitle).Do(ctx); err != nil {
				return err
			}

			// If verification page, wait longer
			if strings.Contains(strings.ToLower(pageTitle), "verif") ||
			   strings.Contains(strings.ToLower(pageTitle), "human") {
				log.Printf("Verification page detected, waiting...")
				for i := 0; i < 10; i++ {
					time.Sleep(3 * time.Second)
					var newTitle string
					chromedp.Title(&newTitle).Do(ctx)
					if !strings.Contains(strings.ToLower(newTitle), "verif") &&
					   !strings.Contains(strings.ToLower(newTitle), "human") {
						pageTitle = newTitle
						break
					}
				}
			}
			return nil
		}),

		chromedp.Sleep(3 * time.Second),

		// Extract property data using JavaScript
		chromedp.ActionFunc(func(ctx context.Context) error {
			script := `
				(function() {
					const props = [];
					const cards = document.querySelectorAll('.unitListBody');

					for (let i = 0; i < cards.length; i++) {
						const card = cards[i];
						const text = card.textContent;

						// Extract link
						const linkEl = card.querySelector('a');
						let link = linkEl ? linkEl.getAttribute('href') : '';
						if (link && !link.startsWith('http')) {
							link = 'https://www.homes.co.jp' + link;
						}

						// Extract price (X万円)
						const priceMatch = text.match(/(\d{1,3}(?:,\d{3})*\s*万円)/);
						const priceDisplay = priceMatch ? priceMatch[1] : '';

						// Extract area (XX.XX㎡)
						const areaMatch = text.match(/(\d+(?:\.\d+)?)\s*㎡/);
						const area = areaMatch ? areaMatch[1] : '';

						// Extract layout (1LDK, 2DK, etc.)
						const layoutMatch = text.match(/[1-3][LDK]/);
						const layout = layoutMatch ? layoutMatch[0] : '';

						// Extract floor
						const floorMatch = text.match(/(\d+)階/);
						const floor = floorMatch ? floorMatch[1] : '';

						// Extract title/name - usually first meaningful text
						const lines = text.split(/[\n\t]+/).map(s => s.trim()).filter(s => s);
						let title = '';
						for (const line of lines) {
							if (line.length > 3 && !line.match(/^[0-9]+$/) &&
							    !line.match(/万円/) && !line.match(/㎡/) &&
							    !line.match(/階/) && !line.match(/分/)) {
								title = line.substring(0, 50);
								break;
							}
						}

						// Extract image
						const imgEl = card.querySelector('img[src*="homes"], img[src*="apamanshop"]');
						const img = imgEl ? imgEl.getAttribute('src') : '';

						// Generate property ID from link
						let propId = '';
						if (link) {
							const idMatch = link.match(/\/room\/([a-f0-9]+)\//);
							propId = idMatch ? idMatch[1] : link.substring(link.lastIndexOf('/') + 1);
						}

						props.push({
							property_id: propId || ('homes-' + i),
							title: title || 'Property ' + (i + 1),
							price_display: priceDisplay,
							area: area,
							layout: layout,
							floor: floor,
							detail_url: link,
							image_url: img
						});
					}

					return JSON.stringify(props);
				})();
			`
			return chromedp.Evaluate(script, &propertiesJSON).Do(ctx)
		}),
	}

	if err := chromedp.Run(taskCtx, tasks...); err != nil {
		return nil, fmt.Errorf("failed to scrape HOMES: %w", err)
	}

	log.Printf("HOMES page title: %s", pageTitle)

	// Parse JSON result
	var rawProps []map[string]string
	if err := json.Unmarshal([]byte(propertiesJSON), &rawProps); err != nil {
		return nil, fmt.Errorf("failed to parse properties JSON: %w", err)
	}

	log.Printf("Extracted %d properties from HOMES", len(rawProps))

	// Convert to models.Property
	properties := make([]*models.Property, 0, len(rawProps))
	for _, raw := range rawProps {
		property := h.convertToProperty(raw, listingType)
		properties = append(properties, property)
	}

	return properties, nil
}

// convertToProperty converts raw data to models.Property
func (h *HOMESCrawlerV3) convertToProperty(raw map[string]string, listingType models.ListingType) *models.Property {
	price := h.parsePrice(raw["price_display"], listingType)
	area, _ := strconv.ParseFloat(raw["area"], 64)

	// Only include non-empty image URLs
	var imageURLs []string
	if raw["image_url"] != "" {
		imageURLs = []string{raw["image_url"]}
	}

	return &models.Property{
		Source:         models.SourceHOMES,
		PropertyID:     raw["property_id"],
		ListingType:    listingType,
		Title:          raw["title"],
		Price:          price,
		PriceDisplay:   raw["price_display"],
		Area:           area,
		Layout:         raw["layout"],
		Floor:          raw["floor"],
		BuildingType:   "",
		StationName:    "",
		WalkingMinutes: 0,
		ImageURLs:      imageURLs,
		DetailURL:      raw["detail_url"],
	}
}

// parsePrice converts price string to integer (yen)
func (h *HOMESCrawlerV3) parsePrice(priceStr string, listingType models.ListingType) int {
	priceStr = strings.TrimSpace(priceStr)
	priceStr = strings.ReplaceAll(priceStr, ",", "")

	re := regexp.MustCompile(`(\d+(?:\.\d+)?)`)
	match := re.FindStringSubmatch(priceStr)
	if len(match) < 2 {
		return 0
	}

	value, _ := strconv.ParseFloat(match[1], 64)

	// Convert based on unit
	if strings.Contains(priceStr, "万") {
		return int(value * 10000)
	} else if strings.Contains(priceStr, "億") {
		return int(value * 100000000)
	}
	return int(value)
}
