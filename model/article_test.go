package model

import (
	"os"
	"testing"
)

type articleData struct {
	Title    string
	AuthorId int
	Content  string
	Depth    int
	Link     string
}

func TestArticleValid(t *testing.T) {
	largeText, err := os.ReadFile("testdata/rfc791.txt")
	if err != nil {
		t.Errorf("read file content error: %v", err)
	}

	tests := []struct {
		desc     string
		in       *articleData
		isUpdate bool
		valid    bool
	}{
		{
			desc: "All valid",
			in: &articleData{
				"This is Title",
				1,
				"This is content",
				0,
				"https://test.com",
			},
			isUpdate: false,
			valid:    true,
		},
		{
			desc: "Title is required",
			in: &articleData{
				" ",
				1,
				"This is content",
				0,
				"https://test.com",
			},
			isUpdate: false,
			valid:    false,
		},
		{
			desc: "Author is required",
			in: &articleData{
				"This is Title",
				0,
				"This is content",
				0,
				"https://test.com",
			},
			isUpdate: false,
			valid:    false,
		},
		{
			desc: "Content is optional",
			in: &articleData{
				"This is Title",
				1,
				" ",
				0,
				"https://test.com",
			},
			isUpdate: false,
			valid:    true,
		},
		{
			desc: "Title limit",
			in: &articleData{
				"This is Title This is Title This is Title This is Title This is Title This is Title This is Title This is Title",
				1,
				"This is content",
				0,
				"https://test.com",
			},
			isUpdate: false,
			valid:    false,
		},
		{
			desc: "Content limit",
			in: &articleData{
				"This is Title",
				1,
				string(largeText),
				0,
				"https://test.com",
			},
			isUpdate: false,
			valid:    false,
		},
		{
			desc: "Update root article without title",
			in: &articleData{
				"",
				1,
				"This is Content",
				0,
				"https://test.com",
			},
			isUpdate: true,
			valid:    false,
		},
		{
			desc: "Update reply without title",
			in: &articleData{
				"",
				0,
				"This is Content",
				1,
				"https://test.com",
			},
			isUpdate: true,
			valid:    true,
		},
		{
			desc: "Update without author",
			in: &articleData{
				"",
				0,
				"This is Content",
				1,
				"https://test.com",
			},
			isUpdate: true,
			valid:    true,
		},
		{
			desc: "Update without content",
			in: &articleData{
				"This is Title",
				0,
				"",
				1,
				"https://test.com",
			},
			isUpdate: true,
			valid:    false,
		},
		{
			desc: "URL is optinal",
			in: &articleData{
				"This is Title",
				1,
				"",
				0,
				"",
			},
			isUpdate: false,
			valid:    true,
		},
		{
			desc: "URL format",
			in: &articleData{
				"This is Title",
				1,
				"",
				0,
				"abc.com",
			},
			isUpdate: false,
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			a := &Article{
				Title:      tt.in.Title,
				AuthorId:   tt.in.AuthorId,
				Content:    tt.in.Content,
				ReplyDepth: tt.in.Depth,
				Link:       tt.in.Link,
			}
			err := a.Valid(tt.isUpdate)
			got := err == nil
			want := tt.valid

			if got != want {
				t.Errorf("article: %+v \nvalidate result should be %t, but got %t, error: %v", tt.in, want, got, err)
			}
		})
	}
}
