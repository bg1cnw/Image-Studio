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
	"gioui.org/op/clip"
	"gioui.org/op/paint"
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
			return a.controlsList.Layout(gtx, 1, func(gtx layout.Context, _ int) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
					layout.Rigid(a.layoutWorkbenchCard),
					layout.Rigid(a.layoutModeCard),
					layout.Rigid(a.layoutPromptCard),
					layout.Rigid(a.layoutComposeCard),
					layout.Rigid(a.layoutAdvancedCard),
					layout.Rigid(a.layoutActions),
				)
			})
		})
	})
}

func (a *App) layoutWorkbenchCard(gtx layout.Context) layout.Dimensions {
	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, "图像工作台", unit.Sp(18), fluent.text, font.SemiBold)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, "保持界面简洁，把注意力留给 prompt、参考图和结果。", unit.Sp(12), fluent.textMuted, font.Normal)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.badge(gtx, a.modeLabel(), fluent.accentSoft, fluent.accent)
			}),
		)
	})
}

func (a *App) layoutModeCard(gtx layout.Context) layout.Dimensions {
	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
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
	for a.promptHelperButton.Clicked(gtx) {
		a.promptHelperOpen = !a.promptHelperOpen
	}
	for a.optimizePromptButton.Clicked(gtx) {
		a.startPromptOptimize()
	}
	snap := a.readSnapshot()
	promptSuggestions := buildPromptSuggestions(snap.PromptHistory, snap.History)
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
	footer := "建议把主体、场景、镜头、材质和光照拆成短句，后续更方便复用模板与历史。"
	if a.mode == string(client.ModeEdit) {
		count := len(kernel.ParseSourcePaths(a.sourcePathsInput.Text()))
		footer = fmt.Sprintf("图生图模式 · 当前 %d 张参考图", count)
	}

	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, "PROMPT 提示词", unit.Sp(11), fluent.textMuted, font.Bold)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, fmt.Sprintf("%d", promptLen), unit.Sp(11), fluent.textDim, font.Medium)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return fixedHeight(gtx, unit.Dp(136), func(gtx layout.Context) layout.Dimensions {
					return a.borderedSurface(gtx, fluent.surface, unit.Dp(4), fluent.border2, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: 10, Bottom: 10, Left: 10, Right: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.editorText(gtx, &a.promptInput, "输入生成或编辑要求", unit.Sp(13))
						})
					})
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, footer, unit.Sp(10), fluent.textDim, font.Normal)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return a.pillButton(gtx, &a.promptHelperButton, "模板 / 历史", a.promptHelperOpen)
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						label := "AI 优化"
						if snap.OptimizingPrompt {
							label = "优化中..."
						}
						return a.compactButton(gtx, &a.optimizePromptButton, label, snap.OptimizingPrompt)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, "支持从历史与预设补全文案，也可以直接让当前上游帮你润色提示词。", unit.Sp(10), fluent.textDim, font.Normal)
			}),
		)
	})
}

func (a *App) layoutPromptHelperPanel(gtx layout.Context, presets []sharedCompat.Preset, suggestions []string) layout.Dimensions {
	return a.borderedSurface(gtx, fluent.surface2, unit.Dp(6), fluent.border, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			children := []layout.FlexChild{}
			if len(presets) > 0 {
				children = append(children,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, "预设", unit.Sp(11), fluent.textMuted, font.Medium)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutPromptHelperButtons(gtx, "preset:", presetLabels(presets))
					}),
				)
			}
			if len(suggestions) > 0 {
				if len(children) > 0 {
					children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout))
				}
				children = append(children,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, "最近提示词", unit.Sp(11), fluent.textMuted, font.Medium)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutPromptHelperButtons(gtx, "prompt-history:", promptLabels(suggestions))
					}),
				)
			}
			if len(children) == 0 {
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, "还没有可用的模板或提示词历史。", unit.Sp(10), fluent.textDim, font.Normal)
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}

func (a *App) layoutPromptHelperModal(gtx layout.Context) layout.Dimensions {
	for a.closePromptHelperButton.Clicked(gtx) {
		a.promptHelperOpen = false
	}
	snap := a.readSnapshot()
	suggestions := buildPromptSuggestions(snap.PromptHistory, snap.History)
	paint.FillShape(gtx.Ops, rgba(0x000000, 0x52), clip.Rect{Max: gtx.Constraints.Max}.Op())
	gtx.Constraints.Min = gtx.Constraints.Max
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = image.Point{}
		return fixedWidth(gtx, unit.Dp(560), func(gtx layout.Context) layout.Dimensions {
			return a.borderedSurface(gtx, fluent.surface, unit.Dp(8), fluent.border, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.label(gtx, "模板 / 历史", unit.Sp(18), fluent.text, font.SemiBold)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.label(gtx, "从最近提示词和已保存预设里快速补全当前提示词。", unit.Sp(11), fluent.textMuted, font.Normal)
										}),
									)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return fixedWidth(gtx, unit.Dp(88), func(gtx layout.Context) layout.Dimensions {
										return a.button(gtx, &a.closePromptHelperButton, "关闭", fluent.surface2, fluent.text)
									})
								}),
							)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.layoutPromptHelperPanel(gtx, snap.Presets, suggestions)
						}),
					)
				})
			})
		})
	})
}

