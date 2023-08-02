package model

import (
	"io/ioutil"
	"testing"
)

type articleData struct {
	Title    string
	AuthorId int
	Content  string
}

func TestArticleValid(t *testing.T) {
	largeText, err := ioutil.ReadFile("testdata/rfc791.txt")
	if err != nil {
		t.Errorf("read file content error: %v", err)
	}

	tests := []struct {
		desc  string
		in    *articleData
		valid bool
	}{
		{
			desc: "All valid",
			in: &articleData{
				"This is Title",
				1,
				"This is content",
			},
			valid: true,
		},
		{
			desc: "Title is required",
			in: &articleData{
				" ",
				1,
				"This is content",
			},
			valid: false,
		},
		{
			desc: "Author is required",
			in: &articleData{
				"This is Title",
				0,
				"This is content",
			},
			valid: false,
		},
		{
			desc: "Content is required",
			in: &articleData{
				"This is Title",
				1,
				" ",
			},
			valid: false,
		},
		{
			desc: "Title limit",
			in: &articleData{
				"This is Title This is Title This is Title This is Title This is Title This is Title This is Title This is Title",
				1,
				"This is content",
			},
			valid: false,
		},
		{
			desc: "Content limit",
			in: &articleData{
				"This is Title",
				1,
				string(largeText),
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			a := &Article{
				Title:    tt.in.Title,
				AuthorId: tt.in.AuthorId,
				Content:  tt.in.Content,
			}
			err := a.Valid()
			got := err == nil
			want := tt.valid

			if got != want {
				t.Errorf("article: %+v \nvalidate result should be %t, but got %t, error: %v", tt.in, want, got, err)
			}
		})
	}
}
