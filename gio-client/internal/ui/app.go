package ui

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	gioCompat "image-studio/gio-client/internal/compat"
	"image-studio/gio-client/internal/kernel"
	sharedCompat "image-studio/shared/compat"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/yuanhua/image-gptcodex/pkg/client"
	_ "golang.org/x/image/webp"
)

type choice struct {
	Label string
	Value string
}

var (
	modeChoices = []choice{
		{"文生图", string(client.ModeGenerate)},
		{"图生图", string(client.ModeEdit)},
	}
	apiChoices = []choice{
		{"Responses", string(client.APIModeResponses)},
		{"Images", string(client.APIModeImages)},
	}
	sizeChoices = []choice{
		{"自适应 auto", "auto"},
		{"方形 1024×1024", "1024x1024"},
		{"横版 1536×1024", "1536x1024"},
		{"竖版 1024×1536", "1024x1536"},
		{"2K 方形 2048×2048", "2048x2048"},
		{"2K 横版 2048×1360", "2048x1360"},
		{"2K 竖版 1360×2048", "1360x2048"},
		{"2K 横版 2048×1152", "2048x1152"},
		{"2K 竖版 1152×2048", "1152x2048"},
		{"4K 方形 2880×2880", "2880x2880"},
		{"4K 横版 3456×2304", "3456x2304"},
		{"4K 竖版 2304×3456", "2304x3456"},
		{"4K 横版 3840×2160", "3840x2160"},
		{"4K 竖版 2160×3840", "2160x3840"},
	}
	qualityChoices = []choice{
		{"自适应 auto", "auto"},
		{"高质量 high", "high"},
		{"中等 medium", "medium"},
		{"快速草稿 low", "low"},
	}
	formatChoices = []choice{
		{"PNG", "png"},
		{"JPEG", "jpeg"},
		{"WebP", "webp"},
	}
	policyChoices = []choice{
		{"OpenAI 标准", string(client.RequestPolicyOpenAI)},
		{"兼容中转扩展", string(client.RequestPolicyCompat)},
	}
	proxyChoices = []choice{
		{"系统配置", client.ProxyModeSystem},
		{"不使用", client.ProxyModeNone},
		{"自定义", client.ProxyModeCustom},
	}
)

type resultState struct {
	Image         image.Image
	SavedPath     string
	RawPath       string
	RevisedPrompt string
	SourceEvent   string
	Rev           int
}

type snapshot struct {
	Running           bool
	Status            string
	Logs              []string
	History           []sharedCompat.HistoryItem
	Result            resultState
	SavePromptVisible bool
}

type App struct {
	th     *material.Theme
	runner kernel.Runner

	controlsList widget.List
	logList      widget.List
	historyList  widget.List

	apiKeyInput         widget.Editor
	baseURLInput        widget.Editor
	textModelInput      widget.Editor
	imageModelInput     widget.Editor
	promptInput         widget.Editor
	sourcePathsInput    widget.Editor
	outputDirInput      widget.Editor
	seedInput           widget.Editor
	negativePromptInput widget.Editor
	partialImagesInput  widget.Editor
	proxyURLInput       widget.Editor
	savePromptPathInput widget.Editor

	mode    string
	api     string
	size    string
	quality string
	format  string
	policy  string
	proxy   string

	modeButtons          []widget.Clickable
	apiButtons           []widget.Clickable
	sizeButtons          []widget.Clickable
	qualityButtons       []widget.Clickable
	formatButtons        []widget.Clickable
	policyButtons        []widget.Clickable
	proxyButtons         []widget.Clickable
	runButton            widget.Clickable
	cancelButton         widget.Clickable
	clearLogButton       widget.Clickable
	savePromptSaveButton widget.Clickable
	savePromptSkipButton widget.Clickable
	savePromptNeverAsk   widget.Bool

	mu         sync.Mutex
	running    bool
	cancel     context.CancelFunc
	status     string
	logs       []string
	history    []sharedCompat.HistoryItem
	result     resultState
	imageOp    paint.ImageOp
	imageOpRev int

	savePromptVisible    bool
	savePromptSuppressed bool
	savePromptSourcePath string

	invalidate func()
}

