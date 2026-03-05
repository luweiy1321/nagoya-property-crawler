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

// AtHomeCrawler implements crawling for at-home website
type AtHomeCrawler struct {
	baseURL   string
	rentURL   string
	saleURL   string
	headless  bool
	userAgent string
	timeout   time.Duration
	waitAfter time.Duration
}

// NewAtHomeCrawler creates a new at-home crawler
func NewAtHomeCrawler(headless bool, userAgent string, timeout, waitAfter time.Duration) *AtHomeCrawler {
	return &AtHomeCrawler{
		baseURL:   "https://www.at-home.co.jp",
		rentURL:   "https://www.at-home.co.jp/chintai/aichi/nagoya-city/list/",
		saleURL:   "https://www.at-home.co.jp/chuko/aichi/nagoya-city/list/",
		headless:  headless,
		userAgent: userAgent,
		timeout:   timeout,
		waitAfter: waitAfter,
	}
}

// AtHomeProperty represents a property listing from at-home
type AtHomeProperty struct {
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

// ScrapeRentListings scrapes rental listings from at-home
func (a *AtHomeCrawler) ScrapeRentListings(ctx context.Context, page int) ([]*models.Property, error) {
	url := fmt.Sprintf("%s?page=%d", a.rentURL, page)
	return a.scrapeListings(ctx, url, models.ListingTypeRent)
}

// ScrapeSaleListings scrapes sale listings from at-home
func (a *AtHomeCrawler) ScrapeSaleListings(ctx context.Context, page int) ([]*models.Property, error) {
	url := fmt.Sprintf("%s?page=%d", a.saleURL, page)
	return a.scrapeListings(ctx, url, models.ListingTypeSale)
}

// scrapeListings is the main scraping function
func (a *AtHomeCrawler) scrapeListings(ctx context.Context, url string, listingType models.ListingType) ([]*models.Property, error) {
	// Configure chromedp options with compatibility fixes
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", a.headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.UserAgent(a.userAgent),
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

	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, a.timeout)
	defer cancelTimeout()

	var rawProperties []map[string]string

	tasks := []chromedp.Action{
		chromedp.Navigate(url),
		chromedp.Sleep(a.waitAfter),
		chromedp.WaitVisible(`.bukenList`, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var html string
			if err := chromedp.OuterHTML("body", &html, chromedp.ByQuery).Do(ctx); err != nil {
				return err
			}

			rawProperties = a.parseProperties(html, listingType)
			return nil
		}),
	}

	if err := chromedp.Run(taskCtx, tasks...); err != nil {
		return nil, fmt.Errorf("failed to scrape at-home: %w", err)
	}

	log.Printf("Scraped %d properties from at-home", len(rawProperties))

	properties := make([]*models.Property, 0, len(rawProperties))
	for _, raw := range rawProperties {
		property := a.convertToProperty(raw, listingType)
		properties = append(properties, property)
	}

	return properties, nil
}

