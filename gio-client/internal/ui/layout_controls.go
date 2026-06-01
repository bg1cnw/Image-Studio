package ui

import (
	"fmt"
	"image"
	"image/color"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"image-studio/gio-client/internal/kernel"
	sharedCompat "image-studio/shared/compat"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/yuanhua/image-gptcodex/pkg/client"
)

type promptHelperItem struct {
	ID     string
	Title  string
	Detail string
}

var promptTemplates = []promptHelperItem{
	{ID: "photoreal", Title: "写实摄影", Detail: "photorealistic, professional photography, 35mm, natural lighting, sharp focus, high detail"},
	{ID: "cinematic", Title: "电影感", Detail: "cinematic, dramatic lighting, shallow depth of field, film grain, anamorphic, 2.39:1"},
	{ID: "anime", Title: "二次元", Detail: "anime style, vibrant colors, cel shading, detailed illustration"},
	{ID: "oil", Title: "油画", Detail: "oil painting, thick brush strokes, classical art style, warm tones"},
	{ID: "watercolor", Title: "水彩", Detail: "watercolor painting, soft edges, pastel colors, paper texture"},
	{ID: "flat", Title: "扁平插画", Detail: "flat illustration, minimalist, geometric shapes, vector style"},
	{ID: "render3d", Title: "3D 渲染", Detail: "3D render, octane render, ray tracing, glossy, studio lighting"},
	{ID: "pixel", Title: "像素风", Detail: "pixel art, 16-bit, retro game style, limited palette"},
}

type settingsOptionChoice struct {
	Title  string
	Detail string
	Value  string
}

func (a *App) layoutControls(gtx layout.Context) layout.Dimensions {
	for a.composeToggleButton.Clicked(gtx) {
		a.composeOpen = !a.composeOpen
	}
	for a.advancedToggleButton.Clicked(gtx) {
		a.advancedOpen = !a.advancedOpen
	}
	for a.manageUpstreamButton.Clicked(gtx) {
		a.openSettingsModal()
	}

	return a.borderedSurface(gtx, fluent.sidebar, unit.Dp(0), fluent.border, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		return layout.Inset{Top: 16, Bottom: 16, Left: 20, Right: 20}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			snap := a.readSnapshot()
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.controlsList.Layout(gtx, 1, func(gtx layout.Context, _ int) layout.Dimensions {
						children := []layout.FlexChild{
							layout.Rigid(a.layoutWorkbenchCard),
						}
						if strings.TrimSpace(snap.LastErrorMessage) != "" {
							children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.layoutErrorNoticeCard(gtx, snap)
							}))
						}
						children = append(children,
							layout.Rigid(a.layoutModeCard),
							layout.Rigid(a.layoutPromptCard),
							layout.Rigid(a.layoutComposeCard),
							layout.Rigid(a.layoutAdvancedCard),
						)
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx, children...)
					})
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.layoutSubmitDock(gtx)
				}),
			)
		})
	})
}

func (a *App) layoutSubmitDock(gtx layout.Context) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			size := gtx.Constraints.Min
			if size.X == 0 {
				size.X = gtx.Constraints.Max.X
			}
			if size.Y == 0 {
				size.Y = gtx.Dp(unit.Dp(156))
			}
			paintLinearGradient(gtx, image.Rect(0, 0, size.X, size.Y), 0, rgba(0xffffff, 0x00), withAlpha(fluent.sidebar, 0xf4))
			return layout.Dimensions{Size: size}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 10, Bottom: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return a.layoutActions(gtx)
			})
		}),
	)
}

func (a *App) layoutErrorNoticeCard(gtx layout.Context, snap snapshot) layout.Dimensions {
	for a.retryLastRunButton.Clicked(gtx) {
		a.retryLastRun()
	}
	for a.openRawResponseButton.Clicked(gtx) {
		raw := strings.TrimSpace(snap.Result.RawPath)
		if raw == "" {
			continue
		}
		a.openRawResponseModal(raw)
	}
	for a.dismissErrorButton.Clicked(gtx) {
		a.dismissFailureState()
	}

	return a.borderedSurface(gtx, dangerAlpha(0x16), unit.Dp(10), dangerAlpha(0x2f), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			rows := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Start}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "最近一次请求失败", unit.Sp(12), fluent.danger, font.SemiBold)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.ghostIconButton(gtx, &a.dismissErrorButton, uiIconClose, false)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, snap.LastErrorMessage, unit.Sp(11), fluent.danger, font.Normal)
				}),
			}
			if snap.LastRunAvailable || strings.TrimSpace(snap.Result.RawPath) != "" {
				rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					buttons := []layout.FlexChild{}
					if snap.LastRunAvailable {
						buttons = append(buttons, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.compactButton(gtx, &a.retryLastRunButton, "重试上次请求", false)
						}))
					}
					if strings.TrimSpace(snap.Result.RawPath) != "" {
						if len(buttons) > 0 {
							buttons = append(buttons, layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout))
						}
						buttons = append(buttons, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.compactButton(gtx, &a.openRawResponseButton, "查看日志", false)
						}))
					}
					return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, buttons...)
					})
				}))
			}
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx, rows...)
		})
	})
}

func (a *App) submitActionButton(
	gtx layout.Context,
	btn *widget.Clickable,
	text string,
	bg color.NRGBA,
	hoverBg color.NRGBA,
	border color.NRGBA,
	fg color.NRGBA,
) layout.Dimensions {
	return fixedHeight(gtx, unit.Dp(48), func(gtx layout.Context) layout.Dimensions {
		return a.surfaceButton(
			gtx,
			btn,
			bg,
			hoverBg,
			border,
			unit.Dp(10),
			layout.Inset{Top: 0, Bottom: 0, Left: 12, Right: 12},
			func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, text, unit.Sp(13), fg, font.SemiBold)
				})
			},
		)
	})
}

func (a *App) layoutWorkbenchCard(gtx layout.Context) layout.Dimensions {
	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		modeSummary := a.modeLabel()
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(3))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.titleLabel(gtx, "图像工作台", unit.Sp(18))
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.singleLineLabel(gtx, "保持界面简洁，把注意力留给 prompt、参考图和结果。", unit.Sp(11), fluent.textMuted, font.Normal)
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.staticPill(gtx, modeSummary, true, false)
					}),
				)
			}),
		)
	})
}

func (a *App) layoutModeCard(gtx layout.Context) layout.Dimensions {
	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, "模式", unit.Sp(11), fluent.textMuted, font.SemiBold)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.borderedSurface(gtx, accentAlpha(0x06), unit.Dp(10), fluent.border, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(2)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						children := make([]layout.FlexChild, 0, len(modeChoices))
						for idx := range modeChoices {
							idx := idx
							for a.modeButtons[idx].Clicked(gtx) {
								a.mode = modeChoices[idx].Value
							}
							children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								active := a.mode == modeChoices[idx].Value
								icon := uiIconPlay
								if modeChoices[idx].Value == string(client.ModeEdit) {
									icon = uiIconEdit
								}
								return a.surfaceButton(
									gtx,
									&a.modeButtons[idx],
									chooseColor(active, fluent.surface, rgba(0xffffff, 0x00)),
									chooseColor(active, fluent.surface, fluent.surface),
									chooseColor(active, accentAlpha(0x14), rgba(0xffffff, 0x00)),
									fluentControlRadius,
									layout.Inset{Top: 9, Bottom: 9, Left: 10, Right: 10},
									func(gtx layout.Context) layout.Dimensions {
										fg := chooseColor(active, fluent.text, fluent.textMuted)
										return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return fixedWidth(gtx, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
														return fixedHeight(gtx, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
															return icon.Layout(gtx, fg)
														})
													})
												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return a.label(gtx, modeChoices[idx].Label, unit.Sp(11), fg, chooseFontWeight(active))
												}),
											)
										})
									},
								)
							}))
						}
						return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(2))}.Layout(gtx, children...)
					})
				})
			}),
		)
	})
}

