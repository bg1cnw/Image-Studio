package ui

import (
	"fmt"
	"image"
	"path/filepath"
	"strconv"
	"strings"

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
		a.settingsModalOpen = true
	}

	return a.borderedSurface(gtx, fluent.sidebar, unit.Dp(0), fluent.border, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		return layout.Inset{Top: 12, Bottom: 12, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.controlsList.Layout(gtx, 1, func(gtx layout.Context, _ int) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
							layout.Rigid(a.layoutWorkbenchCard),
							layout.Rigid(a.layoutModeCard),
							layout.Rigid(a.layoutPromptCard),
							layout.Rigid(a.layoutComposeCard),
							layout.Rigid(a.layoutAdvancedCard),
						)
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
	return a.borderedSurface(gtx, fluent.sidebar, fluentCardRadius, rgba(0xffffff, 0x00), func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 4, Bottom: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.layoutActions(gtx)
		})
	})
}

func (a *App) layoutWorkbenchCard(gtx layout.Context) layout.Dimensions {
	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "图像工作台", unit.Sp(18), fluent.text, font.SemiBold)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "保持界面简洁，把注意力留给 prompt、参考图和结果。", unit.Sp(11), fluent.textMuted, font.Normal)
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.badge(gtx, a.modeLabel(), fluent.accentSoft, fluent.accent)
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
				return a.sectionEyebrow(gtx, "模式")
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.segmented(gtx, modeChoices, a.mode, a.modeButtons, func(value string) { a.mode = value })
			}),
		)
	})
}

func (a *App) layoutPromptCard(gtx layout.Context) layout.Dimensions {
	snap := a.readSnapshot()
	promptSuggestions := buildPromptSuggestions(snap.PromptHistory, snap.History)
	defaultPromptHelperTab := "templates"
	if len(snap.Presets) == 0 && len(promptSuggestions) > 0 {
		defaultPromptHelperTab = "history"
	}
	for a.promptHelperButton.Clicked(gtx) {
		if !a.promptHelperOpen {
			a.promptHelperTab = defaultPromptHelperTab
		}
		a.promptHelperOpen = !a.promptHelperOpen
	}
	for a.optimizePromptButton.Clicked(gtx) {
		a.startPromptOptimize()
	}
	for idx := range promptSuggestions {
		btn := a.promptButton(fmt.Sprintf("prompt-history:%d", idx))
		text := promptSuggestions[idx]
		for btn.Clicked(gtx) {
			a.applyPromptSuggestion(text)
		}
	}
	for _, preset := range snap.Presets {
		btn := a.promptButton("preset:" + preset.ID)
		preset := preset
		for btn.Clicked(gtx) {
			a.applyPreset(preset)
		}
	}

	promptLen := len([]rune(strings.TrimSpace(a.promptInput.Text())))
	title := "提示词"
	hint := "主体 / 场景 / 光照 / 镜头 / 风格"
	if a.mode == string(client.ModeEdit) {
		title = "修改要求"
		hint = "主体保持不变，替换背景或补充材质、光照、构图要求"
	}

	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		base := func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, title, unit.Sp(11), fluent.textMuted, font.Bold)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, fmt.Sprintf("%d", promptLen), unit.Sp(11), fluent.textDim, font.Medium)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedHeight(gtx, unit.Dp(124), func(gtx layout.Context) layout.Dimensions {
						return a.borderedSurface(gtx, fluent.surface, fluentControlRadius, fluent.border2, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Top: 10, Bottom: 10, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.editorText(gtx, &a.promptInput, hint, unit.Sp(13))
							})
						})
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.compactIconTextButton(gtx, &a.promptHelperButton, uiIconHistory, "模板 / 历史", a.promptHelperOpen)
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
							return a.compactIconTextButton(gtx, &a.optimizePromptButton, icon, label, snap.OptimizingPrompt)
						}),
						layout.Flexed(1, layout.Spacer{}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.staticPill(gtx, "Ctrl+Enter", false, true)
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
					return a.layoutPromptHelperInline(gtx, snap.Presets, promptSuggestions)
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

func (a *App) layoutPromptHelperPanel(gtx layout.Context, presets []sharedCompat.Preset, suggestions []string) layout.Dimensions {
	return a.borderedSurface(gtx, fluent.surface, fluentCardRadius, fluent.border, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			children := []layout.FlexChild{}
			if a.promptHelperTab != "history" && len(presets) > 0 {
				children = append(children,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutPromptHelperButtons(gtx, "preset:", presetLabels(presets))
					}),
				)
			}
			if a.promptHelperTab == "history" && len(suggestions) > 0 {
				children = append(children,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutPromptHelperButtons(gtx, "prompt-history:", promptLabels(suggestions))
					}),
				)
			}
			if len(children) == 0 {
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					text := "还没有可用的模板。"
					if a.promptHelperTab == "history" {
						text = "还没有提交过提示词。"
					}
					return a.label(gtx, text, unit.Sp(10), fluent.textDim, font.Normal)
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}

