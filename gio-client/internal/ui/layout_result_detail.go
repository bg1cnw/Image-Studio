package ui

import (
	"fmt"
	"image"
	"strings"

	sharedCompat "image-studio/shared/compat"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

func (a *App) layoutResultDetailModal(gtx layout.Context) layout.Dimensions {
	for a.closeResultDetailButton.Clicked(gtx) {
		a.closeResultDetail()
	}
	snap := a.readSnapshot()
	item := snap.ActiveResultDetail
	if item.ID == "" && strings.TrimSpace(item.SavedPath) == "" {
		return layout.Dimensions{}
	}
	paint.FillShape(gtx.Ops, rgba(0x000000, 0x52), clip.Rect{Max: gtx.Constraints.Max}.Op())
	gtx.Constraints.Min = gtx.Constraints.Max
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = image.Point{}
		return fixedWidth(gtx, unit.Dp(760), func(gtx layout.Context) layout.Dimensions {
			return fixedHeight(gtx, unit.Dp(620), func(gtx layout.Context) layout.Dimensions {
				return a.borderedSurface(gtx, fluent.surface, unit.Dp(8), fluent.border, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return a.label(gtx, "结果详情", unit.Sp(18), fluent.text, font.SemiBold)
											}),
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return a.label(gtx, detailHeadline(item), unit.Sp(11), fluent.textMuted, font.Normal)
											}),
										)
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return fixedWidth(gtx, unit.Dp(88), func(gtx layout.Context) layout.Dimensions {
											return a.button(gtx, &a.closeResultDetailButton, "关闭", fluent.surface2, fluent.text)
										})
									}),
								)
							}),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(16))}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return fixedWidth(gtx, unit.Dp(280), func(gtx layout.Context) layout.Dimensions {
											return a.layoutResultDetailPreview(gtx, item)
										})
									}),
									layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										return a.layoutResultDetailSections(gtx, item)
									}),
								)
							}),
						)
					})
				})
			})
		})
	})
}

func (a *App) layoutResultDetailPreview(gtx layout.Context, item sharedCompat.HistoryItem) layout.Dimensions {
	img, _ := a.imageForHistoryItem(item)
	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.imageThumb(gtx, img, unit.Dp(244), unit.Dp(244), unit.Dp(6))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if strings.TrimSpace(item.SavedPath) == "" {
					return layout.Dimensions{}
				}
				return a.label(gtx, historyPathText(item.SavedPath), unit.Sp(10), fluent.textDim, font.Normal)
			}),
		)
	})
}

func (a *App) layoutResultDetailSections(gtx layout.Context, item sharedCompat.HistoryItem) layout.Dimensions {
	return a.settingsList.Layout(gtx, 3, func(gtx layout.Context, index int) layout.Dimensions {
		var widget layout.Widget
		switch index {
		case 0:
			widget = func(gtx layout.Context) layout.Dimensions { return a.layoutResultDetailMeta(gtx, item) }
		case 1:
			widget = func(gtx layout.Context) layout.Dimensions {
				return a.layoutResultDetailTextSection(gtx, "原始提示词", item.Prompt)
			}
		default:
			widget = func(gtx layout.Context) layout.Dimensions {
				return a.layoutResultDetailTextSection(gtx, "优化后提示词", item.RevisedPrompt)
			}
		}
		return layout.Inset{Bottom: unit.Dp(12)}.Layout(gtx, widget)
	})
}

func (a *App) layoutResultDetailMeta(gtx layout.Context, item sharedCompat.HistoryItem) layout.Dimensions {
	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		rows := []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.sectionEyebrow(gtx, "参数")
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.detailKV(gtx, "模式", chooseModeLabel(item.Mode))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.detailKV(gtx, "尺寸", item.Size)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.detailKV(gtx, "质量", item.Quality)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.detailKV(gtx, "格式", strings.ToUpper(strings.TrimSpace(item.OutputFormat)))
			}),
		}
		if item.Seed != 0 {
			rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.detailKV(gtx, "Seed", detailValue(item.Seed))
			}))
		}
		if strings.TrimSpace(item.StyleTag) != "" {
			rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.detailKV(gtx, "风格", "#"+styleChoiceLabel(item.StyleTag))
			}))
		}
		rows = append(rows,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.detailKV(gtx, "创建时间", formatHistoryClock(item.CreatedAt))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if item.ElapsedSec <= 0 {
					return layout.Dimensions{}
				}
				return a.detailKV(gtx, "耗时", detailValue(item.ElapsedSec)+"s")
			}),
		)
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
	})
}

func (a *App) layoutResultDetailTextSection(gtx layout.Context, title string, text string) layout.Dimensions {
	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.sectionEyebrow(gtx, title)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				content := strings.TrimSpace(text)
				if content == "" {
					content = "(空)"
				}
				return a.borderedSurface(gtx, fluent.surface2, unit.Dp(6), fluent.border, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, content, unit.Sp(11), fluent.textMuted, font.Normal)
					})
				})
			}),
		)
	})
}

func (a *App) detailKV(gtx layout.Context, label string, value string) layout.Dimensions {
	value = strings.TrimSpace(value)
	if value == "" {
		return layout.Dimensions{}
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedWidth(gtx, unit.Dp(68), func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, label, unit.Sp(10), fluent.textDim, font.Medium)
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, value, unit.Sp(11), fluent.text, font.Normal)
		}),
	)
}

func detailHeadline(item sharedCompat.HistoryItem) string {
	return chooseModeLabel(item.Mode) + " · " + historyMetaText(item)
}

func chooseModeLabel(mode string) string {
	if mode == "edit" {
		return "图生图"
	}
	return "文生图"
}

func detailValue[T any](value T) string {
	return strings.TrimSpace(fmt.Sprint(value))
}