func (a *App) layoutPromptCard(gtx layout.Context) layout.Dimensions {
	snap := a.readSnapshot()
	promptSuggestions := buildPromptSuggestions(snap.PromptHistory, snap.History)
	for a.promptHelperButton.Clicked(gtx) {
		if !a.promptHelperOpen {
			a.promptHelperTab = "templates"
		}
		a.promptHelperOpen = !a.promptHelperOpen
	}
	for a.optimizePromptButton.Clicked(gtx) {
		a.startPromptOptimize()
	}
	for _, item := range promptTemplates {
		btn := a.promptButton("prompt-template:" + item.ID)
		item := item
		for btn.Clicked(gtx) {
			a.applyPromptSuggestion(item.Detail)
		}
	}
	for idx := range promptSuggestions {
		btn := a.promptButton(fmt.Sprintf("prompt-history:%d", idx))
		text := promptSuggestions[idx]
		for btn.Clicked(gtx) {
			a.applyPromptSuggestion(text)
		}
	}

	promptLen := len([]rune(strings.TrimSpace(a.promptInput.Text())))
	title := "提示词"
	hint := "主体 / 场景 / 光照 / 镜头 / 风格\n例如：一只橘猫坐在雨夜窗边，电影级侧逆光，50mm，浅景深，写实摄影"
	if a.mode == string(client.ModeEdit) {
		title = "修改要求"
		hint = "主体保持不变\n把背景换成夜空，补一圈冷色边缘光，保留原有构图"
	}

	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		base := func(gtx layout.Context) layout.Dimensions {
			promptBorder := fluent.border2
			if gtx.Focused(&a.promptInput) {
				promptBorder = accentAlpha(0xb8)
			}
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, title, unit.Sp(10), fluent.textMuted, font.Medium)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.monoLabel(gtx, fmt.Sprintf("%d", promptLen), unit.Sp(11), fluent.textDim, font.Normal)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedHeight(gtx, unit.Dp(124), func(gtx layout.Context) layout.Dimensions {
						return a.borderedSurface(gtx, fluent.surface, fluentControlRadius, promptBorder, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Top: 10, Bottom: 10, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.editorText(gtx, &a.promptInput, hint, unit.Sp(13))
							})
						})
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.ghostIconTextButton(gtx, &a.promptHelperButton, uiIconHistory, "模板 / 历史", a.promptHelperOpen)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							label := "AI 优化"
							if snap.OptimizingPrompt {
								label = "优化中..."
							}
							icon := uiIconSpark
							if snap.OptimizingPrompt {
								icon = uiIconRefresh
							}
							return a.ghostIconTextButton(gtx, &a.optimizePromptButton, icon, label, snap.OptimizingPrompt)
						}),
						layout.Flexed(1, layout.Spacer{}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.monoLabel(gtx, "Ctrl+Enter", unit.Sp(11), fluent.textDim, font.Normal)
						}),
					)
				}),
			)
		}
		if !a.promptHelperOpen {
			return base(gtx)
		}
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(base),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				macro := op.Record(gtx.Ops)
				overlayDims := layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.layoutPromptHelperInline(gtx, promptSuggestions)
				})
				call := macro.Stop()
				offsetY := gtx.Dp(unit.Dp(152))
				if offsetY+overlayDims.Size.Y > gtx.Constraints.Max.Y {
					offsetY = gtx.Constraints.Max.Y - overlayDims.Size.Y
				}
				if offsetY < 0 {
					offsetY = 0
				}
				trans := op.Offset(image.Pt(0, offsetY)).Push(gtx.Ops)
				call.Add(gtx.Ops)
				trans.Pop()
				return layout.Dimensions{}
			}),
		)
	})
}

func (a *App) layoutPromptHelperPanel(gtx layout.Context, suggestions []string) layout.Dimensions {
	items := promptTemplates
	prefix := "prompt-template:"
	emptyText := "还没有提交过 prompt"
	if a.promptHelperTab == "history" {
		items = promptLabels(suggestions)
		prefix = "prompt-history:"
		emptyText = "还没有提交过提示词。"
	}
	if len(items) == 0 {
		return a.borderedSurface(gtx, fluent.surface, unit.Dp(10), fluent.border, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, emptyText, unit.Sp(11), fluent.textDim, font.Normal)
				})
			})
		})
	}
	return fixedHeight(gtx, unit.Dp(308), func(gtx layout.Context) layout.Dimensions {
		return a.promptHelperList.Layout(gtx, len(items), func(gtx layout.Context, index int) layout.Dimensions {
			item := items[index]
			return layout.Inset{Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return a.layoutPromptHelperItem(gtx, prefix+item.ID, item)
			})
		})
	})
}

func (a *App) layoutPromptHelperInline(gtx layout.Context, suggestions []string) layout.Dimensions {
	for a.closePromptHelperButton.Clicked(gtx) {
		a.promptHelperOpen = false
	}
	for a.promptHelperTemplatesButton.Clicked(gtx) {
		a.promptHelperTab = "templates"
	}
	for a.promptHelperHistoryButton.Clicked(gtx) {
		a.promptHelperTab = "history"
	}
	return fixedWidth(gtx, unit.Dp(360), func(gtx layout.Context) layout.Dimensions {
		return a.elevatedBorderedSurface(gtx, fluent.surface, fluentCardRadius, fluent.border, image.Pt(0, 8), func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: 8, Bottom: 8, Left: 8, Right: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return a.layoutPromptHelperTabs(gtx, len(promptTemplates), len(suggestions))
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.ghostIconButton(gtx, &a.closePromptHelperButton, uiIconClose, false)
							}),
						)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedHeight(gtx, unit.Dp(1), func(gtx layout.Context) layout.Dimensions {
						return a.surface(gtx, fluent.border, 0, layout.Spacer{}.Layout)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: 8, Bottom: 8, Left: 8, Right: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.layoutPromptHelperPanel(gtx, suggestions)
					})
				}),
			)
		})
	})
}

func (a *App) layoutPromptHelperModal(gtx layout.Context) layout.Dimensions {
	for a.closePromptHelperButton.Clicked(gtx) {
		a.promptHelperOpen = false
	}
	for a.promptHelperTemplatesButton.Clicked(gtx) {
		a.promptHelperTab = "templates"
	}
	for a.promptHelperHistoryButton.Clicked(gtx) {
		a.promptHelperTab = "history"
	}
	snap := a.readSnapshot()
	suggestions := buildPromptSuggestions(snap.PromptHistory, snap.History)
	return a.layoutStandardModal(
		gtx,
		unit.Dp(560),
		0,
		"模板 / 历史",
		"",
		&a.closePromptHelperButton,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.layoutPromptHelperTabs(gtx, len(promptTemplates), len(suggestions))
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.layoutPromptHelperPanel(gtx, suggestions)
				}),
			)
		},
	)
}

func chooseFontWeight(active bool) font.Weight {
	if active {
		return font.SemiBold
	}
	return font.Medium
}

func (a *App) layoutPromptHelperTabs(gtx layout.Context, templateCount int, historyCount int) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.surfaceButton(
				gtx,
				&a.promptHelperTemplatesButton,
				chooseColor(a.promptHelperTab != "history", fluent.accentSoft, rgba(0xffffff, 0x00)),
				fluent.surface2,
				rgba(0xffffff, 0x00),
				fluentControlRadius,
				layout.Inset{Top: 8, Bottom: 8, Left: 10, Right: 10},
				func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, "模板", unit.Sp(11), chooseColor(a.promptHelperTab != "history", fluent.accent, fluent.textMuted), chooseFontWeight(a.promptHelperTab != "history"))
					})
				},
			)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			label := fmt.Sprintf("历史 (%d)", historyCount)
			return a.surfaceButton(
				gtx,
				&a.promptHelperHistoryButton,
				chooseColor(a.promptHelperTab == "history", fluent.accentSoft, rgba(0xffffff, 0x00)),
				fluent.surface2,
				rgba(0xffffff, 0x00),
				fluentControlRadius,
				layout.Inset{Top: 8, Bottom: 8, Left: 10, Right: 10},
				func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, label, unit.Sp(11), chooseColor(a.promptHelperTab == "history", fluent.accent, fluent.textMuted), chooseFontWeight(a.promptHelperTab == "history"))
					})
				},
			)
		}),
	)
}

