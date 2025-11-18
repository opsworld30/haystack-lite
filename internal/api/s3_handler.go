package api

import (
	"crypto/md5"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"haystack-lite/internal/storage"

	"github.com/gin-gonic/gin"
)

type S3Handler struct {
	store *storage.Store
}

func NewS3Handler(store *storage.Store) *S3Handler {
	return &S3Handler{store: store}
}

type ListBucketResult struct {
	XMLName     xml.Name   `xml:"ListBucketResult"`
	Name        string     `xml:"Name"`
	Prefix      string     `xml:"Prefix"`
	MaxKeys     int        `xml:"MaxKeys"`
	IsTruncated bool       `xml:"IsTruncated"`
	Contents    []S3Object `xml:"Contents"`
}

type S3Object struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         uint32 `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
}

type S3Error struct {
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
}

func (h *S3Handler) PutObject(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	if bucket == "" || key == "" {
		h.sendS3Error(c, "InvalidRequest", "Bucket and key are required")
		return
	}

	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.sendS3Error(c, "InternalError", err.Error())
		return
	}

	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	filename := fmt.Sprintf("%s/%s", bucket, key)
	id, err := h.store.WriteWithMetadata(data, filename, contentType)
	if err != nil {
		h.sendS3Error(c, "InternalError", err.Error())
		return
	}

	etag := fmt.Sprintf("%x", md5.Sum(data))
	c.Header("ETag", fmt.Sprintf("\"%s\"", etag))
	c.Header("x-amz-request-id", fmt.Sprintf("%d", id))
	c.Status(http.StatusOK)
}

func (h *S3Handler) GetObject(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	if bucket == "" || key == "" {
		h.sendS3Error(c, "InvalidRequest", "Bucket and key are required")
		return
	}

	filename := fmt.Sprintf("%s/%s", bucket, key)
	meta, err := h.store.FindByFilename(filename)
	if err != nil {
		h.sendS3Error(c, "NoSuchKey", "The specified key does not exist")
		return
	}

	data, metadata, err := h.store.ReadWithMetadata(meta.ID)
	if err != nil {
		h.sendS3Error(c, "InternalError", err.Error())
		return
	}

	c.Header("Content-Type", metadata.MimeType)
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Header("ETag", fmt.Sprintf("\"%s\"", metadata.MD5))
	c.Header("Last-Modified", time.Unix(metadata.CreateTime, 0).Format(http.TimeFormat))
	c.Data(http.StatusOK, metadata.MimeType, data)
}

func (h *S3Handler) DeleteObject(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	if bucket == "" || key == "" {
		h.sendS3Error(c, "InvalidRequest", "Bucket and key are required")
		return
	}

	filename := fmt.Sprintf("%s/%s", bucket, key)
	meta, err := h.store.FindByFilename(filename)
	if err != nil {
		c.Status(http.StatusNoContent)
		return
	}

	if err := h.store.Delete(meta.ID); err != nil {
		h.sendS3Error(c, "InternalError", err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *S3Handler) ListObjects(c *gin.Context) {
	bucket := c.Param("bucket")
	prefix := c.Query("prefix")
	maxKeys := 1000

	if maxKeysStr := c.Query("max-keys"); maxKeysStr != "" {
		if mk, err := strconv.Atoi(maxKeysStr); err == nil && mk > 0 {
			maxKeys = mk
		}
	}

	searchPrefix := bucket + "/"
	if prefix != "" {
		searchPrefix = bucket + "/" + prefix
	}

	files, err := h.store.ListByPrefix(searchPrefix, maxKeys)
	if err != nil {
		h.sendS3Error(c, "InternalError", err.Error())
		return
	}

	result := ListBucketResult{
		Name:        bucket,
		Prefix:      prefix,
		MaxKeys:     maxKeys,
		IsTruncated: false,
		Contents:    make([]S3Object, 0, len(files)),
	}

	for _, file := range files {
		key := strings.TrimPrefix(file.FileName, bucket+"/")
		result.Contents = append(result.Contents, S3Object{
			Key:          key,
			LastModified: time.Unix(file.CreateTime, 0).Format(time.RFC3339),
			ETag:         fmt.Sprintf("\"%s\"", file.MD5),
			Size:         file.Size,
			StorageClass: "STANDARD",
		})
	}

	c.XML(http.StatusOK, result)
}

func (h *S3Handler) HeadObject(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	if bucket == "" || key == "" {
		h.sendS3Error(c, "InvalidRequest", "Bucket and key are required")
		return
	}

	filename := fmt.Sprintf("%s/%s", bucket, key)
	meta, err := h.store.FindByFilename(filename)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	metadata, err := h.store.GetMetadata(meta.ID)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.Header("Content-Type", metadata.MimeType)
	c.Header("Content-Length", strconv.FormatUint(uint64(metadata.Size), 10))
	c.Header("ETag", fmt.Sprintf("\"%s\"", metadata.MD5))
	c.Header("Last-Modified", time.Unix(metadata.CreateTime, 0).Format(http.TimeFormat))
	c.Status(http.StatusOK)
}

func (h *S3Handler) sendS3Error(c *gin.Context, code, message string) {
	statusCode := http.StatusBadRequest
	switch code {
	case "NoSuchKey":
		statusCode = http.StatusNotFound
	case "InternalError":
		statusCode = http.StatusInternalServerError
	}

	c.XML(statusCode, S3Error{
		Code:    code,
		Message: message,
	})
}
