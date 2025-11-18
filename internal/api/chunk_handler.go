package api

import (
	"io"
	"net/http"
	"strconv"

	"haystack-lite/internal/storage"

	"github.com/gin-gonic/gin"
)

// ChunkHandler 分片上传处理器
type ChunkHandler struct {
	store   *storage.Store
	manager *storage.ChunkManager
}

// NewChunkHandler 创建分片上传处理器
func NewChunkHandler(store *storage.Store, tempDir string) *ChunkHandler {
	return &ChunkHandler{
		store:   store,
		manager: storage.NewChunkManager(tempDir),
	}
}

// InitChunkUpload 初始化分片上传
func (h *ChunkHandler) InitChunkUpload(c *gin.Context) {
	var req struct {
		FileName    string `json:"filename" binding:"required"`
		TotalChunks int    `json:"total_chunks" binding:"required"`
		ChunkSize   int64  `json:"chunk_size" binding:"required"`
		TotalSize   int64  `json:"total_size" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	uploadID, err := h.manager.InitUpload(req.FileName, req.TotalChunks, req.ChunkSize, req.TotalSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"upload_id": uploadID,
		"filename":  req.FileName,
	})
}

// UploadChunk 上传分片
func (h *ChunkHandler) UploadChunk(c *gin.Context) {
	uploadID := c.Param("upload_id")
	chunkIndexStr := c.Param("chunk_index")

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chunk index"})
		return
	}

	file, err := c.FormFile("chunk")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no chunk uploaded"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open chunk"})
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read chunk"})
		return
	}

	if err := h.manager.UploadChunk(uploadID, chunkIndex, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 检查是否完成
	if h.manager.IsUploadComplete(uploadID) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "complete",
			"chunk":    chunkIndex,
			"complete": true,
		})
	} else {
		uploaded, total, _ := h.manager.GetUploadProgress(uploadID)
		c.JSON(http.StatusOK, gin.H{
			"status":   "uploading",
			"chunk":    chunkIndex,
			"uploaded": uploaded,
			"total":    total,
			"complete": false,
		})
	}
}

// CompleteChunkUpload 完成分片上传
func (h *ChunkHandler) CompleteChunkUpload(c *gin.Context) {
	uploadID := c.Param("upload_id")

	// 检查是否完成
	if !h.manager.IsUploadComplete(uploadID) {
		uploaded, total, _ := h.manager.GetUploadProgress(uploadID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":    "upload incomplete",
			"uploaded": uploaded,
			"total":    total,
		})
		return
	}

	// 合并分片
	data, filename, err := h.manager.MergeChunks(uploadID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 保存文件
	id, err := h.store.WriteWithMetadata(data, filename, "application/octet-stream")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 清理临时文件
	h.manager.CleanupUpload(uploadID)

	c.JSON(http.StatusOK, gin.H{
		"id":       id,
		"filename": filename,
		"size":     len(data),
	})
}

// GetChunkUploadProgress 获取上传进度
func (h *ChunkHandler) GetChunkUploadProgress(c *gin.Context) {
	uploadID := c.Param("upload_id")

	uploaded, total, err := h.manager.GetUploadProgress(uploadID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	complete := h.manager.IsUploadComplete(uploadID)
	progress := 0.0
	if total > 0 {
		progress = float64(uploaded) / float64(total) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"upload_id": uploadID,
		"uploaded":  uploaded,
		"total":     total,
		"progress":  progress,
		"complete":  complete,
	})
}

// CancelChunkUpload 取消分片上传
func (h *ChunkHandler) CancelChunkUpload(c *gin.Context) {
	uploadID := c.Param("upload_id")

	if err := h.manager.CleanupUpload(uploadID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "upload cancelled"})
}

// ListChunkUploads 列出所有上传
func (h *ChunkHandler) ListChunkUploads(c *gin.Context) {
	uploads := h.manager.ListUploads()
	c.JSON(http.StatusOK, gin.H{
		"uploads": uploads,
		"total":   len(uploads),
	})
}
