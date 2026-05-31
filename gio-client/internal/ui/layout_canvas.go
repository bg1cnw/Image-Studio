package ui

import (
	"image"
	"image/color"
	"path/filepath"
	"strconv"
	"strings"

	"image-studio/gio-client/internal/kernel"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func (a *App) layoutCanvas(gtx layout.Context) layout.Dimensions {
	snap := a.readSnapshot()
	sourcePaths := kernel.ParseSourcePaths(a.sourcePathsInput.Text())
	showSourceStrip := a.mode == "edit" || len(sourcePaths) > 0

	children := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedHeight(gtx, unit.Dp(48), func(gtx layout.Context) layout.Dimensions {
				return a.canvasToolbar(gtx, snap)
			})
		}),
	}
	if showSourceStrip {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedHeight(gtx, unit.Dp(64), func(gtx layout.Context) layout.Dimensions {
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
	sizeLabel := sizeChoiceLabel(a.size)
	qualityLabel := qualityChoiceLabel(a.quality)
	if snap.Result.HasItem {
		if strings.TrimSpace(snap.Result.Item.Size) != "" {
			sizeLabel = snap.Result.Item.Size
		}
		if strings.TrimSpace(snap.Result.Item.Quality) != "" {
			qualityLabel = snap.Result.Item.Quality
		}
	}
	trailing := strings.TrimSpace(snap.Result.SourceEvent)
	if trailing == "" {
		trailing = snap.Status
	}
	showReturnLatest := len(snap.History) > 0 && snap.SelectedHistoryID != "" && snap.SelectedHistoryID != snap.History[0].ID

	return a.borderedSurface(gtx, fluent.panel2, unit.Dp(0), fluent.border, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		return layout.Inset{Top: 8, Bottom: 8, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.toolPill(gtx, "画布", true)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.toolPill(gtx, a.modeLabel(), false)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.toolPill(gtx, sizeLabel, false)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.toolPill(gtx, qualityLabel, false)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.toolPill(gtx, strings.ToUpper(a.format), false)
						}),
						layout.Flexed(1, layout.Spacer{}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.badge(gtx, trailing, fluent.surface, fluent.textMuted)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.toolbarGroupCard(gtx, "变换", func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.compactButton(gtx, &a.rotateLeftButton, "左转", false)
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.compactButton(gtx, &a.rotateRightButton, "右转", false)
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.compactButton(gtx, &a.flipHorizontalButton, "水平翻转", false)
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.compactButton(gtx, &a.flipVerticalButton, "竖直翻转", false)
									}),
								)
							})
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.toolbarGroupCard(gtx, "结果", func(gtx layout.Context) layout.Dimensions {
								children := []layout.FlexChild{
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.compactButton(gtx, &a.clearCurrentButton, "清空", false)
									}),
								}
								if showReturnLatest {
									children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.compactButton(gtx, &a.latestResultButton, "返回当前图", false)
									}))
								}
								if hasCurrentGroup && len(currentGroup.Items) > 1 {
									children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.compactButton(gtx, &a.currentGroupButton, "网格 "+strconv.Itoa(len(currentGroup.Items)), snap.ActivePromptGroup.Key == currentGroup.Key)
									}))
								}
								return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx, children...)
							})
						}),
						layout.Flexed(1, layout.Spacer{}.Layout),
						func() layout.FlexChild {
							return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.toolbarGroupCard(gtx, "文件", func(gtx layout.Context) layout.Dimensions {
									children := []layout.FlexChild{}
									if strings.TrimSpace(snap.Result.SavedPath) != "" {
										children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return fixedWidth(gtx, unit.Dp(88), func(gtx layout.Context) layout.Dimensions {
												return a.button(gtx, &a.saveAsButton, "另存为", fluent.accent, fluent.white)
											})
										}))
									}
									if snap.Result.HasItem {
										children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return a.compactButton(gtx, &a.resultDetailButton, "详情", false)
										}))
									}
									label := "全屏"
									if snap.Fullscreen {
										label = "退出全屏"
									}
									children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.compactButton(gtx, &a.fullscreenButton, label, snap.Fullscreen)
									}))
									return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx, children...)
								})
							})
						}(),
					)
				}),
			)
		})
	})
}

