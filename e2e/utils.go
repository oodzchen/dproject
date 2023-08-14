package main

import (
	"math/rand"

	"github.com/brianvoe/gofakeit/v6"
)

type testUser struct {
	email string
	name  string
}

type testArticle struct {
	title   string
	content string
}

func getRandUser() *testUser {
	return &testUser{
		email: gofakeit.Email(),
		name:  gofakeit.Name(),
	}
}

// var chars []string = []string{"零", "一", "二", "三", "四", "五", "六", "七", "八", "九"}

// func genUserName(i int) string {
// 	if i < 10 {
// 		return chars[i]
// 	} else if i < 100 {
// 		return chars[(i/10)%10] + chars[i%10]
// 	} else {
// 		return chars[i/100] + chars[(i/10)%10] + chars[i%10]
// 	}
// }

func genArticle() *testArticle {
	return &testArticle{
		title:   gofakeit.Sentence(3 + rand.Intn(9)),
		content: gofakeit.Paragraph(2+rand.Intn(5), 1+rand.Intn(6), 100, "\n\n"),
	}
}
