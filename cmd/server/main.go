package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nagoya-property-crawler/internal/config"
	"nagoya-property-crawler/internal/database"
	"nagoya-property-crawler/internal/web"
)

var (
	configPath = flag.String("config", "config.yaml", "配置文件路径")
	port       = flag.Int("port", 0, "服务器端口 (覆盖配置文件)")
)

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("加载配置失败: %v，使用默认配置", err)
		cfg = config.Default()
	}

	// Override port if specified
	if *port > 0 {
		cfg.Server.Port = *port
	}

	// Connect to database
	db, err := database.NewPostgresDB(&cfg.Database)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	log.Println("成功连接到数据库")

	// Create web server
	server, err := web.NewServer(db)
	if err != nil {
		log.Fatalf("创建服务器失败: %v", err)
	}

	// Create a custom handler that combines all routes
	handler := http.NewServeMux()

	// Register static files first
	fs := http.FileServer(http.Dir(cfg.Server.StaticDir))
	handler.Handle("/static/", http.StripPrefix("/static/", fs))

	// Register application routes
	server.RegisterRoutes(handler)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Handle shutdown gracefully
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		<-sigChan
		log.Println("收到退出信号，正在关闭服务器...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("服务器关闭失败: %v", err)
		}
	}()

	// Start server
	log.Printf("启动服务器，监听端口 %d", cfg.Server.Port)
	log.Printf("访问地址: http://localhost:%d", cfg.Server.Port)

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("服务器启动失败: %v", err)
	}

	log.Println("服务器已关闭")
}
