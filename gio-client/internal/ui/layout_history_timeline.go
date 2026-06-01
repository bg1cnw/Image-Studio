package ui

import (
	"strconv"
	"strings"

	sharedCompat "image-studio/shared/compat"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
)

func (a *App) layoutHistoryTimelineModal(gtx layout.Context) layout.Dimensions {
	for a.closeHistoryTimelineButton.Clicked(gtx) {
		a.closeHistoryTimeline()
	}
	for a.historyTimelineModePickerButton.Clicked(gtx) {
		a.historyTimelineModePickerOpen = !a.historyTimelineModePickerOpen
		if a.historyTimelineModePickerOpen {
			a.historyTimelineDatePickerOpen = false
		}
	}
	for a.historyTimelineDatePickerButton.Clicked(gtx) {
		a.historyTimelineDatePickerOpen = !a.historyTimelineDatePickerOpen
		if a.historyTimelineDatePickerOpen {
			a.historyTimelineModePickerOpen = false
		}
	}
	for idx, value := range []string{"all", "generate", "edit"} {
		for a.historyTimelineModeButtons[idx].Clicked(gtx) {
			a.historyTimelineModeFilter = value
			a.historyTimelineModePickerOpen = false
		}
	}
	for idx, value := range []string{"all", "today", "week"} {
		for a.historyTimelineDateButtons[idx].Clicked(gtx) {
			a.historyTimelineDateFilter = value
			a.historyTimelineDatePickerOpen = false
		}
	}

	snap := a.readSnapshot()
	if !snap.HistoryTimelineOpen {
		return layout.Dimensions{}
	}

	filtered := a.filteredTimelineHistory(snap.History)
	dayGroups := buildHistoryDayGroups(filtered)
	for _, dayGroup := range dayGroups {
		for _, entry := range dayGroup.Entries {
			if entry.Kind == "group" {
				summaryBtn := a.historyButton("timeline-group:" + entry.Group.Key)
				for summaryBtn.Clicked(gtx) {
					if err := a.loadHistoryPreview(entry.Group.Representative, true); err != nil && !isMissingPreview(err) {
						a.appendLog("载入历史结果失败: " + err.Error())
					} else {
						a.closeHistoryTimeline()
					}
				}
				openBtn := a.historyActionButton("timeline-group-open:" + entry.Group.Key)
				for openBtn.Clicked(gtx) {
					a.openPromptGroup(entry.Group)
					a.closeHistoryTimeline()
				}
				continue
			}

			item := entry.Item
			rowBtn := a.historyButton("timeline-row:" + item.ID)
			for rowBtn.Clicked(gtx) {
				if err := a.loadHistoryPreview(item, true); err != nil && !isMissingPreview(err) {
					a.appendLog("载入历史结果失败: " + err.Error())
				} else {
					a.closeHistoryTimeline()
				}
			}
			detailBtn := a.historyActionButton("timeline-detail:" + item.ID)
			for detailBtn.Clicked(gtx) {
				a.openResultDetail(item)
				a.closeHistoryTimeline()
			}
			reuseBtn := a.historyActionButton("timeline-reuse:" + item.ID)
			for reuseBtn.Clicked(gtx) {
				a.reuseHistoryItemAsSource(item)
				a.closeHistoryTimeline()
			}
			deleteBtn := a.historyActionButton("timeline-delete:" + item.ID)
			for deleteBtn.Clicked(gtx) {
				a.deleteHistoryItem(item.ID)
			}
		}
	}

	countText := strconv.Itoa(len(filtered))
	if len(filtered) != len(snap.History) {
		countText += " / " + strconv.Itoa(len(snap.History))
	}
	return a.layoutStandardModal(
		gtx,
		unit.Dp(920),
		unit.Dp(660),
		"更多历史",
		countText+" 项",
		&a.closeHistoryTimelineButton,
		func(gtx layout.Context) layout.Dimensions {
			base := func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.searchField(gtx, &a.historyTimelineQueryInput, "搜索 prompt / revised prompt...")
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutTimelineFilterRow(gtx)
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if len(dayGroups) == 0 {
							return a.emptyPanel(gtx, "没有匹配的历史记录")
						}
						return a.historyTimelineList.Layout(gtx, len(dayGroups), func(gtx layout.Context, index int) layout.Dimensions {
							return layout.Inset{Bottom: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.layoutHistoryTimelineDayGroup(gtx, dayGroups[index], snap.SelectedHistoryID)
							})
						})
					}),
				)
			}
			if !a.historyTimelineModePickerOpen && !a.historyTimelineDatePickerOpen {
				return base(gtx)
			}
			return layout.Stack{}.Layout(gtx,
				layout.Stacked(base),
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					macro := op.Record(gtx.Ops)
					overlayDims := layout.Inset{Top: unit.Dp(84)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.layoutTimelineFilterMenus(gtx)
					})
					call := macro.Stop()
					call.Add(gtx.Ops)
					return overlayDims
				}),
			)
		},
	)
}

