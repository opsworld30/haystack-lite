package api

import (
	"net/http"
	"strconv"
	"strings"

	"haystack-lite/internal/storage"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Preview(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	data, metadata, err := h.store.ReadWithMetadata(id)
	if err != nil {
		if err == storage.ErrNeedleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	mimeType := metadata.MimeType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	if isPreviewable(mimeType) {
		c.Header("Content-Disposition", "inline")
		if !hasCharset(mimeType) && isTextType(mimeType) {
			mimeType = mimeType + "; charset=utf-8"
		}
		c.Data(http.StatusOK, mimeType, data)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":     "file type not previewable",
			"mime_type": mimeType,
			"supported": []string{
				"image/*",
				"video/*",
				"audio/*",
				"application/pdf",
				"text/*",
			},
		})
	}
}

func isPreviewable(mimeType string) bool {
	previewableTypes := []string{
		"image/",
		"video/",
		"audio/",
		"application/pdf",
		"text/",
	}

	for _, t := range previewableTypes {
		if strings.HasPrefix(mimeType, t) {
			return true
		}
	}
	return false
}

func hasCharset(mimeType string) bool {
	for i := 0; i < len(mimeType); i++ {
		if mimeType[i] == ';' {
			return true
		}
	}
	return false
}

func isTextType(mimeType string) bool {
	textTypes := []string{"text/", "application/json", "application/xml", "application/javascript"}
	for _, t := range textTypes {
		if len(mimeType) >= len(t) && mimeType[:len(t)] == t {
			return true
		}
	}
	return false
}
