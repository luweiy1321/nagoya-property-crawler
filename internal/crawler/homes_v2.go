package crawler

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"nagoya-property-crawler/internal/models"
)

// HOMESCrawlerV2 implements anti-detection crawling for HOMES website
type HOMESCrawlerV2 struct {
	baseURL   string
	rentURL   string
	saleURL   string
	headless  bool
	userAgent string
	timeout   time.Duration
}

// NewHOMESCrawlerV2 creates a new anti-detection HOMES crawler
func NewHOMESCrawlerV2(headless bool, userAgent string, timeout time.Duration) *HOMESCrawlerV2 {
	return &HOMESCrawlerV2{
		baseURL:   "https://www.homes.co.jp",
		rentURL:   "https://www.homes.co.jp/chintai/aichi/nagoya/list/",
		saleURL:   "https://www.homes.co.jp/chuko/aichi/nagoya/list/",
		headless:  headless,
		userAgent: userAgent,
		timeout:   timeout,
	}
}

// ScrapeRentListings scrapes rental listings from HOMES
func (h *HOMESCrawlerV2) ScrapeRentListings(ctx context.Context, page int) ([]*models.Property, error) {
	url := fmt.Sprintf("%s?page=%d", h.rentURL, page)
	return h.scrapeListings(ctx, url, models.ListingTypeRent)
}

// ScrapeSaleListings scrapes sale listings from HOMES
func (h *HOMESCrawlerV2) ScrapeSaleListings(ctx context.Context, page int) ([]*models.Property, error) {
	url := fmt.Sprintf("%s?page=%d", h.saleURL, page)
	return h.scrapeListings(ctx, url, models.ListingTypeSale)
}

// scrapeListings is the main scraping function with anti-detection
func (h *HOMESCrawlerV2) scrapeListings(ctx context.Context, url string, listingType models.ListingType) ([]*models.Property, error) {
	// Configure chromedp with anti-detection options
	opts := []chromedp.ExecAllocatorOption{
		// Basic options
		chromedp.Flag("headless", h.headless),
		chromedp.Flag("disable-gpu", false),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),

		// Window size
		chromedp.Flag("start-maximized", true),
		chromedp.Flag("window-size", "1920,1080"),

		// User agent
		chromedp.UserAgent(h.userAgent),

		// Language
		chromedp.Flag("lang", "ja-JP"),

		// Disable automation indicators
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("exclude-switches", "enable-automation"),

		// Disable infobars
		chromedp.Flag("disable-infobars", true),

		// Disable extensions
		chromedp.Flag("disable-extensions", true),

		// Ignore certificate errors
		chromedp.Flag("ignore-certificate-errors", true),

		// Allow running insecure content
		chromedp.Flag("allow-running-insecure-content", true),

		// Disable web security
		chromedp.Flag("disable-web-security", true),

		// Disable features
		chromedp.Flag("disable-features", "VizDisplayCompositor"),

		// User data dir (creates persistent session)
		chromedp.Flag("user-data-dir", "/tmp/chrome-homes-profile"),
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, h.timeout)
	defer cancelTimeout()

	var rawProperties []map[string]string
	var pageTitle string

	tasks := []chromedp.Action{
		// Navigate to the page
		chromedp.Navigate(url),

		// Wait for initial load
		chromedp.Sleep(2 * time.Second),

		// Inject JavaScript to hide automation
		chromedp.ActionFunc(func(ctx context.Context) error {
			script := `
				Object.defineProperty(navigator, 'webdriver', {
					get: () => undefined
				});
				Object.defineProperty(navigator, 'plugins', {
					get: () => [1, 2, 3, 4, 5]
				});
				Object.defineProperty(navigator, 'languages', {
					get: () => ['ja-JP', 'ja', 'en']
				});
				window.chrome = {
					runtime: {}
				};
			`
			return chromedp.Evaluate(script, nil).Do(ctx)
		}),

		// Scroll page to simulate human behavior
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Random scroll
			scrollTo := rand.Intn(500) + 200
			script := fmt.Sprintf("window.scrollTo(0, %d);", scrollTo)
			return chromedp.Evaluate(script, nil).Do(ctx)
		}),

		// Wait more for Cloudflare
		chromedp.Sleep(5 * time.Second),

		// Check if we hit Cloudflare
		chromedp.ActionFunc(func(ctx context.Context) error {
			var title string
			if err := chromedp.Title(&title).Do(ctx); err != nil {
				return err
			}
			pageTitle = title

			// If we see verification page, wait longer
			if strings.Contains(strings.ToLower(title), "verif") ||
			   strings.Contains(strings.ToLower(title), "human") ||
			   strings.Contains(strings.ToLower(title), "チェック") {
				log.Printf("Detected verification page, waiting longer...")
				// Wait for Cloudflare verification (up to 30 seconds)
				for i := 0; i < 10; i++ {
					time.Sleep(3 * time.Second)
					var newTitle string
					chromedp.Title(&newTitle).Do(ctx)
					if !strings.Contains(strings.ToLower(newTitle), "verif") &&
					   !strings.Contains(strings.ToLower(newTitle), "human") {
						log.Printf("Verification passed!")
						pageTitle = newTitle
						break
					}
				}
			}
			return nil
		}),

		// Final wait
		chromedp.Sleep(3 * time.Second),

		// Get page HTML
		chromedp.ActionFunc(func(ctx context.Context) error {
			var html string
			if err := chromedp.OuterHTML("body", &html, chromedp.ByQuery).Do(ctx); err != nil {
				return err
			}

			log.Printf("HOMES page title: %s, HTML length: %d", pageTitle, len(html))

			// Check if we're still on verification page
			if strings.Contains(strings.ToLower(pageTitle), "verif") ||
			   strings.Contains(strings.ToLower(pageTitle), "human") {
				return fmt.Errorf("still on verification page")
			}

			rawProperties = h.parseProperties(html, listingType)
			return nil
		}),
	}

	if err := chromedp.Run(taskCtx, tasks...); err != nil {
		return nil, fmt.Errorf("failed to scrape HOMES: %w", err)
	}

	log.Printf("Scraped %d properties from HOMES", len(rawProperties))

	properties := make([]*models.Property, 0, len(rawProperties))
	for _, raw := range rawProperties {
		property := h.convertToProperty(raw, listingType)
		properties = append(properties, property)
	}

	return properties, nil
}

