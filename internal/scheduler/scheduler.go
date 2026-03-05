package scheduler

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"nagoya-property-crawler/internal/crawler"
	"nagoya-property-crawler/internal/database"
	"nagoya-property-crawler/internal/models"
)

// Scheduler handles periodic crawling tasks
type Scheduler struct {
	cron        *cron.Cron
	db          *database.PostgresDB
	config      *Config
	running     bool
	mu          sync.RWMutex
	cancelFuncs map[string]context.CancelFunc
	nextRun     time.Time
	lastRun     time.Time
}

// Config holds scheduler configuration
type Config struct {
	CronExpression string            `yaml:"cron"` // Cron expression (e.g., "0 0 */2 * * *" for every 2 days)
	Enabled        bool              `yaml:"enabled"`
	Sources        map[string]Source `yaml:"sources"`
}

// Source holds source-specific configuration
type Source struct {
	Enabled  bool   `yaml:"enabled"`
	BaseURL  string `yaml:"base_url"`
	RentURL  string `yaml:"rent_url"`
	SaleURL  string `yaml:"sale_url"`
	MaxPages int    `yaml:"max_pages"`
}

// DefaultConfig returns default scheduler configuration
func DefaultConfig() *Config {
	return &Config{
		CronExpression: "0 0 0 */2 * *", // Every 2 days at midnight
		Enabled:        true,
		Sources: map[string]Source{
			"suumo": {
				Enabled:  true,
				BaseURL:  "https://suumo.jp",
				RentURL:  "https://suumo.jp/chintai/aichi/nc_nagoya/",
				SaleURL:  "https://suumo.jp/ms/chuko/aichi/nc_nagoya/",
				MaxPages: 5,
			},
			"homes": {
				Enabled:  true,
				BaseURL:  "https://www.homes.co.jp",
				RentURL:  "https://www.homes.co.jp/chintai/aichi/nagoya/list/",
				SaleURL:  "https://www.homes.co.jp/chuko/aichi/nagoya/list/",
				MaxPages: 5,
			},
			"athome": {
				Enabled:  true,
				BaseURL:  "https://www.at-home.co.jp",
				RentURL:  "https://www.at-home.co.jp/chintai/aichi/nagoya-city/list/",
				SaleURL:  "https://www.at-home.co.jp/chuko/aichi/nagoya-city/list/",
				MaxPages: 5,
			},
		},
	}
}

// NewScheduler creates a new scheduler
func NewScheduler(db *database.PostgresDB, config *Config) *Scheduler {
	return &Scheduler{
		cron:        cron.New(cron.WithSeconds()),
		db:          db,
		config:      config,
		cancelFuncs: make(map[string]context.CancelFunc),
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	if !s.config.Enabled {
		log.Println("Scheduler is disabled")
		return nil
	}

	// Add the main cron job
	id, err := s.cron.AddFunc(s.config.CronExpression, func() {
		s.mu.Lock()
		s.lastRun = time.Now()
		s.mu.Unlock()
		s.runAllCrawlers()
	})
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.cron.Start()
	s.running = true

	// Set initial next run time
	entries := s.cron.Entries()
	for _, entry := range entries {
		if entry.ID == id {
			s.mu.Lock()
			s.nextRun = entry.Next
			s.mu.Unlock()
			break
		}
	}

	log.Printf("Scheduler started with cron expression: %s", s.config.CronExpression)
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	ctx := s.cron.Stop()
	<-ctx.Done()

	// Cancel all running jobs
	for name, cancel := range s.cancelFuncs {
		log.Printf("Canceling job: %s", name)
		cancel()
	}

	s.running = false
	log.Println("Scheduler stopped")
}

// IsRunning returns whether the scheduler is running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// RunNow runs all crawlers immediately
func (s *Scheduler) RunNow() {
	s.runAllCrawlers()
}

// runAllCrawlers runs all enabled crawlers
func (s *Scheduler) runAllCrawlers() {
	log.Println("Starting scheduled crawl run...")

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	// Run each source in a goroutine
	var wg sync.WaitGroup
	results := make(chan string, len(s.config.Sources))

	for name, source := range s.config.Sources {
		if !source.Enabled {
			continue
		}

		wg.Add(1)
		go func(sourceName string, sourceConfig Source) {
			defer wg.Done()

			// Store cancel function for this job
			jobCtx, jobCancel := context.WithCancel(ctx)
			s.mu.Lock()
			s.cancelFuncs[sourceName] = jobCancel
			s.mu.Unlock()
			defer func() {
				s.mu.Lock()
				delete(s.cancelFuncs, sourceName)
				s.mu.Unlock()
			}()

			result := s.runSource(jobCtx, sourceName, sourceConfig)
			results <- result
		}(name, source)
	}

	// Wait for all crawlers to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for result := range results {
		log.Println(result)
	}

	// Clean up expired listings
	count, err := s.db.DeleteExpired()
	if err != nil {
		log.Printf("Error deleting expired properties: %v", err)
	} else if count > 0 {
		log.Printf("Deleted %d expired properties", count)
	}

	log.Println("Scheduled crawl run completed")
}