func (a *App) layoutPromptHelperItem(gtx layout.Context, buttonID string, item promptHelperItem) layout.Dimensions {
	btn := a.promptButton(buttonID)
	return a.surfaceButton(
		gtx,
		btn,
		rgba(0xffffff, 0x00),
		fluent.accentSoft,
		rgba(0xffffff, 0x00),
		unit.Dp(10),
		layout.Inset{Top: 10, Bottom: 10, Left: 12, Right: 12},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.clampedLabel(gtx, item.Title, unit.Sp(12), fluent.text, font.SemiBold, 2)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if strings.TrimSpace(item.Detail) == "" || strings.TrimSpace(item.Detail) == strings.TrimSpace(item.Title) {
						return layout.Dimensions{}
					}
					return a.clampedLabel(gtx, item.Detail, unit.Sp(10), fluent.textDim, font.Normal, 3)
				}),
			)
		},
	)
}

func (a *App) layoutSettingsModal(gtx layout.Context) layout.Dimensions {
	for a.closeSettingsButton.Clicked(gtx) {
		a.closeSettingsModal()
	}
	for a.settingsHelpButton.Clicked(gtx) {
		a.settingsHelpOpen = true
	}
	for a.closeSettingsHelpButton.Clicked(gtx) {
		a.settingsHelpOpen = false
	}
	for a.saveSettingsButton.Clicked(gtx) {
		if !a.settingsDraftReady() {
			continue
		}
		if err := a.saveSettingsSelection(); err != nil {
			a.appendLog("保存配置失败: " + err.Error())
			continue
		}
		a.closeSettingsModal()
	}
	for a.toggleAPIKeyMaskButton.Clicked(gtx) {
		a.apiKeyVisible = !a.apiKeyVisible
	}
	for a.settingsTestUpstreamButton.Clicked(gtx) {
		if !a.settingsDraftReady() {
			continue
		}
		if err := a.saveSettingsSelection(); err != nil {
			a.appendLog("保存配置失败: " + err.Error())
			continue
		}
		if strings.TrimSpace(a.settingsSelectedProfileID) != "" && a.settingsSelectedProfileID != a.activeProfileID {
			if err := a.activateStoredProfile(a.settingsSelectedProfileID); err != nil {
				a.appendLog("切换上游失败: " + err.Error())
				continue
			}
		}
		a.closeSettingsModal()
		a.startUpstreamProbe()
	}
	for a.createProfileButton.Clicked(gtx) {
		if err := a.createSettingsProfile(string(client.APIModeResponses)); err != nil {
			a.appendLog("创建配置失败: " + err.Error())
		}
	}
	for a.createImagesProfileButton.Clicked(gtx) {
		if err := a.createSettingsProfile(string(client.APIModeImages)); err != nil {
			a.appendLog("创建配置失败: " + err.Error())
		}
	}
	snap := a.readSnapshot()
	activeName := strings.TrimSpace(activeProfileName(snap.Profiles, snap.ActiveProfileID))
	if activeName == "" {
		activeName = "未命名配置"
	}
	activeMode := activeProfileAPIMode(snap.Profiles, snap.ActiveProfileID)
	if activeMode == "" {
		activeMode = a.api
	}
	if a.apiKeyVisible {
		a.apiKeyInput.Mask = 0
	} else {
		a.apiKeyInput.Mask = '*'
	}
	if len(snap.Profiles) == 0 {
		return a.layoutStandardModal(
			gtx,
			unit.Dp(760),
			0,
			"上游配置",
			"",
			&a.closeSettingsButton,
			a.layoutSettingsEmptyState,
		)
	}
	return a.layoutStandardModal(
		gtx,
		unit.Dp(760),
		unit.Dp(680),
		"上游配置",
		"",
		&a.closeSettingsButton,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedWidth(gtx, unit.Dp(240), func(gtx layout.Context) layout.Dimensions {
						return a.layoutSettingsProfileRail(gtx, snap)
					})
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.layoutSettingsEditorPane(gtx, snap)
				}),
			)
		},
	)
}

func (a *App) layoutSettingsHelpModal(gtx layout.Context) layout.Dimensions {
	return a.layoutStandardModal(
		gtx,
		unit.Dp(620),
		unit.Dp(640),
		"接口说明",
		"上游配置 / 常见问题",
		&a.closeSettingsHelpButton,
		func(gtx layout.Context) layout.Dimensions {
			sections := []layout.Widget{
				func(gtx layout.Context) layout.Dimensions {
					return a.helpInfoCard(gtx, "Responses API 与 Images API 怎么选?", "最关键的一条。先看你的 key 绑在哪个分组。", func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "Responses API 走 /v1/responses + image_generation 工具，SSE 保活，长推理更稳，Cloudflare 524 风险更低。", unit.Sp(11), fluent.textMuted, font.Normal)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "Images API 走标准 /v1/images/generations 与 /v1/images/edits，兼容性最广，但没有 SSE 保活。", unit.Sp(11), fluent.textMuted, font.Normal)
							}),
						)
					})
				},
				func(gtx layout.Context) layout.Dimensions {
					return a.helpInfoCard(gtx, "支持哪些上游中转站?", "", func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "不内置任何默认上游。首次打开时填写你自己的 BASE_URL、API Key，再选择 API 形态。", unit.Sp(11), fluent.textMuted, font.Normal)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "只提供 /v1/chat/completions 的中转站不兼容；本应用不发 chat 请求。", unit.Sp(11), fluent.textMuted, font.Normal)
							}),
						)
					})
				},
				func(gtx layout.Context) layout.Dimensions {
					return a.helpInfoCard(gtx, "模型 ID 怎么填?", "", func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "Responses API 会同时用到文本模型 ID 与图像模型 ID；Images API 只读取图像模型 ID。", unit.Sp(11), fluent.textMuted, font.Normal)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "留空时默认文本模型是 gpt-5.5，默认图像模型是 gpt-image-2。", unit.Sp(11), fluent.textMuted, font.Normal)
							}),
						)
					})
				},
				func(gtx layout.Context) layout.Dimensions {
					return a.helpInfoCard(gtx, "BASE_URL 与参数策略", "", func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "BASE_URL 只填中转站根地址。应用会按当前 API 形态自动拼接 /v1/...，不要手动把完整路径贴进来。", unit.Sp(11), fluent.textMuted, font.Normal)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "OpenAI 标准只发官方公开字段；兼容中转扩展会额外附带 relay 常见扩展字段，例如 seed / negative_prompt。", unit.Sp(11), fluent.textMuted, font.Normal)
							}),
						)
					})
				},
				func(gtx layout.Context) layout.Dimensions {
					return a.helpInfoCard(gtx, "生成失败 / 524 怎么办?", "", func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "优先检查 key 是否过期、是否绑对分组、余额是否足够。Responses API 通常比 Images API 更抗超时。", unit.Sp(11), fluent.textMuted, font.Normal)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "排查时可从历史结果详情或运行日志里打开 Raw response 文件，看上游实际返回。", unit.Sp(11), fluent.textMuted, font.Normal)
							}),
						)
					})
				},
				func(gtx layout.Context) layout.Dimensions {
					return a.helpInfoCard(gtx, "数据存在哪里?", "", func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "API Key 只保存在系统凭据存储中；历史记录与配置元数据在本地兼容状态文件里；生成图片默认保存在输出目录。", unit.Sp(11), fluent.textMuted, font.Normal)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "除了向你配置的上游发请求，本应用不会把这些数据上传到其他服务器。", unit.Sp(11), fluent.textMuted, font.Normal)
							}),
						)
					})
				},
				func(gtx layout.Context) layout.Dimensions {
					return a.helpInfoCard(gtx, "快捷键", "", func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.monoLabel(gtx, "Ctrl+Enter 提交生成  ·  Ctrl+T 新建标签  ·  Ctrl+W 关闭标签", unit.Sp(10), fluent.textDim, font.Normal)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.monoLabel(gtx, "Ctrl+C / Ctrl+V 复制粘贴图片  ·  Delete 删除标注  ·  Esc 关闭弹层", unit.Sp(10), fluent.textDim, font.Normal)
							}),
						)
					})
				},
			}
			return a.settingsList.Layout(gtx, len(sections), func(gtx layout.Context, index int) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, sections[index])
			})
		},
	)
}

func (a *App) helpInfoCard(gtx layout.Context, title string, hint string, body layout.Widget) layout.Dimensions {
	return a.borderedSurface(gtx, fluent.surface, fluentCardRadius, fluent.border, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, title, unit.Sp(12), fluent.text, font.SemiBold)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if strings.TrimSpace(hint) == "" {
						return layout.Dimensions{}
					}
					return a.label(gtx, hint, unit.Sp(10), fluent.textDim, font.Normal)
				}),
				layout.Rigid(body),
			)
		})
	})
}

