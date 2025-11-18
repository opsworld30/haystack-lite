package api

import (
	"haystack-lite/internal/storage"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, store *storage.Store) {
	r.Use(Logger())
	r.Use(Recovery())

	handler := NewHandler(store)
	chunkHandler := NewChunkHandler(store, "./data/chunks")
	webdavHandler := NewWebDAVHandler(store)
	s3Handler := NewS3Handler(store)
	healthHandler := NewHealthHandler(store)
	metricsHandler := NewMetricsHandler(store)

	setupWebRoutes(r)
	setupFileRoutes(r, handler)
	setupBatchRoutes(r, handler)
	setupChunkUploadRoutes(r, chunkHandler)
	setupWebDAVRoutes(r, webdavHandler)
	setupS3Routes(r, s3Handler)
	setupManagementRoutes(r, store, handler)
	setupHealthRoutes(r, healthHandler, metricsHandler)
}

func setupWebRoutes(r *gin.Engine) {
	r.Static("/static", "./web/static")
	r.StaticFile("/", "./web/index.html")
}

func setupFileRoutes(r *gin.Engine, handler *Handler) {
	files := r.Group("/file")
	{
		files.POST("", handler.Upload)
		files.GET("/:id/preview", handler.Preview)
		files.GET("/:id/info", handler.GetFileInfo)
		files.GET("/:id", handler.Download)
		files.DELETE("/:id", handler.Delete)
	}

	r.GET("/files", handler.ListFiles)
}

func setupBatchRoutes(r *gin.Engine, handler *Handler) {
	batch := r.Group("/files/batch")
	{
		batch.POST("", handler.BatchUpload)
		batch.POST("/download", handler.BatchDownload)
		batch.POST("/delete", handler.BatchDelete)
	}
}

func setupChunkUploadRoutes(r *gin.Engine, handler *ChunkHandler) {
	upload := r.Group("/upload")
	{
		upload.POST("/init", handler.InitChunkUpload)
		upload.POST("/:upload_id/chunk/:chunk_index", handler.UploadChunk)
		upload.POST("/:upload_id/complete", handler.CompleteChunkUpload)
		upload.GET("/:upload_id/progress", handler.GetChunkUploadProgress)
		upload.DELETE("/:upload_id", handler.CancelChunkUpload)
	}

	r.GET("/uploads", handler.ListChunkUploads)
}

func setupWebDAVRoutes(r *gin.Engine, handler *WebDAVHandler) {
	webdav := r.Group("/webdav")
	{
		webdav.Handle("OPTIONS", "/*path", handler.Options)
		webdav.Handle("PROPFIND", "/*path", handler.PropFind)
		webdav.GET("/*path", handler.Get)
		webdav.PUT("/*path", handler.Put)
		webdav.DELETE("/*path", handler.Delete)
		webdav.Handle("MKCOL", "/*path", handler.MkCol)
	}
}

func setupS3Routes(r *gin.Engine, handler *S3Handler) {
	s3 := r.Group("/s3")
	{
		s3.PUT("/:bucket/*key", handler.PutObject)
		s3.GET("/:bucket/*key", func(c *gin.Context) {
			if c.Query("prefix") != "" || c.Query("max-keys") != "" {
				handler.ListObjects(c)
			} else {
				handler.GetObject(c)
			}
		})
		s3.HEAD("/:bucket/*key", handler.HeadObject)
		s3.DELETE("/:bucket/*key", handler.DeleteObject)
	}
}

func setupManagementRoutes(r *gin.Engine, store *storage.Store, handler *Handler) {
	r.GET("/status", handler.Status)

	compaction := r.Group("/compaction")
	{
		compaction.GET("/stats", func(c *gin.Context) {
			stats := store.GetCompactionStats()
			c.JSON(200, stats)
		})
		compaction.POST("/run", func(c *gin.Context) {
			if err := store.RunCompactionNow(); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"message": "compaction triggered"})
		})
	}
}

func setupHealthRoutes(r *gin.Engine, healthHandler *HealthHandler, metricsHandler *MetricsHandler) {
	health := r.Group("/health")
	{
		health.GET("", healthHandler.Health)
		health.GET("/live", healthHandler.Liveness)
		health.GET("/ready", healthHandler.Readiness)
	}

	r.GET("/metrics", metricsHandler.Metrics)
}