func New() *App {
	cfg := kernel.DefaultConfig()
	compatState, compatPath, compatErr := gioCompat.LoadState()
	if compatErr == nil {
		cfg = gioCompat.ConfigFromState(cfg, compatState)
	}
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	th.Palette = material.Palette{
		Bg:         rgb(0x0f1117),
		Fg:         rgb(0xf4f7fb),
		ContrastBg: rgb(0x3b82f6),
		ContrastFg: rgb(0xffffff),
	}
	th.TextSize = unit.Sp(14)
	a := &App{
		th:                   th,
		runner:               kernel.Runner{},
		mode:                 string(cfg.Mode),
		api:                  string(cfg.APIMode),
		size:                 cfg.Size,
		quality:              cfg.Quality,
		format:               cfg.OutputFormat,
		policy:               string(cfg.RequestPolicy),
		proxy:                cfg.ProxyMode,
		modeButtons:          make([]widget.Clickable, len(modeChoices)),
		apiButtons:           make([]widget.Clickable, len(apiChoices)),
		sizeButtons:          make([]widget.Clickable, len(sizeChoices)),
		qualityButtons:       make([]widget.Clickable, len(qualityChoices)),
		formatButtons:        make([]widget.Clickable, len(formatChoices)),
		policyButtons:        make([]widget.Clickable, len(policyChoices)),
		proxyButtons:         make([]widget.Clickable, len(proxyChoices)),
		status:               "Gio 原生客户端就绪",
		logs:                 []string{"独立 Gio 高性能测试客户端已启动。"},
		history:              append([]sharedCompat.HistoryItem(nil), compatState.History...),
		savePromptSuppressed: gioCompat.SavePromptSuppressed(compatState),
	}
	a.savePromptNeverAsk.Value = a.savePromptSuppressed
	if compatPath != "" {
		a.logs = append(a.logs, "兼容状态文件: "+compatPath)
	}
	if compatErr != nil {
		a.logs = append(a.logs, "兼容状态读取失败: "+compatErr.Error())
	}
	a.controlsList.List.Axis = layout.Vertical
	a.logList.List.Axis = layout.Vertical
	a.historyList.List.Axis = layout.Vertical
	a.configureEditors(cfg)
	return a
}

func (a *App) configureEditors(cfg kernel.Config) {
	singleLine := []*widget.Editor{
		&a.apiKeyInput,
		&a.baseURLInput,
		&a.textModelInput,
		&a.imageModelInput,
		&a.outputDirInput,
		&a.seedInput,
		&a.partialImagesInput,
		&a.proxyURLInput,
		&a.savePromptPathInput,
	}
	for _, editor := range singleLine {
		editor.SingleLine = true
	}
	a.apiKeyInput.Mask = '*'
	a.seedInput.Filter = "0123456789"
	a.partialImagesInput.Filter = "0123456789"
	a.apiKeyInput.SetText(cfg.APIKey)
	a.baseURLInput.SetText(cfg.BaseURL)
	a.textModelInput.SetText(cfg.TextModelID)
	a.imageModelInput.SetText(cfg.ImageModelID)
	a.outputDirInput.SetText(cfg.OutputDir)
	a.partialImagesInput.SetText(strconv.Itoa(cfg.PartialImages))
	a.proxyURLInput.SetText(cfg.ProxyURL)
	a.promptInput.SetText("")
}

func (a *App) Run(w *app.Window) error {
	a.invalidate = w.Invalidate
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			a.saveCurrentConfig()
			a.cancelRun()
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			a.layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func (a *App) layout(gtx layout.Context) layout.Dimensions {
	for a.runButton.Clicked(gtx) {
		a.startRun()
	}
	for a.cancelButton.Clicked(gtx) {
		a.cancelRun()
	}
	for a.clearLogButton.Clicked(gtx) {
		a.clearLogs()
	}

	paint.FillShape(gtx.Ops, rgb(0x0d1016), clip.Rect{Max: gtx.Constraints.Max}.Op())
	dims := layout.UniformInset(unit.Dp(14)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
			layout.Rigid(a.layoutHeader),
			layout.Flexed(1, a.layoutBody),
		)
	})
	if a.readSnapshot().SavePromptVisible {
		a.layoutSavePrompt(gtx)
	}
	return dims
}