func (a *App) layoutPromptHelperInline(gtx layout.Context, presets []sharedCompat.Preset, suggestions []string) layout.Dimensions {
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
		return a.borderedSurface(gtx, fluent.surface, fluentCardRadius, fluent.border, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										label := "模板"
										if len(presets) > 0 {
											label = fmt.Sprintf("模板 %d", len(presets))
										}
										return a.pillButton(gtx, &a.promptHelperTemplatesButton, label, a.promptHelperTab != "history")
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										label := fmt.Sprintf("历史 %d", len(suggestions))
										return a.pillButton(gtx, &a.promptHelperHistoryButton, label, a.promptHelperTab == "history")
									}),
								)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.ghostIconButton(gtx, &a.closePromptHelperButton, uiIconClose, false)
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutPromptHelperPanel(gtx, presets, suggestions)
					}),
				)
			})
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
					return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							label := "模板"
							if len(snap.Presets) > 0 {
								label = fmt.Sprintf("模板 %d", len(snap.Presets))
							}
							return a.pillButton(gtx, &a.promptHelperTemplatesButton, label, a.promptHelperTab != "history")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							label := fmt.Sprintf("历史 %d", len(suggestions))
							return a.pillButton(gtx, &a.promptHelperHistoryButton, label, a.promptHelperTab == "history")
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.layoutPromptHelperPanel(gtx, snap.Presets, suggestions)
				}),
			)
		},
	)
}

func (a *App) layoutSettingsModal(gtx layout.Context) layout.Dimensions {
	for a.closeSettingsButton.Clicked(gtx) {
		a.settingsModalOpen = false
		a.saveCurrentConfig()
	}
	for a.toggleAPIKeyMaskButton.Clicked(gtx) {
		a.apiKeyVisible = !a.apiKeyVisible
	}
	for a.settingsTestUpstreamButton.Clicked(gtx) {
		a.saveCurrentConfig()
		a.startUpstreamProbe()
	}
	for a.createImagesProfileButton.Clicked(gtx) {
		a.createBlankProfileWithMode(string(client.APIModeImages))
	}
	snap := a.readSnapshot()
	for _, profile := range snap.Profiles {
		btn := a.settingsProfileButton("settings-profile:" + profile.ID)
		profile := profile
		for btn.Clicked(gtx) {
			a.switchActiveProfile(profile.ID)
		}
	}
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
		activeName+" · "+apiChoiceLabel(activeMode),
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
									return a.label(gtx, "Responses API", unit.Sp(13), fluent.text, font.SemiBold)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, "首选。支持 SSE 保活，长任务更稳。", unit.Sp(11), fluent.textMuted, font.Normal)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, "适合 GPT 图像链路和提示词优化。", unit.Sp(10), fluent.textDim, font.Normal)
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
									return a.label(gtx, "Images API", unit.Sp(13), fluent.text, font.SemiBold)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, "兼容性更广，接标准 generations / edits。", unit.Sp(11), fluent.textMuted, font.Normal)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, "适合只想尽快接上常规生图接口。", unit.Sp(10), fluent.textDim, font.Normal)
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
	set func(string),
) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, title, unit.Sp(11), fluent.textMuted, font.Medium)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			rows := make([]layout.FlexChild, 0, (len(options)+1)/2)
			for row := 0; row < len(options); row += 2 {
				row := row
				rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					cells := make([]layout.FlexChild, 0, 2)
					for col := 0; col < 2; col++ {
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
								unit.Dp(6),
								layout.Inset{Top: 10, Bottom: 10, Left: 10, Right: 10},
								func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.label(gtx, options[idx].Title, unit.Sp(11), chooseColor(active, fluent.accent, fluent.text), font.Medium)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.label(gtx, options[idx].Detail, unit.Sp(10), chooseColor(active, fluent.accent, fluent.textDim), font.Normal)
										}),
									)
								},
							)
						}))
					}
					return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx, cells...)
					})
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
		}),
	)
}

