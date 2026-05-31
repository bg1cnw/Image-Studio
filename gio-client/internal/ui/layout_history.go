package ui

import (
	"image"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	sharedCompat "image-studio/shared/compat"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

func (a *App) layoutHistoryAndLogs(gtx layout.Context) layout.Dimensions {
	for a.profilePickerButton.Clicked(gtx) {
		a.profilePickerOpen = !a.profilePickerOpen
	}
	for a.historyCollapseButton.Clicked(gtx) {
		a.historyRailCollapsed = !a.historyRailCollapsed
	}
	for idx, value := range []string{"all", "generate", "edit"} {
		for a.historyModeButtons[idx].Clicked(gtx) {
			a.historyModeFilter = value
		}
	}
	for idx, value := range []string{"all", "today", "week"} {
		for a.historyDateButtons[idx].Clicked(gtx) {
			a.historyDateFilter = value
		}
	}

	snap := a.readSnapshot()
	filtered := a.filteredHistory(snap.History)
	entries := buildHistoryPromptEntries(filtered)
	generateCount, editCount := historyCounts(snap.History)
	latest, hasLatest := newestHistoryItem(filtered)
	visible := entries
	if len(visible) > 18 {
		visible = visible[:18]
	}

	for _, profile := range snap.Profiles {
		button := a.profileButton("profile:" + profile.ID)
		for button.Clicked(gtx) {
			a.switchActiveProfile(profile.ID)
		}
	}
	for _, entry := range visible {
		if entry.Kind == "group" {
			button := a.historyButton("group:" + entry.Group.Key)
			for button.Clicked(gtx) {
				if err := a.loadHistoryPreview(entry.Group.Representative, true); err != nil && !isMissingPreview(err) {
					a.appendLog("载入历史结果失败: " + err.Error())
				}
			}
			expand := a.historyButton("expand:" + entry.Group.Key)
			for expand.Clicked(gtx) {
				a.openPromptGroup(entry.Group)
			}
			continue
		}
		button := a.historyButton("row:" + entry.Item.ID)
		for button.Clicked(gtx) {
			if err := a.loadHistoryPreview(entry.Item, true); err != nil && !isMissingPreview(err) {
				a.appendLog("载入历史结果失败: " + err.Error())
			}
		}
	}
	if hasLatest {
		button := a.historyButton("feature:" + latest.ID)
		for button.Clicked(gtx) {
			if err := a.loadHistoryPreview(latest, true); err != nil && !isMissingPreview(err) {
				a.appendLog("载入最近作品失败: " + err.Error())
			}
		}
		detailBtn := a.historyActionButton("feature-detail:" + latest.ID)
		for detailBtn.Clicked(gtx) {
			a.openResultDetail(latest)
		}
		reuseBtn := a.historyActionButton("feature-reuse:" + latest.ID)
		for reuseBtn.Clicked(gtx) {
			a.reuseHistoryItemAsSource(latest)
		}
		deleteBtn := a.historyActionButton("feature-delete:" + latest.ID)
		for deleteBtn.Clicked(gtx) {
			a.deleteHistoryItem(latest.ID)
		}
	}
	for _, entry := range visible {
		if entry.Kind != "item" {
			continue
		}
		item := entry.Item
		detailBtn := a.historyActionButton("row-detail:" + item.ID)
		for detailBtn.Clicked(gtx) {
			a.openResultDetail(item)
		}
		reuseBtn := a.historyActionButton("row-reuse:" + item.ID)
		for reuseBtn.Clicked(gtx) {
			a.reuseHistoryItemAsSource(item)
		}
		deleteBtn := a.historyActionButton("row-delete:" + item.ID)
		for deleteBtn.Clicked(gtx) {
			a.deleteHistoryItem(item.ID)
		}
	}

	return a.borderedSurface(gtx, fluent.inspector, unit.Dp(0), fluent.border, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		return layout.Inset{Top: 12, Bottom: 12, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			children := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.layoutUpstreamCard(gtx, snap)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.layoutHistorySummaryCard(gtx, snap, filtered, generateCount, editCount)
				}),
			}

			if !a.historyRailCollapsed && hasLatest {
				children = append(children,
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutLatestHistoryCard(gtx, latest, snap.SelectedHistoryID == latest.ID)
					}),
				)
			}

			children = append(children,
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					if a.historyRailCollapsed {
						return a.layoutLogsCard(gtx, snap)
					}
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.layoutHistoryResultsCard(gtx, snap, filtered, entries, visible)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return fixedHeight(gtx, unit.Dp(196), func(gtx layout.Context) layout.Dimensions {
								return a.layoutLogsCard(gtx, snap)
							})
						}),
					)
				}),
			)
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}

