package cache

import "github.com/redis/go-redis/v9"

type Cache struct {
	rdb *redis.Client
}

type ArticleCache struct {
	Cache
}

func (ac *ArticleCache) List() {
	//...
}
