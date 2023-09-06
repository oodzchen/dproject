package main

import (
	"strconv"
	"text/template"
	"time"

	i18nc "github.com/oodzchen/dproject/i18n"
	"github.com/oodzchen/dproject/utils"
	"github.com/xeonx/timeago"
)

var TmplFuncs = template.FuncMap{
	"timeAgo":    formatTimeAgo,
	"timeFormat": utils.FormatTime,
	"intRange":   intRange,
	"local": func(id string, data ...any) string {
		var tplData = make(map[any]any)
		for idx, item := range data {
			if idx%2 == 0 {
				val := data[idx+1]
				if item == "Count" {
					switch v := val.(type) {
					case string:
						tplData[item], _ = strconv.Atoi(v)
					case int32:
						tplData[item] = int(v)
					case int64:
						tplData[item] = int(v)
					case int:
						tplData[item] = v
					case float32:
						tplData[item] = int(v)
					case float64:
						tplData[item] = int(v)
					default:
						// fmt.Println("Count data type: ", reflect.TypeOf(v))
						tplData[item] = 0

					}
				} else {
					tplData[item] = data[idx+1]
				}
			}
		}

		return i18nc.MustLocalize(id, tplData, tplData["Count"])
	},
}

func formatTimeAgo(t time.Time) string {
	return timeago.English.Format(t)
}

func intRange(start, end int) []int {
	// fmt.Println("start: ", start, "end", end)
	n := end - start + 1
	result := make([]int, n)
	for i := 0; i < n; i++ {
		result[i] = start + i
	}
	return result
}
