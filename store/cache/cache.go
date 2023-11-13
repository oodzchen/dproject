package cache

import (
	"encoding/json"
	"fmt"

	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store/pgstore"
	"github.com/redis/go-redis/v9"
)

type ArticleCache struct {
	*pgstore.Article
	Rdb *redis.Client
}

func (ac *ArticleCache) List(page, pageSize, userId int, sortType model.ArticleSortType) ([]*model.Article, int, error) {
	return ac.Article.List(page, pageSize, userId, sortType)
}

func (ac *ArticleCache) SetList(page, pageSize, userId int, sortType model.ArticleSortType, list []*model.Article) error {
	//...
	jsonStr, err := json.Marshal(list)
	if err != nil {
		return err
	}
	fmt.Println("article_list json string:", jsonStr)
	return nil
}