// parseProperties parses the HTML and extracts property data
func (h *HOMESCrawlerV2) parseProperties(html string, listingType models.ListingType) []map[string]string {
	properties := make([]map[string]string, 0)

	// Try to find property cards with the new structure
	patterns := []string{
		// Try new pattern first
		`<div class="unitListBody[^"]*"[^>]*>(.*?)</div>\s*<!-- /unitListBody -->`,
		// Fallback to older patterns
		`<div class="prg-estateListItem[^"]*"[^>]*>(.*?)</div>\s*<!-- /unit -->`,
		// Generic pattern
		`<div class="[^"]*unit[^"]*body[^"]*"[^>]*>(.*?)(?:<div class="[^"]*unit[^"]*body|$)`,
	}

	for _, patternStr := range patterns {
		pattern := regexp.MustCompile(patternStr)
		matches := pattern.FindAllStringSubmatch(html, -1)

		if len(matches) > 0 {
			log.Printf("Found %d properties using pattern: %s", len(matches), patternStr)

			for _, match := range matches {
				if len(match) < 2 {
					continue
				}

				propertyHTML := match[1]
				property := h.parsePropertyItem(propertyHTML, listingType)
				if property != nil && property.PropertyID != "" {
					properties = append(properties, map[string]string{
						"property_id":    property.PropertyID,
						"title":          property.Title,
						"price":          strconv.Itoa(property.Price),
						"price_display":  property.PriceDisplay,
						"address":        property.Address,
						"area":           fmt.Sprintf("%.2f", property.Area),
						"layout":         property.Layout,
						"floor":          property.Floor,
						"building_type":  property.BuildingType,
						"station_name":   property.StationName,
						"walk_minutes":   strconv.Itoa(property.WalkMinutes),
						"image_url":      property.ImageURL,
						"detail_url":     property.DetailURL,
					})
				}
			}

			if len(properties) > 0 {
				break
			}
		}
	}

	return properties
}

