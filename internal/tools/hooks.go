package tools

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// Decision 权限策略对一次调用的结论；零值 Ask 即未注入策略时的原有行为。
type Decision int

const (
	Ask   Decision = iota // 走工具内的确认流程
	Allow                 // 本次调用免确认
	Deny                  // 不执行，直接返回拒绝说明
)

// PolicyFn 按工具名与参数 JSON 判定权限。
type PolicyFn func(name, argsJSON string) Decision

// PathGuardFn 把工具收到的路径转成可安全使用的路径；越界时返回错误。
type PathGuardFn func(path string) (string, error)

// RedactFn 对给模型/用户看的文本做脱敏。
type RedactFn func(s string) string

var (
	hookMu    sync.RWMutex
	policy    PolicyFn
	pathGuard PathGuardFn
	redactor  RedactFn

	// execMu 串行化 Exec：preApproved 是「本次调用」级别的状态，
	// 只有同一时刻至多一个工具在执行时才成立（chat 工具循环本就顺序执行）。
	execMu      sync.Mutex
	preApproved atomic.Bool

	// callCtx 是当前 ExecContext 的上下文，仅在 execMu 持有期间有效。
	callCtx   context.Context
	delegator func(context.Context, string) (string, error)
)

// SetPolicy 注入权限策略；nil 表示一律 Ask。
func SetPolicy(fn PolicyFn) { hookMu.Lock(); policy = fn; hookMu.Unlock() }

// SetPathGuard 注入路径检查；nil 表示不检查。
func SetPathGuard(fn PathGuardFn) { hookMu.Lock(); pathGuard = fn; hookMu.Unlock() }

// SetRedactor 注入脱敏；nil 表示原样输出。
func SetRedactor(fn RedactFn) { hookMu.Lock(); redactor = fn; hookMu.Unlock() }

// SetDelegator 注入子任务委派。delegate_task 用当前 Exec 的上下文调用它；nil 时工具返回「委派未配置」。
func SetDelegator(fn func(context.Context, string) (string, error)) {
	hookMu.Lock()
	delegator = fn
	hookMu.Unlock()
}

func currentDelegator() func(context.Context, string) (string, error) {
	hookMu.RLock()
	defer hookMu.RUnlock()
	return delegator
}

func decide(name, argsJSON string) Decision {
	hookMu.RLock()
	fn := policy
	hookMu.RUnlock()
	if fn == nil {
		return Ask
	}
	return fn(name, argsJSON)
}

func guard(path string) (string, error) {
	hookMu.RLock()
	fn := pathGuard
	hookMu.RUnlock()
	if fn == nil {
		return path, nil
	}
	return fn(path)
}

func redact(s string) string {
	hookMu.RLock()
	fn := redactor
	hookMu.RUnlock()
	if fn == nil {
		return s
	}
	return fn(s)
}

// ask 工具内部的确认入口：策略已放行时直接通过；提示先脱敏。
func ask(prompt string) bool {
	if preApproved.Load() {
		return true
	}
	return confirm(redact(prompt))
}

func denied(name string) string {
	return fmt.Sprintf("已拒绝：权限策略不允许 %s", name)
}
