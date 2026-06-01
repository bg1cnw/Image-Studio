package ui

import (
	"image"
	"image/color"
	"strconv"
	"strings"
	"time"

	"image-studio/gio-client/internal/kernel"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/yuanhua/image-gptcodex/pkg/client"
)

func (a *App) layoutCanvas(gtx layout.Context) layout.Dimensions {
	snap := a.readSnapshot()
	sourcePaths := kernel.ParseSourcePaths(a.sourcePathsInput.Text())
	showSourceStrip := a.mode == string(client.ModeEdit) && len(sourcePaths) > 0

	children := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.canvasToolbar(gtx, snap)
		}),
	}
	if showSourceStrip {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedHeight(gtx, unit.Dp(72), func(gtx layout.Context) layout.Dimensions {
				return a.sourceStrip(gtx, sourcePaths)
			})
		}))
	}
	children = append(children,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.resultSurface(gtx, snap)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.canvasStatusBar(gtx, snap)
		}),
	)

	return a.borderedSurface(gtx, fluent.panel2, unit.Dp(0), fluent.border, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

func (a *App) canvasToolbar(gtx layout.Context, snap snapshot) layout.Dimensions {
	if a.saveAsButton.Clicked(gtx) {
		a.openSavePromptForCurrent()
	}
	if a.latestResultButton.Clicked(gtx) {
		if latest, ok := newestHistoryItem(snap.History); ok {
			if err := a.loadHistoryPreview(latest, true); err != nil && !isMissingPreview(err) {
				a.appendLog("载入当前图失败: " + err.Error())
			}
		}
	}
	currentGroup, hasCurrentGroup := findPromptGroupForItem(snap.History, snap.SelectedHistoryID)
	if a.currentGroupButton.Clicked(gtx) {
		if hasCurrentGroup && len(currentGroup.Items) > 1 {
			if snap.ActivePromptGroup.Key == currentGroup.Key {
				a.closePromptGroup()
			} else {
				a.openPromptGroup(currentGroup)
			}
		}
	}
	if a.rotateLeftButton.Clicked(gtx) {
		if next, err := rotateImageFile(snap.Result.SavedPath, -90); err != nil {
			a.appendLog("左转失败: " + err.Error())
		} else if err := a.replaceCurrentResultWithPath(next, "rotate"); err != nil {
			a.appendLog("载入旋转结果失败: " + err.Error())
		}
	}
	if a.rotateRightButton.Clicked(gtx) {
		if next, err := rotateImageFile(snap.Result.SavedPath, 90); err != nil {
			a.appendLog("右转失败: " + err.Error())
		} else if err := a.replaceCurrentResultWithPath(next, "rotate"); err != nil {
			a.appendLog("载入旋转结果失败: " + err.Error())
		}
	}
	if a.flipHorizontalButton.Clicked(gtx) {
		if next, err := flipImageFile(snap.Result.SavedPath, true); err != nil {
			a.appendLog("水平翻转失败: " + err.Error())
		} else if err := a.replaceCurrentResultWithPath(next, "flip"); err != nil {
			a.appendLog("载入翻转结果失败: " + err.Error())
		}
	}
	if a.flipVerticalButton.Clicked(gtx) {
		if next, err := flipImageFile(snap.Result.SavedPath, false); err != nil {
			a.appendLog("竖直翻转失败: " + err.Error())
		} else if err := a.replaceCurrentResultWithPath(next, "flip"); err != nil {
			a.appendLog("载入翻转结果失败: " + err.Error())
		}
	}
	if a.clearCurrentButton.Clicked(gtx) {
		a.clearCurrentResult()
	}
	if a.fullscreenButton.Clicked(gtx) {
		a.toggleFullscreen()
	}
	if a.resultDetailButton.Clicked(gtx) {
		if snap.Result.HasItem {
			a.openResultDetail(snap.Result.Item)
		}
	}
	showReturnLatest := len(snap.History) > 0 && snap.SelectedHistoryID != "" && snap.SelectedHistoryID != snap.History[0].ID

	return a.borderedSurface(gtx, fluent.panel2, unit.Dp(0), fluent.border, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 8, Bottom: 8, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.toolbarIconButton(gtx, &a.rotateLeftButton, uiIconRotateLeft, false)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.toolbarIconButton(gtx, &a.rotateRightButton, uiIconRotateRight, false)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.toolbarIconButton(gtx, &a.flipHorizontalButton, uiIconFlip, false)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.toolbarIconButton(gtx, &a.flipVerticalButton, uiIconFlip, false)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					children := []layout.FlexChild{}
					if showReturnLatest {
						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.compactButton(gtx, &a.latestResultButton, "最近作品", false)
							})
						}))
					}
					if hasCurrentGroup && len(currentGroup.Items) > 1 {
						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.compactButton(gtx, &a.currentGroupButton, "同提示词 "+strconv.Itoa(len(currentGroup.Items)), snap.ActivePromptGroup.Key == currentGroup.Key)
							})
						}))
					}
					return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx, children...)
				}),
				layout.Flexed(1, layout.Spacer{}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !snap.Result.HasItem {
						return layout.Dimensions{}
					}
					return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.staticPill(gtx, chooseModeLabel(snap.Result.Item.Mode), true, false)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if strings.TrimSpace(snap.Result.Item.Size) == "" {
								return layout.Dimensions{}
							}
							return a.staticPill(gtx, snap.Result.Item.Size, false, true)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if strings.TrimSpace(snap.Result.Item.Quality) == "" {
								return layout.Dimensions{}
							}
							return a.staticPill(gtx, snap.Result.Item.Quality, false, true)
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.toolbarSeparator(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					children := []layout.FlexChild{}
					if snap.Result.HasItem {
						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.toolbarIconButton(gtx, &a.resultDetailButton, uiIconInfo, false)
						}))
						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.toolbarIconButton(gtx, &a.clearCurrentButton, uiIconDelete, false)
						}))
					}
					if strings.TrimSpace(snap.Result.SavedPath) != "" {
						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return fixedWidth(gtx, unit.Dp(112), func(gtx layout.Context) layout.Dimensions {
								return a.primaryIconTextButton(gtx, &a.saveAsButton, uiIconDownload, "另存为", fluent.accent, fluent.white)
							})
						}))
					}
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						icon := uiIconFullscreen
						if snap.Fullscreen {
							icon = uiIconFullscreenExit
						}
						return a.toolbarIconButton(gtx, &a.fullscreenButton, icon, snap.Fullscreen)
					}))
					return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx, children...)
				}),
			)
		})
	})
}