func (a *App) layoutSettingsModal(gtx layout.Context) layout.Dimensions {
	for a.closeSettingsButton.Clicked(gtx) {
		a.settingsModalOpen = false
		a.saveCurrentConfig()
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
	paint.FillShape(gtx.Ops, rgba(0x000000, 0x52), clip.Rect{Max: gtx.Constraints.Max}.Op())
	gtx.Constraints.Min = gtx.Constraints.Max
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = image.Point{}
		return fixedWidth(gtx, unit.Dp(760), func(gtx layout.Context) layout.Dimensions {
			return fixedHeight(gtx, unit.Dp(680), func(gtx layout.Context) layout.Dimensions {
				return a.borderedSurface(gtx, fluent.surface, unit.Dp(8), fluent.border, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return a.label(gtx, "设置", unit.Sp(18), fluent.text, font.SemiBold)
											}),
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return a.label(gtx, activeName+" · "+apiChoiceLabel(a.api), unit.Sp(11), fluent.textMuted, font.Normal)
											}),
										)
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return fixedWidth(gtx, unit.Dp(88), func(gtx layout.Context) layout.Dimensions {
											return a.button(gtx, &a.closeSettingsButton, "关闭", fluent.surface2, fluent.text)
										})
									}),
								)
							}),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return fixedWidth(gtx, unit.Dp(220), func(gtx layout.Context) layout.Dimensions {
											return a.layoutSettingsProfileRail(gtx, snap)
										})
									}),
									layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										return a.settingsList.Layout(gtx, 1, func(gtx layout.Context, _ int) layout.Dimensions {
											return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
														return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
															layout.Rigid(func(gtx layout.Context) layout.Dimensions {
																return a.sectionEyebrow(gtx, "上游与请求")
															}),
															layout.Rigid(func(gtx layout.Context) layout.Dimensions {
																return a.field(gtx, "BASE_URL", &a.baseURLInput, "https://example.com", unit.Dp(44))
															}),
															layout.Rigid(func(gtx layout.Context) layout.Dimensions {
																return a.field(gtx, "API Key", &a.apiKeyInput, "sk-...", unit.Dp(44))
															}),
															layout.Rigid(func(gtx layout.Context) layout.Dimensions {
																return a.field(gtx, "文本模型", &a.textModelInput, client.TextModel, unit.Dp(44))
															}),
															layout.Rigid(func(gtx layout.Context) layout.Dimensions {
																return a.field(gtx, "图像模型", &a.imageModelInput, client.ImageModel, unit.Dp(44))
															}),
															layout.Rigid(func(gtx layout.Context) layout.Dimensions {
																return a.segmentedWithTitle(gtx, "请求字段", policyChoices, a.policy, a.policyButtons, func(value string) { a.policy = value })
															}),
														)
													})
												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
														return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
															layout.Rigid(func(gtx layout.Context) layout.Dimensions {
																return a.sectionEyebrow(gtx, "代理与保存")
															}),
															layout.Rigid(func(gtx layout.Context) layout.Dimensions {
																return a.segmentedWithTitle(gtx, "代理", proxyChoices, a.proxy, a.proxyButtons, func(value string) { a.proxy = value })
															}),
															layout.Rigid(func(gtx layout.Context) layout.Dimensions {
																return a.field(gtx, "自定义代理 URL", &a.proxyURLInput, "http://127.0.0.1:7890", unit.Dp(44))
															}),
															layout.Rigid(func(gtx layout.Context) layout.Dimensions {
																return a.field(gtx, "输出目录", &a.outputDirInput, "生成图片保存目录", unit.Dp(44))
															}),
															layout.Rigid(func(gtx layout.Context) layout.Dimensions {
																return a.segmentedWithTitle(gtx, "输出格式", formatChoices, a.format, a.formatButtons, func(value string) { a.format = value })
															}),
														)
													})
												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
														return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
															layout.Rigid(func(gtx layout.Context) layout.Dimensions {
																return a.sectionEyebrow(gtx, "生成参数")
															}),
															layout.Rigid(func(gtx layout.Context) layout.Dimensions {
																return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
																	layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
																		return a.field(gtx, "Seed", &a.seedInput, "0", unit.Dp(44))
																	}),
																	layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
																		return a.field(gtx, "Partial", &a.partialImagesInput, "1", unit.Dp(44))
																	}),
																)
															}),
															layout.Rigid(func(gtx layout.Context) layout.Dimensions {
																return a.field(gtx, "负向提示词", &a.negativePromptInput, "兼容模式可发给部分上游", unit.Dp(64))
															}),
															layout.Rigid(func(gtx layout.Context) layout.Dimensions {
																return a.label(gtx, "修改会在下一次生成时立即生效。顶部主题切换和右侧上游测试会共用这里的配置。", unit.Sp(10), fluent.textDim, font.Normal)
															}),
														)
													})
												}),
											)
										})
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

