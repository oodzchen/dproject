package web

import (
	"reflect"
	"testing"

	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/utils"
)

func TestGenArticleTree(t *testing.T) {
	root := &model.Article{
		Id:            1,
		ReplyTo:       0,
		ChildrenCount: 1,
	}
	tests := []struct {
		desc    string
		root    *model.Article
		replies []*model.Article
		want    *model.Article
	}{
		{
			"normal article tree",
			root,
			[]*model.Article{
				root,
				{
					Id:            3,
					ReplyTo:       2,
					ChildrenCount: 0,
				},
				{
					Id:            2,
					ReplyTo:       1,
					ChildrenCount: 2,
				},
				{
					Id:            5,
					ReplyTo:       4,
					ChildrenCount: 0,
				},
				{
					Id:            4,
					ReplyTo:       2,
					ChildrenCount: 1,
				},
			},
			&model.Article{
				Id:            1,
				ReplyTo:       0,
				ChildrenCount: 1,
				Replies: model.NewArticleList(
					[]*model.Article{
						{
							Id:            2,
							ReplyTo:       1,
							ChildrenCount: 2,
							Replies: model.NewArticleList(
								[]*model.Article{
									{
										Id:            3,
										ReplyTo:       2,
										ChildrenCount: 0,
									},
									{
										Id:            4,
										ReplyTo:       2,
										ChildrenCount: 1,
										Replies: model.NewArticleList(
											[]*model.Article{
												{
													Id:      5,
													ReplyTo: 4,
												},
											},
											model.ReplySortBest,
											1,
											DefaultReplyPageSize,
											1,
										),
									},
								},
								model.ReplySortBest,
								1,
								DefaultReplyPageSize,
								2,
							),
						},
					},
					model.ReplySortBest,
					1,
					DefaultReplyPageSize,
					1,
				),
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
			got, _ := genArticleTree(tt.root, tt.replies, 1, model.ReplySortBest)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("want\n%+v, \nbut got\n%+v", utils.SprintJSONf(tt.want, "", "  "), utils.SprintJSONf(got, "", "  "))
			}
		})
	}
}
