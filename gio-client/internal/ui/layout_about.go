package ui

import (
	"image"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"
	"github.com/yuanhua/image-gptcodex/pkg/client"
)

func (a *App) layoutAboutModal(gtx layout.Context) layout.Dimensions {
	for a.closeAboutButton.Clicked(gtx) {
		a.aboutModalOpen = false
	}
	for a.openAboutRepoButton.Clicked(gtx) {
		if err := openExternalURL(repoURL); err != nil {
			a.appendLog("打开 GitHub 失败: " + err.Error())
		}
	}
	for a.openAboutFeedbackButton.Clicked(gtx) {
		if err := openExternalURL(issuesURL); err != nil {
			a.appendLog("打开反馈页失败: " + err.Error())
		}
	}
	for a.openAboutLicenseButton.Clicked(gtx) {
		if err := openExternalURL(licenseURL); err != nil {
			a.appendLog("打开 License 失败: " + err.Error())
		}
	}

	return a.layoutStandardModal(
		gtx,
		unit.Dp(460),
		0,
		"关于 Image Studio",
		"",
		&a.closeAboutButton,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return fixedWidth(gtx, unit.Dp(56), func(gtx layout.Context) layout.Dimensions {
									return fixedHeight(gtx, unit.Dp(56), func(gtx layout.Context) layout.Dimensions {
										return a.elevatedBorderedSurface(gtx, fluent.white, unit.Dp(12), fluent.border, image.Pt(0, 1), func(gtx layout.Context) layout.Dimensions {
											return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return fixedWidth(gtx, unit.Dp(22), func(gtx layout.Context) layout.Dimensions {
													return fixedHeight(gtx, unit.Dp(22), func(gtx layout.Context) layout.Dimensions {
														return uiIconPhoto.Layout(gtx, fluent.text)
													})
												})
											})
										})
									})
								})
							}),
							layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.titleLabel(gtx, "Image Studio", unit.Sp(18))
							}),
							layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.label(gtx, "v"+client.Version, unit.Sp(11), fluent.textDim, font.Normal)
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.textActionButton(gtx, &a.openAboutLicenseButton, "MIT License", true)
									}),
								)
							}),
						)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, "开源的图片生成 / 编辑客户端。数据都保存在本地机器，不上传任何服务器，API Key 走系统安全存储。", unit.Sp(11), fluent.textMuted, font.Normal)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.helpInfoCard(gtx, "数据", "", func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "本地保存", unit.Sp(11), fluent.text, font.Medium)
							})
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.helpInfoCard(gtx, "运行时", "", func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "Gio Desktop", unit.Sp(11), fluent.text, font.Medium)
							})
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.helpInfoCard(gtx, "技术栈", "", func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "后端: Go >= 1.25 / SSE", unit.Sp(10), fluent.textDim, font.Normal)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "前端: Gio 原生 UI，对齐 Windows Fluent 设计", unit.Sp(10), fluent.textDim, font.Normal)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "支持上游: Responses API / Images API", unit.Sp(10), fluent.textDim, font.Normal)
							}),
						)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.primaryIconTextButton(gtx, &a.openAboutRepoButton, uiIconLaunch, "GitHub 仓库", fluent.accent, fluent.white)
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.compactIconTextButton(gtx, &a.openAboutFeedbackButton, uiIconFeedback, "反馈", false)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, "100% 本地数据 · 无遥测 · 无云端账户 · 无内购", unit.Sp(9), fluent.textDim, font.Normal)
					})
				}),
			)
		},
	)
}
