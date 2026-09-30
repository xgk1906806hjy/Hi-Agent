package chat

// dupTracker 在滑动窗口内统计同工具同参出现次数；达到 limit 则熔断。
// 覆盖「连续同参」与「窗口内非连续同参」两种空转。
type dupTracker struct {
	limit  int
	window int
	keys   []string
	counts map[string]int
}

// newDupTracker 创建检测器；window 若小于 limit 则抬高到 limit。
func newDupTracker(limit, window int) *dupTracker {
	if limit < 1 {
		limit = defaultMaxDupTools
	}
	if window < limit {
		window = limit
	}
	return &dupTracker{
		limit:  limit,
		window: window,
		counts: make(map[string]int),
	}
}

// add 记录一次调用；若该 key 在窗口内出现次数 ≥ limit，返回 true（应熔断）。
func (d *dupTracker) add(key string) bool {
	d.keys = append(d.keys, key)
	d.counts[key]++
	// 超出窗口：弹出队首并递减计数
	for len(d.keys) > d.window {
		old := d.keys[0]
		d.keys = d.keys[1:]
		d.counts[old]--
		if d.counts[old] <= 0 {
			delete(d.counts, old)
		}
	}
	return d.counts[key] >= d.limit
}

// count 返回 key 在当前窗口内的出现次数。
func (d *dupTracker) count(key string) int {
	return d.counts[key]
}
