package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store/pgstore"
	"github.com/redis/go-redis/v9"
)

type ArticleCache struct {
	*pgstore.Article
	Rdb *redis.Client
}

type CachedList struct {
	Total int              `json:"total"`
	List  []*model.Article `json:"list"`
}

func (ac *ArticleCache) List(page, pageSize int, sortType model.ArticleSortType) ([]*model.Article, int, error) {
	cachedList, total, err := ac.getList(page, pageSize, sortType)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			list, total, err := ac.Article.List(page, pageSize, sortType)
			if err != nil {
				return nil, 0, err
			}
			// go func() {
			// 	err := ac.setList(page, pageSize, userId, sortType, list, total)
			// 	if err != nil {
			// 		fmt.Println("redis cache article list error:", err)
			// 	}
			// }()

			return list, total, nil
		}
		return nil, 0, err
	}

	return cachedList, total, nil
}

func (ac *ArticleCache) Create(title, url, content string, authorId, replyTo int) (int, error) {
	// defer func() {
	// 	// fmt.Println("refresh cache list...")
	// 	go logErr("refresh list cache error:", ac.refreshListCache())
	// }()
	// total, _ := ac.Article.Count()
	// fmt.Println("total before create:", total)
	return ac.Article.Create(title, url, content, authorId, replyTo)
}

// Set list data to redis
func (ac *ArticleCache) setList(page, pageSize, userId int, sortType model.ArticleSortType, list []*model.Article, total int) error {
	jsonStr, err := json.Marshal(&CachedList{
		List:  list,
		Total: total,
	})

	if err != nil {
		return err
	}

	// jsonStr, err = json.MarshalIndent(list, "", "  ")
	// if err != nil {
	// 	return err
	// }

	// fmt.Println("article_list json string:", string(jsonStr))
	err = ac.Rdb.Set(context.Background(), fmt.Sprintf("article_list?page=%d&pageSize=%d&userId=%d&sortType=%s", page, pageSize, userId, string(sortType)), string(jsonStr), 30*24*time.Hour).Err()
	if err != nil {
		return err
	}

	return nil
}

// Get list data from redis
func (ac *ArticleCache) getList(page, pageSize int, sortType model.ArticleSortType) ([]*model.Article, int, error) {
	str, err := ac.Rdb.Get(context.Background(), fmt.Sprintf("article_list?page=%d&pageSize=%d&sortType=%s", page, pageSize, string(sortType))).Result()
	if err != nil {
		return nil, 0, err
	}

	var listData CachedList
	err = json.Unmarshal([]byte(str), &listData)
	if err != nil {
		return nil, 0, err
	}

	// fmt.Printf("cached list: %#v\n", listData)

	return listData.List, listData.Total, nil
}

const maxCachedPageNum = 10

func (ac *ArticleCache) refreshListCache() error {
	total, err := ac.Article.Count()
	if err != nil {
		return err
	}
	fmt.Println("total on cache:", total)

	totalPage := int(math.Ceil(float64(total) / float64(pgstore.DefaultPageSize)))
	maxPage := maxCachedPageNum
	if totalPage < maxCachedPageNum {
		maxPage = totalPage
	}

	sortTypes := []model.ArticleSortType{model.ListSortBest, model.ListSortLatest, model.ListSortHot}
	for _, sortType := range sortTypes {
		fmt.Println("cache list sort type:", string(sortType))
		for i := 1; i <= maxPage; i++ {
			fmt.Println("cache list page:", i)
			list, total, err := ac.Article.List(i, pgstore.DefaultPageSize, sortType)
			// fmt.Println("total from db:", total)
			if err != nil {
				return err
			}

			err = ac.setList(i, pgstore.DefaultPageSize, 0, sortType, list, total)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func logErr(msg string, err error) {
	if err != nil {
		fmt.Println(msg, err)
	}
}
