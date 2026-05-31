package ui

import (
	"fmt"
	"strings"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
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
			return fixedHeight(gtx, unit.Dp(36), a.layoutFooter)
		}))
	}
	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	if snap.SavePromptVisible {
		a.layoutSavePrompt(gtx)
	}
	if a.settingsModalOpen {
		a.layoutSettingsModal(gtx)
	}
	if a.promptHelperOpen {
		a.layoutPromptHelperModal(gtx)
	}
	if snap.ActiveResultDetail.ID != "" || snap.ActiveResultDetail.SavedPath != "" {
		a.layoutResultDetailModal(gtx)
	}
	if snap.ActivePromptGroup.Key != "" {
		a.layoutPromptGroupModal(gtx)
	}
	return dims
}

func (a *App) layoutHeader(gtx layout.Context) layout.Dimensions {
	for idx, mode := range []string{"system", "light", "dark"} {
		for a.themeButtons[idx].Clicked(gtx) {
			a.persistThemeMode(mode)
		}
	}
	for a.githubButton.Clicked(gtx) {
		if err := openExternalURL(repoURL); err != nil {
			a.appendLog("打开 GitHub 失败: " + err.Error())
		}
	}
	for a.settingsButton.Clicked(gtx) {
		a.settingsModalOpen = true
	}

	snap := a.readSnapshot()
	return a.borderedSurface(gtx, fluent.toolbar, unit.Dp(0), fluent.border, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		return layout.Inset{Top: 8, Bottom: 8, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedWidth(gtx, unit.Dp(10), func(gtx layout.Context) layout.Dimensions {
						return fixedHeight(gtx, unit.Dp(10), func(gtx layout.Context) layout.Dimensions {
							return a.surface(gtx, fluent.accent, unit.Dp(5), layout.Spacer{}.Layout)
						})
					})
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(1))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "Image Studio", unit.Sp(14), fluent.text, font.SemiBold)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "Windows / Linux 原生客户端", unit.Sp(11), fluent.textMuted, font.Normal)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.badge(gtx, a.modeLabel(), fluent.accentSoft, fluent.accent)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					label := snap.Status
					if snap.Running {
						label = "运行中 - " + label
					}
					return a.badge(gtx, label, fluent.surface, fluent.textMuted)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.compactButton(gtx, &a.themeButtons[0], "系统", a.themeMode == "system")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.compactButton(gtx, &a.themeButtons[1], "浅色", a.themeMode == "light")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.compactButton(gtx, &a.themeButtons[2], "深色", a.themeMode == "dark")
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.compactButton(gtx, &a.githubButton, "GitHub", false)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.compactButton(gtx, &a.settingsButton, "设置", a.settingsModalOpen)
				}),
			)
		})
	})
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
	activeWorkspaceName := "当前标签"
	for _, ws := range a.workspaces {
		if ws.ID == a.activeWorkspaceID {
			activeWorkspaceName = ws.Name
			break
		}
	}
	return a.borderedSurface(gtx, fluent.toolbar, unit.Dp(0), fluent.border, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		return layout.Inset{Top: 8, Bottom: 8, Left: 14, Right: 14}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.compactButton(gtx, &a.footerOutputButton, "输出目录", false)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.compactButton(gtx, &a.footerGithubButton, "GitHub", false)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.compactButton(gtx, &a.footerFeedbackButton, "反馈", false)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, fmt.Sprintf("今日已生图: %d · 总生图: %d", todayCount, len(snap.History)), unit.Sp(11), fluent.textMuted, font.Medium)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(14)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, activeWorkspaceName, unit.Sp(11), fluent.textDim, font.Medium)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return fixedWidth(gtx, unit.Dp(7), func(gtx layout.Context) layout.Dimensions {
								return fixedHeight(gtx, unit.Dp(7), func(gtx layout.Context) layout.Dimensions {
									return a.surface(gtx, dot, unit.Dp(4), layout.Spacer{}.Layout)
								})
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, state, unit.Sp(11), fluent.textMuted, font.Medium)
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
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
	rightWidth := unit.Dp(320)
	leftWidth := unit.Dp(372)
	if width < gtx.Dp(unit.Dp(1180)) {
		rightWidth = unit.Dp(300)
		leftWidth = unit.Dp(336)
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedWidth(gtx, leftWidth, a.layoutControls)
		}),
		layout.Flexed(1, a.layoutCanvas),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedWidth(gtx, rightWidth, a.layoutHistoryAndLogs)
		}),
	)
}

func (a *App) layoutWorkspaceBar(gtx layout.Context) layout.Dimensions {
	for a.addWorkspaceButton.Clicked(gtx) {
		a.createWorkspace()
	}
	for _, ws := range a.workspaces {
		ws := ws
		btn := a.workspaceButton("workspace:" + ws.ID)
		for btn.Clicked(gtx) {
			a.switchWorkspace(ws.ID)
		}
		closeBtn := a.closeWorkspaceButton("workspace-close:" + ws.ID)
		for closeBtn.Clicked(gtx) {
			a.closeWorkspace(ws.ID)
		}
	}

	return a.borderedSurface(gtx, fluent.toolbar, unit.Dp(0), fluent.border, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		return layout.Inset{Top: 4, Bottom: 4, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, len(a.workspaces)+1)
			for _, ws := range a.workspaces {
				ws := ws
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.layoutWorkspaceTab(gtx, ws, ws.ID == a.activeWorkspaceID)
					})
				}))
			}
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.compactButton(gtx, &a.addWorkspaceButton, "+", false)
			}))
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
		})
	})
}

func (a *App) layoutWorkspaceTab(gtx layout.Context, ws workspaceState, active bool) layout.Dimensions {
	btn := a.workspaceButton("workspace:" + ws.ID)
	closeBtn := a.closeWorkspaceButton("workspace-close:" + ws.ID)
	return a.surfaceButton(
		gtx,
		btn,
		chooseColor(active, fluent.surface, fluent.toolbar),
		chooseColor(active, fluent.surface2, fluent.surface2),
		chooseColor(active, fluent.border, rgba(0xffffff, 0x00)),
		unit.Dp(6),
		layout.Inset{Top: 6, Bottom: 6, Left: 10, Right: 10},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					name := ws.Name
					if strings.TrimSpace(name) == "" {
						name = "未命名"
					}
					return fixedWidth(gtx, unit.Dp(132), func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, name, unit.Sp(11), chooseColor(active, fluent.text, fluent.textMuted), font.Medium)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if len(a.workspaces) <= 1 {
						return layout.Dimensions{}
					}
					return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.compactButton(gtx, closeBtn, "×", false)
					})
				}),
			)
		},
	)
}
