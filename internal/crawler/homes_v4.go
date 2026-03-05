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

// HOMESCrawlerV4 implements list + detail page scraping for HOMES
type HOMESCrawlerV4 struct {
	baseURL   string
	rentURL   string
	saleURL   string
	headless  bool
	userAgent string
	timeout   time.Duration
}

// NewHOMESCrawlerV4 creates a crawler that fetches both list and detail pages
func NewHOMESCrawlerV4(headless bool, userAgent string, timeout time.Duration) *HOMESCrawlerV4 {
	return &HOMESCrawlerV4{
		baseURL:   "https://www.homes.co.jp",
		rentURL:   "https://www.homes.co.jp/chintai/aichi/nagoya/list/",
		saleURL:   "https://www.homes.co.jp/chuko/aichi/nagoya/list/",
		headless:  headless,
		userAgent: userAgent,
		timeout:   timeout,
	}
}

// ScrapeRentListings scrapes rental listings with full details
func (h *HOMESCrawlerV4) ScrapeRentListings(ctx context.Context, page int) ([]*models.Property, error) {
	url := fmt.Sprintf("%s?page=%d", h.rentURL, page)
	return h.scrapeListings(ctx, url, models.ListingTypeRent)
}

// ScrapeSaleListings scrapes sale listings with full details
func (h *HOMESCrawlerV4) ScrapeSaleListings(ctx context.Context, page int) ([]*models.Property, error) {
	url := fmt.Sprintf("%s?page=%d", h.saleURL, page)
	return h.scrapeListings(ctx, url, models.ListingTypeSale)
}