func (a *App) layoutHeader(gtx layout.Context) layout.Dimensions {
	snap := a.readSnapshot()
	return a.surface(gtx, rgb(0x161b24), unit.Dp(8), func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 14, Bottom: 14, Left: 18, Right: 18}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(4))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "Image Studio", unit.Sp(24), rgb(0xf8fafc), font.SemiBold)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "Gio Performance Client · Windows / Linux 独立测试版", unit.Sp(13), rgb(0x9aa8bc), font.Normal)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					label := snap.Status
					if snap.Running {
						label = "运行中 · " + label
					}
					return a.badge(gtx, label, rgb(0x1f2937), rgb(0xdbeafe))
				}),
			)
		})
	})
}

func (a *App) layoutBody(gtx layout.Context) layout.Dimensions {
	width := gtx.Constraints.Max.X
	rightWidth := unit.Dp(340)
	leftWidth := unit.Dp(420)
	if width < gtx.Dp(unit.Dp(1180)) {
		rightWidth = unit.Dp(300)
		leftWidth = unit.Dp(390)
	}
	return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedWidth(gtx, leftWidth, a.layoutControls)
		}),
		layout.Flexed(1, a.layoutCanvas),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedWidth(gtx, rightWidth, a.layoutHistoryAndLogs)
		}),
	)
}

func (a *App) layoutControls(gtx layout.Context) layout.Dimensions {
	return a.surface(gtx, rgb(0x151a22), unit.Dp(8), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(14)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.controlsList.Layout(gtx, 1, func(gtx layout.Context, _ int) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.sectionTitle(gtx, "生成参数")
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.segmented(gtx, modeChoices, a.mode, a.modeButtons, func(value string) { a.mode = value })
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.field(gtx, "提示词", &a.promptInput, "输入生成或编辑要求", unit.Dp(132))
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.field(gtx, "参考图路径", &a.sourcePathsInput, "图生图时填写本地图片路径, 多张可换行", unit.Dp(76))
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.segmentedWithTitle(gtx, "API 形态", apiChoices, a.api, a.apiButtons, func(value string) { a.api = value })
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.segmentedGridWithTitle(gtx, "尺寸", sizeChoices, a.size, a.sizeButtons, 2, func(value string) { a.size = value })
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.segmentedWithTitle(gtx, "质量", qualityChoices, a.quality, a.qualityButtons, func(value string) { a.quality = value })
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.segmentedWithTitle(gtx, "格式", formatChoices, a.format, a.formatButtons, func(value string) { a.format = value })
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
					layout.Rigid(a.layoutActions),
				)
			})
		})
	})
}

func (a *App) layoutActions(gtx layout.Context) layout.Dimensions {
	snap := a.readSnapshot()
	return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			txt := "生成"
			bg := rgb(0x2563eb)
			if snap.Running {
				txt = "运行中"
				bg = rgb(0x334155)
			}
			return a.button(gtx, &a.runButton, txt, bg, rgb(0xffffff))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedWidth(gtx, unit.Dp(96), func(gtx layout.Context) layout.Dimensions {
				return a.button(gtx, &a.cancelButton, "取消", rgb(0x2a303b), rgb(0xe5edf8))
			})
		}),
	)
}

