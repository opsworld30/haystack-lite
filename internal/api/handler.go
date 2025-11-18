package api

import (
	"io"
	"net/http"
	"strconv"

	"haystack-lite/internal/storage"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	store *storage.Store
}

func NewHandler(store *storage.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file uploaded"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = detectMimeType(file.Filename, data)
	}

	id, err := h.store.WriteWithMetadata(data, file.Filename, mimeType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        id,
		"size":      len(data),
		"filename":  file.Filename,
		"mime_type": mimeType,
	})
}

func (h *Handler) Download(c *gin.Context) {
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

	// 设置响应头
	if metadata.FileName != "" {
		c.Header("Content-Disposition", "attachment; filename="+metadata.FileName)
	}
	if metadata.MimeType != "" {
		c.Data(http.StatusOK, metadata.MimeType, data)
	} else {
		c.Data(http.StatusOK, "application/octet-stream", data)
	}
}

func (h *Handler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.store.Delete(id); err != nil {
		if err == storage.ErrNeedleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *Handler) Status(c *gin.Context) {
	status := h.store.Status()
	c.JSON(http.StatusOK, status)
}

func (h *Handler) ListFiles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	allFiles, err := h.store.ListAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	total := len(allFiles)

	for i := 0; i < len(allFiles)/2; i++ {
		allFiles[i], allFiles[len(allFiles)-1-i] = allFiles[len(allFiles)-1-i], allFiles[i]
	}

	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		c.JSON(http.StatusOK, gin.H{
			"files":      []gin.H{},
			"total":      total,
			"page":       page,
			"page_size":  pageSize,
			"total_page": (total + pageSize - 1) / pageSize,
		})
		return
	}

	if end > total {
		end = total
	}

	files := allFiles[start:end]
	result := make([]gin.H, 0, len(files))
	for _, f := range files {
		result = append(result, gin.H{
			"id":         f.ID,
			"filename":   f.FileName,
			"mime_type":  f.MimeType,
			"size":       f.Size,
			"md5":        f.MD5,
			"created_at": f.CreateTime,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"files":      result,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_page": (total + pageSize - 1) / pageSize,
	})
}

func detectMimeType(filename string, data []byte) string {
	ext := ""
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			ext = filename[i:]
			break
		}
	}

	mimeTypes := map[string]string{
		".md":   "text/markdown; charset=utf-8",
		".txt":  "text/plain; charset=utf-8",
		".html": "text/html; charset=utf-8",
		".htm":  "text/html; charset=utf-8",
		".css":  "text/css; charset=utf-8",
		".js":   "text/javascript; charset=utf-8",
		".json": "application/json; charset=utf-8",
		".xml":  "application/xml; charset=utf-8",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".pdf":  "application/pdf",
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".zip":  "application/zip",
		".tar":  "application/x-tar",
		".gz":   "application/gzip",
	}

	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}

	detected := http.DetectContentType(data)
	if detected != "application/octet-stream" {
		return detected
	}

	return "application/octet-stream"
}
