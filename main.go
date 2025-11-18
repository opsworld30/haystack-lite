package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"haystack-lite/internal/api"
	"haystack-lite/internal/config"
	"haystack-lite/internal/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建存储
	store, err := storage.NewStore(cfg)
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	log.Printf("Storage initialized with %s: %s", cfg.Database.Type, cfg.Storage.DataDir)

	// 启动后台压缩
	compactionCfg := storage.CompactionConfig{
		Enabled:          cfg.Compaction.Enabled,
		Interval:         cfg.Compaction.Interval,
		DeletedThreshold: cfg.Compaction.DeletedThreshold,
		MinVolumeSize:    cfg.Compaction.MinVolumeSize,
	}
	store.StartCompaction(compactionCfg)

	r := gin.Default()
	api.SetupRoutes(r, store)

	srv := startServer(r, cfg.Server.Port)

	waitForShutdown(srv, store)
}

func loadConfig(path string) (*config.Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("Config file not found: %s, using default config", path)
		return config.Default(), nil
	}

	cfg, err := config.LoadConfig(path)
	if err != nil {
		return nil, err
	}

	log.Printf("Loaded config from: %s", path)
	return cfg, nil
}

func startServer(r *gin.Engine, port string) *http.Server {
	srv := &http.Server{
		Addr:    port,
		Handler: r,
	}

	go func() {
		log.Printf("Server starting on %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	return srv
}

func waitForShutdown(srv *http.Server, store *storage.Store) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Closing storage...")
	if err := store.Close(); err != nil {
		log.Printf("Error closing storage: %v", err)
	}

	log.Println("Server exited")
}