func (a *App) layoutSettingsAPIKeyField(gtx layout.Context) layout.Dimensions {
	label := "显示"
	icon := uiIconVisibility
	if a.apiKeyVisible {
		label = "隐藏"
		icon = uiIconVisibilityOff
	}
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, "API Key", unit.Sp(11), fluent.textMuted, font.Medium)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedHeight(gtx, unit.Dp(44), func(gtx layout.Context) layout.Dimensions {
				return a.borderedSurface(gtx, fluent.surface, unit.Dp(4), fluent.border2, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: 9, Bottom: 9, Left: 10, Right: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return a.editorText(gtx, &a.apiKeyInput, "sk-...", unit.Sp(13))
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.compactIconTextButton(gtx, &a.toggleAPIKeyMaskButton, icon, label, a.apiKeyVisible)
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
	for a.createProfileButton.Clicked(gtx) {
		a.createBlankProfile()
	}
	for a.duplicateProfileButton.Clicked(gtx) {
		a.duplicateActiveProfile()
	}
	for a.deleteProfileButton.Clicked(gtx) {
		a.deleteActiveProfile()
	}

	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.sectionEyebrow(gtx, "配置列表")
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.borderedSurface(gtx, fluent.surface, unit.Dp(8), fluent.border, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						if len(snap.Profiles) == 0 {
							return a.label(gtx, "还没有可用上游配置。", unit.Sp(10), fluent.textDim, font.Normal)
						}
						rows := make([]layout.FlexChild, 0, len(snap.Profiles))
						for idx, profile := range snap.Profiles {
							profile := profile
							rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								btn := a.settingsProfileButton("settings-profile:" + profile.ID)
								active := profile.ID == snap.ActiveProfileID
								return a.surfaceButton(
									gtx,
									btn,
									chooseColor(active, fluent.accentSoft, rgba(0xffffff, 0x00)),
									chooseColor(active, accentAlpha(0x28), fluent.surface2),
									rgba(0xffffff, 0x00),
									unit.Dp(6),
									layout.Inset{Top: 8, Bottom: 8, Left: 8, Right: 8},
									func(gtx layout.Context) layout.Dimensions {
										modeTag := "R"
										if strings.TrimSpace(profile.APIMode) == "images" {
											modeTag = "I"
										}
										return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												dot := fluent.textDim
												if active {
													dot = fluent.accent
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
												return a.singleLineLabel(gtx, strings.TrimSpace(profile.Name), unit.Sp(12), chooseColor(active, fluent.accent, fluent.text), font.Medium)
											}),
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return a.label(gtx, modeTag, unit.Sp(10), fluent.textDim, font.Medium)
											}),
										)
									},
								)
							}))
							if idx != len(snap.Profiles)-1 {
								rows = append(rows, layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout))
							}
						}
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
					})
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return a.compactIconTextButton(gtx, &a.createProfileButton, uiIconAdd, "新建", false)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.historyMiniIconButton(gtx, &a.duplicateProfileButton, uiIconCopy, false)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.historyMiniIconButton(gtx, &a.deleteProfileButton, uiIconDelete, false)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, "点击列表项会立即切换当前生效配置。", unit.Sp(10), fluent.textDim, font.Normal)
			}),
		)
	})
}

