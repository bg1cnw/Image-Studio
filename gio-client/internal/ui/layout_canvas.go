package ui

import (
	"fmt"
	"image"
	"image/color"
	"strconv"
	"strings"
	"time"

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
	for a.closeCompareButton.Clicked(gtx) {
		a.clearCompare()
	}
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
		if len(snap.BatchResults) > 1 {
			if snap.ResultGridOpen {
				a.closeResultGrid()
			} else {
				a.openResultGrid()
			}
		} else if hasCurrentGroup && len(currentGroup.Items) > 1 {
			if snap.ActivePromptGroup.Key == currentGroup.Key {
				a.closePromptGroup()
			} else {
				a.openPromptGroup(currentGroup)
			}
		}
	}
	for a.closeResultGridButton.Clicked(gtx) {
		a.closeResultGrid()
	}
	for _, item := range snap.BatchResults {
		item := item
		btn := a.historyButton("batch-grid:" + item.ID)
		for btn.Clicked(gtx) {
			if err := a.loadHistoryPreview(item, true); err != nil && !isMissingPreview(err) {
				a.appendLog("载入批量结果失败: " + err.Error())
			} else {
				a.closeResultGrid()
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
	hasCanvasResult := snap.Result.HasItem || strings.TrimSpace(snap.Result.SavedPath) != ""
	compareActive := snap.Compare.HasItem && snap.Compare.Image != nil && !snap.ResultGridOpen

	return a.borderedSurface(gtx, fluent.panel2, unit.Dp(0), fluent.border, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 8, Bottom: 8, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !hasCanvasResult {
						return layout.Dimensions{}
					}
					return a.toolbarCluster(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(2))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.toolbarStaticIcon(gtx, uiIconPanTool, true, false)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.toolbarStaticIcon(gtx, uiIconBrush, false, true)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.toolbarStaticIcon(gtx, uiIconAnnotate, false, true)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.toolbarStaticIcon(gtx, uiIconUndo, false, true)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.toolbarStaticIcon(gtx, uiIconRedo, false, true)
							}),
						)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !hasCanvasResult {
						return layout.Dimensions{}
					}
					return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.staticPill(gtx, "浏览", false, true)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !hasCanvasResult {
						return layout.Dimensions{}
					}
					return a.toolbarCluster(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(2))}.Layout(gtx,
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
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					children := []layout.FlexChild{}
					if compareActive {
						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.pillIconTextButton(gtx, &a.closeCompareButton, uiIconCompare, "退出对比", true)
						}))
					}
					if len(snap.BatchResults) > 1 {
						label := fmt.Sprintf("网格 %d", len(snap.BatchResults))
						if snap.ResultGridOpen {
							label = "单图"
						}
						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.pillIconTextButton(gtx, &a.currentGroupButton, uiIconGrid, label, snap.ResultGridOpen)
						}))
					} else if hasCurrentGroup && len(currentGroup.Items) > 1 {
						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.pillIconTextButton(gtx, &a.currentGroupButton, uiIconGrid, "同提示词 "+strconv.Itoa(len(currentGroup.Items)), snap.ActivePromptGroup.Key == currentGroup.Key)
						}))
					}
					if len(children) == 0 {
						return layout.Dimensions{}
					}
					return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.toolbarCluster(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx, children...)
						})
					})
				}),
				layout.Flexed(1, layout.Spacer{}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !snap.Result.HasItem {
						return layout.Dimensions{}
					}
					return a.toolbarCluster(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.metaBadgeRow(gtx, compactNonEmpty([]string{
							chooseModeLabel(snap.Result.Item.Mode),
							sizeDisplayLabel(snap.Result.Item.Size),
							qualityDisplayLabel(snap.Result.Item.Quality),
						}), true)
					})
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					children := []layout.FlexChild{}
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						icon := uiIconFullscreen
						if snap.Fullscreen {
							icon = uiIconFullscreenExit
						}
						return a.toolbarIconButton(gtx, &a.fullscreenButton, icon, snap.Fullscreen)
					}))
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
							return fixedWidth(gtx, unit.Dp(104), func(gtx layout.Context) layout.Dimensions {
								return a.primaryIconTextButton(gtx, &a.saveAsButton, uiIconDownload, "另存为", fluent.accent, fluent.white)
							})
						}))
					}
					return a.toolbarCluster(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(2))}.Layout(gtx, children...)
					})
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