func (a *App) toolbarSeparator(gtx layout.Context) layout.Dimensions {
	return fixedWidth(gtx, unit.Dp(1), func(gtx layout.Context) layout.Dimensions {
		return fixedHeight(gtx, unit.Dp(18), func(gtx layout.Context) layout.Dimensions {
			return a.surface(gtx, rgba(0x000000, 0x18), unit.Dp(0), layout.Spacer{}.Layout)
		})
	})
}

func (a *App) sourceStrip(gtx layout.Context, sourcePaths []string) layout.Dimensions {
	for a.addSourceStripButton.Clicked(gtx) {
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
		btn := a.sourceButton("remove:" + path)
		for btn.Clicked(gtx) {
			a.removeSourcePath(path)
		}
	}

	label := "参考图 " + strconv.Itoa(len(sourcePaths)) + " 张"

	return a.borderedSurface(gtx, fluent.panel2, unit.Dp(0), fluent.border, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		return layout.Inset{Top: 8, Bottom: 8, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			header := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, label, unit.Sp(11), fluent.textMuted, font.Medium)
				}),
			}
			tiles := []layout.FlexChild{}
			for _, path := range sourcePaths {
				path := path
				tiles = append(tiles,
					layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutSourceStripTile(gtx, path)
					}),
				)
			}
			tiles = append(tiles,
				layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.layoutSourceStripAddTile(gtx)
				}),
			)
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, header...)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(14)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, tiles...)
				}),
			)
		})
	})
}

func (a *App) layoutSourceStripTile(gtx layout.Context, path string) layout.Dimensions {
	btn := a.sourceButton("remove:" + path)
	img, _ := a.imageForPath(path)
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return a.imageThumbCover(gtx, img, unit.Dp(56), unit.Dp(56), unit.Dp(4))
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.NW.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(2), Top: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.surface(gtx, rgba(0x111111, 0xb8), unit.Dp(3), func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: 1, Bottom: 1, Left: 4, Right: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, sourceStripIndexLabel(path, kernel.ParseSourcePaths(a.sourcePathsInput.Text())), unit.Sp(8), fluent.white, font.Medium)
						})
					})
				})
			})
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.NE.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(2), Right: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.surfaceButton(
						gtx,
						btn,
						rgba(0x111111, 0xc0),
						dangerAlpha(0xd8),
						rgba(0xffffff, 0x00),
						unit.Dp(3),
						layout.Inset{Top: 2, Bottom: 2, Left: 2, Right: 2},
						func(gtx layout.Context) layout.Dimensions {
							return fixedWidth(gtx, unit.Dp(10), func(gtx layout.Context) layout.Dimensions {
								return fixedHeight(gtx, unit.Dp(10), func(gtx layout.Context) layout.Dimensions {
									return uiIconClose.Layout(gtx, fluent.white)
								})
							})
						},
					)
				})
			})
		}),
	)
}

