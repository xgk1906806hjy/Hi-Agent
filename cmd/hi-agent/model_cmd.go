package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xgk1906806hjy/Hi-Agent/internal/config"
)

// cmdMenu 命令菜单：优先 ↑/↓ 选择，纯文本回退序号。
func (a *app) cmdMenu() bool {
	cmds := slashCommands()
	items := make([]string, len(cmds))
	for i, c := range cmds {
		items[i] = c.Name + "  —  " + c.Desc
	}
	idx, err := a.ui.Select("命令菜单", items, 0)
	if err != nil {
		a.ui.Error(err.Error())
		return false
	}
	if idx < 0 {
		a.ui.Info("（已取消）")
		return false
	}
	return a.handleCommand(cmds[idx].Name)
}

// cmdModel 交互或快捷配置模型供应商（/model）。
func (a *app) cmdModel(args []string) {
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "show", "status":
			a.showModelConfig()
			return
		case "list", "providers":
			a.ui.Info(config.ProviderHelp())
			return
		default:
			modelName := ""
			if len(args) > 1 {
				modelName = strings.Join(args[1:], " ")
			}
			a.applyProviderQuick(args[0], modelName)
			return
		}
	}
	a.modelMenu()
}

func (a *app) showModelConfig() {
	a.ui.Info(fmt.Sprintf(
		"当前模型配置：\n  供应商：%s\n  模型：%s\n  接口：%s\n  密钥：%s",
		a.provider, a.chat.Model(), a.baseURL, config.MaskSecret(a.apiKey),
	))
}

func (a *app) modelMenu() {
	const (
		optProvider = iota
		optModel
		optKey
		optURL
		optShow
		optPersist
	)
	items := []string{
		"切换供应商 / 中转站",
		"修改模型名称",
		"修改 API Key",
		"修改接口地址 (Base URL)",
		"查看当前配置",
		"保存当前配置到全局 ~/.hi-agent/.env",
	}
	idx, err := a.ui.Select("模型配置", items, 0)
	if err != nil {
		a.ui.Error(err.Error())
		return
	}
	if idx < 0 {
		a.ui.Info("（已取消模型配置）")
		return
	}
	switch idx {
	case optProvider:
		a.pageSelectProvider()
	case optModel:
		a.pageEditModel()
	case optKey:
		a.pageEditKey()
	case optURL:
		a.pageEditURL()
	case optShow:
		a.showModelConfig()
	case optPersist:
		ep := config.Endpoint{
			Provider: a.provider,
			BaseURL:  a.baseURL,
			APIKey:   a.apiKey,
			Model:    a.chat.Model(),
		}
		if err := config.ApplyEndpoint(ep, true); err != nil {
			a.ui.Error(err.Error())
			return
		}
		a.ui.Info("已写入 " + config.GlobalEnvPath())
	}
}

func (a *app) pageSelectProvider() {
	providers := config.ListProviders()
	items := make([]string, len(providers))
	selected := 0
	for i, p := range providers {
		url := p.DefaultURL
		if url == "" {
			url = "须自填 URL"
		}
		model := p.DefaultModel
		if model == "" {
			model = "须自填模型"
		}
		cur := ""
		if p.ID == a.provider {
			cur = "  [当前]"
			selected = i
		}
		items[i] = fmt.Sprintf("%s — %s · 默认 %s%s", p.ID, p.Name, model, cur)
	}
	idx, err := a.ui.Select("选择供应商（Enter 进入配置）", items, selected)
	if err != nil {
		a.ui.Error(err.Error())
		return
	}
	if idx < 0 {
		a.ui.Info("（已取消）")
		return
	}
	a.pageConfigureProvider(providers[idx])
}

func (a *app) pageConfigureProvider(p config.Provider) {
	a.ui.Info(fmt.Sprintf("配置供应商：%s（%s）", p.Name, p.ID))

	defaultURL := p.DefaultURL
	if defaultURL == "" {
		defaultURL = a.baseURL
	}
	urlPrompt := "Base URL"
	if p.DefaultURL != "" {
		urlPrompt += "（回车=" + p.DefaultURL + "）"
	} else {
		urlPrompt += "（必填）"
	}
	urlIn, err := a.ui.ReadLine(urlPrompt + " › ")
	if err != nil {
		a.ui.Info("（已取消）")
		return
	}
	baseURL := strings.TrimSpace(urlIn)
	if baseURL == "" {
		baseURL = defaultURL
	}
	if baseURL == "" {
		a.ui.Error("Base URL 不能为空")
		return
	}

	defaultModel := p.DefaultModel
	if defaultModel == "" {
		defaultModel = a.chat.Model()
	}
	modelPrompt := "模型名"
	if defaultModel != "" {
		modelPrompt += "（回车=" + defaultModel + "）"
	}
	modelIn, err := a.ui.ReadLine(modelPrompt + " › ")
	if err != nil {
		a.ui.Info("（已取消）")
		return
	}
	modelName := strings.TrimSpace(modelIn)
	if modelName == "" {
		modelName = defaultModel
	}
	if modelName == "" {
		a.ui.Error("模型名不能为空")
		return
	}

	keyIn, err := a.ui.ReadLine("API Key（回车保留 " + config.MaskSecret(a.apiKey) + "） › ")
	if err != nil {
		a.ui.Info("（已取消）")
		return
	}
	apiKey := strings.TrimSpace(keyIn)
	if apiKey == "" {
		apiKey = a.apiKey
	}
	if apiKey == "" {
		a.ui.Error("API Key 不能为空")
		return
	}

	ep := config.Endpoint{
		Provider: p.ID,
		BaseURL:  strings.TrimRight(baseURL, "/"),
		APIKey:   apiKey,
		Model:    modelName,
	}
	a.commitEndpoint(ep, a.askPersistSelect())
}

