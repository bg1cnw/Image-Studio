package ui

import (
	"fmt"
	"image"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/yuanhua/image-gptcodex/pkg/client"
)

const repoURL = "https://github.com/RoseKhlifa/Image-Studio"
const issuesURL = "https://github.com/RoseKhlifa/Image-Studio/issues"

func (a *App) layout(gtx layout.Context) layout.Dimensions {
	snap := a.readSnapshot()
	for a.runButton.Clicked(gtx) {
		a.startRun()
	}
	for a.cancelButton.Clicked(gtx) {
		a.cancelRun()
	}
	for a.clearLogButton.Clicked(gtx) {
		a.clearLogs()
	}

	paint.FillShape(gtx.Ops, fluent.bg, clip.Rect{Max: gtx.Constraints.Max}.Op())
	if gtx.Constraints.Max.X > 0 && gtx.Constraints.Max.Y > 0 {
		paintLinearGradient(gtx, image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y), 0, withAlpha(fluent.panel, 0x30), rgba(0xffffff, 0x00))
		glowHeight := min(gtx.Dp(unit.Dp(220)), gtx.Constraints.Max.Y)
		if glowHeight > 0 {
			paintLinearGradient(gtx, image.Rect(0, 0, gtx.Constraints.Max.X, glowHeight), 0, fluent.bgGlow, rgba(0xffffff, 0x00))
		}
	}
	children := []layout.FlexChild{}
	if !snap.Fullscreen {
		children = append(children,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return fixedHeight(gtx, unit.Dp(48), a.layoutHeader)
			}),
		)
		if len(a.workspaces) > 1 {
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return fixedHeight(gtx, unit.Dp(38), a.layoutWorkspaceBar)
			}))
		}
	}
	children = append(children, layout.Flexed(1, a.layoutBody))
	if !snap.Fullscreen {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedHeight(gtx, unit.Dp(42), a.layoutFooter)
		}))
	}
	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	if snap.SavePromptVisible {
		a.layoutSavePrompt(gtx)
	}
	if a.settingsModalOpen {
		a.layoutSettingsModal(gtx)
		if a.settingsHelpOpen {
			a.layoutSettingsHelpModal(gtx)
		}
	}
	if snap.ActiveResultDetail.ID != "" || snap.ActiveResultDetail.SavedPath != "" {
		a.layoutResultDetailModal(gtx)
	}
	if snap.ActivePromptGroup.Key != "" {
		a.layoutPromptGroupModal(gtx)
	}
	if snap.HistoryTimelineOpen {
		a.layoutHistoryTimelineModal(gtx)
	}
	return dims
}

func (a *App) layoutHeader(gtx layout.Context) layout.Dimensions {
	for idx, mode := range []string{"system", "light", "dark"} {
		for a.themeButtons[idx].Clicked(gtx) {
			a.persistThemeMode(mode)
		}
	}
	for a.headerAddWorkspaceButton.Clicked(gtx) {
		a.createWorkspace()
	}
	for a.headerQuoteButton.Clicked(gtx) {
		a.headerQuoteIndex = nextHeaderQuoteIndex(a.headerQuoteIndex)
	}
	for a.githubButton.Clicked(gtx) {
		if err := openExternalURL(repoURL); err != nil {
			a.appendLog("打开 GitHub 失败: " + err.Error())
		}
	}
	for a.headerStarButton.Clicked(gtx) {
		if err := openExternalURL(repoURL); err != nil {
			a.appendLog("打开 GitHub 失败: " + err.Error())
		}
	}
	for a.settingsButton.Clicked(gtx) {
		a.settingsModalOpen = true
	}

	return a.borderedSurface(gtx, fluent.toolbar, unit.Dp(0), fluent.border, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		return layout.Inset{Top: 8, Bottom: 8, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.layoutHeaderBrand(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Stack{}.Layout(gtx,
						layout.Stacked(func(gtx layout.Context) layout.Dimensions {
							return a.headerIconButtonIcon(gtx, &a.headerAddWorkspaceButton, uiIconAdd, false)
						}),
						layout.Stacked(func(gtx layout.Context) layout.Dimensions {
							if len(a.workspaces) <= 1 {
								return layout.Dimensions{}
							}
							return layout.NE.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(-2), Right: unit.Dp(-2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return a.badge(gtx, fmt.Sprintf("%d", len(a.workspaces)), fluent.accent, fluent.white)
								})
							})
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.borderedSurface(gtx, fluent.surface, fluentControlRadius, accentAlpha(0x12), func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: 2, Bottom: 2, Left: 2, Right: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(2))}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.headerIconButtonIcon(gtx, &a.themeButtons[0], uiIconSystem, a.themeMode == "system")
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.headerIconButtonIcon(gtx, &a.themeButtons[1], uiIconLight, a.themeMode == "light")
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.headerIconButtonIcon(gtx, &a.themeButtons[2], uiIconDark, a.themeMode == "dark")
								}),
							)
						})
					})
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.headerIconButtonIcon(gtx, &a.githubButton, uiIconLaunch, false)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.headerIconButtonIcon(gtx, &a.headerStarButton, uiIconStar, false)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.headerIconButtonIcon(gtx, &a.settingsButton, uiIconSettings, a.settingsModalOpen)
				}),
			)
		})
	})
}

