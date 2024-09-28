package utils

import "testing"

func TestReplaceLink(t *testing.T){
	tests := []struct{
		in string
		want string
	}{
		{
			"http://example.com",
			"<a title=\"http://example.com\" href=\"http://example.com\">http://example.com</a>",
		},
		{
			"http://example.com",
			"<a title=\"http://example.com\" href=\"http://example.com\">http://example.com</a>",
		},
		{
			"https://example.com",
			"<a title=\"https://example.com\" href=\"https://example.com\">https://example.com</a>",
		},
		{
			"https://example.com/aaa/bbb/333aaa/555",
			"<a title=\"https://example.com/aaa/bbb/333aaa/555\" href=\"https://example.com/aaa/bbb/333aaa/555\">https://example.com/aaa/bbb/333aaa/555</a>",
		},
		{
			"https://example.com/aaa/bbb/333aaa/555?aa=111&bb=dd",
			"<a title=\"https://example.com/aaa/bbb/333aaa/555?aa=111&bb=dd\" href=\"https://example.com/aaa/bbb/333aaa/555?aa=111&bb=dd\">https://example.com/aaa/bbb/333aaa/555?aa=111&bb=dd</a>",
		},
		{
			"https://example.com/aaa/bbb/333aaa/555?a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd",
			"<a title=\"https://example.com/aaa/bbb/333aaa/555?a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd\" href=\"https://example.com/aaa/bbb/333aaa/555?a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd\">https://example.com/aaa/bbb/333aaa/555?a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a=11&b=dd&a...</a>",
		},
		{
			"https://a.com",
			"<a title=\"https://a.com\" href=\"https://a.com\">https://a.com</a>",
		},
		{
			"https://a.com/bb33/cc",
			"<a title=\"https://a.com/bb33/cc\" href=\"https://a.com/bb33/cc\">https://a.com/bb33/cc</a>",
		},
		{
			"https://a.co",
			"<a title=\"https://a.co\" href=\"https://a.co\">https://a.co</a>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := ReplaceLink(tt.in)
			if got != tt.want {
				t.Errorf("got %s but want %s", got, tt.want)
			}
		})
	}
}
