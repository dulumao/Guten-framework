package cache

import (
	"github.com/dulumao/Guten-framework/app/core/env"
	"github.com/dulumao/Guten-utils/conv"
	"github.com/dulumao/Guten-utils/os/cache"
)

var Cache cache.Cache

func New() {
	var cacheConfig = ``

	// memory
	if env.Value.Cache.Driver == "memory" {
		cacheConfig = `{"interval":` + conv.String(env.Value.Cache.Memory.Interval) + `}`
	}

	// file
	if env.Value.Cache.Driver == "file" {
		cacheConfig = `{"CachePath":"` + env.Value.Cache.File.Path + `","FileSuffix":"` + env.Value.Cache.File.FileSuffix + `","DirectoryLevel":` + conv.String(env.Value.Cache.File.DirectoryLevel) + `,"EmbedExpiry":` + conv.String(env.Value.Cache.File.EmbedExpiry) + `}`
	}

	// redis
	if env.Value.Cache.Driver == "redis" {
		cacheConfig = `{"key":` + env.Value.Cache.Redis.Key + `,"conn":` + env.Value.Cache.Redis.Addr + `,"dbNum":"` + conv.String(env.Value.Cache.Redis.DbNumber) + `","password":` + env.Value.Cache.Redis.Password + `}`
	}

	// memcache
	if env.Value.Cache.Driver == "memcache" {
		cacheConfig = `{"conn":` + env.Value.Cache.Memcache.Addr + `}`
	}

	if adapter, err := cache.NewCache(env.Value.Cache.Driver, cacheConfig); err == nil {
		Cache = adapter
	}
}
