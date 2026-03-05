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

// HOMESCrawler implements crawling for HOMES website
type HOMESCrawler struct {
	baseURL   string
	rentURL   string
	saleURL   string
	headless  bool
	userAgent string
	timeout   time.Duration
	waitAfter time.Duration
}

// NewHOMESCrawler creates a new HOMES crawler
func NewHOMESCrawler(headless bool, userAgent string, timeout, waitAfter time.Duration) *HOMESCrawler {
	return &HOMESCrawler{
		baseURL:   "https://www.homes.co.jp",
		rentURL:   "https://www.homes.co.jp/chintai/aichi/nagoya/list/",
		saleURL:   "https://www.homes.co.jp/chuko/aichi/nagoya/list/",
		headless:  headless,
		userAgent: userAgent,
		timeout:   timeout,
		waitAfter: waitAfter,
	}
}

// HOMESProperty represents a property listing from HOMES
type HOMESProperty struct {
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

// ScrapeRentListings scrapes rental listings from HOMES
func (h *HOMESCrawler) ScrapeRentListings(ctx context.Context, page int) ([]*models.Property, error) {
	url := fmt.Sprintf("%s?page=%d", h.rentURL, page)
	return h.scrapeListings(ctx, url, models.ListingTypeRent)
}

// ScrapeSaleListings scrapes sale listings from HOMES
func (h *HOMESCrawler) ScrapeSaleListings(ctx context.Context, page int) ([]*models.Property, error) {
	url := fmt.Sprintf("%s?page=%d", h.saleURL, page)
	return h.scrapeListings(ctx, url, models.ListingTypeSale)
}

// scrapeListings is the main scraping function
func (h *HOMESCrawler) scrapeListings(ctx context.Context, url string, listingType models.ListingType) ([]*models.Property, error) {
	// Configure chromedp options with compatibility fixes
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", h.headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.UserAgent(h.userAgent),
		chromedp.Flag("blink-settings", "imagesEnabled=false"),
		// Add these to avoid Chrome DevTools Protocol issues
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("metrics-recording-only", true),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("safebrowsing-disable-auto-update", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-web-security", true),
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, h.timeout)
	defer cancelTimeout()

	var rawProperties []map[string]string

	tasks := []chromedp.Action{
		chromedp.Navigate(url),
		chromedp.Sleep(h.waitAfter),
		chromedp.WaitVisible(`.module`, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var html string
			if err := chromedp.OuterHTML("body", &html, chromedp.ByQuery).Do(ctx); err != nil {
				return err
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
func (h *HOMESCrawler) parseProperties(html string, listingType models.ListingType) []map[string]string {
	properties := make([]map[string]string, 0)

	// HOMES uses different class names - this is a simplified parser
	// Find all property cards
	propertyPattern := regexp.MustCompile(`<div class="prg-estateListItem[^"]*"[^>]*>(.*?)</div>\s*<!-- /unit -->`)
	matches := propertyPattern.FindAllStringSubmatch(html, -1)

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

	return properties
}

// parsePropertyItem parses a single property item
func (h *HOMESCrawler) parsePropertyItem(html string, listingType models.ListingType) *HOMESProperty {
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

	// Extract title
	titlePattern := regexp.MustCompile(`<h3 class="prg-estateListItem_title"[^>]*>([^<]+)</h3>`)
	if match := titlePattern.FindStringSubmatch(html); len(match) > 1 {
		property.Title = strings.TrimSpace(match[1])
	}

	// Extract price - HOMES has different price display format
	pricePattern := regexp.MustCompile(`<span class="prg-estateListItem_price"[^>]*>([^<]+)</span>`)
	if match := pricePattern.FindStringSubmatch(html); len(match) > 1 {
		priceStr := strings.TrimSpace(match[1])
		property.PriceDisplay = priceStr
		property.Price = h.parsePrice(priceStr, listingType)
	}

	// Extract address
	addressPattern := regexp.MustCompile(`<li class="prg-estateListItem_data"[^>]*>.*?住所.*?<span[^>]*>([^<]+)</span>`)
	if match := addressPattern.FindStringSubmatch(html); len(match) > 1 {
		property.Address = strings.TrimSpace(match[1])
	}

	// Extract area
	areaPattern := regexp.MustCompile(`<li class="prg-estateListItem_data"[^>]*>.*?面積.*?<span[^>]*>([^<]+)</span>`)
	if match := areaPattern.FindStringSubmatch(html); len(match) > 1 {
		areaStr := strings.TrimSpace(match[1])
		property.Area = h.parseArea(areaStr)
	}

	// Extract layout
	layoutPattern := regexp.MustCompile(`<li class="prg-estateListItem_data"[^>]*>.*?間取り.*?<span[^>]*>([^<]+)</span>`)
	if match := layoutPattern.FindStringSubmatch(html); len(match) > 1 {
		property.Layout = strings.TrimSpace(match[1])
	}

	// Extract floor
	floorPattern := regexp.MustCompile(`<li class="prg-estateListItem_data"[^>]*>.*?階.*?<span[^>]*>([^<]+)</span>`)
	if match := floorPattern.FindStringSubmatch(html); len(match) > 1 {
		property.Floor = strings.TrimSpace(match[1])
	}

	// Extract building type
	buildingTypePattern := regexp.MustCompile(`<li class="prg-estateListItem_data"[^>]*>.*?築造.*?<span[^>]*>([^<]+)</span>`)
	if match := buildingTypePattern.FindStringSubmatch(html); len(match) > 1 {
		property.BuildingType = strings.TrimSpace(match[1])
	}

	// Extract station
	stationPattern := regexp.MustCompile(`<li class="prg-estateListItem_data"[^>]*>.*?沿線・駅.*?<span[^>]*>([^<]+)</span>`)
	if match := stationPattern.FindStringSubmatch(html); len(match) > 1 {
		stationText := strings.TrimSpace(match[1])
		parts := strings.Split(stationText, "　")
		if len(parts) > 0 {
			property.StationName = strings.TrimSpace(parts[0])
		}
		// Extract walking minutes
		walkPattern := regexp.MustCompile(`徒歩(\d+)分`)
		if walkMatch := walkPattern.FindStringSubmatch(stationText); len(walkMatch) > 1 {
			if min, err := strconv.Atoi(walkMatch[1]); err == nil {
				property.WalkMinutes = min
			}
		}
	}

	// Extract image URL
	imagePattern := regexp.MustCompile(`<img[^>]+class="prg-estateListItem_thumbnailImage"[^>]+src="([^"]+)"`)
	if match := imagePattern.FindStringSubmatch(html); len(match) > 1 {
		property.ImageURL = match[1]
	}

	return property
}

// parsePrice converts a price string to integer yen value
func (h *HOMESCrawler) parsePrice(priceStr string, listingType models.ListingType) int {
	priceStr = strings.TrimSpace(priceStr)
	priceStr = strings.ReplaceAll(priceStr, ",", "")
	priceStr = strings.ReplaceAll(priceStr, "円", "")
	priceStr = strings.ReplaceAll(priceStr, " ", "")

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
func (h *HOMESCrawler) parseArea(areaStr string) float64 {
	areaStr = strings.TrimSpace(areaStr)
	areaStr = strings.ReplaceAll(areaStr, "㎡", "")
	areaStr = strings.ReplaceAll(areaStr, "m²", "")

	if val, err := strconv.ParseFloat(areaStr, 64); err == nil {
		return val
	}

	return 0
}

// convertToProperty converts HOMESProperty to models.Property
func (h *HOMESCrawler) convertToProperty(raw map[string]string, listingType models.ListingType) *models.Property {
	property := models.NewProperty(models.SourceHOMES, raw["property_id"], listingType)

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
func (h *HOMESCrawler) ScrapeDetailPage(ctx context.Context, detailURL string) (*models.Property, error) {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", h.headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.UserAgent(h.userAgent),
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, h.timeout)
	defer cancelTimeout()

	var html string
	tasks := []chromedp.Action{
		chromedp.Navigate(detailURL),
		chromedp.Sleep(h.waitAfter),
		chromedp.OuterHTML("body", &html, chromedp.ByQuery),
	}

	if err := chromedp.Run(taskCtx, tasks...); err != nil {
		return nil, fmt.Errorf("failed to scrape detail page: %w", err)
	}

	property := h.parseDetailPage(html)

	return property, nil
}

// parseDetailPage parses the detail page HTML
func (h *HOMESCrawler) parseDetailPage(html string) *models.Property {
	property := &models.Property{}

	// Extract contact information
	contactPattern := regexp.MustCompile(`<div class="prg-realestateCompany_name"[^>]*>([^<]+)</div>`)
	if match := contactPattern.FindStringSubmatch(html); len(match) > 1 {
		property.ContactName = strings.TrimSpace(match[1])
	}

	// Extract phone number
	phonePattern := regexp.MustCompile(`class="prg-realestateCompany_tel"[^>]*>([^<]+)</`)
	if match := phonePattern.FindStringSubmatch(html); len(match) > 1 {
		property.ContactPhone = strings.TrimSpace(match[1])
	}

	// Extract all image URLs from gallery
	imagePattern := regexp.MustCompile(`<img[^>]+data-src="([^"]+)"[^>]+class="prg-photoImage"`)
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
