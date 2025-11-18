package api

import (
	"io"
	"net/http"
	"strconv"

	"haystack-lite/internal/storage"

	"github.com/gin-gonic/gin"
)

// BatchUpload 批量上传文件
func (h *Handler) BatchUpload(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files uploaded"})
		return
	}

	results := make([]map[string]interface{}, 0, len(files))
	errors := make([]map[string]interface{}, 0)

	for _, file := range files {
		f, err := file.Open()
		if err != nil {
			errors = append(errors, map[string]interface{}{
				"filename": file.Filename,
				"error":    "failed to open file",
			})
			continue
		}

		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			errors = append(errors, map[string]interface{}{
				"filename": file.Filename,
				"error":    "failed to read file",
			})
			continue
		}

		mimeType := file.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		id, err := h.store.WriteWithMetadata(data, file.Filename, mimeType)
		if err != nil {
			errors = append(errors, map[string]interface{}{
				"filename": file.Filename,
				"error":    err.Error(),
			})
			continue
		}

		results = append(results, map[string]interface{}{
			"id":        id,
			"filename":  file.Filename,
			"size":      len(data),
			"mime_type": mimeType,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": results,
		"errors":  errors,
		"total":   len(files),
	})
}

// BatchDownload 批量下载文件
func (h *Handler) BatchDownload(c *gin.Context) {
	var req struct {
		IDs []uint64 `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	results := make([]map[string]interface{}, 0, len(req.IDs))
	errors := make([]map[string]interface{}, 0)

	for _, id := range req.IDs {
		data, metadata, err := h.store.ReadWithMetadata(id)
		if err != nil {
			if err == storage.ErrNeedleNotFound {
				errors = append(errors, map[string]interface{}{
					"id":    id,
					"error": "file not found",
				})
			} else {
				errors = append(errors, map[string]interface{}{
					"id":    id,
					"error": err.Error(),
				})
			}
			continue
		}

		results = append(results, map[string]interface{}{
			"id":        id,
			"filename":  metadata.FileName,
			"size":      len(data),
			"mime_type": metadata.MimeType,
			"md5":       metadata.MD5,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": results,
		"errors":  errors,
		"total":   len(req.IDs),
	})
}

// BatchDelete 批量删除文件
func (h *Handler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []uint64 `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	results := make([]uint64, 0, len(req.IDs))
	errors := make([]map[string]interface{}, 0)

	for _, id := range req.IDs {
		if err := h.store.Delete(id); err != nil {
			if err == storage.ErrNeedleNotFound {
				errors = append(errors, map[string]interface{}{
					"id":    id,
					"error": "file not found",
				})
			} else {
				errors = append(errors, map[string]interface{}{
					"id":    id,
					"error": err.Error(),
				})
			}
			continue
		}

		results = append(results, id)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": results,
		"errors":  errors,
		"total":   len(req.IDs),
	})
}

// GetFileInfo 获取文件信息
func (h *Handler) GetFileInfo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	metadata, err := h.store.GetMetadata(id)
	if err != nil {
		if err == storage.ErrNeedleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          metadata.ID,
		"filename":    metadata.FileName,
		"size":        metadata.Size,
		"mime_type":   metadata.MimeType,
		"md5":         metadata.MD5,
		"deleted":     metadata.Deleted,
		"create_time": metadata.CreateTime,
		"update_time": metadata.UpdateTime,
	})
}
