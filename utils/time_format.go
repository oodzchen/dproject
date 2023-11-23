package utils

import (
	"strings"
	"time"
)

// Format time to string with "Y"(year), "M"(month), "D"(day), "h"(hour), "m"(minute), "s"(second)
func FormatTime(t time.Time, format string) string {
	formatArr := strings.Split(format, "")
	nativeFormat := make([]string, 0)
	nativeFormatMap := map[string]string{
		"Y": "2006",
		"M": "1",
		"D": "2",
		"h": "15",
		"m": "4",
		"s": "5",
	}
	hour := t.Hour()

	formatArr = Reverse(formatArr)

	prev := ""
	repeatCount := 0
	for _, tp := range formatArr {
		if _, found := nativeFormatMap[tp]; found {
			if tp == prev {
				repeatCount++
			} else {
				repeatCount = 0
			}

			if repeatCount == 0 {
				if tp == "h" && hour < 10 {
					nativeFormat = append(nativeFormat, "3")
				} else {
					nativeFormat = append(nativeFormat, nativeFormatMap[tp])
				}
			} else {
				if (tp == "Y" && repeatCount < 4) || (tp == "h" && hour > 9) {
					prev = tp
					continue
				}
				nativeFormat = append(nativeFormat, "0")
			}

			prev = tp
		} else {
			nativeFormat = append(nativeFormat, tp)
		}
	}

	nativeFormat = Reverse(nativeFormat)
	// log.Printf("nativeFormat: %s\n", strings.Join(nativeFormat, ""))
	return t.Format(strings.Join(nativeFormat, ""))
}

func Reverse[V any](arr []V) []V {
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}
	return arr
}
