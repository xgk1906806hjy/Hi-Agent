package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// codexTheme：浅色中性底 + 单一蓝强调色（发送/选中），贴近 Codex 简洁对话窗。
type codexTheme struct{}

var _ fyne.Theme = (*codexTheme)(nil)

func (c *codexTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0xF5, G: 0xF6, B: 0xF8, A: 0xFF}
	case theme.ColorNameButton:
		return color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 0xE5, G: 0xE5, B: 0xE7, A: 0xFF}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0x1A, G: 0x1A, B: 0x1E, A: 0xFF}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 0x9A, G: 0x9A, B: 0xA0, A: 0xFF}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 0x8A, G: 0x8A, B: 0x90, A: 0xFF}
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0x2F, G: 0x6F, B: 0xED, A: 0xFF} // 发送强调蓝
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	case theme.ColorNameInputBorder:
		return color.NRGBA{R: 0xD8, G: 0xD8, B: 0xDC, A: 0xFF}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 0xE6, G: 0xE6, B: 0xEA, A: 0xFF}
	case theme.ColorNameHover:
		return color.NRGBA{R: 0xEB, G: 0xEB, B: 0xEF, A: 0xFF}
	case theme.ColorNameFocus:
		return color.NRGBA{R: 0x2F, G: 0x6F, B: 0xED, A: 0x66}
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x14}
	case theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground:
		return color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	}
	return theme.DefaultTheme().Color(name, theme.VariantLight)
}

func (c *codexTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (c *codexTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (c *codexTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameInnerPadding:
		return 10
	case theme.SizeNameText:
		return 14
	case theme.SizeNameHeadingText:
		return 16
	case theme.SizeNameCaptionText:
		return 12
	case theme.SizeNameInputBorder:
		return 1
	case theme.SizeNameScrollBar:
		return 10
	}
	return theme.DefaultTheme().Size(name)
}

// 气泡配色（不走 Theme，保证对比稳定）
var (
	bubbleUserBG = color.NRGBA{R: 0xE7, G: 0xEE, B: 0xFE, A: 0xFF}
	bubbleAgentBG = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	bubbleToolBG  = color.NRGBA{R: 0xF0, G: 0xF1, B: 0xF4, A: 0xFF}
	bubbleSysFG   = color.NRGBA{R: 0x6B, G: 0x6B, B: 0x72, A: 0xFF}
	bubbleErrBG   = color.NRGBA{R: 0xFD, G: 0xE8, B: 0xE8, A: 0xFF}
	sidePanelBG   = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	chatBG        = color.NRGBA{R: 0xF5, G: 0xF6, B: 0xF8, A: 0xFF}
	ink           = color.NRGBA{R: 0x1C, G: 0x1C, B: 0x1F, A: 0xFF}
	muted         = color.NRGBA{R: 0x8B, G: 0x8D, B: 0x98, A: 0xFF}
	hairline      = color.NRGBA{R: 0xE6, G: 0xE7, B: 0xEC, A: 0xFF}
	rowActiveBG   = color.NRGBA{R: 0xE7, G: 0xEE, B: 0xFE, A: 0xFF}
	rowActiveFG   = color.NRGBA{R: 0x1D, G: 0x4E, B: 0xD8, A: 0xFF}
)
