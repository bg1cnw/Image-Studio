package ui

import (
	"image/color"

	"gioui.org/widget/material"
)

type fluentColors struct {
	accent     color.NRGBA
	accent2    color.NRGBA
	accentSoft color.NRGBA
	bg         color.NRGBA
	bg2        color.NRGBA
	panel      color.NRGBA
	panel2     color.NRGBA
	surface    color.NRGBA
	surface2   color.NRGBA
	sidebar    color.NRGBA
	inspector  color.NRGBA
	toolbar    color.NRGBA
	border     color.NRGBA
	border2    color.NRGBA
	text       color.NRGBA
	textMuted  color.NRGBA
	textDim    color.NRGBA
	canvasBg   color.NRGBA
	canvasTile color.NRGBA
	success    color.NRGBA
	danger     color.NRGBA
	white      color.NRGBA
}

var fluentLight = fluentColors{
	accent:     rgb(0x005fb8),
	accent2:    rgb(0x0a6fcb),
	accentSoft: rgba(0x005fb8, 0x1f),
	bg:         rgb(0xf3f3f3),
	bg2:        rgb(0xe9e9e9),
	panel:      rgb(0xfbfbfb),
	panel2:     rgb(0xf6f6f6),
	surface:    rgb(0xffffff),
	surface2:   rgb(0xf2f2f2),
	sidebar:    rgb(0xfbfbfb),
	inspector:  rgb(0xf8f8f8),
	toolbar:    rgb(0xf7f7f7),
	border:     rgba(0x000000, 0x14),
	border2:    rgba(0x000000, 0x24),
	text:       rgb(0x1f1f1f),
	textMuted:  rgb(0x5f6368),
	textDim:    rgb(0x8a8f98),
	canvasBg:   rgb(0xeeeeee),
	canvasTile: rgb(0xdedede),
	success:    rgb(0x0f7b0f),
	danger:     rgb(0xc42b1c),
	white:      rgb(0xffffff),
}

var fluentDark = fluentColors{
	accent:     rgb(0x4cc2ff),
	accent2:    rgb(0x66ccff),
	accentSoft: rgba(0x4cc2ff, 0x22),
	bg:         rgb(0x181818),
	bg2:        rgb(0x141414),
	panel:      rgb(0x202020),
	panel2:     rgb(0x252525),
	surface:    rgb(0x2a2a2a),
	surface2:   rgb(0x323232),
	sidebar:    rgb(0x222222),
	inspector:  rgb(0x242424),
	toolbar:    rgb(0x202020),
	border:     rgba(0xffffff, 0x16),
	border2:    rgba(0xffffff, 0x2a),
	text:       rgb(0xf5f5f5),
	textMuted:  rgb(0xc1c1c1),
	textDim:    rgb(0x8e8e8e),
	canvasBg:   rgb(0x1d1d1d),
	canvasTile: rgb(0x262626),
	success:    rgb(0x45d36b),
	danger:     rgb(0xff776b),
	white:      rgb(0xffffff),
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
