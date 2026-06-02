package ui

import (
	"image/color"

	"gioui.org/widget/material"
)

type fluentColors struct {
	accent          color.NRGBA
	accent2         color.NRGBA
	accentSoft      color.NRGBA
	bg              color.NRGBA
	bg2             color.NRGBA
	panel           color.NRGBA
	panel2          color.NRGBA
	surface         color.NRGBA
	surface2        color.NRGBA
	surfaceElevated color.NRGBA
	sidebar         color.NRGBA
	inspector       color.NRGBA
	toolbar         color.NRGBA
	border          color.NRGBA
	border2         color.NRGBA
	text            color.NRGBA
	textMuted       color.NRGBA
	textDim         color.NRGBA
	cardShadow      color.NRGBA
	cardGlow        color.NRGBA
	bgGlow          color.NRGBA
	canvasBg        color.NRGBA
	canvasTile      color.NRGBA
	success         color.NRGBA
	danger          color.NRGBA
	dangerSoft      color.NRGBA
	toolHoverBg     color.NRGBA
	toolHoverText   color.NRGBA
	windowOutline   color.NRGBA
	white           color.NRGBA
}

var fluentLight = fluentColors{
	accent:          rgb(0x0067c0),
	accent2:         rgb(0x0f7ad6),
	accentSoft:      rgba(0x0067c0, 0x1f),
	bg:              rgb(0xf3f3f3),
	bg2:             rgb(0xededed),
	panel:           rgba(0xffffff, 0xe0),
	panel2:          rgba(0xfafafa, 0xeb),
	surface:         rgb(0xfbfbfb),
	surface2:        rgb(0xefefef),
	surfaceElevated: rgba(0xffffff, 0xd6),
	sidebar:         rgba(0xf8f8f8, 0xd1),
	inspector:       rgba(0xf4f4f4, 0xd6),
	toolbar:         rgba(0xf6f6f6, 0xd1),
	border:          rgba(0x000000, 0x17),
	border2:         rgba(0x000000, 0x2e),
	text:            rgb(0x1f1f1f),
	textMuted:       rgba(0x363636, 0xc7),
	textDim:         rgba(0x606060, 0xc2),
	cardShadow:      rgba(0x000000, 0x07),
	cardGlow:        rgba(0xffffff, 0x20),
	bgGlow:          rgba(0xffffff, 0x36),
	canvasBg:        rgb(0xececec),
	canvasTile:      rgb(0xdedede),
	success:         rgb(0x0f7b0f),
	danger:          rgb(0xc42b1c),
	dangerSoft:      rgba(0xc42b1c, 0x1f),
	toolHoverBg:     rgba(0x0067c0, 0x1a),
	toolHoverText:   rgb(0x005a9e),
	windowOutline:   rgba(0xffffff, 0xb3),
	white:           rgb(0xffffff),
}

var fluentDark = fluentColors{
	accent:          rgb(0x4cc2ff),
	accent2:         rgb(0x78d3ff),
	accentSoft:      rgba(0x4cc2ff, 0x2e),
	bg:              rgb(0x202020),
	bg2:             rgb(0x1b1b1b),
	panel:           rgba(0x2a2a2a, 0xe6),
	panel2:          rgba(0x262626, 0xf0),
	surface:         rgb(0x2b2b2b),
	surface2:        rgb(0x353535),
	surfaceElevated: rgba(0x202020, 0xb8),
	sidebar:         rgba(0x232323, 0xdb),
	inspector:       rgba(0x1f1f1f, 0xd6),
	toolbar:         rgba(0x252525, 0xdb),
	border:          rgba(0xffffff, 0x14),
	border2:         rgba(0xffffff, 0x2e),
	text:            rgb(0xf5f5f5),
	textMuted:       rgba(0xf3f3f3, 0xc2),
	textDim:         rgba(0xd7d7d7, 0x99),
	cardShadow:      rgba(0x000000, 0x34),
	cardGlow:        rgba(0xffffff, 0x08),
	bgGlow:          rgba(0xffffff, 0x05),
	canvasBg:        rgb(0x252526),
	canvasTile:      rgb(0x343436),
	success:         rgb(0x6ccb5f),
	danger:          rgb(0xff99a4),
	dangerSoft:      rgba(0xff99a4, 0x1f),
	toolHoverBg:     rgba(0x4cc2ff, 0x24),
	toolHoverText:   rgb(0xb7e8ff),
	windowOutline:   rgba(0xffffff, 0x14),
	white:           rgb(0xffffff),
}

var fluent = fluentLight

func themePalette(mode string) fluentColors {
	if mode == "dark" {
		return fluentDark
	}
	return fluentLight
}

func normalizeThemeMode(mode string) string {
	switch mode {
	case "dark", "light", "system":
		return mode
	default:
		return "system"
	}
}

func resolveThemeMode(mode string) string {
	if normalizeThemeMode(mode) == "dark" {
		return "dark"
	}
	return "light"
}

func (a *App) applyThemeMode(mode string) {
	a.themeMode = normalizeThemeMode(mode)
	fluent = themePalette(resolveThemeMode(a.themeMode))
	a.th.Palette = material.Palette{
		Bg:         fluent.bg,
		Fg:         fluent.text,
		ContrastBg: fluent.accent,
		ContrastFg: fluent.white,
	}
	a.invalidateNow()
}

func rgb(v uint32) color.NRGBA {
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}
}

func rgba(v uint32, alpha uint8) color.NRGBA {
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: alpha}
}

func withAlpha(base color.NRGBA, alpha uint8) color.NRGBA {
	base.A = alpha
	return base
}

func accentAlpha(alpha uint8) color.NRGBA {
	return withAlpha(fluent.accent, alpha)
}

func dangerAlpha(alpha uint8) color.NRGBA {
	return withAlpha(fluent.danger, alpha)
}