// scrapeListings scrapes list and detail pages
func (h *HOMESCrawlerV4) scrapeListings(ctx context.Context, url string, listingType models.ListingType) ([]*models.Property, error) {
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

	var listJSON string
	var pageTitle string
	var listHTML string // Debug: capture list page HTML

	// Step 1: Scrape list page (optimized with reduced wait times)
	listTasks := []chromedp.Action{
		chromedp.Navigate(url),

		// Wait for body
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.WaitReady("body", chromedp.ByQuery).Do(ctx)
		}),

		chromedp.Sleep(3 * time.Second),

		chromedp.ActionFunc(func(ctx context.Context) error {
			script := `
				Object.defineProperty(navigator, 'webdriver', {get: () => undefined});
				Object.defineProperty(navigator, 'plugins', {get: () => [1, 2, 3, 4, 5]});
				Object.defineProperty(navigator, 'languages', {get: () => ['ja-JP', 'ja', 'en']});
				window.chrome = {runtime: {}};
			`
			return chromedp.Evaluate(script, nil).Do(ctx)
		}),

		chromedp.Sleep(3 * time.Second),

		chromedp.ActionFunc(func(ctx context.Context) error {
			if err := chromedp.Title(&pageTitle).Do(ctx); err != nil {
				return err
			}

			if strings.Contains(strings.ToLower(pageTitle), "verif") ||
				strings.Contains(strings.ToLower(pageTitle), "human") ||
				strings.Contains(pageTitle, "チェック") {
				log.Printf("Verification page detected, waiting...")
				for i := 0; i < 15; i++ {
					time.Sleep(2 * time.Second)
					var newTitle string
					chromedp.Title(&newTitle).Do(ctx)
					if !strings.Contains(strings.ToLower(newTitle), "verif") &&
						!strings.Contains(strings.ToLower(newTitle), "human") &&
						!strings.Contains(newTitle, "チェック") {
						pageTitle = newTitle
						break
					}
				}
			}
			return nil
		}),

		chromedp.Sleep(3 * time.Second),

		// Debug: Capture HTML to see what we're getting
		chromedp.ActionFunc(func(ctx context.Context) error {
			script := `document.body.innerHTML`
			return chromedp.Evaluate(script, &listHTML).Do(ctx)
		}),

		chromedp.ActionFunc(func(ctx context.Context) error {
			script := `
				(function() {
					const props = [];

					// Try each selector separately and use the first that returns results
					const selectors = [
						'.unitListBody',
						'[class*="unit"]',
						'.cassetteitem',
						'.mod-summmary',
						'.property-unit',
						'a[href*="/room/"]'
					];

					let cards = null;
					for (const sel of selectors) {
						const result = document.querySelectorAll(sel);
						if (result && result.length > 0) {
							cards = result;
							console.log('Found ' + result.length + ' elements with selector: ' + sel);
							break;
						}
					}

					// If still no cards, try finding all /room/ links directly
					if (!cards || cards.length === 0) {
						const allLinks = document.querySelectorAll('a');
						cards = [];
						for (const link of allLinks) {
							if (link.getAttribute('href') && link.getAttribute('href').includes('/room/')) {
								cards.push(link);
							}
						}
						console.log('Found ' + cards.length + ' /room/ links');
					}

					for (let i = 0; i < cards.length; i++) {
						const card = cards[i];
						let link = '';

						// Extract link from different possible structures
						if (card.tagName === 'A') {
							link = card.getAttribute('href');
						} else {
							const linkEl = card.querySelector('a');
							link = linkEl ? linkEl.getAttribute('href') : '';
						}

						if (link && link.includes('/room/')) {
							// Skip duplicates
							if (props.some(p => p.detail_url === link)) {
								continue;
							}

							if (!link.startsWith('http')) {
								link = 'https://www.homes.co.jp' + link;
							}

							// Extract basic info
							const text = card.textContent || '';
							const priceMatch = text.match(/(\d{1,3}(?:,\d{3})*\s*万円)/);
							const priceDisplay = priceMatch ? priceMatch[1] : '';

							const floorMatch = text.match(/(\d+)階/);
							const floor = floorMatch ? floorMatch[1] : '';

							const layoutMatch = text.match(/[1-4][LDK]+/);
							const layout = layoutMatch ? layoutMatch[0] : '';

							let propId = link;
							const idMatch = link.match(/\/room\/([a-f0-9]+)\//);
							if (idMatch) {
								propId = idMatch[1];
							}

							props.push({
								property_id: propId,
								detail_url: link,
								price_display: priceDisplay,
								floor: floor,
								layout: layout
							});
						}
					}

					return JSON.stringify(props);
				})();
			`
			return chromedp.Evaluate(script, &listJSON).Do(ctx)
		}),
	}

	if err := chromedp.Run(taskCtx, listTasks...); err != nil {
		return nil, fmt.Errorf("failed to scrape list: %w", err)
	}

	log.Printf("HOMES page title: %s", pageTitle)
	log.Printf("HOMES list HTML length: %d bytes", len(listHTML))

	// Debug: Check for verification indicators
	if strings.Contains(strings.ToLower(listHTML), "verif") ||
	   strings.Contains(strings.ToLower(listHTML), "human") ||
	   strings.Contains(listHTML, "チェック") {
		log.Printf("WARNING: Verification page detected in HTML!")
	}

	// Debug: Show first 500 chars of HTML
	if len(listHTML) > 0 {
		preview := listHTML
		if len(preview) > 500 {
			preview = preview[:500]
		}
		log.Printf("HTML preview: %s", preview)
	}

	// Parse list data
	var rawList []map[string]string
	if err := json.Unmarshal([]byte(listJSON), &rawList); err != nil {
		return nil, fmt.Errorf("failed to parse list JSON: %w", err)
	}

	log.Printf("Found %d properties in list", len(rawList))

	if len(rawList) == 0 {
		return []*models.Property{}, nil
	}

	// Step 2: Scrape detail pages for each property
	properties := make([]*models.Property, 0, len(rawList))

	// Track consecutive failures to stop early if blocked
	consecutiveFailures := 0
	maxConsecutiveFailures := 5

	for i, raw := range rawList {
		log.Printf("Scraping detail %d/%d: %s", i+1, len(rawList), raw["property_id"])

		// Scrape detail page with retries
		var detail *models.Property
		var err error

		// Try up to 2 times
		for attempt := 0; attempt < 2; attempt++ {
			detail, err = h.scrapeDetailPage(taskCtx, raw["detail_url"], listingType, raw)
			if err == nil {
				consecutiveFailures = 0 // Reset on success
				break
			}
			if attempt < 1 {
				log.Printf("Retry %d for %s: %v", attempt+1, raw["property_id"], err)
				time.Sleep(5 * time.Second) // Longer delay on retry
			}
		}

		if err != nil {
			consecutiveFailures++
			log.Printf("Error scraping detail %s after retries: %v", raw["detail_url"], err)

			// Stop if too many consecutive failures (likely blocked)
			if consecutiveFailures >= maxConsecutiveFailures {
				log.Printf("Too many consecutive failures (%d), stopping early. Got %d properties.",
					consecutiveFailures, len(properties))
				break
			}

			// Use basic info if detail fails
			detail = h.convertToBasicProperty(raw, listingType)
		}

		properties = append(properties, detail)
		log.Printf("Completed %d/%d: title=%s, address=%s, area=%.2f",
			i+1, len(rawList), detail.Title, detail.Address, detail.Area)

		// Add delay between requests to avoid triggering WAF (30 seconds)
		if i < len(rawList)-1 {
			log.Printf("Waiting 30 seconds before next request...")
			time.Sleep(30 * time.Second)
		}
	}

	log.Printf("Scraped %d properties with details", len(properties))
	return properties, nil
}