func (a *App) layoutSettingsEditorPane(gtx layout.Context, snap snapshot) layout.Dimensions {
	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		children := []layout.Widget{
			func(gtx layout.Context) layout.Dimensions {
				return a.sectionEyebrow(gtx, "编辑配置")
			},
			func(gtx layout.Context) layout.Dimensions {
				return a.field(gtx, "名称", &a.profileNameInput, "配置1", unit.Dp(44))
			},
			func(gtx layout.Context) layout.Dimensions {
				return a.layoutSettingsOptionCards(gtx, "API 形态", []settingsOptionChoice{
					{Title: "Responses API", Detail: "SSE 保活，更适合长推理", Value: string(client.APIModeResponses)},
					{Title: "Images API", Detail: "标准 generations / edits", Value: string(client.APIModeImages)},
				}, a.api, a.apiButtons, func(value string) { a.api = value })
			},
			func(gtx layout.Context) layout.Dimensions {
				return a.layoutSettingsOptionCards(gtx, "参数策略", []settingsOptionChoice{
					{Title: "OpenAI 标准", Detail: "只发送官方公开字段", Value: string(client.RequestPolicyOpenAI)},
					{Title: "兼容中转扩展", Detail: "附带 seed / negative_prompt", Value: string(client.RequestPolicyCompat)},
				}, a.policy, a.policyButtons, func(value string) { a.policy = value })
			},
			func(gtx layout.Context) layout.Dimensions {
				return a.field(gtx, "上游 BASE_URL", &a.baseURLInput, "https://example.com", unit.Dp(44))
			},
			func(gtx layout.Context) layout.Dimensions {
				return a.layoutSettingsAPIKeyField(gtx)
			},
		}
		if a.api == string(client.APIModeResponses) {
			children = append(children, func(gtx layout.Context) layout.Dimensions {
				return a.field(gtx, "文本模型 ID", &a.textModelInput, client.TextModel, unit.Dp(44))
			})
		}
		children = append(children,
			func(gtx layout.Context) layout.Dimensions {
				return a.field(gtx, "图像模型 ID", &a.imageModelInput, client.ImageModel, unit.Dp(44))
			},
			func(gtx layout.Context) layout.Dimensions {
				return a.field(gtx, "并发数量限制", &a.concurrencyLimitInput, "留空 = 不限制", unit.Dp(44))
			},
			func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, "0/留空 = 不限制。填正整数后，此配置跨所有标签页最多同时运行这么多任务。", unit.Sp(10), fluent.textDim, font.Normal)
			},
			layout.Spacer{Height: unit.Dp(4)}.Layout,
			func(gtx layout.Context) layout.Dimensions {
				return a.sectionEyebrow(gtx, "运行与输出")
			},
			func(gtx layout.Context) layout.Dimensions {
				return a.segmentedWithTitle(gtx, "代理", proxyChoices, a.proxy, a.proxyButtons, func(value string) { a.proxy = value })
			},
		)
		if a.proxy == "custom" || strings.TrimSpace(a.proxyURLInput.Text()) != "" {
			children = append(children, func(gtx layout.Context) layout.Dimensions {
				return a.field(gtx, "自定义代理 URL", &a.proxyURLInput, "http://127.0.0.1:7890", unit.Dp(44))
			})
		}
		children = append(children,
			func(gtx layout.Context) layout.Dimensions {
				return a.field(gtx, "输出目录", &a.outputDirInput, "生成图片保存目录", unit.Dp(44))
			},
			layout.Spacer{Height: unit.Dp(4)}.Layout,
			func(gtx layout.Context) layout.Dimensions {
				label := "保存并测试连接"
				if snap.TestingUpstream {
					label = "测试中..."
				}
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return a.primaryIconTextButton(gtx, &a.settingsTestUpstreamButton, uiIconRefresh, label, fluent.accent, fluent.white)
			},
		)
		if strings.TrimSpace(snap.LastProbeSummary) != "" {
			children = append(children, func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, snap.LastProbeSummary, unit.Sp(10), fluent.textDim, font.Normal)
			})
		}
		children = append(children, func(gtx layout.Context) layout.Dimensions {
			if a.api == string(client.APIModeResponses) {
				return a.label(gtx, "Responses API 更适合长推理和需要 SSE 保活的上游。", unit.Sp(10), fluent.textDim, font.Normal)
			}
			return a.borderedSurface(gtx, fluent.accentSoft, unit.Dp(6), accentAlpha(0x28), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, "Images API 兼容性最广，但没有 SSE 保活，长推理更容易遇到超时。", unit.Sp(10), fluent.accent, font.Normal)
				})
			})
		})
		return a.settingsList.Layout(gtx, len(children), func(gtx layout.Context, index int) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, children[index])
		})
	})
}