// runSource runs a single source crawler
func (s *Scheduler) runSource(ctx context.Context, name string, source Source) string {
	log.Printf("[%s] Starting crawl...", name)

	startTime := time.Now()
	totalCount := 0

	// Configure crawler options
	headless := true
	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	timeout := 5 * time.Minute
	waitAfter := 3 * time.Second

	switch name {
	case "suumo":
		suumo := crawler.NewSUUMOCrawler(headless, userAgent, timeout, waitAfter)
		// Crawl rent
		if source.RentURL != "" {
			count := s.crawlSUUMO(ctx, suumo, models.ListingTypeRent, source.MaxPages, name)
			totalCount += count
		}
		// Check context
		select {
		case <-ctx.Done():
			return fmt.Sprintf("[%s] Canceled: %d properties", name, totalCount)
		default:
		}
		// Crawl sale
		if source.SaleURL != "" {
			count := s.crawlSUUMO(ctx, suumo, models.ListingTypeSale, source.MaxPages, name)
			totalCount += count
		}

	case "homes":
		homes := crawler.NewHOMESCrawler(headless, userAgent, timeout, waitAfter)
		// Crawl rent
		if source.RentURL != "" {
			count := s.crawlHOMES(ctx, homes, models.ListingTypeRent, source.MaxPages, name)
			totalCount += count
		}
		// Check context
		select {
		case <-ctx.Done():
			return fmt.Sprintf("[%s] Canceled: %d properties", name, totalCount)
		default:
		}
		// Crawl sale
		if source.SaleURL != "" {
			count := s.crawlHOMES(ctx, homes, models.ListingTypeSale, source.MaxPages, name)
			totalCount += count
		}

	case "athome":
		athome := crawler.NewAtHomeCrawler(headless, userAgent, timeout, waitAfter)
		// Crawl rent
		if source.RentURL != "" {
			count := s.crawlAtHome(ctx, athome, models.ListingTypeRent, source.MaxPages, name)
			totalCount += count
		}
		// Check context
		select {
		case <-ctx.Done():
			return fmt.Sprintf("[%s] Canceled: %d properties", name, totalCount)
		default:
		}
		// Crawl sale
		if source.SaleURL != "" {
			count := s.crawlAtHome(ctx, athome, models.ListingTypeSale, source.MaxPages, name)
			totalCount += count
		}

	default:
		return fmt.Sprintf("[%s] Unknown source", name)
	}

	duration := time.Since(startTime)
	return fmt.Sprintf("[%s] Completed: %d properties in %v", name, totalCount, duration)
}

