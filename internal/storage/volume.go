package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Volume struct {
	ID          uint32
	File        *os.File
	FilePath    string
	MaxSize     int64
	CurrentSize int64
	Active      bool
	NeedleIndex map[uint64]*NeedleInfo
	mu          sync.RWMutex
}

func NewVolume(id uint32, dataDir string, maxSize int64) (*Volume, error) {
	filePath := filepath.Join(dataDir, fmt.Sprintf("volume_%05d.dat", id))

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	v := &Volume{
		ID:          id,
		File:        file,
		FilePath:    filePath,
		MaxSize:     maxSize,
		CurrentSize: stat.Size(),
		Active:      true,
		NeedleIndex: make(map[uint64]*NeedleInfo),
	}

	return v, nil
}

func (v *Volume) WriteNeedle(n *Needle) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.CurrentSize+n.Size() > v.MaxSize {
		v.Active = false
		return ErrVolumeFull
	}

	offset := v.CurrentSize

	if _, err := v.File.Seek(offset, 0); err != nil {
		return err
	}

	if err := n.Write(v.File); err != nil {
		return err
	}

	v.NeedleIndex[n.ID] = &NeedleInfo{
		Offset:   offset,
		Size:     n.DataSize,
		Flags:    n.Flags,
		VolumeID: v.ID,
	}

	v.CurrentSize += n.Size()
	return nil
}

func (v *Volume) ReadNeedle(id uint64) (*Needle, error) {
	v.mu.RLock()
	info, exists := v.NeedleIndex[id]
	v.mu.RUnlock()

	if !exists {
		return nil, ErrNeedleNotFound
	}

	if info.Flags&0x01 != 0 {
		return nil, ErrNeedleNotFound
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	if _, err := v.File.Seek(info.Offset, 0); err != nil {
		return nil, err
	}

	return ReadNeedleFrom(v.File)
}

func (v *Volume) DeleteNeedle(id uint64) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	info, exists := v.NeedleIndex[id]
	if !exists {
		return ErrNeedleNotFound
	}

	info.Flags |= 0x01
	return nil
}

func (v *Volume) LoadIndex() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if _, err := v.File.Seek(0, 0); err != nil {
		return err
	}

	offset := int64(0)
	for {
		n, err := ReadNeedleFrom(v.File)
		if err != nil {
			break
		}

		v.NeedleIndex[n.ID] = &NeedleInfo{
			Offset:   offset,
			Size:     n.DataSize,
			Flags:    n.Flags,
			VolumeID: v.ID,
		}

		offset += n.Size()
	}

	v.CurrentSize = offset
	return nil
}

func (v *Volume) Sync() error {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.File.Sync()
}

func (v *Volume) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.File.Close()
}
