package utils

import (
	"testing"
	"time"
)

func TestTimeFormat(t *testing.T) {
	tTime := time.Date(2023, 7, 24, 8, 24, 23, 00, time.Local)

	tests := []struct {
		in   time.Time
		tpl  string
		want string
	}{
		{
			in:   tTime,
			tpl:  "YYYYMMDD hh:mm:ss",
			want: "20230724 08:24:23",
		},
		{
			in:   tTime,
			tpl:  "DD/MM/YYYY hh:mm:ss",
			want: "24/07/2023 08:24:23",
		},
		{
			in:   tTime,
			tpl:  "YYYY-M-D h:m:s",
			want: "2023-7-24 8:24:23",
		},
		{
			in:   time.Date(2023, 7, 24, 17, 4, 3, 00, time.Local),
			tpl:  "h:m:s",
			want: "17:4:3",
		},
		{
			in:   time.Date(2023, 7, 24, 17, 4, 3, 00, time.Local),
			tpl:  "h:mm:ss",
			want: "17:04:03",
		},
	}

	for _, tt := range tests {
		t.Run(tt.tpl, func(t *testing.T) {
			got := FormatTime(tt.in, tt.tpl)
			if got != tt.want {
				t.Errorf("got time string %s but want %s", got, tt.want)
			}
		})
	}
}