func (a *App) layoutUpstreamCard(gtx layout.Context, snap snapshot) layout.Dimensions {
	for a.upstreamConfigButton.Clicked(gtx) {
		a.settingsModalOpen = true
	}
	for a.testUpstreamButton.Clicked(gtx) {
		a.startUpstreamProbe()
	}

	activeName := strings.TrimSpace(activeProfileName(snap.Profiles, snap.ActiveProfileID))
	if activeName == "" {
		activeName = "还没有上游配置"
	}
	apiModeLabel := "Responses API"
	if a.api == "images" {
		apiModeLabel = "Images API"
	}
	ready := strings.TrimSpace(a.apiKeyInput.Text()) != "" && strings.TrimSpace(a.baseURLInput.Text()) != ""
	statusLabel := "未配置"
	statusColor := fluent.danger
	dotColor := fluent.danger
	if ready {
		statusLabel = "已配置"
		statusColor = fluent.accent
		dotColor = fluent.accent
	}

	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.sectionEyebrow(gtx, "上游")
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return fixedWidth(gtx, unit.Dp(7), func(gtx layout.Context) layout.Dimensions {
							return fixedHeight(gtx, unit.Dp(7), func(gtx layout.Context) layout.Dimensions {
								return a.surface(gtx, dotColor, unit.Dp(4), layout.Spacer{}.Layout)
							})
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, statusLabel, unit.Sp(11), statusColor, font.Medium)
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		}

		if len(snap.Profiles) == 0 {
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, "还没有上游配置，先在左侧高级参数里补上 BASE_URL 和 API Key。", unit.Sp(11), fluent.textMuted, font.Normal)
			}))
		} else {
			children = append(children,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.surfaceButton(
						gtx,
						&a.profilePickerButton,
						chooseColor(a.profilePickerOpen, fluent.surface2, fluent.surface),
						fluent.surface2,
						fluent.border,
						unit.Dp(4),
						layout.Inset{Top: 9, Bottom: 9, Left: 10, Right: 10},
						func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(3))}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.label(gtx, activeName, unit.Sp(12), fluent.text, font.Medium)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.label(gtx, apiModeLabel, unit.Sp(11), fluent.textMuted, font.Normal)
										}),
									)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									state := "展开"
									if a.profilePickerOpen {
										state = "收起"
									}
									return a.label(gtx, state, unit.Sp(11), fluent.textDim, font.Medium)
								}),
							)
						},
					)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			)
			if a.profilePickerOpen {
				for _, profile := range snap.Profiles {
					profile := profile
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutProfileOption(gtx, profile, profile.ID == snap.ActiveProfileID)
					}))
					children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout))
				}
			}
		}

		if strings.TrimSpace(a.baseURLInput.Text()) != "" {
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, apiModeLabel+" · "+strings.TrimSpace(a.baseURLInput.Text()), unit.Sp(11), fluent.textDim, font.Normal)
			}))
		}
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout))
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.button(gtx, &a.upstreamConfigButton, "上游配置", fluent.surface2, fluent.text)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedWidth(gtx, unit.Dp(88), func(gtx layout.Context) layout.Dimensions {
						label := "测试"
						if snap.TestingUpstream {
							label = "检查中"
						}
						bg := fluent.surface2
						fg := fluent.text
						if snap.LastProbeSummary != "" && !snap.TestingUpstream {
							bg = fluent.accentSoft
							fg = fluent.accent
						}
						return a.button(gtx, &a.testUpstreamButton, label, bg, fg)
					})
				}),
			)
		}))
		if strings.TrimSpace(snap.LastProbeSummary) != "" {
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, snap.LastProbeSummary, unit.Sp(10), fluent.textDim, font.Normal)
			}))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