func (a *App) layoutSourceStripAddTile(gtx layout.Context) layout.Dimensions {
	return a.borderedSurface(gtx, fluent.surface, unit.Dp(4), fluent.border2, func(gtx layout.Context) layout.Dimensions {
		return a.surfaceButton(
			gtx,
			&a.addSourceStripButton,
			fluent.surface,
			fluent.surface2,
			rgba(0xffffff, 0x00),
			unit.Dp(4),
			layout.Inset{},
			func(gtx layout.Context) layout.Dimensions {
				return fixedWidth(gtx, unit.Dp(56), func(gtx layout.Context) layout.Dimensions {
					return fixedHeight(gtx, unit.Dp(56), func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return fixedWidth(gtx, unit.Dp(18), func(gtx layout.Context) layout.Dimensions {
								return fixedHeight(gtx, unit.Dp(18), func(gtx layout.Context) layout.Dimensions {
									return uiIconAdd.Layout(gtx, fluent.textDim)
								})
							})
						})
					})
				})
			},
		)
	})
}

func sourceStripIndexLabel(path string, sourcePaths []string) string {
	for idx, value := range sourcePaths {
		if value == path {
			return strconv.Itoa(idx + 1)
		}
	}
	return ""
}

func (a *App) resultSurface(gtx layout.Context, snap snapshot) layout.Dimensions {
	for a.emptyStateImportButton.Clicked(gtx) {
		paths, err := chooseImageFiles()
		if err != nil {
			a.appendLog("选择图片失败: " + err.Error())
		} else if len(paths) > 0 {
			if err := a.replaceCurrentResultWithPath(paths[0], "import"); err != nil {
				a.appendLog("载入本地图片失败: " + err.Error())
			}
		}
	}
	gtx.Constraints.Min = gtx.Constraints.Max
	return a.surface(gtx, fluent.canvasBg, unit.Dp(0), func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		paintCheckerboard(gtx, clip.Rect{Max: gtx.Constraints.Max}.Op(), gtx.Dp(unit.Dp(22)), fluent.canvasBg, fluent.canvasTile)
		if snap.Result.Image == nil {
			return layout.Center.Layout(gtx, a.layoutCanvasEmptyState)
		}
		if snap.Result.Rev != a.imageOpRev {
			a.imageOp = paint.NewImageOp(snap.Result.Image)
			a.imageOpRev = snap.Result.Rev
		}
		return layout.UniformInset(unit.Dp(28)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				img := widget.Image{
					Src:      a.imageOp,
					Fit:      widget.Contain,
					Position: layout.Center,
				}
				return img.Layout(gtx)
			})
		})
	})
}

func (a *App) layoutCanvasEmptyState(gtx layout.Context) layout.Dimensions {
	copy := "先在左侧写提示词，再开始生成第一张图。"
	if a.mode == string(client.ModeEdit) {
		copy = "图生图时可先添加参考图，或从历史结果里挑一张继续编辑。"
	}
	return fixedWidth(gtx, unit.Dp(360), func(gtx layout.Context) layout.Dimensions {
		return a.borderedSurface(gtx, fluent.surface, unit.Dp(16), fluent.border, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(22)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return fixedWidth(gtx, unit.Dp(60), func(gtx layout.Context) layout.Dimensions {
								return fixedHeight(gtx, unit.Dp(60), func(gtx layout.Context) layout.Dimensions {
									return a.borderedSurface(gtx, fluent.accentSoft, unit.Dp(14), accentAlpha(0x22), func(gtx layout.Context) layout.Dimensions {
										return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return fixedWidth(gtx, unit.Dp(24), func(gtx layout.Context) layout.Dimensions {
												return fixedHeight(gtx, unit.Dp(24), func(gtx layout.Context) layout.Dimensions {
													return uiIconPhoto.Layout(gtx, fluent.accent)
												})
											})
										})
									})
								})
							})
						})
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "还没有图片", unit.Sp(18), fluent.text, font.SemiBold)
						})
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return fixedWidth(gtx, unit.Dp(288), func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, copy, unit.Sp(13), fluent.textMuted, font.Normal)
							})
						})
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return fixedWidth(gtx, unit.Dp(188), func(gtx layout.Context) layout.Dimensions {
								return a.surfaceButton(
									gtx,
									&a.emptyStateImportButton,
									fluent.surface,
									fluent.accentSoft,
									fluent.border,
									unit.Dp(10),
									layout.Inset{Top: 9, Bottom: 9, Left: 12, Right: 12},
									func(gtx layout.Context) layout.Dimensions {
										return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(7))}.Layout(gtx,
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return fixedWidth(gtx, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
													return fixedHeight(gtx, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
														return uiIconSource.Layout(gtx, fluent.textMuted)
													})
												})
											}),
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return a.label(gtx, "选择本地图片", unit.Sp(12), fluent.textMuted, font.Medium)
											}),
										)
									},
								)
							})
						})
					}),
				)
			})
		})
	})
}

