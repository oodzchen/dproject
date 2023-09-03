package main

import (
	"fmt"

	"github.com/oodzchen/dproject/config"
	"github.com/oodzchen/dproject/mocktool"
	"github.com/oodzchen/dproject/service"
)

func replyArticle(userSrv *service.User, articleSrv *service.Article) {
	var userIdPool []int
	failedNum := 0
	for i := 0; i < userNum; i++ {
		u := mocktool.GenUser()
		id, err := register(userSrv, u)
		if err != nil {
			failedNum += 1
			fmt.Println("register user error: ", err)
		} else {
			userIdPool = append(userIdPool, id)
			fmt.Println("register user success: ", id)
		}
	}
	fmt.Println("register users complete: success: ", userNum-failedNum, "failed: ", failedNum)

	level := 0
	rnr := 0
	tempId := replyToId
	failedReplyNum := 0
	var replyQueue []int
	for i := 0; i < replyNum; i++ {
		uId := userIdPool[i%len(userIdPool)]
		a := mocktool.GenArticle()
		id, err := createReply(articleSrv, a, uId, tempId)
		if err != nil {
			failedReplyNum += 1
			fmt.Println("create reply error: ", err)
		} else {
			fmt.Println("create reply success: ", id)
			if level > 0 && rnr < replyNumOfReply {
				replyQueue = append(replyQueue, id)
				rnr += 1
				continue
			} else {
				rnr = 0
			}

			if level < maxLevel {
				level += 1
				if len(replyQueue) > 0 {
					tempId = replyQueue[0]
					replyQueue = replyQueue[1:]
				} else {
					tempId = id
				}
			} else {
				level = 0
				tempId = replyToId
			}
		}
	}
	fmt.Println("create replies complete: success: ", replyNum-failedReplyNum, "failed: ", failedReplyNum)
}

func register(srv *service.User, u *mocktool.TestUser) (int, error) {
	return srv.Register(u.Email, config.Config.DB.UserDefaultPassword, u.Name)
}

func createReply(srv *service.Article, a *mocktool.TestArticle, authorId, target int) (int, error) {
	return srv.Reply(target, a.Content, authorId)
}