func (a *App) layoutSettingsEmptyState(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(14))}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.borderedSurface(gtx, fluent.surface2, fluentCardRadius, fluent.border, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return fixedWidth(gtx, unit.Dp(44), func(gtx layout.Context) layout.Dimensions {
								return fixedHeight(gtx, unit.Dp(44), func(gtx layout.Context) layout.Dimensions {
									return a.borderedSurface(gtx, fluent.accentSoft, unit.Dp(14), accentAlpha(0x22), func(gtx layout.Context) layout.Dimensions {
										return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return fixedWidth(gtx, unit.Dp(20), func(gtx layout.Context) layout.Dimensions {
												return fixedHeight(gtx, unit.Dp(20), func(gtx layout.Context) layout.Dimensions {
													return uiIconSpark.Layout(gtx, fluent.accent)
												})
											})
										})
									})
								})
							})
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, "先连上一个可用上游", unit.Sp(16), fluent.text, font.SemiBold)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, "先保存一条可用的 API 中转配置，后面所有生成、编辑、提示词优化都会走这里。", unit.Sp(12), fluent.textMuted, font.Normal)
								}),
							)
						}),
					)
				})
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.surfaceButton(
						gtx,
						&a.createProfileButton,
						fluent.surface,
						fluent.accentSoft,
						fluent.border,
						fluentCardRadius,
						layout.Inset{Top: 14, Bottom: 14, Left: 14, Right: 14},
						func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.badge(gtx, "R", fluent.accentSoft, fluent.accent)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.label(gtx, "Responses API", unit.Sp(13), fluent.text, font.SemiBold)
										}),
									)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, "首选。支持 SSE 保活，长任务更稳。", unit.Sp(11), fluent.textMuted, font.Normal)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, "适合 GPT 图像链路和提示词优化。", unit.Sp(10), fluent.textDim, font.Normal)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(5))}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return fixedWidth(gtx, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
												return fixedHeight(gtx, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
													return uiIconAdd.Layout(gtx, fluent.accent)
												})
											})
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.label(gtx, "新建这类配置", unit.Sp(11), fluent.accent, font.Medium)
										}),
									)
								}),
							)
						},
					)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.surfaceButton(
						gtx,
						&a.createImagesProfileButton,
						fluent.surface,
						fluent.accentSoft,
						fluent.border,
						fluentCardRadius,
						layout.Inset{Top: 14, Bottom: 14, Left: 14, Right: 14},
						func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.badge(gtx, "I", fluent.accentSoft, fluent.accent)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.label(gtx, "Images API", unit.Sp(13), fluent.text, font.SemiBold)
										}),
									)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, "兼容性更广，接标准 generations / edits。", unit.Sp(11), fluent.textMuted, font.Normal)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, "适合只想尽快接上常规生图接口。", unit.Sp(10), fluent.textDim, font.Normal)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(5))}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return fixedWidth(gtx, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
												return fixedHeight(gtx, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
													return uiIconAdd.Layout(gtx, fluent.accent)
												})
											})
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.label(gtx, "新建这类配置", unit.Sp(11), fluent.accent, font.Medium)
										}),
									)
								}),
							)
						},
					)
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.borderedSurface(gtx, fluent.accentSoft, fluentControlRadius, accentAlpha(0x22), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, "保存后会写入系统凭据存储。之后你可以继续新增多个上游配置，再按场景切换。", unit.Sp(10), fluent.accent, font.Normal)
				})
			})
		}),
	)
}

func (a *App) layoutPromptHelperButtons(gtx layout.Context, prefix string, items []promptHelperItem) layout.Dimensions {
	rows := make([]layout.FlexChild, 0, len(items))
	for _, item := range items {
		item := item
		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				btn := a.promptButton(prefix + item.ID)
				return a.surfaceButton(
					gtx,
					btn,
					rgba(0xffffff, 0x00),
					fluent.accentSoft,
					rgba(0xffffff, 0x00),
					fluentControlRadius,
					layout.Inset{Top: 8, Bottom: 8, Left: 10, Right: 10},
					func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(3))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.singleLineLabel(gtx, item.Title, unit.Sp(11), fluent.text, font.Medium)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if item.Detail == "" {
									return layout.Dimensions{}
								}
								return a.singleLineLabel(gtx, item.Detail, unit.Sp(10), fluent.textDim, font.Normal)
							}),
						)
					},
				)
			})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
}

func (a *App) layoutSettingsOptionCards(
	gtx layout.Context,
	title string,
	options []settingsOptionChoice,
	selected string,
	buttons []widget.Clickable,
	columns int,
	set func(string),
) layout.Dimensions {
	if columns <= 0 {
		columns = 2
	}
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, title, unit.Sp(11), fluent.textMuted, font.SemiBold)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			rows := make([]layout.FlexChild, 0, (len(options)+columns-1)/columns)
			for row := 0; row < len(options); row += columns {
				row := row
				rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					cells := make([]layout.FlexChild, 0, columns)
					for col := 0; col < columns; col++ {
						idx := row + col
						if idx >= len(options) {
							cells = append(cells, layout.Flexed(1, layout.Spacer{}.Layout))
							continue
						}
						for buttons[idx].Clicked(gtx) {
							set(options[idx].Value)
						}
						cells = append(cells, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							active := options[idx].Value == selected
							return a.surfaceButton(
								gtx,
								&buttons[idx],
								chooseColor(active, fluent.accentSoft, fluent.surface),
								chooseColor(active, accentAlpha(0x28), fluent.surface2),
								fluent.border,
								unit.Dp(8),
								layout.Inset{Top: 10, Bottom: 10, Left: 12, Right: 12},
								func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(3))}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.label(gtx, options[idx].Title, unit.Sp(12), chooseColor(active, fluent.accent, fluent.text), font.SemiBold)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.label(gtx, options[idx].Detail, unit.Sp(10), chooseColor(active, fluent.accent, fluent.textDim), font.Normal)
										}),
									)
								},
							)
						}))
					}
					return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx, cells...)
					})
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
		}),
	)
}

func (a *App) layoutSettingsAPIKeyField(gtx layout.Context) layout.Dimensions {
	icon := uiIconVisibility
	if a.apiKeyVisible {
		icon = uiIconVisibilityOff
	}
	border := fluent.border2
	if gtx.Focused(&a.apiKeyInput) {
		border = accentAlpha(0xb8)
	}
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, "API Key", unit.Sp(11), fluent.textMuted, font.SemiBold)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedHeight(gtx, unit.Dp(44), func(gtx layout.Context) layout.Dimensions {
				return a.borderedSurface(gtx, fluent.surface, fluentControlRadius, border, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: 9, Bottom: 9, Left: 12, Right: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								style := material.Editor(a.th, &a.apiKeyInput, "sk-...")
								style.Color = fluent.text
								style.HintColor = fluent.textDim
								style.SelectionColor = accentAlpha(0x3d)
								style.TextSize = unit.Sp(13)
								style.Font.Weight = font.Medium
								style.Font.Typeface = uiMonoTypeface
								return style.Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.ghostIconButton(gtx, &a.toggleAPIKeyMaskButton, icon, a.apiKeyVisible)
							}),
						)
					})
				})
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, "API Key 保存在系统凭据存储中，不写入本地状态文件。", unit.Sp(10), fluent.textDim, font.Normal)
		}),
	)
}