func (a *App) toolbarGroupCard(gtx layout.Context, title string, body layout.Widget) layout.Dimensions {
	return a.borderedSurface(gtx, fluent.surface, unit.Dp(6), fluent.border, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, title, unit.Sp(10), fluent.textDim, font.Medium)
				}),
				layout.Rigid(body),
			)
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
	for a.useCurrentAsSourceButton.Clicked(gtx) {
		if current := strings.TrimSpace(a.readSnapshot().Result.SavedPath); current != "" {
			a.appendSourcePath(current)
		}
	}
	for _, path := range sourcePaths {
		path := path
		btn := a.sourceButton("remove:" + path)
		for btn.Clicked(gtx) {
			a.removeSourcePath(path)
		}
	}

	label := "参考图 0 张"
	if len(sourcePaths) > 0 {
		label = "参考图 " + strconv.Itoa(len(sourcePaths)) + " 张"
	}
	currentSaved := strings.TrimSpace(a.readSnapshot().Result.SavedPath)

	return a.borderedSurface(gtx, fluent.panel2, unit.Dp(0), fluent.border, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		return layout.Inset{Top: 8, Bottom: 8, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			children := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, label, unit.Sp(11), fluent.textMuted, font.Medium)
				}),
			}
			if len(sourcePaths) == 0 {
				children = append(children,
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.compactButton(gtx, &a.addSourceStripButton, "添加图片", false)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if currentSaved == "" {
							return a.label(gtx, "先添加本地图片，或把当前结果设为源图。", unit.Sp(11), fluent.textDim, font.Normal)
						}
						return a.compactButton(gtx, &a.useCurrentAsSourceButton, "用当前图作源图", false)
					}),
				)
			} else {
				for _, path := range sourcePaths {
					path := path
					children = append(children,
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.layoutSourceStripTile(gtx, path)
						}),
					)
				}
				children = append(children,
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.compactButton(gtx, &a.addSourceStripButton, "添加图片", false)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.compactButton(gtx, &a.clearSourcesButton, "清空", false)
					}),
				)
			}
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
		})
	})
}

func (a *App) layoutSourceStripTile(gtx layout.Context, path string) layout.Dimensions {
	btn := a.sourceButton("remove:" + path)
	img, _ := a.imageForPath(path)
	name := filepath.Base(strings.TrimSpace(path))
	return a.surfaceButton(
		gtx,
		btn,
		fluent.surface,
		fluent.surface2,
		fluent.border,
		unit.Dp(6),
		layout.Inset{Top: 6, Bottom: 6, Left: 6, Right: 6},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.imageThumb(gtx, img, unit.Dp(48), unit.Dp(48), unit.Dp(4))
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedWidth(gtx, unit.Dp(132), func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(3))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, name, unit.Sp(10), fluent.text, font.Medium)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "点击移除", unit.Sp(9), fluent.textDim, font.Normal)
							}),
						)
					})
				}),
			)
		},
	)
}

func (a *App) resultSurface(gtx layout.Context, snap snapshot) layout.Dimensions {
	gtx.Constraints.Min = gtx.Constraints.Max
	return a.surface(gtx, fluent.canvasBg, unit.Dp(0), func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		paintCheckerboard(gtx, clip.Rect{Max: gtx.Constraints.Max}.Op(), gtx.Dp(unit.Dp(22)), fluent.canvasBg, fluent.canvasTile)
		return layout.UniformInset(unit.Dp(18)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.borderedSurface(gtx, fluent.surface, unit.Dp(8), fluent.border, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min = gtx.Constraints.Max
				if snap.Result.Image == nil {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "等待生成结果", unit.Sp(18), fluent.text, font.Medium)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.label(gtx, "生成完成后，结果会常驻在这里，右侧历史也会同步记录本次输出。", unit.Sp(13), fluent.textMuted, font.Normal)
							}),
						)
					})
				}
				if snap.Result.Rev != a.imageOpRev {
					a.imageOp = paint.NewImageOp(snap.Result.Image)
					a.imageOpRev = snap.Result.Rev
				}
				return layout.UniformInset(unit.Dp(14)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
						return a.label(gtx, snap.Status, unit.Sp(11), fluent.accent, font.Medium)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if strings.TrimSpace(lastLog) == "" {
							return layout.Dimensions{}
						}
						return a.label(gtx, lastLog, unit.Sp(11), fluent.textMuted, font.Normal)
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
						return a.label(gtx, headline, unit.Sp(11), fluent.accent, font.Medium)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, meta, unit.Sp(11), fluent.textMuted, font.Normal)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, revised, unit.Sp(11), fluent.textDim, font.Normal)
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

	paint.FillShape(gtx.Ops, rgba(0x000000, 0x52), clip.Rect{Max: gtx.Constraints.Max}.Op())
	gtx.Constraints.Min = gtx.Constraints.Max
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = image.Point{}
		return fixedWidth(gtx, unit.Dp(520), func(gtx layout.Context) layout.Dimensions {
			return a.borderedSurface(gtx, fluent.surface, unit.Dp(8), fluent.border, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(18)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "图片已生成, 是否另存到指定位置?", unit.Sp(18), fluent.text, font.SemiBold)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "默认目录已保存一份。需要放到项目、相册或其他目录时, 可以现在填写目标路径再保存副本。", unit.Sp(13), fluent.textMuted, font.Normal)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.field(gtx, "保存到", &a.savePromptPathInput, "输入完整文件路径或目录", unit.Dp(48))
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
									return a.button(gtx, &a.savePromptSkipButton, "稍后", fluent.surface2, fluent.text)
								}),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return a.button(gtx, &a.savePromptSaveButton, "保存副本", fluent.accent, fluent.white)
								}),
							)
						}),
					)
				})
			})
		})
	})
}