func (a *App) layoutHeaderBrand(gtx layout.Context) layout.Dimensions {
	quote := currentHeaderQuote(a.headerQuoteIndex)
	quoteText := "图像工作台"
	if quote.Text != "" {
		quoteText = quote.Text
		if quote.From != "" {
			quoteText += " - " + quote.From
		}
	}
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(1))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, "Image Studio", unit.Sp(14), fluent.text, font.SemiBold)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.surfaceButton(
						gtx,
						&a.headerQuoteButton,
						rgba(0xffffff, 0x00),
						rgba(0x000000, 0x06),
						rgba(0xffffff, 0x00),
						unit.Dp(4),
						layout.Inset{Top: 1, Bottom: 1, Left: 0, Right: 4},
						func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(5))}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return a.singleLineLabel(gtx, quoteText, unit.Sp(10), fluent.textMuted, font.Normal)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									if !a.headerQuoteButton.Hovered() {
										return layout.Dimensions{}
									}
									return fixedWidth(gtx, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
										return fixedHeight(gtx, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
											return uiIconRefresh.Layout(gtx, fluent.textDim)
										})
									})
								}),
							)
						},
					)
				}),
			)
		}),
	)
}

func (a *App) layoutFooter(gtx layout.Context) layout.Dimensions {
	for a.footerOutputButton.Clicked(gtx) {
		if err := openPath(a.outputDirInput.Text()); err != nil {
			a.appendLog("打开输出目录失败: " + err.Error())
		}
	}
	for a.footerGithubButton.Clicked(gtx) {
		if err := openExternalURL(repoURL); err != nil {
			a.appendLog("打开 GitHub 失败: " + err.Error())
		}
	}
	for a.footerFeedbackButton.Clicked(gtx) {
		if err := openExternalURL(issuesURL); err != nil {
			a.appendLog("打开反馈页失败: " + err.Error())
		}
	}

	snap := a.readSnapshot()
	state := "就绪"
	dot := fluent.textDim
	if snap.Running {
		state = "运行中"
		dot = fluent.accent
	}
	todayCount := todayHistoryCount(snap.History, time.Now())
	return a.borderedSurface(gtx, fluent.bg2, unit.Dp(0), fluent.border, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		return layout.Inset{Top: 9, Bottom: 9, Left: 18, Right: 18}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.footerIconTextButton(gtx, &a.footerOutputButton, uiIconFolder, "输出目录")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.footerIconTextButton(gtx, &a.footerGithubButton, uiIconLaunch, "GitHub")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.footerIconTextButton(gtx, &a.footerFeedbackButton, uiIconFeedback, "反馈")
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					children := []layout.FlexChild{
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, fmt.Sprintf("今日已生图: %d", todayCount), unit.Sp(11), fluent.textMuted, font.Medium)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "·", unit.Sp(11), fluent.textDim, font.Normal)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, fmt.Sprintf("总生图: %d", len(snap.History)), unit.Sp(11), fluent.textMuted, font.Medium)
						}),
					}
					if snap.Running {
						children = append(children,
							layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "·", unit.Sp(11), fluent.textDim, font.Normal)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "当前标签: 1", unit.Sp(11), fluent.accent, font.Medium)
							}),
						)
					}
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, state, unit.Sp(11), fluent.textMuted, font.Medium)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return fixedWidth(gtx, unit.Dp(7), func(gtx layout.Context) layout.Dimensions {
								return fixedHeight(gtx, unit.Dp(7), func(gtx layout.Context) layout.Dimensions {
									return a.surface(gtx, dot, unit.Dp(4), layout.Spacer{}.Layout)
								})
							})
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, "v"+client.Version, unit.Sp(11), fluent.textDim, font.Normal)
				}),
			)
		})
	})
}