func (a *App) layoutSettingsProfileRail(gtx layout.Context, snap snapshot) layout.Dimensions {
	for _, profile := range snap.Profiles {
		btn := a.settingsProfileButton("settings-profile:" + profile.ID)
		profile := profile
		for btn.Clicked(gtx) {
			if err := a.loadSettingsProfileDraft(profile.ID); err != nil {
				a.appendLog("读取配置失败: " + err.Error())
			}
		}
	}
	for a.duplicateProfileButton.Clicked(gtx) {
		if err := a.duplicateSettingsProfile(); err != nil {
			a.appendLog("复制配置失败: " + err.Error())
		}
	}
	for a.deleteProfileButton.Clicked(gtx) {
		if err := a.deleteSettingsProfile(); err != nil {
			a.appendLog("删除配置失败: " + err.Error())
		}
	}
	for a.settingsActivateProfileButton.Clicked(gtx) {
		if err := a.activateStoredProfile(snap.SettingsSelectedProfileID); err != nil {
			a.appendLog("切换上游失败: " + err.Error())
		}
	}
	activeName := strings.TrimSpace(activeProfileName(snap.Profiles, snap.ActiveProfileID))
	if activeName == "" {
		activeName = "未命名配置"
	}
	selectedID := strings.TrimSpace(snap.SettingsSelectedProfileID)
	if selectedID == "" {
		selectedID = strings.TrimSpace(snap.ActiveProfileID)
	}

	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, "配置列表", unit.Sp(11), fluent.textMuted, font.SemiBold)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if len(snap.Profiles) == 0 {
						return layout.Dimensions{}
					}
					return a.metaBadge(gtx, fmt.Sprintf("%d 项", len(snap.Profiles)), true)
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, "当前生效: "+activeName, unit.Sp(10), fluent.textDim, font.Normal)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedHeight(gtx, unit.Dp(460), func(gtx layout.Context) layout.Dimensions {
				return a.borderedSurface(gtx, fluent.surface, unit.Dp(8), fluent.border, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						if len(snap.Profiles) == 0 {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "还没有配置,点下方新建开始。", unit.Sp(10), fluent.textDim, font.Normal)
							})
						}
						return a.settingsProfileList.Layout(gtx, len(snap.Profiles), func(gtx layout.Context, index int) layout.Dimensions {
							profile := snap.Profiles[index]
							return layout.Inset{Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								btn := a.settingsProfileButton("settings-profile:" + profile.ID)
								selected := profile.ID == selectedID
								active := profile.ID == snap.ActiveProfileID
								modeLabel := "Responses API"
								if strings.TrimSpace(profile.APIMode) == "images" {
									modeLabel = "Images API"
								}
								return a.surfaceButton(
									gtx,
									btn,
									chooseColor(selected, fluent.accentSoft, rgba(0xffffff, 0x00)),
									chooseColor(selected, accentAlpha(0x28), fluent.surface2),
									chooseColor(selected, accentAlpha(0x24), rgba(0xffffff, 0x00)),
									unit.Dp(8),
									layout.Inset{Top: 8, Bottom: 8, Left: 8, Right: 8},
									func(gtx layout.Context) layout.Dimensions {
										return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												dot := fluent.textDim
												if active {
													dot = fluent.accent
												} else if selected {
													dot = withAlpha(fluent.accent, 0xb8)
												}
												return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													return fixedWidth(gtx, unit.Dp(8), func(gtx layout.Context) layout.Dimensions {
														return fixedHeight(gtx, unit.Dp(8), func(gtx layout.Context) layout.Dimensions {
															return a.surface(gtx, dot, unit.Dp(4), layout.Spacer{}.Layout)
														})
													})
												})
											}),
											layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
												return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(2))}.Layout(gtx,
													layout.Rigid(func(gtx layout.Context) layout.Dimensions {
														return a.clampedLabel(gtx, strings.TrimSpace(profile.Name), unit.Sp(12), chooseColor(selected, fluent.accent, fluent.text), font.Medium, 2)
													}),
													layout.Rigid(func(gtx layout.Context) layout.Dimensions {
														return a.singleLineLabel(gtx, modeLabel, unit.Sp(10), chooseColor(selected, fluent.accent, fluent.textDim), font.Normal)
													}),
												)
											}),
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												modeTag := "R"
												if strings.TrimSpace(profile.APIMode) == "images" {
													modeTag = "I"
												}
												return a.metaBadge(gtx, modeTag, true)
											}),
										)
									},
								)
							})
						})
					})
				})
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.compactIconTextButton(gtx, &a.createProfileButton, uiIconAdd, "新建", false)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.compactIconButton(gtx, &a.duplicateProfileButton, uiIconCopy, false)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.compactIconButton(gtx, &a.deleteProfileButton, uiIconDelete, false)
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if selectedID == "" || selectedID == snap.ActiveProfileID {
				return layout.Dimensions{}
			}
			return a.compactButton(gtx, &a.settingsActivateProfileButton, "设为当前激活", true)
		}),
	)
}

func (a *App) layoutSettingsEditorPane(gtx layout.Context, snap snapshot) layout.Dimensions {
	selectedID := strings.TrimSpace(snap.SettingsSelectedProfileID)
	if selectedID == "" {
		selectedID = strings.TrimSpace(snap.ActiveProfileID)
	}
	selectedName := strings.TrimSpace(activeProfileName(snap.Profiles, selectedID))
	if selectedName == "" {
		selectedName = "未命名配置"
	}
	selectedMode := activeProfileAPIMode(snap.Profiles, selectedID)
	if selectedMode == "" {
		selectedMode = a.api
	}
	activeName := strings.TrimSpace(activeProfileName(snap.Profiles, snap.ActiveProfileID))
	if activeName == "" {
		activeName = "未命名配置"
	}
	activeMode := activeProfileAPIMode(snap.Profiles, snap.ActiveProfileID)
	if activeMode == "" {
		activeMode = a.api
	}
	connectionSection := func(gtx layout.Context) layout.Dimensions {
		rows := []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.field(gtx, "名称", &a.profileNameInput, "配置1", unit.Dp(44))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.layoutSettingsOptionCards(gtx, "API 形态", []settingsOptionChoice{
					{Title: "Responses API", Detail: "SSE 保活，更适合长推理", Value: string(client.APIModeResponses)},
					{Title: "Images API", Detail: "标准 generations / edits", Value: string(client.APIModeImages)},
				}, a.api, a.apiButtons, 2, func(value string) { a.api = value })
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.layoutSettingsOptionCards(gtx, "参数策略", []settingsOptionChoice{
					{Title: "OpenAI 标准", Detail: "只发送官方公开字段", Value: string(client.RequestPolicyOpenAI)},
					{Title: "兼容中转扩展", Detail: "附带 seed / negative_prompt", Value: string(client.RequestPolicyCompat)},
				}, a.policy, a.policyButtons, 1, func(value string) { a.policy = value })
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.technicalField(gtx, "上游 BASE_URL", &a.baseURLInput, "https://example.com", unit.Dp(44))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.layoutSettingsAPIKeyField(gtx)
			}),
		}
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx, rows...)
	}
	runtimeSection := func(gtx layout.Context) layout.Dimensions {
		rows := []layout.FlexChild{}
		canSave := strings.TrimSpace(a.baseURLInput.Text()) != "" && strings.TrimSpace(a.apiKeyInput.Text()) != ""
		if a.api == string(client.APIModeResponses) {
			rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.technicalField(gtx, "文本模型 ID", &a.textModelInput, client.TextModel, unit.Dp(44))
			}))
		}
		rows = append(rows,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.technicalField(gtx, "图像模型 ID", &a.imageModelInput, client.ImageModel, unit.Dp(44))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.technicalField(gtx, "并发数量限制", &a.concurrencyLimitInput, "留空 = 不限制", unit.Dp(44))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, "0/留空 = 不限制。填正整数后，此配置跨所有标签页最多同时运行这么多任务。", unit.Sp(10), fluent.textDim, font.Normal)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.segmentedWithTitle(gtx, "代理", proxyChoices, a.proxy, a.proxyButtons, func(value string) { a.proxy = value })
			}),
		)
		if a.proxy == "custom" || strings.TrimSpace(a.proxyURLInput.Text()) != "" {
			rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.technicalField(gtx, "自定义代理 URL", &a.proxyURLInput, "http://127.0.0.1:7890", unit.Dp(44))
			}))
		}
		rows = append(rows,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.technicalField(gtx, "输出目录", &a.outputDirInput, "生成图片保存目录", unit.Dp(44))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				label := "保存并测试连接"
				if snap.TestingUpstream {
					label = "测试中..."
				}
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				if !canSave {
					return a.surfaceButton(
						gtx,
						&a.settingsTestUpstreamButton,
						fluent.surface2,
						fluent.surface2,
						rgba(0xffffff, 0x00),
						fluentControlRadius,
						layout.Inset{Top: 9, Bottom: 9, Left: 12, Right: 12},
						func(gtx layout.Context) layout.Dimensions {
							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, label, unit.Sp(12), withAlpha(fluent.textDim, 0x88), font.SemiBold)
							})
						},
					)
				}
				return a.primaryButton(gtx, &a.settingsTestUpstreamButton, label, fluent.accent, fluent.white)
			}),
		)
		if strings.TrimSpace(snap.LastProbeSummary) != "" {
			rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, snap.LastProbeSummary, unit.Sp(10), fluent.textDim, font.Normal)
			}))
		}
		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if a.api == string(client.APIModeResponses) {
				return a.label(gtx, "Responses API 更适合长推理和需要 SSE 保活的上游。", unit.Sp(10), fluent.textDim, font.Normal)
			}
			return a.borderedSurface(gtx, fluent.accentSoft, unit.Dp(6), accentAlpha(0x28), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, "Images API 兼容性最广，但没有 SSE 保活，长推理更容易遇到超时。", unit.Sp(10), fluent.accent, font.Normal)
				})
			})
		}))
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx, rows...)
	}
	children := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(3))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "上游配置编辑", unit.Sp(11), fluent.text, font.SemiBold)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, selectedName+" · "+apiChoiceLabel(selectedMode), unit.Sp(10), fluent.textDim, font.Normal)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if selectedID == snap.ActiveProfileID {
								return a.label(gtx, "保存后立即更新当前生效配置；API Key 仍保存在系统凭据存储。", unit.Sp(10), fluent.textDim, font.Normal)
							}
							return a.label(gtx, "当前生效: "+activeName+" · "+apiChoiceLabel(activeMode), unit.Sp(10), fluent.textDim, font.Normal)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.compactIconTextButton(gtx, &a.settingsHelpButton, uiIconInfo, "接口说明", true)
				}),
			)
		},
		connectionSection,
		runtimeSection,
	}
	children = append(children, func(gtx layout.Context) layout.Dimensions {
		canSave := a.settingsDraftReady()
		return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return fixedWidth(gtx, unit.Dp(110), func(gtx layout.Context) layout.Dimensions {
					return a.compactButton(gtx, &a.closeSettingsButton, "关闭", false)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if !canSave {
					return a.label(gtx, "BASE_URL 和 API Key 必须填齐才能保存。", unit.Sp(10), fluent.textDim, font.Normal)
				}
				return layout.Dimensions{}
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return fixedWidth(gtx, unit.Dp(120), func(gtx layout.Context) layout.Dimensions {
					if !canSave {
						return a.surfaceButton(
							gtx,
							&a.saveSettingsButton,
							fluent.surface2,
							fluent.surface2,
							rgba(0xffffff, 0x00),
							fluentControlRadius,
							layout.Inset{Top: 8, Bottom: 8, Left: 10, Right: 10},
							func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return fixedWidth(gtx, unit.Dp(16), func(gtx layout.Context) layout.Dimensions {
											return fixedHeight(gtx, unit.Dp(16), func(gtx layout.Context) layout.Dimensions {
												return uiIconSave.Layout(gtx, withAlpha(fluent.textDim, 0x88))
											})
										})
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.label(gtx, "保存", unit.Sp(12), withAlpha(fluent.textDim, 0x88), font.Medium)
									}),
								)
							},
						)
					}
					return a.primaryButton(gtx, &a.saveSettingsButton, "保存", fluent.accent, fluent.white)
				})
			}),
		)
	})
	return a.settingsList.Layout(gtx, len(children), func(gtx layout.Context, index int) layout.Dimensions {
		return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, children[index])
	})
}

