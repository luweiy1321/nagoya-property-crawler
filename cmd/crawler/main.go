package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nagoya-property-crawler/internal/config"
	"nagoya-property-crawler/internal/crawler"
	"nagoya-property-crawler/internal/database"
	"nagoya-property-crawler/internal/models"
)

var (
	configPath = flag.String("config", "config.yaml", "配置文件路径")
	source     = flag.String("source", "all", "数据源 (suumo, homes, athome, all)")
	listingType = flag.String("type", "all", "房源类型 (rent, sale, all)")
	pages      = flag.Int("pages", 5, "爬取页数")
	headless   = flag.Bool("headless", true, "是否使用无头模式")
)

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// Connect to database
	db, err := database.NewPostgresDB(&cfg.Database)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	log.Println("成功连接到数据库")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("收到退出信号，正在清理...")
		cancel()
	}()

	// Configure crawler
	userAgent := cfg.Crawler.UserAgent
	timeout := cfg.Crawler.Timeout
	waitAfter := cfg.Crawler.WaitAfterLoad

	// Run crawlers based on flags
	totalCount := 0

	if *source == "all" || *source == "suumo" {
		if cfg.Sources.SUUMO.Enabled {
			count := runSuumo(ctx, db, userAgent, timeout, waitAfter, *listingType, *pages, *headless)
			totalCount += count
		} else {
			log.Println("SUUMO crawler is disabled in config")
		}
	}

	if *source == "all" || *source == "homes" {
		if cfg.Sources.HOMES.Enabled {
			count := runHomes(ctx, db, userAgent, timeout, waitAfter, *listingType, *pages, *headless)
			totalCount += count
		} else {
			log.Println("HOMES crawler is disabled in config")
		}
	}

	if *source == "all" || *source == "athome" {
		if cfg.Sources.AtHome.Enabled {
			count := runAtHome(ctx, db, userAgent, timeout, waitAfter, *listingType, *pages, *headless)
			totalCount += count
		} else {
			log.Println("at-home crawler is disabled in config")
		}
	}

	// Clean up expired listings
	log.Println("清理过期房源...")
	expiredCount, err := db.DeleteExpired()
	if err != nil {
		log.Printf("清理过期房源失败: %v", err)
	} else {
		log.Printf("已删除 %d 条过期房源", expiredCount)
	}

	log.Printf("爬取完成！共获取 %d 条房源", totalCount)
}

func runSuumo(ctx context.Context, db *database.PostgresDB, userAgent string, timeout, waitAfter time.Duration, listingType string, maxPages int, headless bool) int {
	log.Println("开始爬取 SUUMO...")

	suumo := crawler.NewSUUMOCrawler(headless, userAgent, timeout, waitAfter)
	allProperties := make([]*models.Property, 0)

	// Crawl rent listings
	if listingType == "all" || listingType == "rent" {
		log.Println("爬取 SUUMO 出租房源...")
		for page := 1; page <= maxPages; page++ {
			select {
			case <-ctx.Done():
				log.Println("SUUMO rent crawl canceled")
				break
			default:
			}

			properties, err := suumo.ScrapeRentListings(ctx, page)
			if err != nil {
				log.Printf("SUUMO rent page %d error: %v", page, err)
				continue
			}

			if len(properties) == 0 {
				log.Printf("SUUMO rent no more properties at page %d", page)
				break
			}

			allProperties = append(allProperties, properties...)
			log.Printf("SUUMO rent page %d: %d properties", page, len(properties))

			// Random delay
			time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)
		}
	}

	// Crawl sale listings
	if listingType == "all" || listingType == "sale" {
		log.Println("爬取 SUUMO 出售房源...")
		for page := 1; page <= maxPages; page++ {
			select {
			case <-ctx.Done():
				log.Println("SUUMO sale crawl canceled")
				break
			default:
			}

			properties, err := suumo.ScrapeSaleListings(ctx, page)
			if err != nil {
				log.Printf("SUUMO sale page %d error: %v", page, err)
				continue
			}

			if len(properties) == 0 {
				log.Printf("SUUMO sale no more properties at page %d", page)
				break
			}

			allProperties = append(allProperties, properties...)
			log.Printf("SUUMO sale page %d: %d properties", page, len(properties))

			// Random delay
			time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)
		}
	}

	// Save to database
	if len(allProperties) > 0 {
		log.Printf("保存 %d 条 SUUMO 房源到数据库...", len(allProperties))
		if err := db.UpsertBatch(allProperties); err != nil {
			log.Printf("保存失败: %v", err)
			return 0
		}
	}

	return len(allProperties)
}