func (a *App) layoutCanvas(gtx layout.Context) layout.Dimensions {
	snap := a.readSnapshot()
	return a.surface(gtx, rgb(0x121720), unit.Dp(8), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(14)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							title := "画布"
							if snap.Result.SavedPath != "" {
								title = "画布 · " + snap.Result.SavedPath
							}
							return a.label(gtx, title, unit.Sp(14), rgb(0xcbd5e1), font.Medium)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							source := snap.Result.SourceEvent
							if source == "" {
								source = "idle"
							}
							return a.badge(gtx, source, rgb(0x1e293b), rgb(0x93c5fd))
						}),
					)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.resultSurface(gtx, snap)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					revised := strings.TrimSpace(snap.Result.RevisedPrompt)
					if revised == "" {
						revised = "暂无修订提示词"
					}
					return a.surface(gtx, rgb(0x171d27), unit.Dp(8), func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, revised, unit.Sp(13), rgb(0x9fb0c6), font.Normal)
						})
					})
				}),
			)
		})
	})
}

func (a *App) resultSurface(gtx layout.Context, snap snapshot) layout.Dimensions {
	gtx.Constraints.Min = gtx.Constraints.Max
	return a.surface(gtx, rgb(0x0a0d12), unit.Dp(8), func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		if snap.Result.Image == nil {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, "等待生成结果", unit.Sp(18), rgb(0xd7e1ef), font.Medium)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, "Gio 直接渲染画布, 不经过 WebView2/WebKit", unit.Sp(13), rgb(0x748195), font.Normal)
					}),
				)
			})
		}
		if snap.Result.Rev != a.imageOpRev {
			a.imageOp = paint.NewImageOp(snap.Result.Image)
			a.imageOpRev = snap.Result.Rev
		}
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			img := widget.Image{
				Src:      a.imageOp,
				Fit:      widget.Contain,
				Position: layout.Center,
			}
			return img.Layout(gtx)
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

	paint.FillShape(gtx.Ops, rgba(0x000000, 0x96), clip.Rect{Max: gtx.Constraints.Max}.Op())
	gtx.Constraints.Min = gtx.Constraints.Max
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = image.Point{}
		return fixedWidth(gtx, unit.Dp(520), func(gtx layout.Context) layout.Dimensions {
			return a.surface(gtx, rgb(0x151a22), unit.Dp(10), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(18)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "图片已生成,是否另存到指定位置?", unit.Sp(18), rgb(0xf8fafc), font.SemiBold)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, "默认目录已保存一份。需要放到项目、相册或其他目录时,可以现在填写目标路径再保存副本。", unit.Sp(13), rgb(0xaab8ca), font.Normal)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.field(gtx, "保存到", &a.savePromptPathInput, "输入完整文件路径或目录", unit.Dp(48))
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							style := material.CheckBox(a.th, &a.savePromptNeverAsk, "以后不再提示")
							style.Color = rgb(0xdbe7f5)
							style.IconColor = rgb(0x60a5fa)
							return style.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return a.button(gtx, &a.savePromptSkipButton, "稍后", rgb(0x202938), rgb(0xdbe7f5))
								}),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return a.button(gtx, &a.savePromptSaveButton, "保存副本", rgb(0x2563eb), rgb(0xffffff))
								}),
							)
						}),
					)
				})
			})
		})
	})
}

func (a *App) layoutHistoryAndLogs(gtx layout.Context) layout.Dimensions {
	return a.surface(gtx, rgb(0x151a22), unit.Dp(8), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(14)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
				layout.Flexed(0.58, a.layoutHistory),
				layout.Flexed(0.42, a.layoutLogs),
			)
		})
	})
}

func (a *App) layoutHistory(gtx layout.Context) layout.Dimensions {
	snap := a.readSnapshot()
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.sectionTitle(gtx, "历史记录")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.badge(gtx, strconv.Itoa(len(snap.History)), rgb(0x202938), rgb(0xdbeafe))
				}),
			)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(snap.History) == 0 {
				return a.surface(gtx, rgb(0x111720), unit.Dp(6), func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min = gtx.Constraints.Max
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, "暂无历史", unit.Sp(13), rgb(0x8fa0b6), font.Normal)
					})
				})
			}
			return a.historyList.Layout(gtx, len(snap.History), func(gtx layout.Context, i int) layout.Dimensions {
				return layout.Inset{Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.historyRow(gtx, snap.History[i])
				})
			})
		}),
	)
}

