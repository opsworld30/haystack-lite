package storage

import (
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"haystack-lite/internal/config"
)

type Store struct {
	config      *config.Config
	volumes     map[uint32]*Volume
	activeVolID uint32
	nextID      uint64
	db          *Database
	mu          sync.RWMutex
}

func NewStore(cfg *config.Config) (*Store, error) {
	if err := os.MkdirAll(cfg.Storage.DataDir, 0755); err != nil {
		return nil, err
	}

	// 连接数据库
	dsn := cfg.GetDatabaseDSN()
	db, err := NewDatabase(cfg.Database.Type, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	s := &Store{
		config:  cfg,
		volumes: make(map[uint32]*Volume),
		nextID:  1,
		db:      db,
	}

	if err := s.loadFromDatabase(); err != nil {
		return nil, err
	}

	if len(s.volumes) == 0 {
		if _, err := s.createNewVolume(); err != nil {
			return nil, err
		}
	}

	go s.syncLoop()

	return s, nil
}

func (s *Store) loadFromDatabase() error {
	// 加载 Volume 信息
	volumeInfos, err := s.db.LoadAllVolumeInfo()
	if err != nil {
		return fmt.Errorf("failed to load volume info: %w", err)
	}

	maxID := uint32(0)
	for _, info := range volumeInfos {
		vol, err := NewVolume(info.ID, s.config.Storage.DataDir, s.config.Storage.MaxVolumeSize)
		if err != nil {
			log.Printf("Warning: failed to open volume %d: %v", info.ID, err)
			continue
		}

		vol.CurrentSize = info.CurrentSize
		vol.Active = info.Active

		s.volumes[info.ID] = vol
		if info.Active && info.ID > maxID {
			maxID = info.ID
			s.activeVolID = info.ID
		}
	}

	allMetas, err := s.db.LoadAllFileMetadataIncludingDeleted()
	if err != nil {
		return fmt.Errorf("failed to load file metadata: %w", err)
	}

	activeMetas := 0
	for _, meta := range allMetas {
		if vol, exists := s.volumes[meta.VolumeID]; exists {
			vol.NeedleIndex[meta.ID] = &NeedleInfo{
				Offset:   meta.Offset,
				Size:     meta.Size,
				Flags:    meta.Flags,
				VolumeID: meta.VolumeID,
			}

			if !meta.Deleted {
				activeMetas++
			}
		}

		if meta.ID >= s.nextID {
			s.nextID = meta.ID + 1
		}
	}

	log.Printf("Loaded %d volumes and %d files (%d active, %d deleted) from database",
		len(s.volumes), len(allMetas), activeMetas, len(allMetas)-activeMetas)
	return nil
}

func (s *Store) createNewVolume() (*Volume, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	newID := s.activeVolID + 1
	vol, err := NewVolume(newID, s.config.Storage.DataDir, s.config.Storage.MaxVolumeSize)
	if err != nil {
		return nil, err
	}

	// 保存到数据库
	volumeInfo := &VolumeInfo{
		ID:          newID,
		FilePath:    vol.FilePath,
		MaxSize:     vol.MaxSize,
		CurrentSize: 0,
		Active:      true,
	}
	if err := s.db.SaveVolumeInfo(volumeInfo); err != nil {
		vol.Close()
		return nil, fmt.Errorf("failed to save volume info: %w", err)
	}

	s.volumes[newID] = vol
	s.activeVolID = newID

	log.Printf("Created new volume: %d", newID)
	return vol, nil
}

func (s *Store) Write(data []byte) (uint64, error) {
	return s.WriteWithMetadata(data, "", "")
}

func (s *Store) WriteWithMetadata(data []byte, filename, mimeType string) (uint64, error) {
	if s.config.Storage.ReadOnly {
		return 0, ErrReadOnly
	}

	id := atomic.AddUint64(&s.nextID, 1) - 1

	// 计算 MD5
	md5Hash := fmt.Sprintf("%x", md5.Sum(data))

	needle := &Needle{
		ID:         id,
		Cookie:     uint32(time.Now().Unix()),
		Data:       data,
		DataSize:   uint32(len(data)),
		Flags:      0,
		CreateTime: time.Now().Unix(),
		FileName:   filename,
		MimeType:   mimeType,
		MD5:        md5Hash,
	}

	s.mu.RLock()
	vol := s.volumes[s.activeVolID]
	volID := s.activeVolID
	s.mu.RUnlock()

	err := vol.WriteNeedle(needle)
	if err == ErrVolumeFull {
		// 设置当前 volume 为非活跃
		s.db.SetVolumeInactive(volID)

		vol, err = s.createNewVolume()
		if err != nil {
			return 0, err
		}
		volID = vol.ID
		err = vol.WriteNeedle(needle)
	}

	if err != nil {
		return 0, err
	}

	meta := &FileMetadata{
		ID:         id,
		VolumeID:   volID,
		Offset:     vol.NeedleIndex[id].Offset,
		Size:       needle.DataSize,
		Cookie:     needle.Cookie,
		Flags:      needle.Flags,
		Deleted:    false,
		FileName:   filename,
		MimeType:   mimeType,
		MD5:        md5Hash,
		CreateTime: needle.CreateTime,
	}
	if err := s.db.SaveFileMetadata(meta); err != nil {
		log.Printf("Warning: failed to save metadata to database: %v", err)
	}

	// 更新 Volume 大小
	s.db.UpdateVolumeSize(volID, vol.CurrentSize)

	return id, nil
}

func (s *Store) ReadWithMetadata(id uint64) ([]byte, *FileMetadata, error) {
	// 从数据库获取元数据
	meta, err := s.db.GetFileMetadata(id)
	if err != nil {
		return nil, nil, ErrNeedleNotFound
	}

	// 读取文件数据
	data, err := s.Read(id)
	if err != nil {
		return nil, nil, err
	}

	return data, meta, nil
}

func (s *Store) GetMetadata(id uint64) (*FileMetadata, error) {
	return s.db.GetFileMetadata(id)
}

func (s *Store) Read(id uint64) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, vol := range s.volumes {
		needle, err := vol.ReadNeedle(id)
		if err == nil {
			return needle.Data, nil
		}
	}

	return nil, ErrNeedleNotFound
}

