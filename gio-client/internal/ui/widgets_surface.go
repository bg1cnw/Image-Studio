package ui

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func (a *App) sectionTitle(gtx layout.Context, text string) layout.Dimensions {
	return a.label(gtx, text, unit.Sp(15), fluent.text, font.SemiBold)
}

func (a *App) sectionEyebrow(gtx layout.Context, text string) layout.Dimensions {
	return a.label(gtx, text, unit.Sp(11), fluent.text, font.Bold)
}

func (a *App) button(gtx layout.Context, btn *widget.Clickable, text string, bg color.NRGBA, fg color.NRGBA) layout.Dimensions {
	style := material.Button(a.th, btn, text)
	style.Background = bg
	style.Color = fg
	style.CornerRadius = unit.Dp(4)
	style.TextSize = unit.Sp(12)
	style.Font.Weight = font.Medium
	style.Inset = layout.Inset{Top: 8, Bottom: 8, Left: 10, Right: 10}
	return style.Layout(gtx)
}

func (a *App) badge(gtx layout.Context, text string, bg color.NRGBA, fg color.NRGBA) layout.Dimensions {
	return a.borderedSurface(gtx, bg, unit.Dp(4), fluent.border, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 6, Bottom: 6, Left: 9, Right: 9}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, text, unit.Sp(11), fg, font.Medium)
		})
	})
}

func (a *App) toolPill(gtx layout.Context, text string, active bool) layout.Dimensions {
	bg := fluent.surface
	fg := fluent.textMuted
	if active {
		bg = fluent.accentSoft
		fg = fluent.accent
	}
	return a.badge(gtx, text, bg, fg)
}

func (a *App) surfaceButton(
	gtx layout.Context,
	btn *widget.Clickable,
	bg color.NRGBA,
	hoverBg color.NRGBA,
	border color.NRGBA,
	radius unit.Dp,
	inset layout.Inset,
	w layout.Widget,
) layout.Dimensions {
	fill := bg
	if btn.Hovered() {
		fill = hoverBg
	}
	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return a.borderedSurface(gtx, fill, radius, border, func(gtx layout.Context) layout.Dimensions {
			return inset.Layout(gtx, w)
		})
	})
}

func (a *App) pillButton(gtx layout.Context, btn *widget.Clickable, text string, active bool) layout.Dimensions {
	bg := fluent.surface
	hoverBg := fluent.surface2
	border := fluent.border
	fg := fluent.textMuted
	if active {
		bg = fluent.accentSoft
		hoverBg = rgba(0x005fb8, 0x28)
		fg = fluent.accent
	}
	return a.surfaceButton(
		gtx,
		btn,
		bg,
		hoverBg,
		border,
		unit.Dp(4),
		layout.Inset{Top: 7, Bottom: 7, Left: 10, Right: 10},
		func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, text, unit.Sp(11), fg, font.Medium)
		},
	)
}

func (a *App) compactButton(gtx layout.Context, btn *widget.Clickable, text string, accent bool) layout.Dimensions {
	bg := fluent.surface
	hoverBg := fluent.surface2
	fg := fluent.textMuted
	border := fluent.border
	if accent {
		bg = fluent.accentSoft
		hoverBg = rgba(0x005fb8, 0x28)
		fg = fluent.accent
	}
	return a.surfaceButton(
		gtx,
		btn,
		bg,
		hoverBg,
		border,
		unit.Dp(4),
		layout.Inset{Top: 6, Bottom: 6, Left: 9, Right: 9},
		func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, text, unit.Sp(11), fg, font.Medium)
		},
	)
}

func (a *App) staticPill(gtx layout.Context, text string, accent bool, dimmed bool) layout.Dimensions {
	bg := fluent.surface
	fg := fluent.textMuted
	border := fluent.border
	if accent {
		bg = fluent.accentSoft
		fg = fluent.accent
	}
	if dimmed {
		fg = fluent.textDim
		border = fluent.border2
	}
	return a.borderedSurface(gtx, bg, unit.Dp(4), border, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 7, Bottom: 7, Left: 10, Right: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, text, unit.Sp(11), fg, font.Medium)
		})
	})
}

func (a *App) imageThumb(gtx layout.Context, img image.Image, width unit.Dp, height unit.Dp, radius unit.Dp) layout.Dimensions {
	return fixedWidth(gtx, width, func(gtx layout.Context) layout.Dimensions {
		return fixedHeight(gtx, height, func(gtx layout.Context) layout.Dimensions {
			return a.borderedSurface(gtx, fluent.panel2, radius, fluent.border, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min = gtx.Constraints.Max
				if img == nil {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, "预览", unit.Sp(10), fluent.textDim, font.Medium)
					})
				}
				return layout.UniformInset(unit.Dp(3)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min = gtx.Constraints.Max
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						view := widget.Image{
							Src:      paint.NewImageOp(img),
							Fit:      widget.Contain,
							Position: layout.Center,
						}
						return view.Layout(gtx)
					})
				})
			})
		})
	})
}

func (a *App) label(gtx layout.Context, text string, size unit.Sp, color color.NRGBA, weight font.Weight) layout.Dimensions {
	style := material.Label(a.th, size, text)
	style.Color = color
	style.Font.Weight = weight
	style.WrapPolicy = textWrapWords
	return style.Layout(gtx)
}

func (a *App) card(gtx layout.Context, w layout.Widget) layout.Dimensions {
	return a.borderedSurface(gtx, fluent.surface, unit.Dp(8), fluent.border, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, w)
	})
}

func (a *App) borderedSurface(gtx layout.Context, bg color.NRGBA, radius unit.Dp, border color.NRGBA, w layout.Widget) layout.Dimensions {
	return a.surface(gtx, border, radius, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(1)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.surface(gtx, bg, radius, w)
		})
	})
}

func (a *App) surface(gtx layout.Context, bg color.NRGBA, radius unit.Dp, w layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := w(gtx)
	call := macro.Stop()
	shape := clip.UniformRRect(image.Rectangle{Max: dims.Size}, gtx.Dp(radius)).Op(gtx.Ops)
	paint.FillShape(gtx.Ops, bg, shape)
	call.Add(gtx.Ops)
	return dims
}

func fixedWidth(gtx layout.Context, width unit.Dp, w layout.Widget) layout.Dimensions {
	px := gtx.Dp(width)
	if px > gtx.Constraints.Max.X {
		px = gtx.Constraints.Max.X
	}
	gtx.Constraints.Min.X = px
	gtx.Constraints.Max.X = px
	return w(gtx)
}

func fixedHeight(gtx layout.Context, height unit.Dp, w layout.Widget) layout.Dimensions {
	px := gtx.Dp(height)
	if px > gtx.Constraints.Max.Y {
		px = gtx.Constraints.Max.Y
	}
	gtx.Constraints.Min.Y = px
	gtx.Constraints.Max.Y = px
	return w(gtx)
}

func chooseColor(ok bool, whenTrue color.NRGBA, whenFalse color.NRGBA) color.NRGBA {
	if ok {
		return whenTrue
	}
	return whenFalse
}

var textWrapWords = text.WrapWords