// crawlSUUMO crawls SUUMO for a specific listing type
func (s *Scheduler) crawlSUUMO(ctx context.Context, c *crawler.SUUMOCrawler, listingType models.ListingType, maxPages int, sourceName string) int {
	return s.crawlGeneric(ctx, listingType, maxPages, sourceName,
		func(ctx context.Context, page int) ([]*models.Property, error) {
			if listingType == models.ListingTypeRent {
				return c.ScrapeRentListings(ctx, page)
			}
			return c.ScrapeSaleListings(ctx, page)
		},
	)
}

// crawlHOMES crawls HOMES for a specific listing type
func (s *Scheduler) crawlHOMES(ctx context.Context, c *crawler.HOMESCrawler, listingType models.ListingType, maxPages int, sourceName string) int {
	return s.crawlGeneric(ctx, listingType, maxPages, sourceName,
		func(ctx context.Context, page int) ([]*models.Property, error) {
			if listingType == models.ListingTypeRent {
				return c.ScrapeRentListings(ctx, page)
			}
			return c.ScrapeSaleListings(ctx, page)
		},
	)
}

// crawlAtHome crawls at-home for a specific listing type
func (s *Scheduler) crawlAtHome(ctx context.Context, c *crawler.AtHomeCrawler, listingType models.ListingType, maxPages int, sourceName string) int {
	return s.crawlGeneric(ctx, listingType, maxPages, sourceName,
		func(ctx context.Context, page int) ([]*models.Property, error) {
			if listingType == models.ListingTypeRent {
				return c.ScrapeRentListings(ctx, page)
			}
			return c.ScrapeSaleListings(ctx, page)
		},
	)
}

// scrapeFunc is a function that scrapes a page
type scrapeFunc func(ctx context.Context, page int) ([]*models.Property, error)

// crawlGeneric crawls a specific listing type using a scrape function
func (s *Scheduler) crawlGeneric(ctx context.Context, listingType models.ListingType, maxPages int, sourceName string, scrape scrapeFunc) int {
	log.Printf("[%s] Crawling %s listings...", sourceName, listingType)

	totalCount := 0
	allProperties := make([]*models.Property, 0)

	for page := 1; page <= maxPages; page++ {
		select {
		case <-ctx.Done():
			log.Printf("[%s] Context canceled, stopping %s crawl at page %d", sourceName, listingType, page)
			break
		default:
		}

		properties, err := scrape(ctx, page)
		if err != nil {
			log.Printf("[%s] Error scraping page %d: %v", sourceName, page, err)
			continue
		}

		if len(properties) == 0 {
			log.Printf("[%s] No more properties found on page %d", sourceName, page)
			break
		}

		allProperties = append(allProperties, properties...)
		log.Printf("[%s] Page %d: %d properties", sourceName, page, len(properties))

		// Random delay between pages to avoid blocking
		if page < maxPages && len(properties) > 0 {
			delay := time.Duration(2+rand.Intn(3)) * time.Second
			select {
			case <-ctx.Done():
				break
			case <-time.After(delay):
			}
		}
	}

	// Save all properties to database
	if len(allProperties) > 0 {
		if err := s.db.UpsertBatch(allProperties); err != nil {
			log.Printf("[%s] Error saving properties: %v", sourceName, err)
		} else {
			log.Printf("[%s] Saved %d %s properties", sourceName, len(allProperties), listingType)
			totalCount = len(allProperties)
		}
	}

	return totalCount
}

// GetNextRunTime returns the next scheduled run time
func (s *Scheduler) GetNextRunTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nextRun
}

// GetPreviousRunTime returns the previous scheduled run time
func (s *Scheduler) GetPreviousRunTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastRun
}

// RunSpecificSource runs a specific source crawler
func (s *Scheduler) RunSpecificSource(sourceName string) error {
	source, ok := s.config.Sources[sourceName]
	if !ok {
		return fmt.Errorf("unknown source: %s", sourceName)
	}

	if !source.Enabled {
		return fmt.Errorf("source %s is disabled", sourceName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()

	result := s.runSource(ctx, sourceName, source)
	log.Println(result)

	return nil
}