func (s *Store) Delete(id uint64) error {
	if s.config.Storage.ReadOnly {
		return ErrReadOnly
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, vol := range s.volumes {
		if err := vol.DeleteNeedle(id); err == nil {
			// 更新数据库
			if err := s.db.DeleteFileMetadata(id); err != nil {
				log.Printf("Warning: failed to delete metadata from database: %v", err)
			}
			return nil
		}
	}

	return ErrNeedleNotFound
}

func (s *Store) Status() map[string]interface{} {
	// 从数据库获取统计信息
	stats, err := s.db.GetStats()
	if err != nil {
		log.Printf("Warning: failed to get stats from database: %v", err)
		// 降级到内存统计
		return s.getMemoryStats()
	}

	stats["active_volume"] = s.activeVolID
	stats["next_id"] = s.nextID
	return stats
}

func (s *Store) getMemoryStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalFiles := 0
	totalSize := int64(0)
	deletedFiles := 0

	for _, vol := range s.volumes {
		vol.mu.RLock()
		for _, info := range vol.NeedleIndex {
			totalFiles++
			totalSize += int64(info.Size)
			if info.Flags&0x01 != 0 {
				deletedFiles++
			}
		}
		vol.mu.RUnlock()
	}

	return map[string]interface{}{
		"total_files":   totalFiles,
		"deleted_files": deletedFiles,
		"active_files":  totalFiles - deletedFiles,
		"total_size":    totalSize,
		"volume_count":  len(s.volumes),
		"active_volume": s.activeVolID,
		"next_id":       s.nextID,
	}
}

func (s *Store) syncLoop() {
	ticker := time.NewTicker(time.Duration(s.config.Storage.SyncInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()
		for _, vol := range s.volumes {
			vol.Sync()
		}
		s.mu.RUnlock()
	}
}

func (s *Store) ListAll() ([]*FileMetadata, error) {
	return s.db.LoadAllFileMetadata()
}

func (s *Store) FindByFilename(filename string) (*FileMetadata, error) {
	return s.db.FindByFilename(filename)
}

func (s *Store) ListByPrefix(prefix string, limit int) ([]*FileMetadata, error) {
	return s.db.ListByPrefix(prefix, limit)
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, vol := range s.volumes {
		if err := vol.Close(); err != nil {
			return err
		}
	}

	if s.db != nil {
		return s.db.Close()
	}

	return nil
}