func (a *App) layoutComposeCard(gtx layout.Context) layout.Dimensions {
	activeAspect := deriveAspectPreset(a.size)
	activeResolution := normalizeResolutionChoice(deriveResolutionPreset(a.size), a.api, a.policy, a.imageModelInput.Text())
	currentSaved := strings.TrimSpace(a.readSnapshot().Result.SavedPath)
	sourcePaths := kernel.ParseSourcePaths(a.sourcePathsInput.Text())
	sourceLabel := "文生图"
	if a.mode == string(client.ModeEdit) {
		count := len(sourcePaths)
		if count > 0 {
			sourceLabel = fmt.Sprintf("%d 张源图", count)
		} else if currentSaved != "" {
			sourceLabel = "画板图作源图"
		} else {
			sourceLabel = "未添加源图"
		}
	}
	summary := strings.Join(compactNonEmpty([]string{
		chooseStyleSummary(a.styleTag),
		aspectChoiceLabel(activeAspect),
		choiceLabel(resolutionChoices, activeResolution),
		qualityChoiceLabel(a.quality),
		fmt.Sprintf("%d 张", normalizeBatchCount(a.batchCount)),
		sourceLabel,
	}), " · ")

	return a.borderedSurface(gtx, fluent.surfaceElevated, fluentCardRadius, fluent.border, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			children := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.layoutComposeAccordionHeader(gtx, summary, a.composeOpen)
				}),
			}
			if a.composeOpen {
				children = append(children,
					layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.composeSectionCard(gtx, a.layoutStyleSection)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.composeSectionCard(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.layoutAspectSection(gtx, activeAspect, activeResolution)
						})
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.composeSectionCard(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.layoutResolutionSection(gtx, activeAspect, activeResolution)
						})
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.composeSectionCard(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, "质量", unit.Sp(11), fluent.textMuted, font.SemiBold)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.segmented(gtx, qualityChoices, a.quality, a.qualityButtons, func(value string) { a.quality = value })
								}),
							)
						})
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.composeSectionCard(gtx, a.layoutBatchCountSection)
					}),
				)
				if a.mode == string(client.ModeEdit) {
					children = append(children,
						layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.composeSectionCard(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.layoutSourceInputSection(gtx, sourcePaths, currentSaved)
							})
						}),
					)
				}
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}

func (a *App) composeSectionCard(gtx layout.Context, body layout.Widget) layout.Dimensions {
	return a.borderedSurface(gtx, fluent.surfaceElevated, unit.Dp(8), fluent.border, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, body)
	})
}