func (a *App) layoutProfileOption(gtx layout.Context, profile sharedCompat.UpstreamProfile, active bool) layout.Dimensions {
	btn := a.profileButton("profile:" + profile.ID)
	return a.surfaceButton(
		gtx,
		btn,
		chooseColor(active, fluent.accentSoft, fluent.surface),
		chooseColor(active, rgba(0x005fb8, 0x28), fluent.surface2),
		fluent.border,
		unit.Dp(4),
		layout.Inset{Top: 9, Bottom: 9, Left: 10, Right: 10},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(3))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, strings.TrimSpace(profile.Name), unit.Sp(12), chooseColor(active, fluent.accent, fluent.text), font.Medium)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, strings.ToUpper(strings.TrimSpace(profile.APIMode)), unit.Sp(10), fluent.textDim, font.Normal)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !active {
						return layout.Dimensions{}
					}
					return a.badge(gtx, "当前", fluent.accentSoft, fluent.accent)
				}),
			)
		},
	)
}

func (a *App) layoutHistorySummaryCard(
	gtx layout.Context,
	snap snapshot,
	filtered []sharedCompat.HistoryItem,
	generateCount int,
	editCount int,
) layout.Dimensions {
	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(3))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.sectionEyebrow(gtx, "历史")
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								countText := strconv.Itoa(len(filtered))
								if len(filtered) != len(snap.History) {
									countText += " / " + strconv.Itoa(len(snap.History))
								}
								return a.label(gtx, countText+" 项", unit.Sp(11), fluent.textMuted, font.Normal)
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.compactButton(gtx, &a.historyCollapseButton, chooseHistoryCollapseLabel(a.historyRailCollapsed), a.historyRailCollapsed)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.pillButton(gtx, &a.historyModeButtons[0], "全部 "+strconv.Itoa(len(snap.History)), a.historyModeFilter == "all")
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.pillButton(gtx, &a.historyModeButtons[1], "文生图 "+strconv.Itoa(generateCount), a.historyModeFilter == "generate")
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.pillButton(gtx, &a.historyModeButtons[2], "图生图 "+strconv.Itoa(editCount), a.historyModeFilter == "edit")
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.searchField(gtx, &a.historyQueryInput, "搜索 prompt...")
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return a.pillButton(gtx, &a.historyDateButtons[0], "全部", a.historyDateFilter == "all")
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return a.pillButton(gtx, &a.historyDateButtons[1], "今天", a.historyDateFilter == "today")
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return a.pillButton(gtx, &a.historyDateButtons[2], "本周", a.historyDateFilter == "week")
					}),
				)
			}),
		)
	})
}

func (a *App) layoutLatestHistoryCard(gtx layout.Context, item sharedCompat.HistoryItem, active bool) layout.Dimensions {
	btn := a.historyButton("feature:" + item.ID)
	reuseBtn := a.historyActionButton("feature-reuse:" + item.ID)
	deleteBtn := a.historyActionButton("feature-delete:" + item.ID)
	return a.surfaceButton(
		gtx,
		btn,
		chooseColor(active, fluent.surface2, fluent.surface),
		fluent.surface2,
		chooseColor(active, rgba(0x005fb8, 0x48), fluent.border),
		unit.Dp(6),
		layout.Inset{Top: 10, Bottom: 10, Left: 10, Right: 10},
		func(gtx layout.Context) layout.Dimensions {
			img, _ := a.imageForHistoryItem(item)
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.sectionEyebrow(gtx, "最近作品")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, formatHistoryClock(item.CreatedAt), unit.Sp(11), fluent.textDim, font.Normal)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.imageThumb(gtx, img, unit.Dp(88), unit.Dp(88), unit.Dp(4))
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(5))}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, shortPrompt(item.Prompt), unit.Sp(12), fluent.text, font.Medium)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, historyMetaText(item), unit.Sp(11), fluent.textMuted, font.Normal)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, historyPathText(item.SavedPath), unit.Sp(10), fluent.textDim, font.Normal)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									detailBtn := a.historyActionButton("feature-detail:" + item.ID)
									return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.compactButton(gtx, detailBtn, "详情", false)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.compactButton(gtx, reuseBtn, "设为源图", false)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.compactButton(gtx, deleteBtn, "删除", false)
										}),
									)
								}),
							)
						}),
					)
				}),
			)
		},
	)
}

