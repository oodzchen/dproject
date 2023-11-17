package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
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
	Total int
	List  []*model.Article
}

type ListType string

const (
	ListTypeHome ListType = "home"
	ListTypeTree          = "tree"
)

func (ac *ArticleCache) List(page, pageSize int, sortType model.ArticleSortType, categoryFrontId string) ([]*model.Article, int, error) {
	params := map[string]string{
		"sortType": string(sortType),
		"page":     strconv.Itoa(page),
		"pageSize": strconv.Itoa(pageSize),
	}
	cachedList,
		total, err := ac.getList(ListTypeHome, params)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			list, total, err := ac.Article.List(page, pageSize, sortType, categoryFrontId)
			if err != nil {
				return nil, 0, err
			}
			go func() {
				err := ac.setList(ListTypeHome, params, list, total)
				if err != nil {
					fmt.Println("redis cache article list error:", err)
				}
			}()

			return list, total, nil
		}
		return nil, 0, err
	}

	return cachedList, total, nil
}

// func (ac *ArticleCache) ItemTree(id int, sortType model.ArticleSortType) ([]*model.Article, error) {
// 	params := map[string]string{
// 		"sortType": string(sortType),
// 		"id":       strconv.Itoa(id),
// 	}
// 	cachedList, _, err := ac.getList(ListTypeTree, params)
// 	// fmt.Println("cached tree list len:", len(cachedList))
// 	if err != nil {
// 		if errors.Is(err, redis.Nil) {
// 			list, err := ac.Article.ItemTree(id, sortType)
// 			if err != nil {
// 				return nil, err
// 			}
// 			go func() {
// 				err := ac.setList(ListTypeTree, params, list, len(list))
// 				if err != nil {
// 					fmt.Println("redis cache article tree list error:", err)
// 				}
// 			}()

// 			return list, nil
// 		}
// 		return nil, err
// 	}

// 	return cachedList, nil
// }

func paramsToStr(params map[string]string) string {
	var paramsArr []string
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	// fmt.Println("sorted keys:", keys)

	for _, key := range keys {
		paramsArr = append(paramsArr, fmt.Sprintf("%s=%s", key, params[key]))
	}

	return strings.Join(paramsArr, "&")
}

// Set list data to redis
func (ac *ArticleCache) setList(listType ListType, params map[string]string, list []*model.Article, total int) error {
	jsonStr, err := json.Marshal(&CachedList{
		List:  list,
		Total: total,
	})

	if err != nil {
		return err
	}

	err = ac.Rdb.Set(
		context.Background(),
		fmt.Sprintf("article_list:%s?%s", string(listType), paramsToStr(params)),
		string(jsonStr),
		30*24*time.Hour,
	).Err()
	if err != nil {
		return err
	}

	return nil
}

// Get list data from redis
func (ac *ArticleCache) getList(listType ListType, params map[string]string) ([]*model.Article, int, error) {
	str, err := ac.Rdb.Get(
		context.Background(),
		fmt.Sprintf("article_list:%s?%s", string(listType), paramsToStr(params)),
	).Result()
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

const maxCachedPageNum = 1

func (ac *ArticleCache) RefreshListCache() error {
	total, err := ac.Article.Count()
	if err != nil {
		return err
	}
	// fmt.Println("total on cache:", total)

	totalPage := int(math.Ceil(float64(total) / float64(pgstore.DefaultPageSize)))
	maxPage := maxCachedPageNum
	if totalPage < maxCachedPageNum {
		maxPage = totalPage
	}

	sortTypes := []model.ArticleSortType{model.ListSortBest, model.ListSortLatest, model.ListSortHot}
	for _, sortType := range sortTypes {
		// fmt.Println("cache list sort type:", string(sortType))
		for i := 1; i <= maxPage; i++ {
			// fmt.Println("cache list page:", i)
			list, total, err := ac.Article.List(i, pgstore.DefaultPageSize, sortType, "")
			// fmt.Println("total from db:", total)
			if err != nil {
				return err
			}

			params := map[string]string{
				"sortType": string(sortType),
				"page":     strconv.Itoa(i),
				"pageSize": strconv.Itoa(pgstore.DefaultPageSize),
			}

			err = ac.setList(ListTypeHome, params, list, total)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// func (ac *ArticleCache) RefreshArticleTreeCache(id int) error {
// 	sortTypes := []model.ArticleSortType{model.ReplySortBest, model.ListSortLatest}
// 	for _, sortType := range sortTypes {
// 		list, err := ac.Article.ItemTree(id, sortType)
// 		if err != nil {
// 			return err
// 		}

// 		params := map[string]string{
// 			"sortType": string(sortType),
// 			"id":       strconv.Itoa(id),
// 		}

// 		ac.setList(ListTypeTree, params, list, len(list))
// 	}

// 	return nil
// }

func logErr(msg string, err error) {
	if err != nil {
		fmt.Println(msg, err)
	}
}
