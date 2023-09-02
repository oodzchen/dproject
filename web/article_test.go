package web

import (
	"reflect"
	"testing"

	"github.com/oodzchen/dproject/model"
)

func TestGenArticleTree(t *testing.T) {
	tests := []struct {
		desc    string
		root    *model.Article
		replies []*model.Article
		want    *model.Article
	}{
		{
			"normal article tree",
			&model.Article{
				Id:      1,
				ReplyTo: 0,
			},
			[]*model.Article{
				{
					Id:      3,
					ReplyTo: 2,
				},
				{
					Id:      2,
					ReplyTo: 1,
				},
				{
					Id:      5,
					ReplyTo: 4,
				},
				{
					Id:      4,
					ReplyTo: 2,
				},
			},
			&model.Article{
				Id:      1,
				ReplyTo: 0,
				Replies: &model.ArticleList{
					List: []*model.Article{
						{
							Id:      2,
							ReplyTo: 1,
							Replies: &model.ArticleList{
								List: []*model.Article{
									{
										Id:      3,
										ReplyTo: 2,
									},
									{
										Id:      4,
										ReplyTo: 2,
										Replies: &model.ArticleList{
											List: []*model.Article{
												{
													Id:      5,
													ReplyTo: 4,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			"article with no reply",
			&model.Article{
				Id:      1,
				ReplyTo: 0,
			},
			[]*model.Article{
				{
					Id:      3,
					ReplyTo: 2,
				},
				{
					Id:      2,
					ReplyTo: 0,
				},
				{
					Id:      5,
					ReplyTo: 4,
				},
				{
					Id:      4,
					ReplyTo: 2,
				},
			},
			&model.Article{
				Id:      1,
				ReplyTo: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, _ := genArticleTree(tt.root, tt.replies)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("want\n%+v, but got\n%+v", tt.want, got)
			}
		})
	}
}