func (a *App) layoutPromptGroupModal(gtx layout.Context) layout.Dimensions {
	snap := a.readSnapshot()
	group := snap.ActivePromptGroup
	if group.Key == "" {
		return layout.Dimensions{}
	}
	for a.closePromptGroupButton.Clicked(gtx) {
		a.closePromptGroup()
	}
	for _, item := range group.Items {
		item := item
		btn := a.historyButton("modal:" + item.ID)
		for btn.Clicked(gtx) {
			if err := a.loadHistoryPreview(item, true); err != nil && !isMissingPreview(err) {
				a.appendLog("载入历史结果失败: " + err.Error())
			}
			a.closePromptGroup()
		}
	}

	paint.FillShape(gtx.Ops, rgba(0x000000, 0x52), clip.Rect{Max: gtx.Constraints.Max}.Op())
	gtx.Constraints.Min = gtx.Constraints.Max
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = image.Point{}
		return fixedWidth(gtx, unit.Dp(760), func(gtx layout.Context) layout.Dimensions {
			return fixedHeight(gtx, unit.Dp(560), func(gtx layout.Context) layout.Dimensions {
				return a.borderedSurface(gtx, fluent.surface, unit.Dp(8), fluent.border, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						children := []layout.FlexChild{
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return a.layoutHistoryGroupPile(gtx, group)
											}),
											layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
												return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
													layout.Rigid(func(gtx layout.Context) layout.Dimensions {
														return a.label(gtx, choosePromptGroupTitle(group), unit.Sp(18), fluent.text, font.SemiBold)
													}),
													layout.Rigid(func(gtx layout.Context) layout.Dimensions {
														meta := strconv.Itoa(len(group.Items)) + " 张 · " + historyMetaText(group.Representative)
														return a.label(gtx, meta, unit.Sp(11), fluent.textMuted, font.Normal)
													}),
												)
											}),
										)
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return fixedWidth(gtx, unit.Dp(88), func(gtx layout.Context) layout.Dimensions {
											return a.button(gtx, &a.closePromptGroupButton, "关闭", fluent.surface2, fluent.text)
										})
									}),
								)
							}),
							layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								rowCount := (len(group.Items) + 2) / 3
								return a.promptGroupList.Layout(gtx, rowCount, func(gtx layout.Context, row int) layout.Dimensions {
									return layout.Inset{Bottom: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return a.layoutPromptGroupModalGridRow(gtx, group.Items, row, snap.SelectedHistoryID)
									})
								})
							}),
						}
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
					})
				})
			})
		})
	})
}

func (a *App) layoutPromptGroupModalGridRow(gtx layout.Context, items []sharedCompat.HistoryItem, row int, selectedHistoryID string) layout.Dimensions {
	cells := make([]layout.FlexChild, 0, 3)
	for col := 0; col < 3; col++ {
		idx := row*3 + col
		if idx >= len(items) {
			cells = append(cells, layout.Flexed(1, layout.Spacer{}.Layout))
			continue
		}
		item := items[idx]
		cells = append(cells, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return a.layoutPromptGroupModalTile(gtx, item, selectedHistoryID == item.ID)
			})
		}))
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, cells...)
}

func (a *App) layoutPromptGroupModalTile(gtx layout.Context, item sharedCompat.HistoryItem, active bool) layout.Dimensions {
	btn := a.historyButton("modal:" + item.ID)
	detailBtn := a.historyActionButton("modal-detail:" + item.ID)
	for detailBtn.Clicked(gtx) {
		a.openResultDetail(item)
	}
	return a.surfaceButton(
		gtx,
		btn,
		chooseColor(active, fluent.surface2, fluent.surface),
		fluent.surface2,
		chooseColor(active, rgba(0x005fb8, 0x48), fluent.border),
		unit.Dp(6),
		layout.Inset{Top: 8, Bottom: 8, Left: 8, Right: 8},
		func(gtx layout.Context) layout.Dimensions {
			img, _ := a.imageForHistoryItem(item)
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.imageThumb(gtx, img, unit.Dp(180), unit.Dp(180), unit.Dp(4))
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, shortPrompt(item.Prompt), unit.Sp(11), fluent.text, font.Medium)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, historyMetaText(item), unit.Sp(10), fluent.textMuted, font.Normal)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					detail := strings.Join(compactNonEmpty([]string{
						chooseBatchIndexLabel(item.BatchIndex),
						formatHistoryClock(item.CreatedAt),
					}), " · ")
					return a.label(gtx, detail, unit.Sp(10), fluent.textDim, font.Normal)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if active {
						return a.badge(gtx, "当前", fluent.accentSoft, fluent.accent)
					}
					return layout.Dimensions{}
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.compactButton(gtx, detailBtn, "详情", false)
				}),
			)
		},
	)
}

