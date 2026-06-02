package ui

import (
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func (a *App) field(gtx layout.Context, title string, editor *widget.Editor, hint string, height unit.Dp) layout.Dimensions {
	return a.fieldWithStyle(gtx, title, editor, hint, height, false)
}

func (a *App) technicalField(gtx layout.Context, title string, editor *widget.Editor, hint string, height unit.Dp) layout.Dimensions {
	return a.fieldWithStyle(gtx, title, editor, hint, height, true)
}

func (a *App) fieldWithStyle(gtx layout.Context, title string, editor *widget.Editor, hint string, height unit.Dp, monospace bool) layout.Dimensions {
	border := fluent.border2
	if gtx.Focused(editor) {
		border = accentAlpha(0xb8)
	}
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if title == "" {
				return layout.Dimensions{}
			}
			return a.label(gtx, title, unit.Sp(11), fluent.textMuted, font.SemiBold)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedHeight(gtx, height, func(gtx layout.Context) layout.Dimensions {
				return a.borderedSurface(gtx, fluent.surface, fluentControlRadius, border, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: 9, Bottom: 9, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						style := material.Editor(a.th, editor, hint)
						style.Color = fluent.text
						style.HintColor = fluent.textDim
						style.SelectionColor = accentAlpha(0x3d)
						style.TextSize = unit.Sp(13)
						style.Font.Weight = font.Medium
						if monospace {
							style.Font.Typeface = uiMonoTypeface
						}
						return style.Layout(gtx)
					})
				})
			})
		}),
	)
}

func (a *App) searchField(gtx layout.Context, editor *widget.Editor, hint string) layout.Dimensions {
	border := fluent.border2
	iconColor := fluent.textDim
	if gtx.Focused(editor) {
		border = accentAlpha(0xb8)
		iconColor = fluent.accent
	}
	return fixedHeight(gtx, unit.Dp(34), func(gtx layout.Context) layout.Dimensions {
		return a.borderedSurface(gtx, fluent.surface, fluentControlRadius, border, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 8, Bottom: 8, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return fixedWidth(gtx, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
							return fixedHeight(gtx, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
								return uiIconSearch.Layout(gtx, iconColor)
							})
						})
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						style := material.Editor(a.th, editor, hint)
						style.Color = fluent.text
						style.HintColor = fluent.textDim
						style.SelectionColor = accentAlpha(0x3d)
						style.TextSize = unit.Sp(12)
						style.Font.Weight = font.Medium
						return style.Layout(gtx)
					}),
				)
			})
		})
	})
}

func (a *App) segmentedWithTitle(gtx layout.Context, title string, options []choice, selected string, buttons []widget.Clickable, set func(string)) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, title, unit.Sp(12), fluent.textMuted, font.Medium)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.segmented(gtx, options, selected, buttons, set)
		}),
	)
}

func (a *App) segmentedGridWithTitle(gtx layout.Context, title string, options []choice, selected string, buttons []widget.Clickable, columns int, set func(string)) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, title, unit.Sp(12), fluent.textMuted, font.Medium)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.segmentedGrid(gtx, options, selected, buttons, columns, set)
		}),
	)
}

func (a *App) segmented(gtx layout.Context, options []choice, selected string, buttons []widget.Clickable, set func(string)) layout.Dimensions {
	return a.borderedSurface(gtx, fluent.surface2, unit.Dp(6), fluent.border, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(2)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, len(options))
			for i := range options {
				i := i
				for buttons[i].Clicked(gtx) {
					set(options[i].Value)
				}
				children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					active := options[i].Value == selected
					bg := rgba(0xffffff, 0x00)
					hoverBg := fluent.surface
					fg := fluent.textMuted
					border := rgba(0xffffff, 0x00)
					weight := font.Medium
					if active {
						bg = fluent.surface
						hoverBg = fluent.surface
						fg = fluent.text
						border = withAlpha(fluent.border2, 0x7a)
						weight = font.SemiBold
					}
					return a.surfaceButton(
						gtx,
						&buttons[i],
						bg,
						hoverBg,
						border,
						fluentControlRadius,
						layout.Inset{Top: 9, Bottom: 9, Left: 8, Right: 8},
						func(gtx layout.Context) layout.Dimensions {
							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, options[i].Label, unit.Sp(12), fg, weight)
							})
						},
					)
				}))
			}
			return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(2))}.Layout(gtx, children...)
		})
	})
}

func (a *App) segmentedGrid(gtx layout.Context, options []choice, selected string, buttons []widget.Clickable, columns int, set func(string)) layout.Dimensions {
	if columns <= 0 {
		columns = 2
	}
	rows := (len(options) + columns - 1) / columns
	return a.borderedSurface(gtx, fluent.surface2, unit.Dp(6), fluent.border, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(2)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, rows)
			for row := 0; row < rows; row++ {
				row := row
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					cellChildren := make([]layout.FlexChild, 0, columns)
					for col := 0; col < columns; col++ {
						idx := row*columns + col
						if idx >= len(options) {
							cellChildren = append(cellChildren, layout.Flexed(1, layout.Spacer{}.Layout))
							continue
						}
						for buttons[idx].Clicked(gtx) {
							set(options[idx].Value)
						}
						cellChildren = append(cellChildren, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							active := options[idx].Value == selected
							bg := rgba(0xffffff, 0x00)
							hoverBg := fluent.surface
							fg := fluent.textMuted
							border := rgba(0xffffff, 0x00)
							weight := font.Medium
							if active {
								bg = fluent.surface
								hoverBg = fluent.surface
								fg = fluent.text
								border = withAlpha(fluent.border2, 0x7a)
								weight = font.SemiBold
							}
							return a.surfaceButton(
								gtx,
								&buttons[idx],
								bg,
								hoverBg,
								border,
								fluentControlRadius,
								layout.Inset{Top: 9, Bottom: 9, Left: 8, Right: 8},
								func(gtx layout.Context) layout.Dimensions {
									return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return a.label(gtx, options[idx].Label, unit.Sp(12), fg, weight)
									})
								},
							)
						}))
					}
					bottom := unit.Dp(2)
					if row == rows-1 {
						bottom = 0
					}
					return layout.Inset{Bottom: bottom}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(2))}.Layout(gtx, cellChildren...)
					})
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}