func (a *App) historyRow(gtx layout.Context, item sharedCompat.HistoryItem) layout.Dimensions {
	return a.surface(gtx, rgb(0x111720), unit.Dp(6), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			mode := "文生图"
			if item.Mode == string(client.ModeEdit) {
				mode = "图生图"
			}
			meta := strings.Join(compactNonEmpty([]string{mode, item.Size, item.Quality, item.OutputFormat}), " · ")
			saved := item.SavedPath
			if saved == "" {
				saved = "未登记保存路径"
			}
			return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(5))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, shortPrompt(item.Prompt), unit.Sp(13), rgb(0xe6edf7), font.Medium)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, meta, unit.Sp(11), rgb(0x93a4b8), font.Normal)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, saved, unit.Sp(10), rgb(0x6f7f92), font.Normal)
				}),
			)
		})
	})
}

func (a *App) layoutLogs(gtx layout.Context) layout.Dimensions {
	snap := a.readSnapshot()
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(10))}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.sectionTitle(gtx, "运行日志")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return fixedWidth(gtx, unit.Dp(76), func(gtx layout.Context) layout.Dimensions {
						return a.button(gtx, &a.clearLogButton, "清空", rgb(0x202938), rgb(0xdbe7f5))
					})
				}),
			)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.logList.Layout(gtx, len(snap.Logs), func(gtx layout.Context, i int) layout.Dimensions {
				idx := len(snap.Logs) - 1 - i
				line := snap.Logs[idx]
				return layout.Inset{Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.surface(gtx, rgb(0x111720), unit.Dp(6), func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, line, unit.Sp(12), rgb(0xb6c3d5), font.Normal)
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
			return a.label(gtx, raw, unit.Sp(12), rgb(0x76859a), font.Normal)
		}),
	)
}

func (a *App) field(gtx layout.Context, title string, editor *widget.Editor, hint string, height unit.Dp) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, title, unit.Sp(12), rgb(0x8fa0b6), font.Medium)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedHeight(gtx, height, func(gtx layout.Context) layout.Dimensions {
				return a.surface(gtx, rgb(0x0f141d), unit.Dp(6), func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: 9, Bottom: 9, Left: 10, Right: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						style := material.Editor(a.th, editor, hint)
						style.Color = rgb(0xeaf1fb)
						style.HintColor = rgb(0x5f6d80)
						style.SelectionColor = rgba(0x3b82f6, 0x70)
						style.TextSize = unit.Sp(13)
						return style.Layout(gtx)
					})
				})
			})
		}),
	)
}

func (a *App) segmentedWithTitle(gtx layout.Context, title string, options []choice, selected string, buttons []widget.Clickable, set func(string)) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, title, unit.Sp(12), rgb(0x8fa0b6), font.Medium)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.segmented(gtx, options, selected, buttons, set)
		}),
	)
}

func (a *App) segmentedGridWithTitle(gtx layout.Context, title string, options []choice, selected string, buttons []widget.Clickable, columns int, set func(string)) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, title, unit.Sp(12), rgb(0x8fa0b6), font.Medium)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.segmentedGrid(gtx, options, selected, buttons, columns, set)
		}),
	)
}

func (a *App) segmented(gtx layout.Context, options []choice, selected string, buttons []widget.Clickable, set func(string)) layout.Dimensions {
	children := make([]layout.FlexChild, 0, len(options))
	for i := range options {
		i := i
		for buttons[i].Clicked(gtx) {
			set(options[i].Value)
		}
		children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			bg := rgb(0x111720)
			fg := rgb(0x8fa0b6)
			if options[i].Value == selected {
				bg = rgb(0x2563eb)
				fg = rgb(0xffffff)
			}
			return a.button(gtx, &buttons[i], options[i].Label, bg, fg)
		}))
	}
	return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx, children...)
}