// parseProperties parses the HTML and extracts property data
func (a *AtHomeCrawler) parseProperties(html string, listingType models.ListingType) []map[string]string {
	properties := make([]map[string]string, 0)

	// at-home uses .bukenList_item for property cards
	propertyPattern := regexp.MustCompile(`<li class="bukenList_item"[^>]*>(.*?)</li>`)
	matches := propertyPattern.FindAllStringSubmatch(html, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		propertyHTML := match[1]
		property := a.parsePropertyItem(propertyHTML, listingType)
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
func (a *AtHomeCrawler) parsePropertyItem(html string, listingType models.ListingType) *AtHomeProperty {
	property := &AtHomeProperty{}

	// Extract detail URL and property ID
	hrefPattern := regexp.MustCompile(`href="(/chintai/[^\s"]+|/chuko/[^\s"]+)"`)
	if match := hrefPattern.FindStringSubmatch(html); len(match) > 1 {
		property.DetailURL = "https://www.at-home.co.jp" + match[1]
		// Extract property ID from URL
		idPattern := regexp.MustCompile(`\/([a-zA-Z0-9_-]+)\/?$`)
		if idMatch := idPattern.FindStringSubmatch(match[1]); len(idMatch) > 1 {
			property.PropertyID = idMatch[1]
		}
	}

	// Extract title
	titlePattern := regexp.MustCompile(`<h3 class="bukenList_title"[^>]*>([^<]+)</h3>`)
	if match := titlePattern.FindStringSubmatch(html); len(match) > 1 {
		property.Title = strings.TrimSpace(match[1])
	}

	// Extract price
	pricePattern := regexp.MustCompile(`<p class="bukenList_price"[^>]*>.*?<span[^>]*>([^<]+)</span>`)
	if match := pricePattern.FindStringSubmatch(html); len(match) > 1 {
		priceStr := strings.TrimSpace(match[1])
		property.PriceDisplay = priceStr
		property.Price = a.parsePrice(priceStr, listingType)
	}

	// Extract address
	addressPattern := regexp.MustCompile(`<li class="bukenList_data"[^>]*>.*?住所.*?<span[^>]*>([^<]+)</span>`)
	if match := addressPattern.FindStringSubmatch(html); len(match) > 1 {
		property.Address = strings.TrimSpace(match[1])
	}

	// Extract area
	areaPattern := regexp.MustCompile(`<li class="bukenList_data"[^>]*>.*?面積.*?<span[^>]*>([^<]+)</span>`)
	if match := areaPattern.FindStringSubmatch(html); len(match) > 1 {
		areaStr := strings.TrimSpace(match[1])
		property.Area = a.parseArea(areaStr)
	}

	// Extract layout
	layoutPattern := regexp.MustCompile(`<li class="bukenList_data"[^>]*>.*?間取り.*?<span[^>]*>([^<]+)</span>`)
	if match := layoutPattern.FindStringSubmatch(html); len(match) > 1 {
		property.Layout = strings.TrimSpace(match[1])
	}

	// Extract floor
	floorPattern := regexp.MustCompile(`<li class="bukenList_data"[^>]*>.*?階.*?<span[^>]*>([^<]+)</span>`)
	if match := floorPattern.FindStringSubmatch(html); len(match) > 1 {
		property.Floor = strings.TrimSpace(match[1])
	}

	// Extract building type
	buildingTypePattern := regexp.MustCompile(`<li class="bukenList_data"[^>]*>.*?構造.*?<span[^>]*>([^<]+)</span>`)
	if match := buildingTypePattern.FindStringSubmatch(html); len(match) > 1 {
		property.BuildingType = strings.TrimSpace(match[1])
	}

	// Extract station
	stationPattern := regexp.MustCompile(`<li class="bukenList_data"[^>]*>.*?最寄駅.*?<span[^>]*>([^<]+)</span>`)
	if match := stationPattern.FindStringSubmatch(html); len(match) > 1 {
		stationText := strings.TrimSpace(match[1])
		// Extract station name
		stationParts := strings.Split(stationText, "駅")
		if len(stationParts) > 0 {
			property.StationName = strings.TrimSpace(stationParts[0]) + "駅"
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
	imagePattern := regexp.MustCompile(`<img[^>]+class="bukenList_thumbnail"[^>]+src="([^"]+)"`)
	if match := imagePattern.FindStringSubmatch(html); len(match) > 1 {
		property.ImageURL = match[1]
	}

	return property
}

// parsePrice converts a price string to integer yen value
func (a *AtHomeCrawler) parsePrice(priceStr string, listingType models.ListingType) int {
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

	// Handle sale prices
	if val, err := strconv.Atoi(priceStr); err == nil {
		return val
	}

	return 0
}

// parseArea converts an area string to float64
func (a *AtHomeCrawler) parseArea(areaStr string) float64 {
	areaStr = strings.TrimSpace(areaStr)
	areaStr = strings.ReplaceAll(areaStr, "㎡", "")
	areaStr = strings.ReplaceAll(areaStr, "m²", "")

	if val, err := strconv.ParseFloat(areaStr, 64); err == nil {
		return val
	}

	return 0
}

// convertToProperty converts AtHomeProperty to models.Property
func (a *AtHomeCrawler) convertToProperty(raw map[string]string, listingType models.ListingType) *models.Property {
	property := models.NewProperty(models.SourceAtHome, raw["property_id"], listingType)

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
func (a *AtHomeCrawler) ScrapeDetailPage(ctx context.Context, detailURL string) (*models.Property, error) {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", a.headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.UserAgent(a.userAgent),
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, a.timeout)
	defer cancelTimeout()

	var html string
	tasks := []chromedp.Action{
		chromedp.Navigate(detailURL),
		chromedp.Sleep(a.waitAfter),
		chromedp.OuterHTML("body", &html, chromedp.ByQuery),
	}

	if err := chromedp.Run(taskCtx, tasks...); err != nil {
		return nil, fmt.Errorf("failed to scrape detail page: %w", err)
	}

	property := a.parseDetailPage(html)

	return property, nil
}

// parseDetailPage parses the detail page HTML
func (a *AtHomeCrawler) parseDetailPage(html string) *models.Property {
	property := &models.Property{}

	// Extract contact information
	contactPattern := regexp.MustCompile(`<div class="shopName"[^>]*>([^<]+)</div>`)
	if match := contactPattern.FindStringSubmatch(html); len(match) > 1 {
		property.ContactName = strings.TrimSpace(match[1])
	}

	// Extract phone number
	phonePattern := regexp.MustCompile(`<span class="telNumber"[^>]*>([^<]+)</span>`)
	if match := phonePattern.FindStringSubmatch(html); len(match) > 1 {
		property.ContactPhone = strings.TrimSpace(match[1])
	}

	// Extract all image URLs from gallery
	imagePattern := regexp.MustCompile(`<img[^>]+src="([^"]+)"[^>]+class="galleryImage"`)
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