func (a *App) toolbarCluster(gtx layout.Context, body layout.Widget) layout.Dimensions {
	return body(gtx)
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
			tiles := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.sectionEyebrow(gtx, label)
				}),
			}
			for _, path := range sourcePaths {
				path := path
				tiles = append(tiles,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutSourceStripTile(gtx, path)
					}),
				)
			}
			tiles = append(tiles,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.layoutSourceStripAddTile(gtx)
				}),
			)
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx, tiles...)
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
	return a.borderedSurface(gtx, fluent.surface, unit.Dp(4), fluent.border, func(gtx layout.Context) layout.Dimensions {
		return a.surfaceButton(
			gtx,
			&a.addSourceStripButton,
			fluent.surface,
			fluent.toolHoverBg,
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
		if snap.ResultGridOpen && len(snap.BatchResults) > 1 {
			return a.layoutBatchResultGrid(gtx, snap)
		}
		if snap.Result.Image == nil {
			return layout.Center.Layout(gtx, a.layoutCanvasEmptyState)
		}
		if snap.Result.Rev != a.imageOpRev {
			a.imageOp = paint.NewImageOp(snap.Result.Image)
			a.imageOpRev = snap.Result.Rev
		}
		if snap.Compare.Image != nil {
			return a.layoutCompareSurface(gtx, snap)
		}
		return layout.UniformInset(unit.Dp(28)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return a.layoutCanvasImageContain(gtx, a.imageOp)
			})
		})
	})
}

func (a *App) layoutCanvasImageContain(gtx layout.Context, op paint.ImageOp) layout.Dimensions {
	img := widget.Image{
		Src:      op,
		Fit:      widget.Contain,
		Position: layout.Center,
	}
	return img.Layout(gtx)
}

func (a *App) layoutCompareSurface(gtx layout.Context, snap snapshot) layout.Dimensions {
	compareOp := paint.NewImageOp(snap.Compare.Image)
	split := snap.CompareSplit
	if split < 0 {
		split = 0
	}
	if split > 1 {
		split = 1
	}
	return layout.UniformInset(unit.Dp(28)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return a.borderedSurface(gtx, fluent.surfaceElevated, fluentCardRadius, fluent.border, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min = gtx.Constraints.Max
			return layout.Stack{}.Layout(gtx,
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					return a.layoutCompareViewport(gtx, a.imageOp, compareOp, split)
				}),
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					return layout.NW.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Left: unit.Dp(12), Top: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.layoutCompareBadge(gtx, "A · 当前图", accentAlpha(0xe0))
						})
					})
				}),
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					return layout.NE.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.layoutCompareBadge(gtx, "B · 对比图", rgba(0x6741d9, 0xe0))
						})
					})
				}),
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					return layout.S.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Bottom: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return fixedWidth(gtx, unit.Dp(340), func(gtx layout.Context) layout.Dimensions {
								return a.borderedSurface(gtx, rgba(0x111111, 0xd8), unit.Dp(999), rgba(0xffffff, 0x1a), func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Top: 8, Bottom: 8, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										slider := material.Slider(a.th, &a.compareSplitSlider)
										slider.Color = fluent.accent
										slider.Axis = layout.Horizontal
										return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return a.label(gtx, "拖动调整对比", unit.Sp(11), fluent.white, font.Medium)
											}),
											layout.Flexed(1, slider.Layout),
										)
									})
								})
							})
						})
					})
				}),
			)
		})
	})
}

func (a *App) layoutCompareViewport(gtx layout.Context, currentOp paint.ImageOp, compareOp paint.ImageOp, split float32) layout.Dimensions {
	max := gtx.Constraints.Max
	gtx.Constraints.Min = max
	splitPx := clampInt(int(float32(max.X)*split), 0, max.X)
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			stack := clip.Rect(image.Rect(0, 0, splitPx, max.Y)).Push(gtx.Ops)
			defer stack.Pop()
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return a.layoutCanvasImageContain(gtx, currentOp)
			})
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			stack := clip.Rect(image.Rect(splitPx, 0, max.X, max.Y)).Push(gtx.Ops)
			defer stack.Pop()
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return a.layoutCanvasImageContain(gtx, compareOp)
			})
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			if max.X <= 0 || max.Y <= 0 {
				return layout.Dimensions{Size: max}
			}
			lineLeft := clampInt(splitPx-1, 0, max.X)
			lineRight := clampInt(splitPx+1, 0, max.X)
			if lineRight > lineLeft {
				paint.FillShape(gtx.Ops, accentAlpha(0xf2), clip.Rect(image.Rect(lineLeft, 0, lineRight, max.Y)).Op())
			}
			centerX := clampInt(splitPx, 12, max.X-12)
			handleRect := image.Rect(centerX-14, max.Y/2-14, centerX+14, max.Y/2+14)
			paint.FillShape(gtx.Ops, accentAlpha(0xf0), clip.Ellipse(handleRect).Op(gtx.Ops))
			paint.FillShape(gtx.Ops, fluent.white, clip.Rect(image.Rect(centerX-6, max.Y/2-1, centerX+6, max.Y/2+1)).Op())
			return layout.Dimensions{Size: max}
		}),
	)
}

