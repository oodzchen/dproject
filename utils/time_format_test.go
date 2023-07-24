package utils

import (
	"testing"
	"time"
)

func TestTimeFormat(t *testing.T) {
	tests := []struct {
		in   []int
		tpl  string
		want string
	}{
		{
			in:   []int{2023, 7, 24, 8, 24, 23},
			tpl:  "YYYYMMDD hh:mm:ss",
			want: "20230724 08:24:23",
		},
		{
			in:   []int{2023, 7, 24, 8, 24, 23},
			tpl:  "DD/MM/YYYY hh:mm:ss",
			want: "24/07/2023 08:24:23",
		},
		{
			in:   []int{2023, 7, 24, 8, 24, 23},
			tpl:  "YYYY-M-D h:m:s",
			want: "2023-7-24 8:24:23",
		},
		{
			in:   []int{2023, 7, 24, 17, 4, 3},
			tpl:  "h:m:s",
			want: "17:4:3",
		},
		{
			in:   []int{2023, 7, 24, 17, 4, 3},
			tpl:  "h:mm:ss",
			want: "17:04:03",
		},
	}

	for _, tt := range tests {
		t.Run(tt.tpl, func(t *testing.T) {
			inTime := time.Date(tt.in[0], time.Month(tt.in[1]), tt.in[2], tt.in[3], tt.in[4], tt.in[5], 00, time.Local)
			got := FormatTime(inTime, tt.tpl)
			if got != tt.want {
				t.Errorf("got time string %s but want %s", got, tt.want)
			}
		})
	}
}
