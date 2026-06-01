package ui

import (
	_ "embed"

	"gioui.org/font"
	"gioui.org/font/opentype"
)

const (
	uiSansTypeface = font.Typeface("HarmonyOS Sans SC")
	uiMonoTypeface = font.Typeface("monospace")
)

//go:embed assets/HarmonyOS_SansSC_Regular.ttf
var harmonySansSC []byte

//go:embed assets/JetBrainsMono-Regular.ttf
var jetBrainsMono []byte

func bundledFontCollection() []font.FontFace {
	out := make([]font.FontFace, 0, 4)
	out = append(out, parseBundledFont(harmonySansSC, uiSansTypeface)...)
	out = append(out, parseBundledFont(jetBrainsMono, uiMonoTypeface)...)
	return out
}

func parseBundledFont(src []byte, typeface font.Typeface) []font.FontFace {
	faces, err := opentype.ParseCollection(src)
	if err != nil {
		return nil
	}
	for idx := range faces {
		faces[idx].Font.Typeface = typeface
	}
	return faces
}
