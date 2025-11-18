package storage

import "time"

// FileMetadata 文件元数据表
type FileMetadata struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement:false"`
	VolumeID   uint32    `gorm:"index"`
	Offset     int64     `gorm:"not null"`
	Size       uint32    `gorm:"not null"`
	Cookie     uint32    `gorm:"not null"`
	Flags      uint8     `gorm:"default:0"`
	Deleted    bool      `gorm:"default:false;index"`
	FileName   string    `gorm:"size:255;index"`
	MimeType   string    `gorm:"size:100"`
	MD5        string    `gorm:"size:32;index"`
	CreateTime int64     `gorm:"not null"`
	UpdateTime time.Time `gorm:"autoUpdateTime"`
}

func (FileMetadata) TableName() string {
	return "file_metadata"
}

// VolumeInfo Volume 信息表
type VolumeInfo struct {
	ID          uint32    `gorm:"primaryKey;autoIncrement:false"`
	FilePath    string    `gorm:"size:255;not null"`
	MaxSize     int64     `gorm:"not null"`
	CurrentSize int64     `gorm:"default:0"`
	Active      bool      `gorm:"default:true;index"`
	CreateTime  time.Time `gorm:"autoCreateTime"`
	UpdateTime  time.Time `gorm:"autoUpdateTime"`
}

func (VolumeInfo) TableName() string {
	return "volume_info"
}
