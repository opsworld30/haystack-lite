package storage

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ChunkUpload 分片上传管理
type ChunkUpload struct {
	UploadID    string
	FileName    string
	TotalChunks int
	ChunkSize   int64
	TotalSize   int64
	Chunks      map[int]bool
	TempDir     string
	mu          sync.RWMutex
}

// ChunkManager 分片管理器
type ChunkManager struct {
	uploads map[string]*ChunkUpload
	tempDir string
	mu      sync.RWMutex
}

// NewChunkManager 创建分片管理器
func NewChunkManager(tempDir string) *ChunkManager {
	os.MkdirAll(tempDir, 0755)
	return &ChunkManager{
		uploads: make(map[string]*ChunkUpload),
		tempDir: tempDir,
	}
}

// InitUpload 初始化分片上传
func (cm *ChunkManager) InitUpload(filename string, totalChunks int, chunkSize, totalSize int64) (string, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 生成上传 ID
	uploadID := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s-%d-%d", filename, totalSize, totalChunks))))

	// 创建临时目录
	uploadDir := filepath.Join(cm.tempDir, uploadID)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", err
	}

	upload := &ChunkUpload{
		UploadID:    uploadID,
		FileName:    filename,
		TotalChunks: totalChunks,
		ChunkSize:   chunkSize,
		TotalSize:   totalSize,
		Chunks:      make(map[int]bool),
		TempDir:     uploadDir,
	}

	cm.uploads[uploadID] = upload
	return uploadID, nil
}

// UploadChunk 上传分片
func (cm *ChunkManager) UploadChunk(uploadID string, chunkIndex int, data []byte) error {
	cm.mu.RLock()
	upload, exists := cm.uploads[uploadID]
	cm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("upload not found: %s", uploadID)
	}

	upload.mu.Lock()
	defer upload.mu.Unlock()

	// 检查分片是否已上传
	if upload.Chunks[chunkIndex] {
		return nil
	}

	// 保存分片
	chunkPath := filepath.Join(upload.TempDir, fmt.Sprintf("chunk_%d", chunkIndex))
	if err := os.WriteFile(chunkPath, data, 0644); err != nil {
		return err
	}

	upload.Chunks[chunkIndex] = true
	return nil
}

// GetUploadProgress 获取上传进度
func (cm *ChunkManager) GetUploadProgress(uploadID string) (int, int, error) {
	cm.mu.RLock()
	upload, exists := cm.uploads[uploadID]
	cm.mu.RUnlock()

	if !exists {
		return 0, 0, fmt.Errorf("upload not found: %s", uploadID)
	}

	upload.mu.RLock()
	defer upload.mu.RUnlock()

	uploaded := len(upload.Chunks)
	return uploaded, upload.TotalChunks, nil
}

// IsUploadComplete 检查上传是否完成
func (cm *ChunkManager) IsUploadComplete(uploadID string) bool {
	cm.mu.RLock()
	upload, exists := cm.uploads[uploadID]
	cm.mu.RUnlock()

	if !exists {
		return false
	}

	upload.mu.RLock()
	defer upload.mu.RUnlock()

	return len(upload.Chunks) == upload.TotalChunks
}

// MergeChunks 合并分片
func (cm *ChunkManager) MergeChunks(uploadID string) ([]byte, string, error) {
	cm.mu.RLock()
	upload, exists := cm.uploads[uploadID]
	cm.mu.RUnlock()

	if !exists {
		return nil, "", fmt.Errorf("upload not found: %s", uploadID)
	}

	upload.mu.Lock()
	defer upload.mu.Unlock()

	// 检查是否所有分片都已上传
	if len(upload.Chunks) != upload.TotalChunks {
		return nil, "", fmt.Errorf("upload incomplete: %d/%d chunks", len(upload.Chunks), upload.TotalChunks)
	}

	// 合并分片
	result := make([]byte, 0, upload.TotalSize)
	for i := 0; i < upload.TotalChunks; i++ {
		chunkPath := filepath.Join(upload.TempDir, fmt.Sprintf("chunk_%d", i))
		data, err := os.ReadFile(chunkPath)
		if err != nil {
			return nil, "", err
		}
		result = append(result, data...)
	}

	return result, upload.FileName, nil
}

// CleanupUpload 清理上传临时文件
func (cm *ChunkManager) CleanupUpload(uploadID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	upload, exists := cm.uploads[uploadID]
	if !exists {
		return nil
	}

	// 删除临时目录
	if err := os.RemoveAll(upload.TempDir); err != nil {
		return err
	}

	delete(cm.uploads, uploadID)
	return nil
}

// ListUploads 列出所有上传
func (cm *ChunkManager) ListUploads() []map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make([]map[string]interface{}, 0, len(cm.uploads))
	for _, upload := range cm.uploads {
		upload.mu.RLock()
		result = append(result, map[string]interface{}{
			"upload_id":    upload.UploadID,
			"filename":     upload.FileName,
			"total_chunks": upload.TotalChunks,
			"uploaded":     len(upload.Chunks),
			"total_size":   upload.TotalSize,
		})
		upload.mu.RUnlock()
	}

	return result
}