func (a *App) layoutComposeCard(gtx layout.Context) layout.Dimensions {
	activeAspect := deriveAspectPreset(a.size)
	activeResolution := deriveResolutionPreset(a.size)
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
		activeAspect,
		strings.ToUpper(activeResolution),
		qualityChoiceLabel(a.quality),
		fmt.Sprintf("%d 张", normalizeBatchCount(a.batchCount)),
		sourceLabel,
	}), " · ")

	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.layoutDisclosureHeader(gtx, &a.composeToggleButton, "创作参数", summary, a.composeOpen)
			}),
		}
		if a.composeOpen {
			children = append(children,
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
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
					if a.mode != string(client.ModeEdit) {
						return layout.Dimensions{}
					}
					return a.composeSectionCard(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.layoutSourceInputSection(gtx, sourcePaths, currentSaved)
					})
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.composeSectionCard(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.segmentedWithTitle(gtx, "质量", qualityChoices, a.quality, a.qualityButtons, func(value string) { a.quality = value })
					})
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.composeSectionCard(gtx, a.layoutBatchCountSection)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, "风格、比例、分辨率、质量和批量张数会随当前标签一起保存。", unit.Sp(10), fluent.textDim, font.Normal)
					})
				}),
			)
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

func (a *App) composeSectionCard(gtx layout.Context, body layout.Widget) layout.Dimensions {
	return a.borderedSurface(gtx, fluent.surface, fluentCardRadius, fluent.border, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(10)).Layout(gtx, body)
	})
}

