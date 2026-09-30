package tools

// ConfirmFn 执行危险操作前的确认回调；返回 true 表示允许继续。
type ConfirmFn func(prompt string) bool

// confirm 默认拒绝，避免 REPL 忘记接线时误跑 shell / 写文件。
var confirm ConfirmFn = func(string) bool { return false }

// SetConfirm 由 REPL 注入确认实现（读 stdin 的 y/N）。
// fn 为 nil 时恢复为「一律拒绝」。
func SetConfirm(fn ConfirmFn) {
	if fn == nil {
		confirm = func(string) bool { return false }
		return
	}
	confirm = fn
}