func timelineModeFilterLabel(value string) string {
	switch strings.TrimSpace(value) {
	case "generate":
		return "文生图"
	case "edit":
		return "图生图"
	default:
		return "全部模式"
	}
}

func timelineDateFilterLabel(value string) string {
	switch strings.TrimSpace(value) {
	case "today":
		return "今天"
	case "week":
		return "近 7 天"
	default:
		return "全部日期"
	}
}

func (a *App) timelineFilterButton(gtx layout.Context, btn *widget.Clickable, label string, open bool) layout.Dimensions {
	return a.surfaceButton(
		gtx,
		btn,
		chooseColor(open, fluent.surface2, fluent.surface),
		fluent.surface2,
		fluent.border2,
		fluentControlRadius,
		layout.Inset{Top: 9, Bottom: 9, Left: 10, Right: 10},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.singleLineLabel(gtx, label, unit.Sp(12), fluent.text, font.Normal)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					icon := uiIconExpand
					if open {
						icon = uiIconCollapse
					}
					return fixedWidth(gtx, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
						return fixedHeight(gtx, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
							return icon.Layout(gtx, fluent.textDim)
						})
					})
				}),
			)
		},
	)
}

func (a *App) layoutTimelineFilterRow(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.timelineFilterButton(
				gtx,
				&a.historyTimelineModePickerButton,
				timelineModeFilterLabel(a.historyTimelineModeFilter),
				a.historyTimelineModePickerOpen,
			)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.timelineFilterButton(
				gtx,
				&a.historyTimelineDatePickerButton,
				timelineDateFilterLabel(a.historyTimelineDateFilter),
				a.historyTimelineDatePickerOpen,
			)
		}),
	)
}

func (a *App) layoutTimelineFilterMenus(gtx layout.Context) layout.Dimensions {
	if !a.historyTimelineModePickerOpen && !a.historyTimelineDatePickerOpen {
		return layout.Dimensions{}
	}
	return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if !a.historyTimelineModePickerOpen {
				return layout.Dimensions{}
			}
			return a.timelineFilterMenu(gtx, []timelineFilterOption{
				{Label: "全部模式", Button: &a.historyTimelineModeButtons[0], Active: a.historyTimelineModeFilter == "all"},
				{Label: "文生图", Button: &a.historyTimelineModeButtons[1], Active: a.historyTimelineModeFilter == "generate"},
				{Label: "图生图", Button: &a.historyTimelineModeButtons[2], Active: a.historyTimelineModeFilter == "edit"},
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if !a.historyTimelineDatePickerOpen {
				return layout.Dimensions{}
			}
			return a.timelineFilterMenu(gtx, []timelineFilterOption{
				{Label: "全部日期", Button: &a.historyTimelineDateButtons[0], Active: a.historyTimelineDateFilter == "all"},
				{Label: "今天", Button: &a.historyTimelineDateButtons[1], Active: a.historyTimelineDateFilter == "today"},
				{Label: "近 7 天", Button: &a.historyTimelineDateButtons[2], Active: a.historyTimelineDateFilter == "week"},
			})
		}),
	)
}

type timelineFilterOption struct {
	Label  string
	Button *widget.Clickable
	Active bool
}

func (a *App) timelineFilterMenu(gtx layout.Context, options []timelineFilterOption) layout.Dimensions {
	return a.borderedSurface(gtx, fluent.surface, fluentControlRadius, fluent.border, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, len(options)*2)
			for idx := range options {
				opt := options[idx]
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.surfaceButton(
						gtx,
						opt.Button,
						chooseColor(opt.Active, fluent.accentSoft, rgba(0xffffff, 0x00)),
						chooseColor(opt.Active, accentAlpha(0x28), fluent.surface2),
						rgba(0xffffff, 0x00),
						fluentControlRadius,
						layout.Inset{Top: 8, Bottom: 8, Left: 10, Right: 10},
						func(gtx layout.Context) layout.Dimensions {
							return a.singleLineLabel(gtx, opt.Label, unit.Sp(11), chooseColor(opt.Active, fluent.accent, fluent.text), font.Medium)
						},
					)
				}))
				if idx != len(options)-1 {
					children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout))
				}
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}

func (a *App) layoutHistoryTimelineDayGroup(gtx layout.Context, dayGroup historyDayGroup, selectedHistoryID string) layout.Dimensions {
	children := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedWidth(gtx, unit.Dp(8), func(gtx layout.Context) layout.Dimensions {
						return fixedHeight(gtx, unit.Dp(8), func(gtx layout.Context) layout.Dimensions {
							return a.surface(gtx, fluent.accent, unit.Dp(4), layout.Spacer{}.Layout)
						})
					})
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, dayGroup.Label, unit.Sp(13), fluent.text, font.SemiBold)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.metaBadge(gtx, strconv.Itoa(len(dayGroup.Entries))+" 组", true)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
	}
	for idx, entry := range dayGroup.Entries {
		entry := entry
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.layoutHistoryTimelineEntry(gtx, entry, selectedHistoryID)
		}))
		if idx != len(dayGroup.Entries)-1 {
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
		}
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (a *App) layoutHistoryTimelineEntry(gtx layout.Context, entry historyPromptEntry, selectedHistoryID string) layout.Dimensions {
	if entry.Kind == "group" {
		return a.layoutTimelineTrackRow(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.layoutHistoryTimelineGroupRow(gtx, entry.Group, selectedHistoryID)
		})
	}
	return a.layoutTimelineTrackRow(gtx, func(gtx layout.Context) layout.Dimensions {
		return a.layoutHistoryTimelineRow(gtx, entry.Item, entry.Item.ID == selectedHistoryID)
	})
}

