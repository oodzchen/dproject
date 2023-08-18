package main

import (
	"text/template"
	"time"

	"github.com/oodzchen/dproject/utils"
	"github.com/xeonx/timeago"
)

var TmplFuncs = template.FuncMap{
	"timeAgo":    formatTimeAgo,
	"timeFormat": utils.FormatTime,
	"intRange":   intRange,
}

func formatTimeAgo(t time.Time) string {
	return timeago.English.Format(t)
}

func intRange(start, end int) []int {
	n := end - start + 1
	result := make([]int, n)
	for i := 0; i < n; i++ {
		result[i] = start + i
	}
	return result
}