func (a *app) pageEditModel() {
	cur := a.chat.Model()
	v, err := a.ui.ReadLine("模型名（当前 " + cur + "）› ")
	if err != nil || strings.TrimSpace(v) == "" {
		a.ui.Info("（已取消）")
		return
	}
	ep := config.Endpoint{Provider: a.provider, BaseURL: a.baseURL, APIKey: a.apiKey, Model: strings.TrimSpace(v)}
	a.commitEndpoint(ep, a.askPersistSelect())
}

func (a *app) pageEditKey() {
	v, err := a.ui.ReadLine("API Key（当前 " + config.MaskSecret(a.apiKey) + "）› ")
	if err != nil || strings.TrimSpace(v) == "" {
		a.ui.Info("（已取消）")
		return
	}
	ep := config.Endpoint{Provider: a.provider, BaseURL: a.baseURL, APIKey: strings.TrimSpace(v), Model: a.chat.Model()}
	a.commitEndpoint(ep, a.askPersistSelect())
}

func (a *app) pageEditURL() {
	v, err := a.ui.ReadLine("Base URL（当前 " + a.baseURL + "）› ")
	if err != nil || strings.TrimSpace(v) == "" {
		a.ui.Info("（已取消）")
		return
	}
	ep := config.Endpoint{
		Provider: a.provider,
		BaseURL:  strings.TrimRight(strings.TrimSpace(v), "/"),
		APIKey:   a.apiKey,
		Model:    a.chat.Model(),
	}
	a.commitEndpoint(ep, a.askPersistSelect())
}

func (a *app) askPersistSelect() bool {
	items := []string{"仅当前进程生效", "同时写入全局 ~/.hi-agent/.env"}
	idx, err := a.ui.Select("是否持久化？", items, 0)
	if err != nil || idx < 0 {
		return false
	}
	return idx == 1
}

func (a *app) applyProviderQuick(providerID, modelName string) {
	ep, err := config.ResolveEndpoint(providerID, func(k string) string {
		switch k {
		case "OPENAI_API_KEY", "HI_AGENT_API_KEY", "GEEKAGENT_API_KEY":
			if a.apiKey != "" {
				return a.apiKey
			}
		case "OPENAI_BASE_URL", "HI_AGENT_BASE_URL":
			if strings.EqualFold(providerID, "custom") || strings.EqualFold(providerID, a.provider) {
				return a.baseURL
			}
			return ""
		case "OPENAI_MODEL", "HI_AGENT_MODEL":
			return modelName
		}
		return strings.TrimSpace(os.Getenv(k))
	})
	if err != nil {
		a.ui.Error(err.Error())
		a.ui.Info("可改用 /model 打开菜单补全密钥与地址。")
		return
	}
	if modelName != "" {
		ep.Model = modelName
	}
	a.commitEndpoint(ep, a.askPersistSelect())
}

func (a *app) commitEndpoint(ep config.Endpoint, persist bool) {
	if err := config.ApplyEndpoint(ep, persist); err != nil {
		a.ui.Error("保存配置失败：" + err.Error())
		return
	}
	a.chat.Reconfigure(ep.BaseURL, ep.APIKey, ep.Model)
	a.provider = ep.Provider
	a.baseURL = ep.BaseURL
	a.apiKey = ep.APIKey
	msg := fmt.Sprintf("（已切换）供应商 %s · 模型 %s · %s", ep.Provider, ep.Model, ep.BaseURL)
	if persist {
		msg += "\n已写入 " + config.GlobalEnvPath()
	} else {
		msg += "\n仅当前进程生效；下次启动仍读 .env / 环境变量"
	}
	a.ui.Info(msg)
	a.ui.Refresh()
}
