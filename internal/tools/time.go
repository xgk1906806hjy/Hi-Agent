package tools

import "time"

// registerTime 注册 get_current_time：返回 Asia/Shanghai 本地时间字符串。
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
				// 系统无时区数据时回退到固定 UTC+8。
				loc = time.FixedZone("CST", 8*3600)
			}
			return time.Now().In(loc).Format("2006/1/2 15:04:05"), nil
		},
	})
}