func (a *App) segmentedGrid(gtx layout.Context, options []choice, selected string, buttons []widget.Clickable, columns int, set func(string)) layout.Dimensions {
	if columns <= 0 {
		columns = 2
	}
	rows := (len(options) + columns - 1) / columns
	children := make([]layout.FlexChild, 0, rows)
	for row := 0; row < rows; row++ {
		row := row
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			cellChildren := make([]layout.FlexChild, 0, columns)
			for col := 0; col < columns; col++ {
				idx := row*columns + col
				if idx >= len(options) {
					cellChildren = append(cellChildren, layout.Flexed(1, layout.Spacer{}.Layout))
					continue
				}
				for buttons[idx].Clicked(gtx) {
					set(options[idx].Value)
				}
				cellChildren = append(cellChildren, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					bg := rgb(0x111720)
					fg := rgb(0x8fa0b6)
					if options[idx].Value == selected {
						bg = rgb(0x2563eb)
						fg = rgb(0xffffff)
					}
					return a.button(gtx, &buttons[idx], options[idx].Label, bg, fg)
				}))
			}
			return layout.Inset{Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(6))}.Layout(gtx, cellChildren...)
			})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (a *App) sectionTitle(gtx layout.Context, text string) layout.Dimensions {
	return a.label(gtx, text, unit.Sp(15), rgb(0xe6edf7), font.SemiBold)
}

func (a *App) button(gtx layout.Context, btn *widget.Clickable, text string, bg color.NRGBA, fg color.NRGBA) layout.Dimensions {
	style := material.Button(a.th, btn, text)
	style.Background = bg
	style.Color = fg
	style.CornerRadius = unit.Dp(6)
	style.TextSize = unit.Sp(13)
	style.Font.Weight = font.Medium
	style.Inset = layout.Inset{Top: 10, Bottom: 10, Left: 10, Right: 10}
	return style.Layout(gtx)
}

func (a *App) badge(gtx layout.Context, text string, bg color.NRGBA, fg color.NRGBA) layout.Dimensions {
	return a.surface(gtx, bg, unit.Dp(6), func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 7, Bottom: 7, Left: 10, Right: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, text, unit.Sp(12), fg, font.Medium)
		})
	})
}

func (a *App) label(gtx layout.Context, text string, size unit.Sp, color color.NRGBA, weight font.Weight) layout.Dimensions {
	style := material.Label(a.th, size, text)
	style.Color = color
	style.Font.Weight = weight
	style.WrapPolicy = textWrapWords
	return style.Layout(gtx)
}

func (a *App) surface(gtx layout.Context, bg color.NRGBA, radius unit.Dp, w layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := w(gtx)
	call := macro.Stop()
	shape := clip.UniformRRect(image.Rectangle{Max: dims.Size}, gtx.Dp(radius)).Op(gtx.Ops)
	paint.FillShape(gtx.Ops, bg, shape)
	call.Add(gtx.Ops)
	return dims
}

func fixedWidth(gtx layout.Context, width unit.Dp, w layout.Widget) layout.Dimensions {
	px := gtx.Dp(width)
	if px > gtx.Constraints.Max.X {
		px = gtx.Constraints.Max.X
	}
	gtx.Constraints.Min.X = px
	gtx.Constraints.Max.X = px
	return w(gtx)
}

func fixedHeight(gtx layout.Context, height unit.Dp, w layout.Widget) layout.Dimensions {
	px := gtx.Dp(height)
	if px > gtx.Constraints.Max.Y {
		px = gtx.Constraints.Max.Y
	}
	gtx.Constraints.Min.Y = px
	gtx.Constraints.Max.Y = px
	return w(gtx)
}