// scrapeDetailPage scrapes a single property detail page with new context (matching test script approach)
func (h *HOMESCrawlerV4) scrapeDetailPage(parentCtx context.Context, detailURL string, listingType models.ListingType, basicData map[string]string) (*models.Property, error) {
	// Create new allocator context for each detail page to avoid timeout issues
	// Use same approach as test script that got 409KB successfully
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", h.headless),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("exclude-switches", "enable-automation"),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("user-data-dir", "/tmp/chrome-test-profile"),
		chromedp.UserAgent(h.userAgent),
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Increased timeout to 300 seconds (5 minutes) for each detail page
	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, 300*time.Second)
	defer cancelTimeout()

	var detailHTML string
	var pageTitle string

	// Use same wait times as test script (8s + 5s) that worked
	tasks := []chromedp.Action{
		chromedp.Navigate(detailURL),
		chromedp.Sleep(8 * time.Second),

		// Anti-detection (same as test script)
		chromedp.ActionFunc(func(ctx context.Context) error {
			script := `Object.defineProperty(navigator, 'webdriver', {get: () => undefined});`
			return chromedp.Evaluate(script, nil).Do(ctx)
		}),

		chromedp.Sleep(5 * time.Second),

		// Get title and check for verification
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.Title(&pageTitle).Do(ctx)
		}),

		// If verification detected, wait longer
		chromedp.ActionFunc(func(ctx context.Context) error {
			if contains(pageTitle, "Verif") || contains(pageTitle, "Human") || contains(pageTitle, "チェック") {
				log.Printf("Verification page on detail URL: %s", detailURL)
				chromedp.Sleep(10 * time.Second).Do(ctx)
			}
			return nil
		}),

		// Extract HTML (same as test script)
		chromedp.OuterHTML("body", &detailHTML, chromedp.ByQuery),
	}

	if err := chromedp.Run(taskCtx, tasks...); err != nil {
		return nil, err
	}

	// Debug: Show HTML length and content
	log.Printf("Detail page HTML length: %d bytes", len(detailHTML))

	// If HTML is too short, likely an error page
	if len(detailHTML) < 5000 {
		// Show full HTML for debugging (it's small anyway)
		log.Printf("HTML content (first 1000 chars): %s", detailHTML[:min(1000, len(detailHTML))])
	}
	if len(detailHTML) < 1000 {
		log.Printf("WARNING: Insufficient HTML data, first 500 chars: %s", detailHTML)
		return nil, fmt.Errorf("received insufficient HTML data (%d bytes)", len(detailHTML))
	}

	// Parse detail page HTML
	property := h.parseDetailHTML(detailHTML, detailURL, listingType, basicData)

	// Override with basic data if needed
	if property.PropertyID == "" {
		property.PropertyID = basicData["property_id"]
	}
	if property.Floor == "" {
		property.Floor = basicData["floor"]
	}

	return property, nil
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