func (a *App) layoutPromptHelperButtons(gtx layout.Context, prefix string, items []promptHelperItem) layout.Dimensions {
	rows := make([]layout.FlexChild, 0, len(items))
	for _, item := range items {
		item := item
		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				btn := a.promptButton(prefix + item.ID)
				return a.surfaceButton(
					gtx,
					btn,
					fluent.surface,
					fluent.surface2,
					fluent.border,
					unit.Dp(4),
					layout.Inset{Top: 8, Bottom: 8, Left: 10, Right: 10},
					func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(3))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, item.Title, unit.Sp(11), fluent.text, font.Medium)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if item.Detail == "" {
									return layout.Dimensions{}
								}
								return a.label(gtx, item.Detail, unit.Sp(10), fluent.textDim, font.Normal)
							}),
						)
					},
				)
			})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
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
		children := []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.sectionEyebrow(gtx, "配置")
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		}
		if len(snap.Profiles) == 0 {
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, "还没有可用上游配置。", unit.Sp(10), fluent.textDim, font.Normal)
			}))
		} else {
			for _, profile := range snap.Profiles {
				profile := profile
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn := a.settingsProfileButton("settings-profile:" + profile.ID)
					active := profile.ID == snap.ActiveProfileID
					return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.surfaceButton(
							gtx,
							btn,
							chooseColor(active, fluent.accentSoft, fluent.surface),
							chooseColor(active, rgba(0x005fb8, 0x28), fluent.surface2),
							fluent.border,
							unit.Dp(6),
							layout.Inset{Top: 10, Bottom: 10, Left: 10, Right: 10},
							func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.label(gtx, strings.TrimSpace(profile.Name), unit.Sp(12), chooseColor(active, fluent.accent, fluent.text), font.Medium)
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.label(gtx, strings.ToUpper(strings.TrimSpace(profile.APIMode)), unit.Sp(10), fluent.textDim, font.Normal)
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										if !active {
											return layout.Dimensions{}
										}
										return a.badge(gtx, "当前启用", fluent.accentSoft, fluent.accent)
									}),
								)
							},
						)
					})
				}))
			}
		}
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout))
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.compactButton(gtx, &a.createProfileButton, "新建", false)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.compactButton(gtx, &a.duplicateProfileButton, "复制", false)
				}),
			)
		}))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
		if len(snap.Profiles) > 0 {
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.compactButton(gtx, &a.deleteProfileButton, "删除当前配置", false)
			}))
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, "切换配置会先保存当前编辑内容，再载入目标配置。", unit.Sp(10), fluent.textDim, font.Normal)
		}))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
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
					return a.layoutStyleSection(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.layoutAspectSection(gtx, activeAspect, activeResolution)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.layoutResolutionSection(gtx, activeAspect, activeResolution)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if a.mode != string(client.ModeEdit) {
						return layout.Dimensions{}
					}
					return a.layoutSourceInputSection(gtx, sourcePaths, currentSaved)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.segmentedWithTitle(gtx, "API 形态", apiChoices, a.api, a.apiButtons, func(value string) { a.api = value })
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.segmentedWithTitle(gtx, "质量", qualityChoices, a.quality, a.qualityButtons, func(value string) { a.quality = value })
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.layoutBatchCountSection(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.segmentedWithTitle(gtx, "格式", formatChoices, a.format, a.formatButtons, func(value string) { a.format = value })
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, "这里的参数会随当前上游配置一起保存，并在下一次生成时立即生效。", unit.Sp(10), fluent.textDim, font.Normal)
					})
				}),
			)
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
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
			return a.borderedSurface(gtx, fluent.surface2, unit.Dp(6), fluent.border, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, "当前画布图可直接作为隐式源图使用。", unit.Sp(10), fluent.textDim, font.Normal)
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
				unit.Dp(6),
				layout.Inset{Top: 8, Bottom: 8, Left: 10, Right: 10},
				func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, filepath.Base(path), unit.Sp(11), fluent.text, font.Medium)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "移除", unit.Sp(10), fluent.textDim, font.Normal)
						}),
					)
				},
			)
		}))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
	}

	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return fixedHeight(gtx, unit.Dp(64), func(gtx layout.Context) layout.Dimensions {
			return a.borderedSurface(gtx, fluent.surface, unit.Dp(4), fluent.border2, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 10, Bottom: 10, Left: 10, Right: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.editorText(gtx, &a.sourcePathsInput, "图生图时填写本地图片路径，多张可换行", unit.Sp(12))
				})
			})
		})
	}))
	children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		row := []layout.FlexChild{}
		row = append(row, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.compactButton(gtx, &a.addSourceFilesButton, "添加图片", false)
		}))
		if currentSaved != "" {
			row = append(row, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.compactButton(gtx, &a.useCurrentAsSourceButton, "使用当前图", false)
				})
			}))
		}
		if len(sourcePaths) > 0 {
			row = append(row, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.compactButton(gtx, &a.clearSourcesButton, "清空", false)
				})
			}))
		}
		if len(row) == 0 {
			return a.label(gtx, "支持直接粘贴本地图片路径，或先把当前结果设为源图。", unit.Sp(10), fluent.textDim, font.Normal)
		}
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, row...)
	}))

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
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

	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.layoutDisclosureHeader(gtx, &a.advancedToggleButton, "高级参数", summary, a.advancedOpen)
			}),
		}
		if a.advancedOpen {
			children = append(children,
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.field(gtx, "Seed", &a.seedInput, "0", unit.Dp(44))
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.field(gtx, "Partial", &a.partialImagesInput, "1", unit.Dp(44))
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.field(gtx, "负向提示词", &a.negativePromptInput, "兼容模式可发给部分上游", unit.Dp(64))
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.segmentedWithTitle(gtx, "输出格式", formatChoices, a.format, a.formatButtons, func(value string) { a.format = value })
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "上游、代理、模型与输出目录已收纳到设置面板。", unit.Sp(10), fluent.textDim, font.Normal)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.compactButton(gtx, &a.manageUpstreamButton, "打开设置", false)
						}),
					)
				}),
			)
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

