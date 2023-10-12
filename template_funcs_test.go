package main

import (
	"fmt"
	"net/url"
	"testing"
)

type pageStrData struct {
	path  string
	page  int
	query url.Values
}

func TestPageStr(t *testing.T) {
	tests := []struct {
		in   *pageStrData
		want string
	}{
		{
			&pageStrData{
				path:  "/articles",
				page:  2,
				query: nil,
			},
			"/articles?page=2",
		},
		{
			&pageStrData{
				"/articles",
				2,
				url.Values{
					"type": []string{"reply"},
				},
			},
			"/articles?page=2&type=reply",
		},
		{
			&pageStrData{
				"/articles",
				3,
				url.Values{
					"type": []string{"reply"},
					"page": []string{"2"},
				},
			},
			"/articles?page=3&type=reply",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("generate path %s", tt.want), func(t *testing.T) {
			got := pageStr(tt.in.path, tt.in.page, tt.in.query)
			if got != tt.want {
				t.Errorf("want %s, but got %s", tt.want, got)
			}
		})
	}
}