func (a *App) layoutBody(gtx layout.Context) layout.Dimensions {
	if a.readSnapshot().Fullscreen {
		return a.layoutCanvas(gtx)
	}
	width := gtx.Constraints.Max.X
	leftMin := gtx.Dp(unit.Dp(336))
	leftMax := gtx.Dp(unit.Dp(384))
	rightMin := gtx.Dp(unit.Dp(300))
	rightMax := gtx.Dp(unit.Dp(344))
	centerMin := gtx.Dp(unit.Dp(360))
	leftWidth := clampInt(int(float64(width)*0.25), leftMin, leftMax)
	rightWidth := clampInt(int(float64(width)*0.22), rightMin, rightMax)
	if overflow := leftWidth + rightWidth + centerMin - width; overflow > 0 {
		reduceRight := min(overflow, rightWidth-rightMin)
		rightWidth -= reduceRight
		overflow -= reduceRight
		if overflow > 0 {
			reduceLeft := min(overflow, leftWidth-leftMin)
			leftWidth -= reduceLeft
		}
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedPixelWidth(gtx, leftWidth, a.layoutControls)
		}),
		layout.Flexed(1, a.layoutCanvas),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedPixelWidth(gtx, rightWidth, a.layoutHistoryAndLogs)
		}),
	)
}

func (a *App) layoutWorkspaceBar(gtx layout.Context) layout.Dimensions {
	for {
		event, ok := a.workspaceNameInput.Update(gtx)
		if !ok {
			break
		}
		switch event.(type) {
		case widget.SubmitEvent:
			a.commitWorkspaceRename()
		}
	}
	for a.addWorkspaceButton.Clicked(gtx) {
		a.createWorkspace()
	}
	for _, ws := range a.workspaces {
		ws := ws
		btn := a.workspaceButton("workspace:" + ws.ID)
		for btn.Clicked(gtx) {
			a.handleWorkspacePrimaryClick(ws.ID, gtx.Now)
		}
		closeBtn := a.closeWorkspaceButton("workspace-close:" + ws.ID)
		for closeBtn.Clicked(gtx) {
			a.closeWorkspace(ws.ID)
		}
	}

	return a.borderedSurface(gtx, fluent.toolbar, unit.Dp(0), fluent.border, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		return layout.Inset{Top: 6, Bottom: 0, Left: 10, Right: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, len(a.workspaces)+1)
			for _, ws := range a.workspaces {
				ws := ws
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.layoutWorkspaceTab(gtx, ws, ws.ID == a.activeWorkspaceID)
					})
				}))
			}
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.surfaceButton(
					gtx,
					&a.addWorkspaceButton,
					rgba(0xffffff, 0x00),
					fluent.surface2,
					rgba(0xffffff, 0x00),
					unit.Dp(4),
					layout.Inset{Top: 7, Bottom: 7, Left: 9, Right: 9},
					func(gtx layout.Context) layout.Dimensions {
						return fixedWidth(gtx, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
							return fixedHeight(gtx, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
								return uiIconAdd.Layout(gtx, fluent.textMuted)
							})
						})
					},
				)
			}))
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.End}.Layout(gtx, children...)
		})
	})
}

