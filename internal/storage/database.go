package storage

import (
	"fmt"
	"log"

	"haystack-lite/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	db *gorm.DB
}

func NewDatabase(dbType config.DatabaseType, dsn string) (*Database, error) {
	var dialector gorm.Dialector

	switch dbType {
	case config.DatabaseMySQL:
		dialector = mysql.Open(dsn)
		log.Printf("Connecting to MySQL: %s", maskPassword(dsn))
	case config.DatabaseSQLite:
		dialector = sqlite.Open(dsn)
		log.Printf("Connecting to SQLite: %s", dsn)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// 自动迁移表结构
	if err := db.AutoMigrate(&FileMetadata{}, &VolumeInfo{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Printf("Database (%s) connected and migrated successfully", dbType)

	return &Database{db: db}, nil
}

// maskPassword 隐藏 DSN 中的密码
func maskPassword(dsn string) string {
	// 简单实现，仅用于日志显示
	if len(dsn) > 20 {
		return dsn[:10] + "***" + dsn[len(dsn)-10:]
	}
	return "***"
}

// SaveFileMetadata 保存文件元数据
func (d *Database) SaveFileMetadata(meta *FileMetadata) error {
	return d.db.Create(meta).Error
}

// GetFileMetadata 获取文件元数据
func (d *Database) GetFileMetadata(id uint64) (*FileMetadata, error) {
	var meta FileMetadata
	err := d.db.Where("id = ? AND deleted = ?", id, false).First(&meta).Error
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

// DeleteFileMetadata 逻辑删除文件
func (d *Database) DeleteFileMetadata(id uint64) error {
	return d.db.Model(&FileMetadata{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted": true,
			"flags":   1,
		}).Error
}

func (d *Database) LoadAllFileMetadata() ([]*FileMetadata, error) {
	var metas []*FileMetadata
	err := d.db.Where("deleted = ?", false).Find(&metas).Error
	return metas, err
}

func (d *Database) LoadAllFileMetadataIncludingDeleted() ([]*FileMetadata, error) {
	var metas []*FileMetadata
	err := d.db.Find(&metas).Error
	return metas, err
}

// SaveVolumeInfo 保存 Volume 信息
func (d *Database) SaveVolumeInfo(info *VolumeInfo) error {
	return d.db.Save(info).Error
}

// GetVolumeInfo 获取 Volume 信息
func (d *Database) GetVolumeInfo(id uint32) (*VolumeInfo, error) {
	var info VolumeInfo
	err := d.db.First(&info, id).Error
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// LoadAllVolumeInfo 加载所有 Volume 信息
func (d *Database) LoadAllVolumeInfo() ([]VolumeInfo, error) {
	var infos []VolumeInfo
	err := d.db.Find(&infos).Error
	return infos, err
}

// UpdateVolumeSize 更新 Volume 大小
func (d *Database) UpdateVolumeSize(id uint32, size int64) error {
	return d.db.Model(&VolumeInfo{}).
		Where("id = ?", id).
		Update("current_size", size).Error
}

// SetVolumeInactive 设置 Volume 为非活跃
func (d *Database) SetVolumeInactive(id uint32) error {
	return d.db.Model(&VolumeInfo{}).
		Where("id = ?", id).
		Update("active", false).Error
}

// GetStats 获取统计信息
func (d *Database) GetStats() (map[string]interface{}, error) {
	var totalFiles int64
	var deletedFiles int64
	var totalSize int64

	d.db.Model(&FileMetadata{}).Count(&totalFiles)
	d.db.Model(&FileMetadata{}).Where("deleted = ?", true).Count(&deletedFiles)
	d.db.Model(&FileMetadata{}).Select("COALESCE(SUM(size), 0)").Scan(&totalSize)

	var volumeCount int64
	d.db.Model(&VolumeInfo{}).Count(&volumeCount)

	return map[string]interface{}{
		"total_files":   totalFiles,
		"deleted_files": deletedFiles,
		"active_files":  totalFiles - deletedFiles,
		"total_size":    totalSize,
		"volume_count":  volumeCount,
	}, nil
}

func (d *Database) FindByFilename(filename string) (*FileMetadata, error) {
	var meta FileMetadata
	err := d.db.Where("file_name = ? AND deleted = ?", filename, false).First(&meta).Error
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func (d *Database) ListByPrefix(prefix string, limit int) ([]*FileMetadata, error) {
	var metas []*FileMetadata
	query := d.db.Where("file_name LIKE ? AND deleted = ?", prefix+"%", false)
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&metas).Error
	return metas, err
}

func (d *Database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