func (a *App) startRun() {
	if a.isRunning() {
		return
	}
	cfg := a.currentConfig()
	if err := gioCompat.SaveConfig(cfg); err != nil {
		a.appendLog("兼容配置保存失败: " + err.Error())
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.running = true
	a.cancel = cancel
	a.status = "正在提交请求"
	a.logs = appendBounded(a.logs, "开始任务: "+shortPrompt(cfg.Prompt))
	a.mu.Unlock()
	a.invalidateNow()

	go func() {
		started := time.Now()
		res, err := a.runner.Run(ctx, cfg, kernel.Callbacks{
			Log: func(line string) {
				a.appendLog(line)
			},
			Progress: func(stage string, elapsed int, bytes int64) {
				a.setStatus(fmt.Sprintf("%s · %ds · %s", stage, elapsed, client.FormatBytes(bytes)))
			},
			Partial: func(partial client.PartialImage) {
				a.setStatus(fmt.Sprintf("收到流式预览 #%d", partial.PartialImageIndex))
			},
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				a.finishCancelled()
				return
			}
			a.finishWithError(err, res.RawPath)
			return
		}
		img, err := decodeImageB64(res.ImageB64)
		if err != nil {
			a.finishWithError(err, res.RawPath)
			return
		}
		elapsedSec := time.Since(started).Seconds()
		if err := gioCompat.SaveConfigAndHistory(cfg, res, elapsedSec); err != nil {
			a.appendLog("兼容历史保存失败: " + err.Error())
		}
		compatState, _, _ := gioCompat.LoadState()
		a.mu.Lock()
		a.running = false
		a.cancel = nil
		a.status = fmt.Sprintf("完成 · %.1fs", elapsedSec)
		a.result = resultState{
			Image:         img,
			SavedPath:     res.SavedPath,
			RawPath:       res.RawPath,
			RevisedPrompt: res.RevisedPrompt,
			SourceEvent:   res.SourceEvent,
			Rev:           a.result.Rev + 1,
		}
		a.history = append([]sharedCompat.HistoryItem(nil), compatState.History...)
		if !a.savePromptSuppressed && res.SavedPath != "" {
			a.savePromptVisible = true
			a.savePromptSourcePath = res.SavedPath
			a.savePromptPathInput.SetText(res.SavedPath)
		}
		a.logs = appendBounded(a.logs, "生成完成: "+res.SavedPath)
		a.mu.Unlock()
		a.invalidateNow()
	}()
}

func (a *App) currentConfig() kernel.Config {
	seed, _ := strconv.ParseInt(strings.TrimSpace(a.seedInput.Text()), 10, 64)
	partial, _ := strconv.Atoi(strings.TrimSpace(a.partialImagesInput.Text()))
	return kernel.Config{
		APIKey:         a.apiKeyInput.Text(),
		BaseURL:        a.baseURLInput.Text(),
		TextModelID:    a.textModelInput.Text(),
		ImageModelID:   a.imageModelInput.Text(),
		Prompt:         a.promptInput.Text(),
		Mode:           client.Mode(a.mode),
		APIMode:        client.APIMode(a.api),
		RequestPolicy:  client.RequestPolicy(a.policy),
		Size:           a.size,
		Quality:        a.quality,
		OutputFormat:   a.format,
		ProxyMode:      a.proxy,
		ProxyURL:       a.proxyURLInput.Text(),
		SourcePaths:    kernel.ParseSourcePaths(a.sourcePathsInput.Text()),
		OutputDir:      a.outputDirInput.Text(),
		Seed:           seed,
		NegativePrompt: a.negativePromptInput.Text(),
		PartialImages:  partial,
	}
}

func (a *App) saveCurrentConfig() {
	if err := gioCompat.SaveConfig(a.currentConfig()); err != nil {
		a.appendLog("兼容配置保存失败: " + err.Error())
	}
}

func (a *App) cancelRun() {
	a.mu.Lock()
	cancel := a.cancel
	if cancel != nil {
		a.cancel = nil
		a.running = false
		a.status = "已取消"
		a.logs = appendBounded(a.logs, "任务已取消")
	}
	a.mu.Unlock()
	if cancel != nil {
		cancel()
		a.invalidateNow()
	}
}

func (a *App) finishWithError(err error, rawPath string) {
	a.mu.Lock()
	a.running = false
	a.cancel = nil
	a.status = "失败"
	if rawPath != "" {
		a.result.RawPath = rawPath
	}
	a.logs = appendBounded(a.logs, "失败: "+err.Error())
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) finishCancelled() {
	a.mu.Lock()
	a.running = false
	a.cancel = nil
	a.status = "已取消"
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) appendLog(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	a.mu.Lock()
	a.logs = appendBounded(a.logs, line)
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) setStatus(status string) {
	a.mu.Lock()
	a.status = status
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) clearLogs() {
	a.mu.Lock()
	a.logs = nil
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) closeSavePrompt() {
	a.mu.Lock()
	a.savePromptVisible = false
	a.savePromptSourcePath = ""
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) setSavePromptSuppressed(value bool) {
	a.mu.Lock()
	a.savePromptSuppressed = value
	a.savePromptNeverAsk.Value = value
	a.mu.Unlock()
	if err := gioCompat.SetSavePromptSuppressed(value); err != nil {
		a.appendLog("保存提示设置失败: " + err.Error())
	}
	a.invalidateNow()
}

func (a *App) savePromptCopy() {
	a.mu.Lock()
	src := a.savePromptSourcePath
	dst := a.savePromptPathInput.Text()
	a.mu.Unlock()
	saved, err := copyImageFile(src, dst)
	if err != nil {
		a.appendLog("另存失败: " + err.Error())
		return
	}
	a.appendLog("已另存图片: " + saved)
	a.closeSavePrompt()
}

func (a *App) readSnapshot() snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	logs := append([]string(nil), a.logs...)
	history := append([]sharedCompat.HistoryItem(nil), a.history...)
	return snapshot{
		Running:           a.running,
		Status:            a.status,
		Logs:              logs,
		History:           history,
		Result:            a.result,
		SavePromptVisible: a.savePromptVisible,
	}
}

