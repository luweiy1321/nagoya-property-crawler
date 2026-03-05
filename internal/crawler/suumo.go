package crawler

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"nagoya-property-crawler/internal/models"
)

// SUUMOCrawler implements crawling for SUUMO website
type SUUMOCrawler struct {
	baseURL    string
	rentURL    string
	saleURL    string
	headless   bool
	userAgent  string
	timeout    time.Duration
	waitAfter  time.Duration
}

// NewSUUMOCrawler creates a new SUUMO crawler
func NewSUUMOCrawler(headless bool, userAgent string, timeout, waitAfter time.Duration) *SUUMOCrawler {
	return &SUUMOCrawler{
		baseURL:   "https://suumo.jp",
		rentURL:   "https://suumo.jp/chintai/aichi/nc_nagoya/",
		saleURL:   "https://suumo.jp/ms/chuko/aichi/nc_nagoya/",
		headless:  headless,
		userAgent: userAgent,
		timeout:   timeout,
		waitAfter: waitAfter,
	}
}

// SUUMOProperty represents a property listing from SUUMO
type SUUMOProperty struct {
	PropertyID   string
	Title        string
	Price        int
	PriceDisplay string
	Address      string
	Area         float64
	Layout       string
	Floor        string
	BuildingType string
	StationName  string
	WalkMinutes  int
	ImageURL     string
	DetailURL    string
}

// ScrapeRentListings scrapes rental listings from SUUMO
func (s *SUUMOCrawler) ScrapeRentListings(ctx context.Context, page int) ([]*models.Property, error) {
	url := fmt.Sprintf("%s?page=%d", s.rentURL, page)
	return s.scrapeListings(ctx, url, models.ListingTypeRent)
}

// ScrapeSaleListings scrapes sale listings from SUUMO
func (s *SUUMOCrawler) ScrapeSaleListings(ctx context.Context, page int) ([]*models.Property, error) {
	url := fmt.Sprintf("%s?page=%d", s.saleURL, page)
	return s.scrapeListings(ctx, url, models.ListingTypeSale)
}

// scrapeListings is the main scraping function
func (s *SUUMOCrawler) scrapeListings(ctx context.Context, url string, listingType models.ListingType) ([]*models.Property, error) {
	// Configure chromedp options with compatibility fixes
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", s.headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.UserAgent(s.userAgent),
		chromedp.Flag("blink-settings", "imagesEnabled=false"),
		// Add these to avoid Chrome DevTools Protocol issues
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("metrics-recording-only", true),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("safebrowsing-disable-auto-update", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-web-security", true),
	}

	// Create allocator context
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// Create context
	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set timeout
	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, s.timeout)
	defer cancelTimeout()

	// Scrape the page
	var rawProperties []map[string]string
	var propertyCount int
	var hasNextPage bool

	tasks := []chromedp.Action{
		chromedp.Navigate(url),
		chromedp.Sleep(s.waitAfter),
		chromedp.WaitVisible(`.cassetteitem`, chromedp.ByQuery),
	}

	// Get property count
	tasks = append(tasks, chromedp.ActionFunc(func(ctx context.Context) error {
		// Extract property data
		var html string
		if err := chromedp.OuterHTML("body", &html, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}

		// Parse the HTML to extract properties
		rawProperties = s.parseProperties(html, listingType)
		propertyCount = len(rawProperties)

		// Check if there's a next page
		hasNextPage = s.checkNextPage(html)

		return nil
	}))

	if err := chromedp.Run(taskCtx, tasks...); err != nil {
		return nil, fmt.Errorf("failed to scrape SUUMO: %w", err)
	}

	log.Printf("Scraped %d properties from SUUMO, has next: %v", propertyCount, hasNextPage)

	// Convert to models.Property
	properties := make([]*models.Property, 0, len(rawProperties))
	for _, raw := range rawProperties {
		property := s.convertToProperty(raw, listingType)
		properties = append(properties, property)
	}

	return properties, nil
}