func (a *App) layoutHistoryTimelineGroupRow(gtx layout.Context, group historyPromptGroup, selectedHistoryID string) layout.Dimensions {
	active := historyPromptGroupContains(group, selectedHistoryID)
	summaryBtn := a.historyButton("timeline-group:" + group.Key)
	openBtn := a.historyActionButton("timeline-group-open:" + group.Key)
	prompt := group.Prompt
	if prompt == "" {
		prompt = "(无 prompt)"
	}
	meta := strconv.Itoa(len(group.Items)) + " 张 · " + historyRailMetaText(group.Representative)

	return a.borderedSurface(gtx, chooseColor(active, fluent.surface2, fluent.surface), unit.Dp(6), chooseColor(active, accentAlpha(0x48), fluent.border), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return summaryBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.layoutHistoryGroupPile(gtx, group)
					})
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return summaryBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return fixedWidth(gtx, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
											return fixedHeight(gtx, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
												return uiIconGrid.Layout(gtx, fluent.accent)
											})
										})
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.sectionEyebrow(gtx, "同提示词")
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.metaBadge(gtx, strconv.Itoa(len(group.Items))+" 张", true)
									}),
								)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.clampedLabel(gtx, shortPrompt(prompt), unit.Sp(12), fluent.text, font.Medium, 2)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.singleLineLabel(gtx, meta, unit.Sp(10), fluent.textMuted, font.Normal)
							}),
						)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.compactButton(gtx, summaryBtn, "查看最新", active)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.compactIconButton(gtx, openBtn, uiIconGrid, false)
				}),
			)
		})
	})
}

func (a *App) layoutTimelineTrackRow(gtx layout.Context, body layout.Widget) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Start}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedWidth(gtx, unit.Dp(120), func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return fixedWidth(gtx, unit.Dp(10), func(gtx layout.Context) layout.Dimensions {
								return fixedHeight(gtx, unit.Dp(10), func(gtx layout.Context) layout.Dimensions {
									return a.surface(gtx, fluent.accent, unit.Dp(5), layout.Spacer{}.Layout)
								})
							})
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return fixedWidth(gtx, unit.Dp(1), func(gtx layout.Context) layout.Dimensions {
								return a.surface(gtx, withAlpha(fluent.textDim, 0x22), 0, layout.Spacer{}.Layout)
							})
						}),
					)
				})
			})
		}),
		layout.Flexed(1, body),
	)
}

func (a *App) layoutHistoryTimelineRow(gtx layout.Context, item sharedCompat.HistoryItem, active bool) layout.Dimensions {
	rowBtn := a.historyButton("timeline-row:" + item.ID)
	detailBtn := a.historyActionButton("timeline-detail:" + item.ID)
	reuseBtn := a.historyActionButton("timeline-reuse:" + item.ID)
	deleteBtn := a.historyActionButton("timeline-delete:" + item.ID)
	return a.surfaceButton(
		gtx,
		rowBtn,
		chooseColor(active, fluent.surface2, fluent.surface),
		fluent.surface2,
		chooseColor(active, accentAlpha(0x48), fluent.border),
		unit.Dp(6),
		layout.Inset{Top: 8, Bottom: 8, Left: 8, Right: 8},
		func(gtx layout.Context) layout.Dimensions {
			img, _ := a.imageForHistoryItem(item)
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.layoutHistoryModeThumb(gtx, img, item.Mode, unit.Dp(92), unit.Dp(72))
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.clampedLabel(gtx, shortPrompt(item.Prompt), unit.Sp(12), fluent.text, font.Medium, 2)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.metaBadgeRow(gtx, historyMetaBadgeItems(item), true)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.singleLineLabel(gtx, formatHistoryClock(item.CreatedAt), unit.Sp(10), fluent.textDim, font.Normal)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if strings.TrimSpace(item.RevisedPrompt) == "" {
								return layout.Dimensions{}
							}
							return a.borderedSurface(gtx, fluent.surface2, unit.Dp(8), rgba(0xffffff, 0x00), func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(2))}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.label(gtx, "优化后", unit.Sp(10), fluent.textMuted, font.SemiBold)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.clampedLabel(gtx, item.RevisedPrompt, unit.Sp(10), fluent.textDim, font.Normal, 2)
										}),
									)
								})
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.compactButton(gtx, detailBtn, "查看", active)
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
