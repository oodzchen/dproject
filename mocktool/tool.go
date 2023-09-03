package mocktool

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/brianvoe/gofakeit/v6"
)

type TestUser struct {
	Email string
	Name  string
}

type TestArticle struct {
	Title   string
	Content string
}

func GenUser() *TestUser {
	return &TestUser{
		Email: gofakeit.Email(),
		Name:  gofakeit.Name(),
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

func GenArticle() *TestArticle {
	title := gofakeit.Sentence(3 + rand.Intn(9))
	content := gofakeit.Paragraph(1+rand.Intn(3), 1+rand.Intn(3), 30, "\n\n")
	if len(title) > 80 {
		title = title[:80]
	}

	if len(content) > 20000 {
		content = content[:20000]
	}

	return &TestArticle{
		Title:   title,
		Content: content,
	}
}

func LogFailed(err error) {
	LogErrf("FAILED: %v", err)
}

func LogErrf(msg string, err error) {
	if err != nil {
		log.Fatalf(msg, err)
	}
}

func Logln(data ...any) {
	fmt.Println(data...)
}