// parseProperties parses the HTML and extracts property data
func (s *SUUMOCrawler) parseProperties(html string, listingType models.ListingType) []map[string]string {
	// This is a simplified parser - in production, you'd use a proper HTML parser
	properties := make([]map[string]string, 0)

	// Find all cassette items (property cards)
	cassettePattern := regexp.MustCompile(`<div class="cassetteitem"(.*?)</div>\s*<!-- / cassetteitem -->`)
	matches := cassettePattern.FindAllStringSubmatch(html, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		cassetteHTML := match[1]
		property := s.parseCassetteItem(cassetteHTML, listingType)
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

	return properties
}

// parseCassetteItem parses a single cassette item
func (s *SUUMOCrawler) parseCassetteItem(html string, listingType models.ListingType) *SUUMOProperty {
	property := &SUUMOProperty{}

	// Extract property ID from href
	hrefPattern := regexp.MustCompile(`href="(/chintai/[^\s"]+|/ms/chuko/[^\s"]+)"`)
	if match := hrefPattern.FindStringSubmatch(html); len(match) > 1 {
		property.DetailURL = "https://suumo.jp" + match[1]
		// Extract property ID from URL
		idPattern := regexp.MustCompile(`\/([a-z0-9]+)_`)
		if idMatch := idPattern.FindStringSubmatch(match[1]); len(idMatch) > 1 {
			property.PropertyID = idMatch[1]
		}
	}

	// Extract title
	titlePattern := regexp.MustCompile(`<div class="cassetteitem_content-title"[^>]*>([^<]+)</div>`)
	if match := titlePattern.FindStringSubmatch(html); len(match) > 1 {
		property.Title = strings.TrimSpace(match[1])
	}

	// Extract price
	pricePattern := regexp.MustCompile(`<span class="cassetteitem_price casseteitem_price--[^\s]+"><span class="cassetteitem_price--[^"]+">([^<]+)</span>`)
	if match := pricePattern.FindStringSubmatch(html); len(match) > 1 {
		priceStr := strings.TrimSpace(match[1])
		property.PriceDisplay = priceStr
		property.Price = s.parsePrice(priceStr, listingType)
	}

	// Extract address
	addressPattern := regexp.MustCompile(`<li class="cassetteitem_detail-col3"><div class="cassetteitem_detail-text"[^>]*>([^<]+)</div></li>`)
	if match := addressPattern.FindStringSubmatch(html); len(match) > 1 {
		property.Address = strings.TrimSpace(match[1])
	}

	// Extract area and layout
	areaLayoutPattern := regexp.MustCompile(`(?:<li class="cassetteitem_detail-col2"[^>]*>.*?<div class="cassetteitem_detail-text"[^>]*>)([^<]+)(?:</div>.*?<li class="cassetteitem_detail-col2"[^>]*>.*?<div class="cassetteitem_detail-text"[^>]*>)([^<]+)`)
	if match := areaLayoutPattern.FindStringSubmatch(html); len(match) > 2 {
		areaStr := strings.TrimSpace(match[1])
		property.Layout = strings.TrimSpace(match[2])
		property.Area = s.parseArea(areaStr)
	}

	// Extract floor
	floorPattern := regexp.MustCompile(`<li class="cassetteitem_detail-col2"[^>]*>.*?<div class="cassetteitem_detail-text"[^>]*>([^<]+)</div>.*?階</li>`)
	if match := floorPattern.FindStringSubmatch(html); len(match) > 1 {
		property.Floor = strings.TrimSpace(match[1])
	}

	// Extract building type
	buildingTypePattern := regexp.MustCompile(`<li class="cassetteitem_detail-col1"[^>]*>.*?<div class="cassetteitem_detail-text"[^>]*>([^<]+)</div></li>`)
	if match := buildingTypePattern.FindStringSubmatch(html); len(match) > 1 {
		property.BuildingType = strings.TrimSpace(match[1])
	}

	// Extract station and walking time
	stationPattern := regexp.MustCompile(`<div class="cassetteitem_station-text"[^>]*>([^<]+)</div>\s*<div class="cassetteitem_station-text"[^>]*>徒歩([^<]+)</div>`)
	if match := stationPattern.FindStringSubmatch(html); len(match) > 2 {
		property.StationName = strings.TrimSpace(match[1])
		walkStr := strings.TrimSpace(match[2])
		walkStr = strings.ReplaceAll(walkStr, "分", "")
		if min, err := strconv.Atoi(walkStr); err == nil {
			property.WalkMinutes = min
		}
	}

	// Extract image URL
	imagePattern := regexp.MustCompile(`<img[^>]+class="cassetteitem_object-img"[^>]+src="([^"]+)"`)
	if match := imagePattern.FindStringSubmatch(html); len(match) > 1 {
		property.ImageURL = match[1]
	}

	return property
}

// checkNextPage checks if there's a next page
func (s *SUUMOCrawler) checkNextPage(html string) bool {
	nextPagePattern := regexp.MustCompile(`<li class="pagination-item-next"><a`)
	return nextPagePattern.MatchString(html)
}

// parsePrice converts a price string to integer yen value
func (s *SUUMOCrawler) parsePrice(priceStr string, listingType models.ListingType) int {
	priceStr = strings.TrimSpace(priceStr)
	priceStr = strings.ReplaceAll(priceStr, ",", "")
	priceStr = strings.ReplaceAll(priceStr, "円", "")

	// Handle rental prices (e.g., "8.5万円" = 85000 yen)
	if strings.Contains(priceStr, "万") {
		priceStr = strings.ReplaceAll(priceStr, "万", "")
		if val, err := strconv.ParseFloat(priceStr, 64); err == nil {
			return int(val * 10000)
		}
	}

	// Handle sale prices (e.g., "2980万円" = 29800000 yen)
	if val, err := strconv.Atoi(priceStr); err == nil {
		return val
	}

	return 0
}

// parseArea converts an area string to float64
func (s *SUUMOCrawler) parseArea(areaStr string) float64 {
	areaStr = strings.TrimSpace(areaStr)
	areaStr = strings.ReplaceAll(areaStr, "㎡", "")
	areaStr = strings.ReplaceAll(areaStr, "m²", "")

	if val, err := strconv.ParseFloat(areaStr, 64); err == nil {
		return val
	}

	return 0
}

// convertToProperty converts SUUMOProperty to models.Property
func (s *SUUMOCrawler) convertToProperty(raw map[string]string, listingType models.ListingType) *models.Property {
	property := models.NewProperty(models.SourceSUUMO, raw["property_id"], listingType)

	property.Title = raw["title"]
	property.PriceDisplay = raw["price_display"]
	if price, err := strconv.Atoi(raw["price"]); err == nil {
		property.Price = price
	}
	property.Address = raw["address"]
	if area, err := strconv.ParseFloat(raw["area"], 64); err == nil {
		property.Area = area
	}
	property.Layout = raw["layout"]
	property.Floor = raw["floor"]
	property.BuildingType = raw["building_type"]
	property.StationName = raw["station_name"]
	if walk, err := strconv.Atoi(raw["walk_minutes"]); err == nil {
		property.WalkingMinutes = walk
	}
	property.DetailURL = raw["detail_url"]

	if raw["image_url"] != "" {
		property.ImageURLs = models.StringArray{raw["image_url"]}
	}

	return property
}

// ScrapeDetailPage scrapes additional details from a property detail page
func (s *SUUMOCrawler) ScrapeDetailPage(ctx context.Context, detailURL string) (*models.Property, error) {
	// Configure chromedp options
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", s.headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.UserAgent(s.userAgent),
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, s.timeout)
	defer cancelTimeout()

	var html string
	tasks := []chromedp.Action{
		chromedp.Navigate(detailURL),
		chromedp.Sleep(s.waitAfter),
		chromedp.OuterHTML("body", &html, chromedp.ByQuery),
	}

	if err := chromedp.Run(taskCtx, tasks...); err != nil {
		return nil, fmt.Errorf("failed to scrape detail page: %w", err)
	}

	// Parse additional details from detail page
	property := s.parseDetailPage(html)

	return property, nil
}

// parseDetailPage parses the detail page HTML
func (s *SUUMOCrawler) parseDetailPage(html string) *models.Property {
	property := &models.Property{}

	// Extract contact information
	contactPattern := regexp.MustCompile(`<div class="section_hospital-item-title"[^>]*>([^<]+)</div>`)
	if match := contactPattern.FindStringSubmatch(html); len(match) > 1 {
		property.ContactName = strings.TrimSpace(match[1])
	}

	// Extract phone number
	phonePattern := regexp.MustCompile(`0\d{1,4}-\d{1,4}-\d{4}`)
	if match := phonePattern.FindString(html); match != "" {
		property.ContactPhone = match
	}

	// Extract all image URLs
	imagePattern := regexp.MustCompile(`<img[^>]+class="gallery_img"[^>]+src="([^"]+)"`)
	matches := imagePattern.FindAllStringSubmatch(html, -1)
	var images models.StringArray
	for _, match := range matches {
		if len(match) > 1 {
			images = append(images, match[1])
		}
	}
	if len(images) > 0 {
		property.ImageURLs = images
	}

	return property
}