func (a *App) layoutAspectButton(gtx layout.Context, btn *widget.Clickable, choice aspectChoice, active bool) layout.Dimensions {
	return a.surfaceButton(
		gtx,
		btn,
		chooseColor(active, fluent.accentSoft, fluent.surface),
		chooseColor(active, rgba(0x005fb8, 0x28), fluent.surface2),
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

	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		children := make([]layout.FlexChild, 0, 4)
		if !ready {
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.borderedSurface(gtx, fluent.accentSoft, unit.Dp(6), rgba(0x005fb8, 0x28), func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "还没有可用上游配置", unit.Sp(12), fluent.accent, font.SemiBold)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "先配置 BASE_URL 和 API Key，才能开始测试连接或生成。", unit.Sp(11), fluent.textMuted, font.Normal)
							}),
						)
					})
				})
			}))
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout))
		}

		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			flexChildren := []layout.FlexChild{
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					if !ready {
						return a.button(gtx, &a.manageUpstreamButton, "配置上游", fluent.accent, fluent.white)
					}
					label := "生成"
					bg := fluent.accent
					if a.mode == string(client.ModeEdit) {
						label = "编辑"
					}
					if snap.Running {
						label = "运行中"
						bg = fluent.textMuted
					}
					return a.button(gtx, &a.runButton, label, bg, fluent.white)
				}),
			}
			if snap.Running {
				flexChildren = append(flexChildren, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedWidth(gtx, unit.Dp(92), func(gtx layout.Context) layout.Dimensions {
						return a.button(gtx, &a.cancelButton, "取消", fluent.surface2, fluent.text)
					})
				}))
			} else if ready {
				flexChildren = append(flexChildren, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedWidth(gtx, unit.Dp(92), func(gtx layout.Context) layout.Dimensions {
						return a.button(gtx, &a.manageUpstreamButton, "上游", fluent.surface2, fluent.text)
					})
				}))
			}
			return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx, flexChildren...)
		}))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

func (a *App) layoutDisclosureHeader(gtx layout.Context, btn *widget.Clickable, title string, summary string, open bool) layout.Dimensions {
	stateText := "展开"
	if open {
		stateText = "收起"
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
					return a.label(gtx, stateText, unit.Sp(11), fluent.textDim, font.Medium)
				}),
			)
		},
	)
}

func (a *App) editorText(gtx layout.Context, editor *widget.Editor, hint string, size unit.Sp) layout.Dimensions {
	style := material.Editor(a.th, editor, hint)
	style.Color = fluent.text
	style.HintColor = fluent.textDim
	style.SelectionColor = rgba(0x005fb8, 0x3d)
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
