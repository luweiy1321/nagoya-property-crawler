package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	url := "https://www.homes.co.jp/chintai/aichi/nagoya/list/"

	var html string
	var title string

	tasks := []chromedp.Action{
		chromedp.Navigate(url),
		chromedp.Sleep(10 * time.Second),
		chromedp.Title(&title),
		chromedp.OuterHTML("body", &html, chromedp.ByQuery),
	}

	if err := chromedp.Run(taskCtx, tasks...); err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("=== HOMES Analysis ===\n")
	log.Printf("URL: %s\n", url)
	log.Printf("Title: %s\n", title)
	log.Printf("HTML length: %d\n\n", len(html))

	// Show first part of HTML
	log.Printf("=== First 2000 chars of HTML ===\n")
	if len(html) > 2000 {
		log.Printf("%s\n", html[:2000])
	} else {
		log.Printf("%s\n", html)
	}

	// Check for errors
	errorIndicators := []string{
		"404",
		"Error",
		"エラー",
		"見つかりません",
		"not found",
	}
	log.Printf("\n=== Error Check ===\n")
	htmlLower := strings.ToLower(html)
	for _, indicator := range errorIndicators {
		if strings.Contains(htmlLower, strings.ToLower(indicator)) {
			log.Printf("WARNING: Found '%s'\n", indicator)
		}
	}
}
