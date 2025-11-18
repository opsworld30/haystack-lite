package storage

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// CompactionConfig 压缩配置
type CompactionConfig struct {
	Enabled          bool    // 是否启用
	Interval         int     // 压缩间隔（秒）
	DeletedThreshold float64 // 删除文件比例阈值（0-1）
	MinVolumeSize    int64   // 最小压缩 Volume 大小
}

// StartCompaction 启动后台压缩
func (s *Store) StartCompaction(cfg CompactionConfig) {
	if !cfg.Enabled {
		log.Println("Compaction disabled")
		return
	}

	go func() {
		ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
		defer ticker.Stop()

		log.Printf("Compaction started, interval: %d seconds", cfg.Interval)

		for range ticker.C {
			if err := s.runCompaction(cfg); err != nil {
				log.Printf("Compaction error: %v", err)
			}
		}
	}()
}

func (s *Store) runCompaction(cfg CompactionConfig) error {
	s.mu.RLock()
	volumes := make([]*Volume, 0, len(s.volumes))
	for _, vol := range s.volumes {
		if vol.CurrentSize >= cfg.MinVolumeSize {
			volumes = append(volumes, vol)
		}
	}
	s.mu.RUnlock()

	for _, vol := range volumes {
		if err := s.compactVolume(vol, cfg); err != nil {
			log.Printf("Failed to compact volume %d: %v", vol.ID, err)
		}
	}

	return nil
}

func (s *Store) RunCompactionNow() error {
	cfg := CompactionConfig{
		Enabled:          true,
		DeletedThreshold: 0.3,
		MinVolumeSize:    1048576,
	}
	return s.runCompaction(cfg)
}

// compactVolume 压缩单个 Volume
func (s *Store) compactVolume(vol *Volume, cfg CompactionConfig) error {
	vol.mu.RLock()
	totalFiles := len(vol.NeedleIndex)
	deletedFiles := 0
	for _, info := range vol.NeedleIndex {
		if info.Flags&0x01 != 0 {
			deletedFiles++
		}
	}
	vol.mu.RUnlock()

	if totalFiles == 0 {
		return nil
	}

	deletedRatio := float64(deletedFiles) / float64(totalFiles)
	if deletedRatio < cfg.DeletedThreshold {
		return nil
	}

	log.Printf("Compacting volume %d: %d/%d files deleted (%.2f%%)",
		vol.ID, deletedFiles, totalFiles, deletedRatio*100)

	// 创建临时 Volume
	tempID := vol.ID + 10000
	tempVol, err := NewVolume(tempID, s.config.Storage.DataDir, s.config.Storage.MaxVolumeSize)
	if err != nil {
		return fmt.Errorf("failed to create temp volume: %w", err)
	}
	defer tempVol.Close()

	// 复制未删除的文件
	vol.mu.RLock()
	needles := make([]uint64, 0, totalFiles-deletedFiles)
	for id, info := range vol.NeedleIndex {
		if info.Flags&0x01 == 0 {
			needles = append(needles, id)
		}
	}
	vol.mu.RUnlock()

	copiedCount := 0
	for _, id := range needles {
		needle, err := vol.ReadNeedle(id)
		if err != nil {
			log.Printf("Failed to read needle %d: %v", id, err)
			continue
		}

		if err := tempVol.WriteNeedle(needle); err != nil {
			log.Printf("Failed to write needle %d: %v", id, err)
			continue
		}

		copiedCount++
	}

	// 关闭原 Volume
	vol.mu.Lock()
	vol.File.Close()
	vol.mu.Unlock()

	// 重命名文件
	oldPath := vol.FilePath
	newPath := filepath.Join(s.config.Storage.DataDir, fmt.Sprintf("volume_%05d.dat", vol.ID))
	tempPath := tempVol.FilePath

	// 删除旧文件
	if err := os.Remove(oldPath); err != nil {
		log.Printf("Failed to remove old volume: %v", err)
	}

	// 重命名临时文件
	tempVol.File.Close()
	if err := os.Rename(tempPath, newPath); err != nil {
		return fmt.Errorf("failed to rename temp volume: %w", err)
	}

	// 重新打开 Volume
	newVol, err := NewVolume(vol.ID, s.config.Storage.DataDir, s.config.Storage.MaxVolumeSize)
	if err != nil {
		return fmt.Errorf("failed to reopen volume: %w", err)
	}

	// 重建索引
	if err := newVol.LoadIndex(); err != nil {
		newVol.Close()
		return fmt.Errorf("failed to load index: %w", err)
	}

	// 更新 Store
	s.mu.Lock()
	s.volumes[vol.ID] = newVol
	s.mu.Unlock()

	// 更新数据库
	s.db.UpdateVolumeSize(vol.ID, newVol.CurrentSize)

	log.Printf("Compaction completed for volume %d: %d files copied, saved %.2f MB",
		vol.ID, copiedCount, float64(vol.CurrentSize-newVol.CurrentSize)/(1024*1024))

	return nil
}

// GetCompactionStats 获取压缩统计信息
func (s *Store) GetCompactionStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalFiles := 0
	deletedFiles := 0
	totalSize := int64(0)
	wastedSize := int64(0)

	for _, vol := range s.volumes {
		vol.mu.RLock()
		for _, info := range vol.NeedleIndex {
			totalFiles++
			size := int64(info.Size)
			totalSize += size
			if info.Flags&0x01 != 0 {
				deletedFiles++
				wastedSize += size
			}
		}
		vol.mu.RUnlock()
	}

	wastedRatio := 0.0
	if totalSize > 0 {
		wastedRatio = float64(wastedSize) / float64(totalSize)
	}

	return map[string]interface{}{
		"total_files":   totalFiles,
		"deleted_files": deletedFiles,
		"total_size":    totalSize,
		"wasted_size":   wastedSize,
		"wasted_ratio":  wastedRatio,
	}
}