// Helper function to find index of substring
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Helper function to get minimum
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// parseDetailHTML extracts property data from detail page HTML
func (h *HOMESCrawlerV4) parseDetailHTML(html, detailURL string, listingType models.ListingType, basicData map[string]string) *models.Property {
	property := &models.Property{
		Source:        models.SourceHOMES,
		PropertyID:    h.truncateString(basicData["property_id"], 100),
		ListingType:   listingType,
		DetailURL:     h.truncateString(detailURL, 500),
		PriceDisplay:  h.truncateString(basicData["price_display"], 100),
		Floor:         h.truncateString(basicData["floor"], 50),
		Layout:        h.truncateString(basicData["layout"], 50),
	}

	// Try JSON-LD structured data first (most reliable)
	if jsonLDProp := h.parseJSONLD(html); jsonLDProp != nil {
		// Merge with existing property
		if jsonLDProp.Title != "" && property.Title == "" {
			property.Title = jsonLDProp.Title
		}
		if jsonLDProp.Address != "" && property.Address == "" {
			property.Address = jsonLDProp.Address
		}
		if jsonLDProp.Area > 0 && property.Area == 0 {
			property.Area = jsonLDProp.Area
		}
		if jsonLDProp.Floor != "" && property.Floor == "" {
			property.Floor = jsonLDProp.Floor
		}
		if jsonLDProp.Layout != "" && property.Layout == "" {
			property.Layout = jsonLDProp.Layout
		}
		if jsonLDProp.BuildingType != "" && property.BuildingType == "" {
			property.BuildingType = jsonLDProp.BuildingType
		}
		if jsonLDProp.ConstructionYear > 0 && property.ConstructionYear == 0 {
			property.ConstructionYear = jsonLDProp.ConstructionYear
		}
		if jsonLDProp.StationName != "" && property.StationName == "" {
			property.StationName = jsonLDProp.StationName
		}
		if jsonLDProp.WalkingMinutes > 0 && property.WalkingMinutes == 0 {
			property.WalkingMinutes = jsonLDProp.WalkingMinutes
		}
	}

	// Extract title
	titlePatterns := []string{
		`<h1[^>]*class="[^"]*"[^>]*>([^<]+)</h1>`,
		`<h2[^>]*class="[^"]*propertyTitle[^"]*"[^>]*>([^<]+)</h2>`,
		`<title>([^<]+)の物件情報</title>`,
	}
	for _, pattern := range titlePatterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(html); len(match) > 1 {
			property.Title = h.truncateString(strings.TrimSpace(match[1]), 200)
			break
		}
	}

	// Extract address - use the actual HOMES structure (dt/dd tags)
	addrPatterns := []string{
		// HOMES uses definition lists: <dt>所在地</dt><dd><p>ADDRESS</p></dd>
		`<dt[^>]*>所在地</dt>\s*<dd[^>]*>.*?<p[^>]*>([^<]+)</p>`,
		`<dt[^>]*>所在地</dt>\s*<dd[^>]*>([^<]+)</dd>`,
		// Also try other patterns
		`<td[^>]*class="[^"]*address[^"]*"[^>]*>([^<]+)</td>`,
		`<td[^>]*class="[^"]*所在地[^"]*"[^>]*>\s*(?:<span[^>]*>)?\s*([^<<]+)(?:</span>)?\s*</td>`,
		`所在地.*?<td[^>]*>\s*(?:<span[^>]*>)?\s*([^<]+)(?:</span>)?\s*</td>`,
		`住所.*?<td[^>]*>\s*(?:<span[^>]*>)?\s*([^<]+)(?:</span>)?\s*</td>`,
		`<th[^>]*>所在地</th>\s*<td[^>]*>([^<]+)</td>`,
		`<th[^>]*>所在地</th>\s*<td[^>]*><span[^>]*>([^<]+)</span></td>`,
		`<li[^>]*class="[^"]*所在地[^"]*"[^>]*>.*?<span[^>]*>([^<]+)</span>`,
		`<div[^>]*class="[^"]*address[^"]*"[^>]*>([^<]+)</div>`,
		`所在地[:\s]*([^\n<]{10,100})`,
		// Data attribute patterns
		`data-address="([^"]+)"`,
		`data-location="([^"]+)"`,
		// Try to extract from HTML input values
		`prefecture[^>]*>.*?value="([^"]+)"`,
		`city[^>]*>.*?value="([^"]+)"`,
		// JSON-LD patterns (last resort, but be specific about schema.org)
		`"@type":\s*"PostalAddress"[\s\S]*?"streetAddress":\s*"([^"]+)"`,
		`"streetAddress"\s*:\s*"([^"]{20,})"`,
	}

	for _, pattern := range addrPatterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(html); len(match) > 1 {
			addr := strings.TrimSpace(match[1])
			// Clean up common unwanted text
			addr = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(addr, "")
			property.Address = h.truncateString(addr, 200)
			log.Printf("Found address using pattern: %s", regexp.MustCompile(`^.{30}`).ReplaceAllString(pattern, "..."))
			break
		}
	}

	// Debug: If address still not found, search for any context around "所在地"
	if property.Address == "" && strings.Contains(html, "所在地") {
		idx := strings.Index(html, "所在地")
		if idx >= 0 {
			context := html[idx:min(idx+200, len(html))]
			log.Printf("Address context (first 200 chars after '所在地'): %s", context)
		}
	}

	// Extract area
	areaPattern := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*㎡`)
	if match := areaPattern.FindStringSubmatch(html); len(match) > 1 {
		property.Area, _ = strconv.ParseFloat(match[1], 64)
	}

	// Extract building type and construction year
	buildPattern := regexp.MustCompile(`建物種類.*?<td[^>]*>([^<]+)</td>`)
	if match := buildPattern.FindStringSubmatch(html); len(match) > 1 {
		property.BuildingType = h.truncateString(strings.TrimSpace(match[1]), 100)
	}

	yearPattern := regexp.MustCompile(`築.*?(\d{4})年`)
	if match := yearPattern.FindStringSubmatch(html); len(match) > 1 {
		property.ConstructionYear, _ = strconv.Atoi(match[1])
	}

	// Extract station info
	stationPattern := regexp.MustCompile(`([^0-9]{2,})駅.*?(\d+)分`)
	if match := stationPattern.FindStringSubmatch(html); len(match) > 2 {
		property.StationName = h.truncateString(strings.TrimSpace(match[1])+"駅", 100)
		property.WalkingMinutes, _ = strconv.Atoi(match[2])
	}

	// Extract price
	property.Price = h.parsePrice(property.PriceDisplay, listingType)

	return property
}

// convertToBasicProperty creates a property from basic list data
func (h *HOMESCrawlerV4) convertToBasicProperty(raw map[string]string, listingType models.ListingType) *models.Property {
	price := h.parsePrice(raw["price_display"], listingType)

	return &models.Property{
		Source:        models.SourceHOMES,
		PropertyID:    h.truncateString(raw["property_id"], 100),
		ListingType:   listingType,
		Title:         "詳細取得中...",
		Price:         price,
		PriceDisplay:  h.truncateString(raw["price_display"], 100),
		Floor:         h.truncateString(raw["floor"], 50),
		Layout:        h.truncateString(raw["layout"], 50),
		DetailURL:     h.truncateString(raw["detail_url"], 500),
	}
}

// parseJSONLD extracts property data from JSON-LD structured data
func (h *HOMESCrawlerV4) parseJSONLD(html string) *models.Property {
	// Find JSON-LD script tags
	jsonLDPattern := regexp.MustCompile(`<script type="application/ld\+json">(.*?)</script>`)
	matches := jsonLDPattern.FindAllStringSubmatch(html, -1)

	for _, match := range matches {
		if len(match) > 1 {
			var jsonData map[string]interface{}
			if err := json.Unmarshal([]byte(match[1]), &jsonData); err == nil {
				return h.extractFromJSONLD(jsonData)
			}
		}
	}

	// Try without type attribute
	jsonPattern := regexp.MustCompile(`<script[^>]*>[\s\S]*?application/ld\+json[\s\S]*?({.*?})[\s\S]*?</script>`)
	matches = jsonPattern.FindAllStringSubmatch(html, -1)

	for _, match := range matches {
		if len(match) > 1 {
			var jsonData map[string]interface{}
			if err := json.Unmarshal([]byte(match[1]), &jsonData); err == nil {
				return h.extractFromJSONLD(jsonData)
			}
		}
	}

	return nil
}

// extractFromJSONLD extracts property data from parsed JSON-LD
func (h *HOMESCrawlerV4) extractFromJSONLD(data map[string]interface{}) *models.Property {

	// Handle @graph format (common in JSON-LD)
	if graph, ok := data["@graph"].([]interface{}); ok {
		for _, item := range graph {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if itemType, ok := itemMap["@type"].(string); ok && itemType == "Product" {
					return h.extractFromProduct(itemMap)
				}
				if itemType, ok := itemMap["@type"].(string); ok && itemType == "Offer" {
					if product, ok := itemMap["itemOffered"].(map[string]interface{}); ok {
						return h.extractFromProduct(product)
					}
				}
			}
		}
	}

	// Direct Product type
	if itemType, ok := data["@type"].(string); ok && itemType == "Product" {
		return h.extractFromProduct(data)
	}

	return nil
}

// extractFromProduct extracts data from a Product JSON-LD object
func (h *HOMESCrawlerV4) extractFromProduct(product map[string]interface{}) *models.Property {
	prop := &models.Property{}

	// Extract name
	if name, ok := product["name"].(string); ok {
		prop.Title = h.truncateString(name, 200)
	}

	// Extract address
	if address, ok := product["address"].(map[string]interface{}); ok {
		if addr, ok := address["streetAddress"].(string); ok {
			prop.Address = h.truncateString(addr, 200)
		}
		if addr, ok := address["addressLocality"].(string); ok {
			if prop.Address != "" {
				prop.Address += " " + addr
			} else {
				prop.Address = h.truncateString(addr, 200)
			}
		}
		if addr, ok := address["addressRegion"].(string); ok {
			if prop.Address != "" {
				prop.Address += " " + addr
			}
		}
	}

	// Extract floor
	if floor, ok := product["floorLevel"].(string); ok {
		prop.Floor = h.truncateString(floor, 50)
	}

	// Extract area
	if areaStr, ok := product["floorSize"].(string); ok {
		re := regexp.MustCompile(`(\d+(?:\.\d+)?)`)
		if match := re.FindStringSubmatch(areaStr); len(match) > 1 {
			if val, err := strconv.ParseFloat(match[1], 64); err == nil {
				prop.Area = val
			}
		}
	}

	// Extract offers/pricing
	if offers, ok := product["offers"].([]interface{}); ok && len(offers) > 0 {
		if offer, ok := offers[0].(map[string]interface{}); ok {
			if price, ok := offer["price"].(string); ok {
				prop.PriceDisplay = h.truncateString(price, 100)
				prop.Price = h.parsePrice(price, models.ListingTypeRent)
			}
		}
	}

	return prop
}

// parsePrice converts price string to integer (yen)
func (h *HOMESCrawlerV4) parsePrice(priceStr string, listingType models.ListingType) int {
	priceStr = strings.TrimSpace(priceStr)
	priceStr = strings.ReplaceAll(priceStr, ",", "")

	re := regexp.MustCompile(`(\d+(?:\.\d+)?)`)
	match := re.FindStringSubmatch(priceStr)
	if len(match) < 2 {
		return 0
	}

	value, _ := strconv.ParseFloat(match[1], 64)

	if strings.Contains(priceStr, "万") {
		return int(value * 10000)
	} else if strings.Contains(priceStr, "億") {
		return int(value * 100000000)
	}
	return int(value)
}

// truncateString truncates a string to max length
func (h *HOMESCrawlerV4) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// Truncate to maxLen bytes (approximate for UTF-8)
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}
