package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type DatabaseType string

const (
	DatabaseMySQL  DatabaseType = "mysql"
	DatabaseSQLite DatabaseType = "sqlite"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Storage    StorageConfig    `yaml:"storage"`
	Database   DatabaseConfig   `yaml:"database"`
	Compaction CompactionConfig `yaml:"compaction"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type StorageConfig struct {
	DataDir       string `yaml:"data_dir"`
	MaxVolumeSize int64  `yaml:"max_volume_size"`
	VolumeFileExt string `yaml:"volume_file_ext"`
	SyncInterval  int    `yaml:"sync_interval"`
	ReadOnly      bool   `yaml:"read_only"`
}

type CompactionConfig struct {
	Enabled          bool    `yaml:"enabled"`
	Interval         int     `yaml:"interval"`
	DeletedThreshold float64 `yaml:"deleted_threshold"`
	MinVolumeSize    int64   `yaml:"min_volume_size"`
}

type DatabaseConfig struct {
	Type   DatabaseType `yaml:"type"`
	SQLite SQLiteConfig `yaml:"sqlite"`
	MySQL  MySQLConfig  `yaml:"mysql"`
}

type SQLiteConfig struct {
	Path string `yaml:"path"`
}

type MySQLConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	User      string `yaml:"user"`
	Password  string `yaml:"password"`
	Database  string `yaml:"database"`
	Charset   string `yaml:"charset"`
	ParseTime bool   `yaml:"parse_time"`
	Loc       string `yaml:"loc"`
}

// LoadConfig 从 YAML 文件加载配置
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// GetDatabaseDSN 根据配置生成数据库 DSN
func (c *Config) GetDatabaseDSN() string {
	switch c.Database.Type {
	case DatabaseSQLite:
		return c.Database.SQLite.Path
	case DatabaseMySQL:
		mysql := c.Database.MySQL
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
			mysql.User,
			mysql.Password,
			mysql.Host,
			mysql.Port,
			mysql.Database,
			mysql.Charset,
			mysql.ParseTime,
			mysql.Loc,
		)
	default:
		return ""
	}
}

// Default 返回默认配置
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port: ":8080",
		},
		Storage: StorageConfig{
			DataDir:       "./data",
			MaxVolumeSize: 1 << 30,
			VolumeFileExt: ".dat",
			SyncInterval:  60,
			ReadOnly:      false,
		},
		Compaction: CompactionConfig{
			Enabled:          true,
			Interval:         3600,
			DeletedThreshold: 0.3,
			MinVolumeSize:    10485760,
		},
		Database: DatabaseConfig{
			Type: DatabaseSQLite,
			SQLite: SQLiteConfig{
				Path: "./data/haystack.db",
			},
			MySQL: MySQLConfig{
				Host:      "127.0.0.1",
				Port:      3306,
				User:      "root",
				Password:  "password",
				Database:  "haystack",
				Charset:   "utf8mb4",
				ParseTime: true,
				Loc:       "Local",
			},
		},
	}
}