func paintCheckerboard(gtx layout.Context, area clip.Op, tile int, first color.NRGBA, second color.NRGBA) {
	if tile <= 0 {
		tile = 16
	}
	paint.FillShape(gtx.Ops, first, area)
	max := gtx.Constraints.Max
	for y := 0; y < max.Y; y += tile {
		for x := 0; x < max.X; x += tile {
			if ((x/tile)+(y/tile))%2 == 0 {
				continue
			}
			rect := image.Rect(x, y, min(x+tile, max.X), min(y+tile, max.Y))
			paint.FillShape(gtx.Ops, second, clip.Rect(rect).Op())
		}
	}
}

func (a *App) canvasStatusBar(gtx layout.Context, snap snapshot) layout.Dimensions {
	lastLog := ""
	if len(snap.Logs) > 0 {
		lastLog = snap.Logs[len(snap.Logs)-1]
	}

	return a.borderedSurface(gtx, fluent.panel2, unit.Dp(0), fluent.border, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 9, Bottom: 9, Left: 14, Right: 14}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			if snap.Running {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.badge(gtx, "运行中", fluent.accentSoft, fluent.accent)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, snap.Status, unit.Sp(11), fluent.text, font.Medium)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, time.Now().Format("15:04"), unit.Sp(11), fluent.textDim, font.Normal)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if strings.TrimSpace(lastLog) == "" {
							return layout.Dimensions{}
						}
						return a.singleLineLabel(gtx, lastLog, unit.Sp(11), fluent.textMuted, font.Normal)
					}),
				)
			}

			if snap.Result.HasItem {
				headline := "生成结果"
				if snap.Result.Item.Mode == "edit" {
					headline = "编辑结果"
				}
				meta := historyMetaText(snap.Result.Item)
				revised := strings.TrimSpace(snap.Result.RevisedPrompt)
				if revised == "" {
					revised = "暂无修订提示词"
				}
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.badge(gtx, headline, fluent.accentSoft, fluent.accent)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.staticPill(gtx, meta, false, true)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if snap.Result.Item.CreatedAt <= 0 {
							return layout.Dimensions{}
						}
						return a.label(gtx, formatHistoryClock(snap.Result.Item.CreatedAt), unit.Sp(11), fluent.textDim, font.Normal)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return a.singleLineLabel(gtx, revised, unit.Sp(11), fluent.textDim, font.Normal)
					}),
				)
			}

			return a.label(gtx, "准备就绪", unit.Sp(11), fluent.textMuted, font.Normal)
		})
	})
}

func (a *App) layoutSavePrompt(gtx layout.Context) layout.Dimensions {
	if a.savePromptNeverAsk.Update(gtx) {
		a.setSavePromptSuppressed(a.savePromptNeverAsk.Value)
	}
	for a.savePromptSkipButton.Clicked(gtx) {
		a.closeSavePrompt()
	}
	for a.savePromptSaveButton.Clicked(gtx) {
		a.savePromptCopy()
	}

	return a.layoutStandardModal(
		gtx,
		unit.Dp(520),
		0,
		"图片已生成，是否另存到指定位置？",
		"",
		nil,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, "默认目录已保存一份。需要放到项目、相册或其他目录时，可以现在填写目标路径再保存副本。", unit.Sp(13), fluent.textMuted, font.Normal)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.technicalField(gtx, "保存到", &a.savePromptPathInput, "输入完整文件路径或目录", unit.Dp(48))
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					style := material.CheckBox(a.th, &a.savePromptNeverAsk, "以后不再提示")
					style.Color = fluent.text
					style.IconColor = fluent.accent
					return style.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.compactButton(gtx, &a.savePromptSkipButton, "稍后", false)
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.primaryButton(gtx, &a.savePromptSaveButton, "保存副本", fluent.accent, fluent.white)
						}),
					)
				}),
			)
		},
	)
}