func (a *App) layoutAspectSection(gtx layout.Context, activeAspect string, currentResolution string) layout.Dimensions {
	children := make([]layout.FlexChild, 0, 2)
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return a.label(gtx, "比例", unit.Sp(11), fluent.textMuted, font.Medium)
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
					a.size = buildAspectSizeSelection(choice.Value, currentResolution)
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
	children := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, "风格", unit.Sp(11), fluent.textMuted, font.Medium)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if a.styleTag == "" {
						return a.label(gtx, "默认风格", unit.Sp(10), fluent.textDim, font.Normal)
					}
					return a.staticPill(gtx, "已选 "+styleChoiceLabel(a.styleTag), true, false)
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
						return a.pillButton(gtx, btn, choice.Label, a.styleTag == choice.Value)
					})
				}))
			}
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, row...)
		}))
	}
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		if a.styleTag == "" {
			return layout.Dimensions{}
		}
		return a.label(gtx, styleSuffixes[a.styleTag], unit.Sp(10), fluent.textDim, font.Normal)
	}))
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
	for a.useCurrentAsSourceButton.Clicked(gtx) {
		if currentSaved != "" {
			a.appendSourcePath(currentSaved)
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
			title := "源图片 / 参考图"
			if len(sourcePaths) > 0 {
				title += fmt.Sprintf(" · %d 张", len(sourcePaths))
			}
			return a.label(gtx, title, unit.Sp(11), fluent.textMuted, font.Medium)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
	}

	if len(sourcePaths) == 0 && currentSaved != "" {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.borderedSurface(gtx, fluent.surface2, fluentControlRadius, fluent.border, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, "(画板当前图 · 隐式源图)", unit.Sp(10), fluent.textDim, font.Normal)
				})
			})
		}))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
	}

	for _, path := range sourcePaths {
		path := path
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := a.sourceButton("panel-remove:" + path)
			return a.surfaceButton(
				gtx,
				btn,
				fluent.surface,
				fluent.surface2,
				fluent.border,
				fluentControlRadius,
				layout.Inset{Top: 8, Bottom: 8, Left: 10, Right: 10},
				func(gtx layout.Context) layout.Dimensions {
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
							return a.historyMiniIconButton(gtx, btn, uiIconDelete, false)
						}),
					)
				},
			)
		}))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
	}

	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		row := []layout.FlexChild{}
		row = append(row, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.compactIconTextButton(gtx, &a.addSourceFilesButton, uiIconAdd, "添加图片", false)
		}))
		if currentSaved != "" {
			row = append(row, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.compactIconTextButton(gtx, &a.useCurrentAsSourceButton, uiIconPhoto, "使用当前图", false)
				})
			}))
		}
		if len(sourcePaths) > 0 {
			row = append(row, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.compactIconTextButton(gtx, &a.clearSourcesButton, uiIconDelete, "清空", false)
				})
			}))
		}
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, row...)
	}))
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		if currentSaved != "" && len(sourcePaths) == 0 {
			return a.label(gtx, "可直接复用当前画布结果，或继续添加本地图片。", unit.Sp(10), fluent.textDim, font.Normal)
		}
		if len(sourcePaths) > 0 {
			return a.label(gtx, "多张参考图会按当前顺序发送给图生图请求。", unit.Sp(10), fluent.textDim, font.Normal)
		}
		return a.label(gtx, "使用系统文件选择器添加参考图。", unit.Sp(10), fluent.textDim, font.Normal)
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
	choices := visibleResolutionChoices(activeAspect)
	children := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, "分辨率", unit.Sp(11), fluent.textMuted, font.Medium)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
	}
	row := make([]layout.FlexChild, 0, len(choices))
	for idx := range choices {
		idx := idx
		for a.resolutionButtons[idx].Clicked(gtx) {
			a.size = buildResolutionSizeSelection(activeAspect, choices[idx].Value)
		}
		row = append(row, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.pillButton(gtx, &a.resolutionButtons[idx], choices[idx].Label, activeResolution == choices[idx].Value)
		}))
	}
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx, row...)
	}))
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (a *App) layoutBatchCountSection(gtx layout.Context) layout.Dimensions {
	batchCount := normalizeBatchCount(a.batchCount)
	children := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, "出图张数", unit.Sp(11), fluent.textMuted, font.Medium)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, fmt.Sprintf("当前 %d 张", batchCount), unit.Sp(10), fluent.textDim, font.Normal)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
	}
	rows := 2
	cols := 3
	for row := 0; row < rows; row++ {
		row := row
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			cells := make([]layout.FlexChild, 0, cols)
			for col := 0; col < cols; col++ {
				idx := row*cols + col
				if idx >= len(batchCountChoices) {
					cells = append(cells, layout.Flexed(1, layout.Spacer{}.Layout))
					continue
				}
				label := batchCountChoices[idx].Label
				value, _ := strconv.Atoi(batchCountChoices[idx].Value)
				btn := &a.batchCountButtons[idx]
				for btn.Clicked(gtx) {
					a.batchCount = normalizeBatchCount(value)
				}
				cells = append(cells, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.pillButton(gtx, btn, label, batchCount == value)
				}))
			}
			return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx, cells...)
			})
		}))
	}
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return a.label(gtx, "多张结果会自动按同一提示词归到一组，方便回看和挑图。", unit.Sp(10), fluent.textDim, font.Normal)
	}))
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (a *App) layoutAdvancedCard(gtx layout.Context) layout.Dimensions {
	summary := strings.Join(compactNonEmpty([]string{
		negativePromptSummary(a.negativePromptInput.Text()),
		strings.ToUpper(strings.TrimSpace(a.format)),
		seedSummary(a.seedInput.Text()),
	}), " · ")

	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.borderedSurface(gtx, fluent.surface, fluentCardRadius, fluent.border, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.layoutDisclosureHeader(gtx, &a.advancedToggleButton, "高级参数", summary, a.advancedOpen)
				})
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !a.advancedOpen {
				return layout.Dimensions{}
			}
			return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.composeSectionCard(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.field(gtx, "负向提示词", &a.negativePromptInput, "兼容模式可发给部分上游", unit.Dp(64))
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.composeSectionCard(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.segmentedWithTitle(gtx, "输出格式", formatChoices, a.format, a.formatButtons, func(value string) { a.format = value })
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.composeSectionCard(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return a.field(gtx, "Seed", &a.seedInput, "0", unit.Dp(44))
								}),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return a.field(gtx, "Partial", &a.partialImagesInput, "1", unit.Dp(44))
								}),
							)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "上游、代理、模型与输出目录已收纳到上游配置面板。", unit.Sp(10), fluent.textDim, font.Normal)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.compactIconTextButton(gtx, &a.manageUpstreamButton, uiIconSettings, "打开上游配置", false)
							}),
						)
					}),
				)
			})
		}),
	)
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
				)
			})
		},
	)
}

