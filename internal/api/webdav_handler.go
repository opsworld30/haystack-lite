package api

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"haystack-lite/internal/storage"

	"github.com/gin-gonic/gin"
)

type WebDAVHandler struct {
	store *storage.Store
}

func NewWebDAVHandler(store *storage.Store) *WebDAVHandler {
	return &WebDAVHandler{store: store}
}

type Multistatus struct {
	XMLName   xml.Name   `xml:"D:multistatus"`
	Xmlns     string     `xml:"xmlns:D,attr"`
	Responses []Response `xml:"D:response"`
}

type Response struct {
	Href     string   `xml:"D:href"`
	Propstat Propstat `xml:"D:propstat"`
}

type Propstat struct {
	Prop   Prop   `xml:"D:prop"`
	Status string `xml:"D:status"`
}

type Prop struct {
	DisplayName      string        `xml:"D:displayname,omitempty"`
	CreationDate     string        `xml:"D:creationdate,omitempty"`
	GetLastModified  string        `xml:"D:getlastmodified,omitempty"`
	GetContentLength string        `xml:"D:getcontentlength,omitempty"`
	GetContentType   string        `xml:"D:getcontenttype,omitempty"`
	ResourceType     *ResourceType `xml:"D:resourcetype,omitempty"`
}

type ResourceType struct {
	Collection *struct{} `xml:"D:collection,omitempty"`
}

func (h *WebDAVHandler) Options(c *gin.Context) {
	c.Header("DAV", "1, 2")
	c.Header("Allow", "OPTIONS, GET, HEAD, POST, PUT, DELETE, PROPFIND, MKCOL")
	c.Status(http.StatusOK)
}

func (h *WebDAVHandler) PropFind(c *gin.Context) {
	urlPath := c.Param("path")
	if urlPath == "" {
		urlPath = "/"
	}

	depth := c.GetHeader("Depth")
	if depth == "" {
		depth = "infinity"
	}

	multistatus := Multistatus{
		Xmlns:     "DAV:",
		Responses: []Response{},
	}

	if urlPath == "/" {
		multistatus.Responses = append(multistatus.Responses, Response{
			Href: "/",
			Propstat: Propstat{
				Prop: Prop{
					DisplayName:  "root",
					ResourceType: &ResourceType{Collection: &struct{}{}},
				},
				Status: "HTTP/1.1 200 OK",
			},
		})

		if depth != "0" {
			files, err := h.store.ListAll()
			if err == nil {
				for _, file := range files {
					multistatus.Responses = append(multistatus.Responses, h.fileToResponse(file))
				}
			}
		}
	} else {
		cleanPath := strings.TrimPrefix(urlPath, "/")
		meta, err := h.store.FindByFilename(cleanPath)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		metadata, err := h.store.GetMetadata(meta.ID)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		multistatus.Responses = append(multistatus.Responses, h.metadataToResponse(metadata))
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.XML(http.StatusMultiStatus, multistatus)
}

func (h *WebDAVHandler) Get(c *gin.Context) {
	urlPath := strings.TrimPrefix(c.Param("path"), "/")
	if urlPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}

	meta, err := h.store.FindByFilename(urlPath)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	data, metadata, err := h.store.ReadWithMetadata(meta.ID)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.Header("Content-Type", metadata.MimeType)
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Header("Last-Modified", time.Unix(metadata.CreateTime, 0).Format(http.TimeFormat))
	c.Data(http.StatusOK, metadata.MimeType, data)
}

func (h *WebDAVHandler) Put(c *gin.Context) {
	urlPath := strings.TrimPrefix(c.Param("path"), "/")
	if urlPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}

	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	existingMeta, _ := h.store.FindByFilename(urlPath)
	if existingMeta != nil {
		if err := h.store.Delete(existingMeta.ID); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
	}

	_, err = h.store.WriteWithMetadata(data, urlPath, contentType)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	if existingMeta != nil {
		c.Status(http.StatusNoContent)
	} else {
		c.Status(http.StatusCreated)
	}
}

func (h *WebDAVHandler) Delete(c *gin.Context) {
	urlPath := strings.TrimPrefix(c.Param("path"), "/")
	if urlPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}

	meta, err := h.store.FindByFilename(urlPath)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	if err := h.store.Delete(meta.ID); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *WebDAVHandler) MkCol(c *gin.Context) {
	c.Status(http.StatusCreated)
}

func (h *WebDAVHandler) fileToResponse(file *storage.FileMetadata) Response {
	return Response{
		Href: "/" + file.FileName,
		Propstat: Propstat{
			Prop: Prop{
				DisplayName:      path.Base(file.FileName),
				CreationDate:     time.Unix(file.CreateTime, 0).Format(time.RFC3339),
				GetLastModified:  time.Unix(file.CreateTime, 0).Format(http.TimeFormat),
				GetContentLength: strconv.FormatUint(uint64(file.Size), 10),
				GetContentType:   file.MimeType,
			},
			Status: "HTTP/1.1 200 OK",
		},
	}
}

func (h *WebDAVHandler) metadataToResponse(metadata *storage.FileMetadata) Response {
	return Response{
		Href: "/" + metadata.FileName,
		Propstat: Propstat{
			Prop: Prop{
				DisplayName:      path.Base(metadata.FileName),
				CreationDate:     time.Unix(metadata.CreateTime, 0).Format(time.RFC3339),
				GetLastModified:  time.Unix(metadata.CreateTime, 0).Format(http.TimeFormat),
				GetContentLength: fmt.Sprintf("%d", metadata.Size),
				GetContentType:   metadata.MimeType,
			},
			Status: "HTTP/1.1 200 OK",
		},
	}
}