func (a *App) layoutCompareBadge(gtx layout.Context, text string, bg color.NRGBA) layout.Dimensions {
	return a.surface(gtx, bg, unit.Dp(4), func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 3, Bottom: 3, Left: 8, Right: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, text, unit.Sp(10), fluent.white, font.Medium)
		})
	})
}

func (a *App) layoutBatchResultGrid(gtx layout.Context, snap snapshot) layout.Dimensions {
	items := snap.BatchResults
	columns := 3
	if len(items) <= 2 {
		columns = 2
	} else if len(items) <= 4 {
		columns = 2
	}
	rows := (len(items) + columns - 1) / columns
	return layout.Inset{Top: unit.Dp(16), Bottom: unit.Dp(16), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return a.borderedSurface(gtx, fluent.surfaceElevated, fluentModalRadius, fluent.border, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				children := []layout.FlexChild{
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, fmt.Sprintf("本批结果 · %d 张", len(items)), unit.Sp(12), fluent.text, font.SemiBold)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.compactButton(gtx, &a.closeResultGridButton, "返回当前图", false)
							}),
						)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				}
				for row := 0; row < rows; row++ {
					row := row
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						cells := make([]layout.FlexChild, 0, columns)
						for col := 0; col < columns; col++ {
							idx := row*columns + col
							if idx >= len(items) {
								cells = append(cells, layout.Flexed(1, layout.Spacer{}.Layout))
								continue
							}
							item := items[idx]
							cells = append(cells, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Right: chooseBatchGridInset(col, columns), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return a.layoutBatchGridTile(gtx, item, idx, snap.SelectedHistoryID == item.ID)
								})
							}))
						}
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, cells...)
					}))
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
			})
		})
	})
}

func chooseBatchGridInset(col int, columns int) unit.Dp {
	if col == columns-1 {
		return 0
	}
	return unit.Dp(10)
}

func (a *App) layoutBatchGridTile(gtx layout.Context, item sharedCompat.HistoryItem, index int, active bool) layout.Dimensions {
	btn := a.historyButton("batch-grid:" + item.ID)
	img, _ := a.imageForHistoryItem(item)
	return fixedHeight(gtx, unit.Dp(196), func(gtx layout.Context) layout.Dimensions {
		return a.surfaceButton(
			gtx,
			btn,
			chooseColor(active, fluent.surface2, fluent.surface),
			fluent.surface2,
			chooseColor(active, accentAlpha(0x48), fluent.border),
			fluentCardRadius,
			layout.Inset{},
			func(gtx layout.Context) layout.Dimensions {
				return layout.Stack{}.Layout(gtx,
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						return a.borderedSurface(gtx, fluent.panel2, fluentCardRadius, rgba(0xffffff, 0x00), func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min = gtx.Constraints.Max
							if img == nil {
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, "预览", unit.Sp(10), fluent.textDim, font.Medium)
								})
							}
							view := widget.Image{
								Src:      paint.NewImageOp(img),
								Fit:      widget.Contain,
								Position: layout.Center,
							}
							return view.Layout(gtx)
						})
					}),
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						return layout.NW.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(8), Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.surface(gtx, rgba(0x111111, 0xba), unit.Dp(4), func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Top: 2, Bottom: 2, Left: 6, Right: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return a.label(gtx, fmt.Sprintf("#%d", index+1), unit.Sp(9), fluent.white, font.Medium)
									})
								})
							})
						})
					}),
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						if item.ElapsedSec <= 0 {
							return layout.Dimensions{}
						}
						return layout.NE.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Right: unit.Dp(8), Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.surface(gtx, rgba(0x111111, 0xba), unit.Dp(4), func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Top: 2, Bottom: 2, Left: 6, Right: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return a.label(gtx, fmt.Sprintf("%.0fs", item.ElapsedSec), unit.Sp(9), fluent.white, font.Medium)
									})
								})
							})
						})
					}),
				)
			},
		)
	})
}

