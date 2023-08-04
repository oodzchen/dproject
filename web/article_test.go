package web

import (
	"reflect"
	"testing"

	"github.com/oodzchen/dproject/model"
)

// func TestFormatArticlesTree(t *testing.T) {
// 	root := &articleWithReplies{
// 		Article: &model.Article{
// 			Id:      1,
// 			ReplyTo: 0,
// 		},
// 		Replies: make([]*articleWithReplies, 0),
// 	}

// 	list := []*model.Article{
// 		{
// 			Id:      6,
// 			ReplyTo: 2,
// 		},
// 		{
// 			Id:      2,
// 			ReplyTo: 1,
// 		},
// 		{
// 			Id:      3,
// 			ReplyTo: 1,
// 		},
// 		{
// 			Id:      4,
// 			ReplyTo: 2,
// 		},
// 		{
// 			Id:      5,
// 			ReplyTo: 4,
// 		},
// 	}

// 	want := &articleWithReplies{
// 		Article: &model.Article{
// 			Id:      1,
// 			ReplyTo: 0,
// 		},
// 		Replies: []*articleWithReplies{
// 			{
// 				&model.Article{
// 					Id:      2,
// 					ReplyTo: 1,
// 				},
// 				[]*articleWithReplies{
// 					{
// 						&model.Article{
// 							Id:      6,
// 							ReplyTo: 2,
// 						},
// 						make([]*articleWithReplies, 0),
// 					},
// 					{
// 						&model.Article{
// 							Id:      4,
// 							ReplyTo: 2,
// 						},
// 						[]*articleWithReplies{
// 							{
// 								&model.Article{
// 									Id:      5,
// 									ReplyTo: 4,
// 								},
// 								make([]*articleWithReplies, 0),
// 							},
// 						},
// 					},
// 				},
// 			},
// 			{
// 				&model.Article{
// 					Id:      3,
// 					ReplyTo: 1,
// 				},
// 				make([]*articleWithReplies, 0),
// 			},
// 		},
// 	}

// 	got := formatArticlesToTree(root, list)

// 	if !reflect.DeepEqual(got, want) {
// 		t.Errorf("result should be %+v, but got %+v", want, got)
// 	}
// }

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
				Replies: []*model.Article{
					{
						Id:      2,
						ReplyTo: 1,
						Replies: []*model.Article{
							{
								Id:      3,
								ReplyTo: 2,
							},
							{
								Id:      4,
								ReplyTo: 2,
								Replies: []*model.Article{
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
			nil,
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