func (a *App) layoutHistoryResultsCard(
	gtx layout.Context,
	snap snapshot,
	filtered []sharedCompat.HistoryItem,
	entries []historyPromptEntry,
	visible []historyPromptEntry,
) layout.Dimensions {
	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return a.sectionEyebrow(gtx, "结果")
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						label := strconv.Itoa(len(visible))
						if len(entries) > len(visible) {
							label += " / " + strconv.Itoa(len(entries))
						}
						return a.label(gtx, label, unit.Sp(11), fluent.textMuted, font.Normal)
					}),
				)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(visible) == 0 {
					text := "还没有结果"
					if len(filtered) == 0 && len(snap.History) > 0 {
						text = "没有匹配项"
					}
					return a.emptyPanel(gtx, text)
				}
				return a.historyList.Layout(gtx, len(visible), func(gtx layout.Context, i int) layout.Dimensions {
					entry := visible[i]
					return layout.Inset{Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						if entry.Kind == "group" {
							return a.layoutHistoryGroupRow(gtx, entry.Group, snap.SelectedHistoryID)
						}
						return a.layoutHistoryRow(gtx, entry.Item, entry.Item.ID == snap.SelectedHistoryID)
					})
				})
			}),
		)
	})
}

func (a *App) layoutHistoryGroupRow(gtx layout.Context, group historyPromptGroup, selectedHistoryID string) layout.Dimensions {
	active := historyPromptGroupContains(group, selectedHistoryID)
	summaryBtn := a.historyButton("group:" + group.Key)
	expandBtn := a.historyButton("expand:" + group.Key)
	prompt := group.Prompt
	if prompt == "" {
		prompt = "(无 prompt)"
	}
	meta := historyMetaText(group.Representative)
	if len(group.Items) > 1 {
		meta = strconv.Itoa(len(group.Items)) + " 张 · " + meta
	}

	return a.borderedSurface(gtx, chooseColor(active, fluent.surface2, fluent.surface), unit.Dp(6), chooseColor(active, rgba(0x005fb8, 0x48), fluent.border), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			children := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return summaryBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.layoutHistoryGroupPile(gtx, group)
									}),
									layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return a.label(gtx, shortPrompt(prompt), unit.Sp(12), fluent.text, font.Medium)
											}),
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return a.label(gtx, meta, unit.Sp(10), fluent.textMuted, font.Normal)
											}),
										)
									}),
								)
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.compactButton(gtx, expandBtn, "展开", false)
						}),
					)
				}),
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}

func (a *App) layoutHistoryGroupPile(gtx layout.Context, group historyPromptGroup) layout.Dimensions {
	return fixedWidth(gtx, unit.Dp(58), func(gtx layout.Context) layout.Dimensions {
		return fixedHeight(gtx, unit.Dp(44), func(gtx layout.Context) layout.Dimensions {
			maxThumbs := min(3, len(group.Items))
			return layout.Stack{}.Layout(gtx,
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					return layout.Dimensions{Size: image.Pt(gtx.Constraints.Min.X, gtx.Constraints.Min.Y)}
				}),
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					for idx := maxThumbs - 1; idx >= 0; idx-- {
						item := group.Items[idx]
						offset := idx * 6
						img, _ := a.imageForHistoryItem(item)
						layout.Inset{
							Left: unit.Dp(float32(offset)),
							Top:  unit.Dp(float32(offset) / 2),
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.imageThumb(gtx, img, unit.Dp(40), unit.Dp(30), unit.Dp(4))
						})
					}
					return layout.Dimensions{Size: image.Pt(gtx.Constraints.Min.X, gtx.Constraints.Min.Y)}
				}),
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					return layout.S.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Right: unit.Dp(2), Bottom: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.badge(gtx, strconv.Itoa(len(group.Items)), fluent.accentSoft, fluent.accent)
						})
					})
				}),
			)
		})
	})
}

