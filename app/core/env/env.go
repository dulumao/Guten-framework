package env

import (
	"github.com/BurntSushi/toml"
	"path/filepath"
)

type tomlConfig struct {
	Server    server
	Framework framework
	Session   session
	Database  database
	Cache     cache
}

type framework struct {
	TemplateCached bool     `toml:"template_cached"`
	TemplateDirs   []string `toml:"template_dirs"`
	AdminPath      string   `toml:"admin_path"`
}

type server struct {
	Debug    bool   `toml:"debug"`
	Addr     string `toml:"addr"`
	Timezone string `toml:"timezone"`
	LogLevel string `toml:"log_level"`
	LogFile  string `toml:"log_file"`
	HashKey  string `toml:"hashKey"`
}

type session struct {
	Driver string `toml:"driver"`
	Name   string `toml:"name"`
	// Encrypt  bool   `toml:"encrypt"`
	Path     string `toml:"path"`
	Lifetime int    `toml:"lifetime"`
	Secure   bool   `toml:"secure"`
	HTTPOnly bool   `toml:"http_only"`

	File struct {
		Path string `toml:"path"`
	}

	Redis struct {
		Addr     string `toml:"addr"`
		Password string `toml:"password"`
	}
}

type database struct {
	Driver  string `toml:"driver"`
	MaxOpen int    `toml:"max_open"`
	MaxIdle int    `toml:"max_idle"`
	Debug   bool   `toml:"debug"`

	Mysql struct {
		Host          string `toml:"host"`
		Port          int    `toml:"port"`
		Username      string `toml:"username"`
		Password      string `toml:"password"`
		Database      string `toml:"database"`
		Charset       string `toml:"charset"`
		ExplainEnable bool   `toml:"explain_enable"`
	}
	Sqlite3 struct {
		Database string `toml:"database"`
	}
}

type cache struct {
	Driver string `toml:"driver"`

	Memory struct {
		Interval int `toml:"interval"`
	}

	File struct {
		Path           string `toml:"path"`
		FileSuffix     string `toml:"file_suffix"`
		DirectoryLevel int    `toml:"directory_level"`
		EmbedExpiry    int    `toml:"embed_expiry"`
	}

	Redis struct {
		Key      string `toml:"key"`
		Addr     string `toml:"addr"`
		DbNumber int    `toml:"db_number"`
		Password string `toml:"password"`
	}

	Memcache struct {
		Addr string `toml:"addr"`
	}
}

var Value *tomlConfig

func New() (error) {
	filePath, err := filepath.Abs("web/env.toml")

	if err != nil {
		return err
	}

	if _, err := toml.DecodeFile(filePath, &Value); err != nil {
		return err
	}

	return nil
}