func (a *App) layoutActions(gtx layout.Context) layout.Dimensions {
	snap := a.readSnapshot()
	ready := strings.TrimSpace(a.apiKeyInput.Text()) != "" && strings.TrimSpace(a.baseURLInput.Text()) != ""
	children := make([]layout.FlexChild, 0, 6)
	if snap.Status == "失败" {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.borderedSurface(gtx, dangerAlpha(0x16), unit.Dp(8), dangerAlpha(0x2f), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "最近一次请求失败", unit.Sp(12), fluent.danger, font.SemiBold)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							text := "可打开日志或 Raw response 排查。"
							if strings.TrimSpace(snap.Result.RawPath) != "" {
								text = "Raw response: " + snap.Result.RawPath
							}
							return a.label(gtx, text, unit.Sp(10), fluent.textMuted, font.Normal)
						}),
					)
				})
			})
		}))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout))
	}
	if !ready {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.borderedSurface(gtx, fluent.accentSoft, unit.Dp(8), accentAlpha(0x28), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "还没有可用上游配置", unit.Sp(12), fluent.accent, font.SemiBold)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "先打开上游配置创建一条可用配置，才能测试连接或开始生成。", unit.Sp(11), fluent.textMuted, font.Normal)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.compactIconTextButton(gtx, &a.manageUpstreamButton, uiIconSettings, "配置上游", true)
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
			return a.primaryIconTextButton(gtx, &a.manageUpstreamButton, uiIconSettings, "配置上游", fluent.accent, fluent.white)
		}
		if snap.Running {
			return a.primaryIconTextButton(gtx, &a.cancelButton, uiIconCancel, "取消生成", dangerAlpha(0x12), fluent.danger)
		}
		label := "生成"
		icon := uiIconPlay
		if a.mode == string(client.ModeEdit) {
			label = "编辑"
			icon = uiIconBuild
		}
		return a.primaryIconTextButton(gtx, &a.runButton, icon, label, fluent.accent, fluent.white)
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
		unit.Dp(6),
		layout.Inset{Top: 10, Bottom: 10, Left: 10, Right: 10},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, title, unit.Sp(11), fluent.text, font.Bold)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, summary, unit.Sp(11), fluent.textMuted, font.Normal)
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