func (a *App) layoutCanvasEmptyState(gtx layout.Context) layout.Dimensions {
	copy := "先在左侧写提示词，再开始生成第一张图。"
	if a.mode == string(client.ModeEdit) {
		copy = "图生图时可先添加参考图，或从历史结果里挑一张继续编辑。"
	}
	return fixedWidth(gtx, unit.Dp(360), func(gtx layout.Context) layout.Dimensions {
		return a.borderedSurface(gtx, fluent.surface, unit.Dp(8), fluent.border, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(22)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return fixedWidth(gtx, unit.Dp(60), func(gtx layout.Context) layout.Dimensions {
								return fixedHeight(gtx, unit.Dp(60), func(gtx layout.Context) layout.Dimensions {
									return a.borderedSurface(gtx, fluent.accentSoft, unit.Dp(8), accentAlpha(0x22), func(gtx layout.Context) layout.Dimensions {
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
								return a.label(gtx, copy, unit.Sp(12), fluent.textMuted, font.Normal)
							})
						})
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return fixedWidth(gtx, unit.Dp(188), func(gtx layout.Context) layout.Dimensions {
								return a.compactIconTextButton(gtx, &a.emptyStateImportButton, uiIconSource, "选择本地图片", false)
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
				return layout.Stack{}.Layout(gtx,
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return fixedWidth(gtx, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
									return fixedHeight(gtx, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
										return uiIconRefresh.Layout(gtx, fluent.accent)
									})
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, chooseStatusText(snap.Status), unit.Sp(11), fluent.text, font.Medium)
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return a.staticPill(gtx, time.Now().Format("15:04"), false, true)
								})
							}),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								if strings.TrimSpace(lastLog) == "" {
									return layout.Dimensions{}
								}
								return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return a.singleLineLabel(gtx, lastLog, unit.Sp(11), fluent.textMuted, font.Normal)
								})
							}),
						)
					}),
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						return layout.S.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return fixedHeight(gtx, unit.Dp(2), func(gtx layout.Context) layout.Dimensions {
								return a.surface(gtx, accentAlpha(0x58), 0, layout.Spacer{}.Layout)
							})
						})
					}),
				)
			}

			if snap.Result.HasItem {
				headline := "生成结果"
				if snap.Result.Item.Mode == "edit" {
					headline = "编辑结果"
				}
				revised := strings.TrimSpace(snap.Result.RevisedPrompt)
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return fixedWidth(gtx, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
							return fixedHeight(gtx, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
								return uiIconCheck.Layout(gtx, fluent.accent)
							})
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, headline, unit.Sp(11), fluent.accent, font.Medium)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.metaBadgeRow(gtx, historyMetaBadgeItems(snap.Result.Item), true)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if snap.Result.Item.CreatedAt <= 0 {
							return layout.Dimensions{}
						}
						return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.singleLineLabel(gtx, formatHistoryClock(snap.Result.Item.CreatedAt), unit.Sp(11), fluent.textDim, font.Normal)
						})
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if revised == "" {
							return layout.Dimensions{}
						}
						return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.singleLineLabel(gtx, revised, unit.Sp(11), fluent.textMuted, font.Normal)
						})
					}),
				)
			}

			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedWidth(gtx, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
						return fixedHeight(gtx, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
							return uiIconCheck.Layout(gtx, fluent.textDim)
						})
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, "准备就绪", unit.Sp(11), fluent.textMuted, font.Normal)
					})
				}),
			)
		})
	})
}

func chooseStatusText(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return "正在请求..."
	}
	return status
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

	snap := a.readSnapshot()
	item := snap.Result.Item
	img := snap.Result.Image
	return a.layoutStandardModal(
		gtx,
		unit.Dp(520),
		0,
		"是否另存这张图片?",
		"",
		nil,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.borderedSurface(gtx, fluent.surface, unit.Dp(10), fluent.border, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return a.imageThumb(gtx, img, unit.Dp(116), unit.Dp(116), unit.Dp(8))
								})
							})
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, "图片已生成并保存在默认输出目录。", unit.Sp(13), fluent.text, font.Medium)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.label(gtx, "需要放到项目、相册或其他目录时，可以现在填写目标位置另存一份。", unit.Sp(11), fluent.textMuted, font.Normal)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									path := strings.TrimSpace(item.SavedPath)
									if path == "" {
										path = strings.TrimSpace(a.savePromptSourcePath)
									}
									if path == "" {
										return layout.Dimensions{}
									}
									return a.borderedSurface(gtx, fluent.surface2, unit.Dp(8), fluent.border, func(gtx layout.Context) layout.Dimensions {
										return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return a.label(gtx, path, unit.Sp(10), fluent.textDim, font.Normal)
										})
									})
								}),
							)
						}),
					)
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
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return fixedWidth(gtx, unit.Dp(110), func(gtx layout.Context) layout.Dimensions {
								return a.compactButton(gtx, &a.savePromptSkipButton, "稍后", false)
							})
						}),
						layout.Flexed(1, layout.Spacer{}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return fixedWidth(gtx, unit.Dp(152), func(gtx layout.Context) layout.Dimensions {
								return a.primaryIconTextButton(gtx, &a.savePromptSaveButton, uiIconFolder, "保存到指定位置", fluent.accent, fluent.white)
							})
						}),
					)
				}),
			)
		},
	)
}