func (a *App) layoutAspectSection(gtx layout.Context, activeAspect string, currentResolution string) layout.Dimensions {
	children := make([]layout.FlexChild, 0, 2)
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return a.label(gtx, "比例", unit.Sp(11), fluent.textMuted, font.SemiBold)
	}))
	children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout))

	rows := 2
	cols := 3
	for row := 0; row < rows; row++ {
		row := row
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			cells := make([]layout.FlexChild, 0, cols)
			for col := 0; col < cols; col++ {
				idx := row*cols + col
				if idx >= len(aspectChoices) {
					cells = append(cells, layout.Flexed(1, layout.Spacer{}.Layout))
					continue
				}
				choice := aspectChoices[idx]
				for a.aspectButtons[idx].Clicked(gtx) {
					a.size = buildAspectSizeSelection(choice.Value, currentResolution, a.api, a.policy, a.imageModelInput.Text())
				}
				cells = append(cells, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.layoutAspectButton(gtx, &a.aspectButtons[idx], choice, activeAspect == choice.Value)
				}))
			}
			return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx, cells...)
			})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (a *App) layoutStyleSection(gtx layout.Context) layout.Dimensions {
	for a.clearStyleButton.Clicked(gtx) {
		a.styleTag = ""
	}
	children := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, "风格", unit.Sp(11), fluent.textMuted, font.SemiBold)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if a.styleTag == "" {
						return a.metaBadge(gtx, "默认风格", true)
					}
					return a.ghostIconTextButton(gtx, &a.clearStyleButton, uiIconClear, "清除", true)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
	}
	rows := [][]choice{
		styleChoices[:3],
		styleChoices[3:],
	}
	base := 0
	for _, rowChoices := range rows {
		rowChoices := rowChoices
		rowStart := base
		base += len(rowChoices)
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			row := make([]layout.FlexChild, 0, len(rowChoices))
			for idx := range rowChoices {
				choice := rowChoices[idx]
				btnIdx := rowStart + idx
				row = append(row, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn := &a.styleButtons[btnIdx]
					for btn.Clicked(gtx) {
						if a.styleTag == choice.Value {
							a.styleTag = ""
						} else {
							a.styleTag = choice.Value
						}
					}
					return layout.Inset{Right: unit.Dp(8), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.surfaceButton(
							gtx,
							btn,
							chooseColor(a.styleTag == choice.Value, fluent.accentSoft, rgba(0xffffff, 0x00)),
							chooseColor(a.styleTag == choice.Value, accentAlpha(0x28), fluent.surface2),
							chooseColor(a.styleTag == choice.Value, accentAlpha(0x28), fluent.border),
							fluentControlRadius,
							layout.Inset{Top: 6, Bottom: 6, Left: 10, Right: 10},
							func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, choice.Label, unit.Sp(11), chooseColor(a.styleTag == choice.Value, fluent.accent, fluent.textMuted), font.Medium)
							},
						)
					})
				}))
			}
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, row...)
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (a *App) layoutSourceInputSection(gtx layout.Context, sourcePaths []string, currentSaved string) layout.Dimensions {
	for a.addSourceFilesButton.Clicked(gtx) {
		paths, err := chooseImageFiles()
		if err != nil {
			a.appendLog("选择源图失败: " + err.Error())
		} else {
			for _, path := range paths {
				a.appendSourcePath(path)
			}
		}
	}
	for a.clearSourcesButton.Clicked(gtx) {
		a.setSourcePaths(nil)
	}
	for _, path := range sourcePaths {
		path := path
		btn := a.sourceButton("panel-remove:" + path)
		for btn.Clicked(gtx) {
			a.removeSourcePath(path)
		}
	}

	children := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, "源图片 / 参考图", unit.Sp(11), fluent.textMuted, font.SemiBold)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if len(sourcePaths) == 0 {
						return layout.Dimensions{}
					}
					return a.metaBadge(gtx, fmt.Sprintf("%d 张", len(sourcePaths)), true)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
	}

	if len(sourcePaths) == 0 && currentSaved != "" {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.borderedSurface(gtx, fluent.surface2, fluentControlRadius, fluent.border, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.singleLineLabel(gtx, "(画板当前图 · 隐式源图)", unit.Sp(10), fluent.textDim, font.Normal)
				})
			})
		}))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
	}

	for _, path := range sourcePaths {
		path := path
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := a.sourceButton("panel-remove:" + path)
			return a.borderedSurface(gtx, fluent.surface, fluentControlRadius, fluent.border, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 8, Bottom: 8, Left: 10, Right: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							idx := indexOfSourcePath(path, sourcePaths) + 1
							if idx <= 0 {
								return layout.Dimensions{}
							}
							return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, strconv.Itoa(idx)+".", unit.Sp(10), fluent.textDim, font.Medium)
							})
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.singleLineLabel(gtx, filepath.Base(path), unit.Sp(11), fluent.text, font.Medium)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.compactIconTextButton(gtx, btn, uiIconDelete, "移除", false)
						}),
					)
				})
			})
		}))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
	}

	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		row := []layout.FlexChild{}
		row = append(row, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.compactIconTextButton(gtx, &a.addSourceFilesButton, uiIconAdd, "添加图片", false)
		}))
		if len(sourcePaths) > 0 {
			row = append(row, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.compactIconTextButton(gtx, &a.clearSourcesButton, uiIconDelete, "清空", false)
				})
			}))
		}
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, row...)
	}))

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func indexOfSourcePath(path string, sourcePaths []string) int {
	for idx, value := range sourcePaths {
		if value == path {
			return idx
		}
	}
	return -1
}

func (a *App) layoutResolutionSection(gtx layout.Context, activeAspect string, activeResolution string) layout.Dimensions {
	choices := visibleResolutionChoices(a.api, a.policy, a.imageModelInput.Text())
	children := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, "分辨率", unit.Sp(11), fluent.textMuted, font.SemiBold)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
	}
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return a.segmented(gtx, choices, activeResolution, a.resolutionButtons, func(value string) {
			a.size = buildResolutionSizeSelection(activeAspect, value, a.api, a.policy, a.imageModelInput.Text())
		})
	}))
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		hint := sizeCapabilityHint(a.api, a.policy, a.imageModelInput.Text())
		if hint == "" {
			return layout.Dimensions{}
		}
		return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, hint, unit.Sp(10), fluent.textDim, font.Normal)
		})
	}))
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (a *App) layoutBatchCountSection(gtx layout.Context) layout.Dimensions {
	batchCount := normalizeBatchCount(a.batchCount)
	children := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, "出图张数", unit.Sp(11), fluent.textMuted, font.SemiBold)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.metaBadge(gtx, fmt.Sprintf("%dx", batchCount), true)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
	}
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return a.segmentedGrid(gtx, batchCountChoices, strconv.Itoa(batchCount), a.batchCountButtons, 3, func(value string) {
			n, _ := strconv.Atoi(value)
			a.batchCount = normalizeBatchCount(n)
		})
	}))
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{}
	}))
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (a *App) layoutAdvancedCard(gtx layout.Context) layout.Dimensions {
	summary := strings.Join(compactNonEmpty([]string{
		negativePromptSummary(a.negativePromptInput.Text()),
		strings.ToUpper(strings.TrimSpace(a.format)),
		seedSummary(a.seedInput.Text()),
	}), " · ")

	return a.borderedSurface(gtx, fluent.surfaceElevated, fluentCardRadius, fluent.border, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			children := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.layoutAdvancedAccordionHeader(gtx, summary, a.advancedOpen)
				}),
			}
			if a.advancedOpen {
				children = append(children,
					layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.advancedSectionCard(gtx, "负向提示词", "", func(gtx layout.Context) layout.Dimensions {
							border := fluent.border2
							if gtx.Focused(&a.negativePromptInput) {
								border = accentAlpha(0xb8)
							}
							return fixedHeight(gtx, unit.Dp(96), func(gtx layout.Context) layout.Dimensions {
								return a.borderedSurface(gtx, fluent.surface, fluentControlRadius, border, func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Top: 10, Bottom: 10, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return a.editorText(gtx, &a.negativePromptInput, "例如：不要文字、不要水印、不要多余肢体、不要过曝", unit.Sp(13))
									})
								})
							})
						})
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.advancedSectionCard(gtx, "输出格式", "JPEG/WebP 体积更小；落盘扩展名 jpeg -> .jpg", func(gtx layout.Context) layout.Dimensions {
							return a.segmented(gtx, formatChoices, a.format, a.formatButtons, func(value string) { a.format = value })
						})
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.advancedSectionCard(gtx, "随机种子", "", func(gtx layout.Context) layout.Dimensions {
							for a.randomSeedButton.Clicked(gtx) {
								a.seedInput.SetText(strconv.FormatInt(time.Now().UnixNano()%1000000007, 10))
							}
							for a.clearSeedButton.Clicked(gtx) {
								a.seedInput.SetText("")
							}
							return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
										layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
											return a.field(gtx, "Seed", &a.seedInput, "0", unit.Dp(42))
										}),
										layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
											return a.field(gtx, "Partial", &a.partialImagesInput, "1", unit.Dp(42))
										}),
									)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
										layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
											return a.compactIconTextButton(gtx, &a.randomSeedButton, uiIconRefresh, "随机", false)
										}),
										layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
											return a.compactIconTextButton(gtx, &a.clearSeedButton, uiIconClear, "清空", false)
										}),
									)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, "留空或 0 使用随机种子；Partial 仅影响流式预览帧数。", unit.Sp(10), fluent.textDim, font.Normal)
								}),
							)
						})
					}),
				)
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}

func (a *App) layoutAspectButton(gtx layout.Context, btn *widget.Clickable, choice aspectChoice, active bool) layout.Dimensions {
	return a.surfaceButton(
		gtx,
		btn,
		chooseColor(active, fluent.accentSoft, fluent.surface),
		chooseColor(active, accentAlpha(0x28), fluent.surface2),
		fluent.border,
		unit.Dp(6),
		layout.Inset{Top: 8, Bottom: 8, Left: 8, Right: 8},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return fixedWidth(gtx, unit.Dp(float32(choice.W)+10), func(gtx layout.Context) layout.Dimensions {
							return fixedHeight(gtx, unit.Dp(float32(choice.H)+10), func(gtx layout.Context) layout.Dimensions {
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return a.borderedSurface(gtx, chooseColor(active, fluent.surface, fluent.panel2), unit.Dp(3), chooseColor(active, fluent.accent, fluent.textDim), func(gtx layout.Context) layout.Dimensions {
										return fixedWidth(gtx, unit.Dp(float32(choice.W)), func(gtx layout.Context) layout.Dimensions {
											return fixedHeight(gtx, unit.Dp(float32(choice.H)), func(gtx layout.Context) layout.Dimensions {
												return layout.Dimensions{Size: gtx.Constraints.Min}
											})
										})
									})
								})
							})
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, choice.Label, unit.Sp(10), chooseColor(active, fluent.accent, fluent.textMuted), font.Medium)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !choice.Auto {
							return layout.Dimensions{}
						}
						return a.singleLineLabel(gtx, "让上游决定尺寸", unit.Sp(9), chooseColor(active, fluent.accent, fluent.textDim), font.Normal)
					}),
				)
			})
		},
	)
}

