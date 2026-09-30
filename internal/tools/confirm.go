package tools

// ConfirmFn 执行前确认；返回 true 表示允许执行。
type ConfirmFn func(prompt string) bool

// 未注入时一律拒绝，避免无人确认时跑危险命令。
var confirm ConfirmFn = func(string) bool { return false }

// SetConfirm 注入确认实现（由 REPL 接线）。
func SetConfirm(fn ConfirmFn) {
	if fn == nil {
		confirm = func(string) bool { return false }
		return
	}
	confirm = fn
}
