package tools

import "time"

func registerTime() {
	Register(Tool{
		Name:        "get_current_time",
		Description: "获取当前本地时间（Asia/Shanghai）。",
		Parameters: map[string]any{
			"type":                 "object",
			"properties":           map[string]any{},
			"additionalProperties": false,
		},
		Run: func(_ string) (string, error) {
			loc, err := time.LoadLocation("Asia/Shanghai")
			if err != nil {
				loc = time.FixedZone("CST", 8*3600)
			}
			return time.Now().In(loc).Format("2006/1/2 15:04:05"), nil
		},
	})
}