func (a *App) layoutActions(gtx layout.Context) layout.Dimensions {
	snap := a.readSnapshot()
	ready := strings.TrimSpace(a.apiKeyInput.Text()) != "" && strings.TrimSpace(a.baseURLInput.Text()) != ""
	children := make([]layout.FlexChild, 0, 4)
	if !ready {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.borderedSurface(gtx, fluent.accentSoft, unit.Dp(8), accentAlpha(0x22), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "还没有可用上游配置", unit.Sp(11), fluent.accent, font.SemiBold)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "先配置 BASE_URL 和 API Key，才能测试连接或开始生成。", unit.Sp(10), fluent.accent, font.Normal)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.surfaceButton(
									gtx,
									&a.manageUpstreamButton,
									withAlpha(fluent.white, 0xe6),
									fluent.surface,
									accentAlpha(0x1c),
									unit.Dp(8),
									layout.Inset{Top: 6, Bottom: 6, Left: 10, Right: 10},
									func(gtx layout.Context) layout.Dimensions {
										return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return fixedWidth(gtx, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
													return fixedHeight(gtx, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
														return uiIconSettings.Layout(gtx, fluent.accent)
													})
												})
											}),
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return a.label(gtx, "配置上游", unit.Sp(10), fluent.accent, font.Medium)
											}),
										)
									},
								)
							})
						}),
					)
				})
			})
		}))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout))
	}

	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		if !ready {
			return a.submitActionButton(gtx, &a.manageUpstreamButton, "配置上游", fluent.accent, fluent.accent2, accentAlpha(0x58), fluent.white)
		}
		if snap.Running {
			return a.submitActionButton(gtx, &a.cancelButton, "取消生成", fluent.dangerSoft, dangerAlpha(0x2a), dangerAlpha(0x30), fluent.danger)
		}
		label := "生成"
		if a.mode == string(client.ModeEdit) {
			label = "编辑"
		}
		return a.submitActionButton(gtx, &a.runButton, label, fluent.accent, fluent.accent2, accentAlpha(0x58), fluent.white)
	}))
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (a *App) layoutDisclosureHeader(gtx layout.Context, btn *widget.Clickable, title string, summary string, open bool) layout.Dimensions {
	stateText := "展开"
	stateIcon := uiIconExpand
	if open {
		stateText = "收起"
		stateIcon = uiIconCollapse
	}
	return a.surfaceButton(
		gtx,
		btn,
		chooseColor(open, fluent.surface2, fluent.surface),
		fluent.surface2,
		fluent.border,
		fluentCardRadius,
		layout.Inset{Top: 12, Bottom: 12, Left: 12, Right: 12},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.sectionEyebrow(gtx, title)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.singleLineLabel(gtx, summary, unit.Sp(12), fluent.textMuted, font.Normal)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return fixedWidth(gtx, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
								return fixedHeight(gtx, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
									return stateIcon.Layout(gtx, fluent.textDim)
								})
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, stateText, unit.Sp(11), fluent.textDim, font.Medium)
						}),
					)
				}),
			)
		},
	)
}

func (a *App) layoutComposeAccordionHeader(gtx layout.Context, summary string, open bool) layout.Dimensions {
	stateText := "展开"
	stateIcon := uiIconExpand
	if open {
		stateText = "收起"
		stateIcon = uiIconCollapse
	}
	return a.composeToggleButton.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		fg := fluent.textMuted
		if a.composeToggleButton.Hovered() {
			fg = fluent.text
		}
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, "创作参数", unit.Sp(11), fluent.text, font.SemiBold)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.singleLineLabel(gtx, summary, unit.Sp(12), fluent.textMuted, font.Normal)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.surface(gtx, chooseColor(a.composeToggleButton.Hovered(), fluent.toolHoverBg, rgba(0xffffff, 0x00)), fluentControlRadius, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: 6, Bottom: 6, Left: 8, Right: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return fixedWidth(gtx, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
									return fixedHeight(gtx, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
										return stateIcon.Layout(gtx, fg)
									})
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, stateText, unit.Sp(12), fg, font.Normal)
							}),
						)
					})
				})
			}),
		)
	})
}

func (a *App) layoutAdvancedAccordionHeader(gtx layout.Context, summary string, open bool) layout.Dimensions {
	stateText := "展开"
	stateIcon := uiIconExpand
	if open {
		stateText = "收起"
		stateIcon = uiIconCollapse
	}
	return a.surfaceButton(
		gtx,
		&a.advancedToggleButton,
		chooseColor(open, fluent.surface2, fluent.surfaceElevated),
		fluent.surface2,
		fluent.border,
		fluentCardRadius,
		layout.Inset{Top: 10, Bottom: 10, Left: 12, Right: 12},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "高级参数", unit.Sp(11), fluent.textMuted, font.SemiBold)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.singleLineLabel(gtx, summary, unit.Sp(12), fluent.textMuted, font.Normal)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return fixedWidth(gtx, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
								return fixedHeight(gtx, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
									return stateIcon.Layout(gtx, fluent.textDim)
								})
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, stateText, unit.Sp(12), fluent.textDim, font.Normal)
						}),
					)
				}),
			)
		},
	)
}

func (a *App) advancedSectionCard(gtx layout.Context, title string, hint string, body layout.Widget) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, title, unit.Sp(11), fluent.textMuted, font.SemiBold)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.borderedSurface(gtx, fluent.surface, unit.Dp(8), fluent.border, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
						layout.Rigid(body),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if strings.TrimSpace(hint) == "" {
								return layout.Dimensions{}
							}
							return a.label(gtx, hint, unit.Sp(10), fluent.textDim, font.Normal)
						}),
					)
				})
			})
		}),
	)
}

func (a *App) editorText(gtx layout.Context, editor *widget.Editor, hint string, size unit.Sp) layout.Dimensions {
	style := material.Editor(a.th, editor, hint)
	style.Color = fluent.text
	style.HintColor = fluent.textDim
	style.SelectionColor = accentAlpha(0x3d)
	style.TextSize = size
	return style.Layout(gtx)
}

func apiChoiceLabel(value string) string {
	return choiceLabel(apiChoices, value)
}

func policyChoiceLabel(value string) string {
	return choiceLabel(policyChoices, value)
}

func proxyChoiceLabel(value string) string {
	return choiceLabel(proxyChoices, value)
}

func negativePromptSummary(value string) string {
	if strings.TrimSpace(value) == "" {
		return "无负向限制"
	}
	return "已填负向提示词"
}

func seedSummary(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "0" {
		return "随机 Seed"
	}
	return "Seed " + value
}

func presetLabels(presets []sharedCompat.Preset) []promptHelperItem {
	items := make([]promptHelperItem, 0, len(presets))
	for _, preset := range presets {
		detail := strings.Join(compactNonEmpty([]string{
			preset.Size,
			preset.Quality,
			strings.ToUpper(strings.TrimSpace(preset.OutputFormat)),
			fmt.Sprintf("%d 张", normalizeBatchCount(preset.BatchCount)),
		}), " · ")
		items = append(items, promptHelperItem{
			ID:     preset.ID,
			Title:  strings.TrimSpace(preset.Name),
			Detail: detail,
		})
	}
	return items
}

func promptLabels(values []string) []promptHelperItem {
	items := make([]promptHelperItem, 0, len(values))
	for idx, value := range values {
		items = append(items, promptHelperItem{
			ID:     fmt.Sprintf("%d", idx),
			Title:  shortPrompt(value),
			Detail: value,
		})
	}
	return items
}
