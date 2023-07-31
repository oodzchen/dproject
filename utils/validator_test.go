package utils

import "testing"

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{
			in:   "abc@test.com",
			want: true,
		},
		{
			in:   "abc.123.def@test.com",
			want: true,
		},
		{
			in:   "123@test.com.cn",
			want: true,
		},
		{
			in:   "abc@test.com'org",
			want: false,
		},
	}

	for _, tt := range tests {
		got := ValidateEmail(tt.in)
		if got != tt.want {
			t.Errorf("%s validate result should be %t, but got %t", tt.in, tt.want, got)
		}
	}
}
