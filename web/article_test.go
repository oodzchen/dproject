package web

import (
	"reflect"
	"testing"

	"github.com/oodzchen/dproject/model"
)

func TestFormatArticlesTree(t *testing.T) {
	root := &articleWithReplies{
		Article: &model.Article{
			Id:      1,
			ReplyTo: 0,
		},
		Replies: make([]*articleWithReplies, 0),
	}

	list := []*model.Article{
		{
			Id:      2,
			ReplyTo: 1,
		},
		{
			Id:      3,
			ReplyTo: 1,
		},
		{
			Id:      4,
			ReplyTo: 2,
		},
		{
			Id:      5,
			ReplyTo: 4,
		},
	}

	want := &articleWithReplies{
		Article: &model.Article{
			Id:      1,
			ReplyTo: 0,
		},
		Replies: []*articleWithReplies{
			{
				&model.Article{
					Id:      2,
					ReplyTo: 1,
				},
				[]*articleWithReplies{
					{
						&model.Article{
							Id:      4,
							ReplyTo: 2,
						},
						[]*articleWithReplies{
							{
								&model.Article{
									Id:      5,
									ReplyTo: 4,
								},
								make([]*articleWithReplies, 0),
							},
						},
					},
				},
			},
			{
				&model.Article{
					Id:      3,
					ReplyTo: 1,
				},
				make([]*articleWithReplies, 0),
			},
		},
	}

	got := formatArticlesToTree(root, list)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("result should be %+v, but got %+v", want, got)
	}
}