// parsePropertyItem parses a single property item
func (h *HOMESCrawlerV2) parsePropertyItem(html string, listingType models.ListingType) *HOMESProperty {
	property := &HOMESProperty{}

	// Extract detail URL and property ID
	hrefPattern := regexp.MustCompile(`href="(/chintai/[^\s"]+|/chuko/[^\s"]+)"`)
	if match := hrefPattern.FindStringSubmatch(html); len(match) > 1 {
		property.DetailURL = "https://www.homes.co.jp" + match[1]
		// Extract property ID from URL
		idPattern := regexp.MustCompile(`\/([a-zA-Z0-9_-]+)\/?$`)
		if idMatch := idPattern.FindStringSubmatch(match[1]); len(idMatch) > 1 {
			property.PropertyID = idMatch[1]
		}
	}

	// Extract title - try multiple patterns
	titlePatterns := []string{
		`<h3[^>]*class="[^"]*"[^>]*>([^<]+)</h3>`,
		`<h4[^>]*class="[^"]*"[^>]*>([^<]+)</h4>`,
		`class="[^"]*title[^"]*"[^>]*>([^<]+)<`,
	}
	for _, pattern := range titlePatterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(html); len(match) > 1 {
			property.Title = strings.TrimSpace(match[1])
			break
		}
	}

	// Extract price
	pricePattern := regexp.MustCompile(`(\d{1,3}(?:,\d{3})*(?:\.\d+)?)\s*(万円|円)`)
	if match := pricePattern.FindStringSubmatch(html); len(match) > 2 {
		property.PriceDisplay = match[1] + match[2]
		property.Price = h.parsePrice(property.PriceDisplay, listingType)
	}

	// Extract area
	areaPattern := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*㎡`)
	if match := areaPattern.FindStringSubmatch(html); len(match) > 1 {
		property.Area = parseFloat(match[1])
	}

	// Extract layout
	layoutPattern := regexp.MustCompile(`([1-3][LDK][A-Z]?)`)
	if match := layoutPattern.FindStringSubmatch(html); len(match) > 1 {
		property.Layout = match[1]
	}

	// Extract image
	imgPattern := regexp.MustCompile(`<img[^>]*src="([^"]+)"[^>]*>`)
	if match := imgPattern.FindStringSubmatch(html); len(match) > 1 {
		property.ImageURL = match[1]
	}

	return property
}

// convertToProperty converts HOMESProperty to models.Property
func (h *HOMESCrawlerV2) convertToProperty(raw map[string]string, listingType models.ListingType) *models.Property {
	price, _ := strconv.Atoi(raw["price"])
	area, _ := strconv.ParseFloat(raw["area"], 64)
	walkMinutes, _ := strconv.Atoi(raw["walk_minutes"])

	return &models.Property{
		Source:         "homes",
		PropertyID:     raw["property_id"],
		ListingType:    listingType,
		Title:          raw["title"],
		Price:          price,
		PriceDisplay:   raw["price_display"],
		Address:        raw["address"],
		Area:           area,
		Layout:         raw["layout"],
		Floor:          raw["floor"],
		BuildingType:   raw["building_type"],
		StationName:    raw["station_name"],
		WalkingMinutes: walkMinutes,
		ImageURLs:      []string{raw["image_url"]},
		DetailURL:      raw["detail_url"],
	}
}

// parsePrice parses price string to integer (yen)
func (h *HOMESCrawlerV2) parsePrice(priceStr string, listingType models.ListingType) int {
	priceStr = strings.TrimSpace(priceStr)
	priceStr = strings.ReplaceAll(priceStr, ",", "")

	re := regexp.MustCompile(`(\d+(?:\.\d+)?)`)
	match := re.FindStringSubmatch(priceStr)
	if len(match) < 2 {
		return 0
	}

	value := parseFloat(match[1])

	// Convert based on unit
	if strings.Contains(priceStr, "万") {
		return int(value * 10000)
	} else if strings.Contains(priceStr, "億") {
		return int(value * 100000000)
	}
	return int(value)
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
