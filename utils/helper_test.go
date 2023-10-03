package utils

import (
	"fmt"
	"testing"
)

func TestExtractNameFromEmail(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{
			"abc@test.com",
			"abc",
		},
		{
			"$abc#@test.com",
			"abc",
		},
		{
			"a@test.com",
			"a",
		},
		{
			"abc.efg@test.com",
			"abc.efg",
		},
		{
			".abc_@test.com",
			"abc",
		},
		{
			"a#b^c@test.com",
			"a.b.c",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("extract name from %s", tt.in), func(t *testing.T) {
			got := ExtractNameFromEmail(tt.in)
			if got != tt.want {
				t.Errorf("want %s, but got %s", tt.want, got)
			}
		})
	}
}