func runHomes(ctx context.Context, db *database.PostgresDB, userAgent string, timeout, waitAfter time.Duration, listingType string, maxPages int, headless bool) int {
	log.Println("开始爬取 HOMES (V4 - List + Detail pages)...")

	homes := crawler.NewHOMESCrawlerV4(headless, userAgent, timeout)
	allProperties := make([]*models.Property, 0)

	// Crawl rent listings
	if listingType == "all" || listingType == "rent" {
		log.Println("爬取 HOMES 出租房源...")
		for page := 1; page <= maxPages; page++ {
			select {
			case <-ctx.Done():
				log.Println("HOMES rent crawl canceled")
				break
			default:
			}

			properties, err := homes.ScrapeRentListings(ctx, page)
			if err != nil {
				log.Printf("HOMES rent page %d error: %v", page, err)
				continue
			}

			if len(properties) == 0 {
				log.Printf("HOMES rent no more properties at page %d", page)
				break
			}

			allProperties = append(allProperties, properties...)
			log.Printf("HOMES rent page %d: %d properties", page, len(properties))

			// Random delay
			time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)
		}
	}

	// Crawl sale listings
	if listingType == "all" || listingType == "sale" {
		log.Println("爬取 HOMES 出售房源...")
		for page := 1; page <= maxPages; page++ {
			select {
			case <-ctx.Done():
				log.Println("HOMES sale crawl canceled")
				break
			default:
			}

			properties, err := homes.ScrapeSaleListings(ctx, page)
			if err != nil {
				log.Printf("HOMES sale page %d error: %v", page, err)
				continue
			}

			if len(properties) == 0 {
				log.Printf("HOMES sale no more properties at page %d", page)
				break
			}

			allProperties = append(allProperties, properties...)
			log.Printf("HOMES sale page %d: %d properties", page, len(properties))

			// Random delay
			time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)
		}
	}

	// Save to database
	if len(allProperties) > 0 {
		log.Printf("保存 %d 条 HOMES 房源到数据库...", len(allProperties))
		if err := db.UpsertBatch(allProperties); err != nil {
			log.Printf("保存失败: %v", err)
			return 0
		}
	}

	return len(allProperties)
}

func runAtHome(ctx context.Context, db *database.PostgresDB, userAgent string, timeout, waitAfter time.Duration, listingType string, maxPages int, headless bool) int {
	log.Println("开始爬取 at-home...")

	athome := crawler.NewAtHomeCrawler(headless, userAgent, timeout, waitAfter)
	allProperties := make([]*models.Property, 0)

	// Crawl rent listings
	if listingType == "all" || listingType == "rent" {
		log.Println("爬取 at-home 出租房源...")
		for page := 1; page <= maxPages; page++ {
			select {
			case <-ctx.Done():
				log.Println("at-home rent crawl canceled")
				break
			default:
			}

			properties, err := athome.ScrapeRentListings(ctx, page)
			if err != nil {
				log.Printf("at-home rent page %d error: %v", page, err)
				continue
			}

			if len(properties) == 0 {
				log.Printf("at-home rent no more properties at page %d", page)
				break
			}

			allProperties = append(allProperties, properties...)
			log.Printf("at-home rent page %d: %d properties", page, len(properties))

			// Random delay
			time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)
		}
	}

	// Crawl sale listings
	if listingType == "all" || listingType == "sale" {
		log.Println("爬取 at-home 出售房源...")
		for page := 1; page <= maxPages; page++ {
			select {
			case <-ctx.Done():
				log.Println("at-home sale crawl canceled")
				break
			default:
			}

			properties, err := athome.ScrapeSaleListings(ctx, page)
			if err != nil {
				log.Printf("at-home sale page %d error: %v", page, err)
				continue
			}

			if len(properties) == 0 {
				log.Printf("at-home sale no more properties at page %d", page)
				break
			}

			allProperties = append(allProperties, properties...)
			log.Printf("at-home sale page %d: %d properties", page, len(properties))

			// Random delay
			time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)
		}
	}

	// Save to database
	if len(allProperties) > 0 {
		log.Printf("保存 %d 条 at-home 房源到数据库...", len(allProperties))
		if err := db.UpsertBatch(allProperties); err != nil {
			log.Printf("保存失败: %v", err)
			return 0
		}
	}

	return len(allProperties)
}