func (a *App) layoutHistoryRow(gtx layout.Context, item sharedCompat.HistoryItem, active bool) layout.Dimensions {
	btn := a.historyButton("row:" + item.ID)
	detailBtn := a.historyActionButton("row-detail:" + item.ID)
	reuseBtn := a.historyActionButton("row-reuse:" + item.ID)
	deleteBtn := a.historyActionButton("row-delete:" + item.ID)
	return a.surfaceButton(
		gtx,
		btn,
		chooseColor(active, fluent.surface2, fluent.surface),
		fluent.surface2,
		chooseColor(active, rgba(0x005fb8, 0x48), fluent.border),
		unit.Dp(6),
		layout.Inset{Top: 7, Bottom: 7, Left: 7, Right: 7},
		func(gtx layout.Context) layout.Dimensions {
			img, _ := a.imageForHistoryItem(item)
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.imageThumb(gtx, img, unit.Dp(48), unit.Dp(48), unit.Dp(4))
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, shortPrompt(item.Prompt), unit.Sp(12), fluent.text, font.Medium)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, historyMetaText(item), unit.Sp(10), fluent.textMuted, font.Normal)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, historyPathText(item.SavedPath), unit.Sp(10), fluent.textDim, font.Normal)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Alignment: layout.End, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if active {
								return a.badge(gtx, "当前", fluent.accentSoft, fluent.accent)
							}
							return layout.Dimensions{}
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.compactButton(gtx, detailBtn, "详情", false)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.compactButton(gtx, reuseBtn, "源图", false)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.compactButton(gtx, deleteBtn, "删除", false)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, formatHistoryClock(item.CreatedAt), unit.Sp(10), fluent.textDim, font.Normal)
						}),
					)
				}),
			)
		},
	)
}

func (a *App) layoutLogsCard(gtx layout.Context, snap snapshot) layout.Dimensions {
	return a.card(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return a.sectionEyebrow(gtx, "运行日志")
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.compactButton(gtx, &a.clearLogButton, "清空", false)
					}),
				)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(snap.Logs) == 0 {
					return a.emptyPanel(gtx, "暂无日志")
				}
				return a.logList.Layout(gtx, len(snap.Logs), func(gtx layout.Context, i int) layout.Dimensions {
					idx := len(snap.Logs) - 1 - i
					line := snap.Logs[idx]
					return layout.Inset{Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.borderedSurface(gtx, fluent.surface, unit.Dp(4), fluent.border, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, line, unit.Sp(11), fluent.textMuted, font.Normal)
							})
						})
					})
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				raw := strings.TrimSpace(snap.Result.RawPath)
				if raw == "" {
					raw = "Raw response: 暂无"
				} else {
					raw = "Raw response: " + raw
				}
				return a.label(gtx, raw, unit.Sp(10), fluent.textDim, font.Normal)
			}),
		)
	})
}

func (a *App) emptyPanel(gtx layout.Context, text string) layout.Dimensions {
	return a.surface(gtx, fluent.surface2, unit.Dp(6), func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, text, unit.Sp(12), fluent.textMuted, font.Normal)
		})
	})
}

func chooseHistoryCollapseLabel(collapsed bool) string {
	if collapsed {
		return "展开"
	}
	return "折叠"
}

func choosePromptGroupTitle(group historyPromptGroup) string {
	if strings.TrimSpace(group.Prompt) == "" {
		return "同提示词结果"
	}
	return group.Prompt
}

func chooseBatchIndexLabel(batchIndex int) string {
	if batchIndex < 0 {
		return "历史结果"
	}
	return "第 " + strconv.Itoa(batchIndex+1) + " 张"
}

func historyMetaText(item sharedCompat.HistoryItem) string {
	mode := "文生图"
	if item.Mode == "edit" {
		mode = "图生图"
	}
	format := strings.ToUpper(strings.TrimSpace(item.OutputFormat))
	style := ""
	if strings.TrimSpace(item.StyleTag) != "" {
		style = "#" + styleChoiceLabel(item.StyleTag)
	}
	return strings.Join(compactNonEmpty([]string{mode, item.Size, item.Quality, style, format}), " · ")
}

func historyPathText(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "未登记保存路径"
	}
	return filepath.Base(path)
}

func formatHistoryClock(createdAt int64) string {
	if createdAt <= 0 {
		return ""
	}
	return time.UnixMilli(createdAt).Format("15:04")
}