func (a *App) layoutWorkspaceTab(gtx layout.Context, ws workspaceState, active bool) layout.Dimensions {
	btn := a.workspaceButton("workspace:" + ws.ID)
	closeBtn := a.closeWorkspaceButton("workspace-close:" + ws.ID)
	editing := a.workspaceRenameID == ws.ID
	running := a.isRunning() && ws.ID == a.activeWorkspaceID
	bg := chooseColor(active, fluent.surface, rgba(0xffffff, 0x00))
	hoverBg := chooseColor(active, fluent.surface, fluent.surface2)
	border := chooseColor(active, fluent.border, rgba(0xffffff, 0x00))
	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		fill := bg
		if btn.Hovered() {
			fill = hoverBg
		}
		return a.borderedTopTabSurface(gtx, fill, border, unit.Dp(6), func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 7, Bottom: 6, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if editing {
							return fixedWidth(gtx, unit.Dp(132), func(gtx layout.Context) layout.Dimensions {
								border := fluent.border2
								if gtx.Focused(&a.workspaceNameInput) {
									border = accentAlpha(0xb8)
								}
								return a.borderedSurface(gtx, fluent.surface, fluentControlRadius, border, func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Top: 7, Bottom: 7, Left: 8, Right: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return a.editorText(gtx, &a.workspaceNameInput, "未命名", unit.Sp(11))
									})
								})
							})
						}
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if !running {
									return layout.Dimensions{}
								}
								return fixedWidth(gtx, unit.Dp(8), func(gtx layout.Context) layout.Dimensions {
									return fixedHeight(gtx, unit.Dp(8), func(gtx layout.Context) layout.Dimensions {
										return a.surface(gtx, fluent.accent, unit.Dp(4), layout.Spacer{}.Layout)
									})
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return fixedWidth(gtx, unit.Dp(132), func(gtx layout.Context) layout.Dimensions {
									weight := font.Medium
									if active {
										weight = font.SemiBold
									}
									return a.singleLineLabel(gtx, a.displayedWorkspaceName(ws), unit.Sp(11), chooseColor(active, fluent.text, fluent.textMuted), weight)
								})
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if len(a.workspaces) <= 1 || editing {
							return layout.Dimensions{}
						}
						if !active && !btn.Hovered() {
							return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return fixedWidth(gtx, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
									return layout.Dimensions{Size: image.Pt(gtx.Constraints.Min.X, 0)}
								})
							})
						}
						return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.surfaceButton(
								gtx,
								closeBtn,
								chooseColor(active, rgba(0x000000, 0x00), rgba(0x000000, 0x00)),
								fluent.surface2,
								rgba(0xffffff, 0x00),
								unit.Dp(3),
								layout.Inset{Top: 2, Bottom: 2, Left: 4, Right: 4},
								func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, "×", unit.Sp(10), fluent.textDim, font.Medium)
								},
							)
						})
					}),
				)
			})
		})
	})
}

func (a *App) footerIconTextButton(gtx layout.Context, btn *widget.Clickable, icon *widget.Icon, text string) layout.Dimensions {
	fg := fluent.textMuted
	if btn.Hovered() {
		fg = fluent.toolHoverText
	}
	return a.surfaceButton(
		gtx,
		btn,
		rgba(0xffffff, 0x00),
		fluent.toolHoverBg,
		rgba(0xffffff, 0x00),
		unit.Dp(6),
		layout.Inset{Top: 4, Bottom: 4, Left: 7, Right: 7},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(5))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedWidth(gtx, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
						return fixedHeight(gtx, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
							return icon.Layout(gtx, fg)
						})
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, text, unit.Sp(11), fg, font.Medium)
				}),
			)
		},
	)
}