func (a *App) isRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.running
}

func (a *App) invalidateNow() {
	if a.invalidate != nil {
		a.invalidate()
	}
}

func decodeImageB64(imageB64 string) (image.Image, error) {
	data, err := base64.StdEncoding.DecodeString(imageB64)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image bytes: %w", err)
	}
	return img, nil
}

func appendBounded(logs []string, line string) []string {
	const maxLogs = 240
	logs = append(logs, time.Now().Format("15:04:05")+"  "+line)
	if len(logs) > maxLogs {
		copy(logs, logs[len(logs)-maxLogs:])
		logs = logs[:maxLogs]
	}
	return logs
}

func shortPrompt(prompt string) string {
	prompt = strings.Join(strings.Fields(prompt), " ")
	if len([]rune(prompt)) <= 40 {
		return prompt
	}
	runes := []rune(prompt)
	return string(runes[:40]) + "..."
}

func compactNonEmpty(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func copyImageFile(src, dst string) (string, error) {
	src = strings.TrimSpace(src)
	dst = strings.TrimSpace(strings.Trim(dst, `"'`))
	if src == "" {
		return "", errors.New("源图片路径为空")
	}
	if dst == "" {
		return "", errors.New("目标路径为空")
	}
	if strings.HasSuffix(dst, string(os.PathSeparator)) || strings.HasSuffix(dst, "/") || strings.HasSuffix(dst, `\`) {
		dst = filepath.Join(dst, filepath.Base(src))
	}
	if info, err := os.Stat(dst); err == nil && info.IsDir() {
		dst = filepath.Join(dst, filepath.Base(src))
	}
	if filepath.Ext(dst) == "" {
		dst += filepath.Ext(src)
	}
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return "", err
	}
	absDst, err := filepath.Abs(dst)
	if err != nil {
		return "", err
	}
	if filepath.Clean(absSrc) == filepath.Clean(absDst) {
		return absDst, nil
	}
	if err := os.MkdirAll(filepath.Dir(absDst), 0o700); err != nil {
		return "", err
	}
	in, err := os.Open(absSrc)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.OpenFile(absDst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	ok := false
	defer func() {
		_ = out.Close()
		if !ok {
			_ = os.Remove(absDst)
		}
	}()
	if _, err := io.Copy(out, in); err != nil {
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	ok = true
	return absDst, nil
}

func rgb(v uint32) color.NRGBA {
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}
}

func rgba(v uint32, alpha uint8) color.NRGBA {
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: alpha}
}

var textWrapWords = text.WrapWords
