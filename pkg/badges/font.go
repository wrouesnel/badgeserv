package badges

import (
	"github.com/golang/freetype/truetype"
	"go.uber.org/zap"
	"golang.org/x/image/math/fixed"
)

type FontCalculator struct {
	font *truetype.Font
}

func NewFontCalculator(font *truetype.Font) *FontCalculator {
	if font == nil {
		zap.L().Warn("Initialized FontCalculator with nil font. Returning nil!")
		return nil
	}

	return &FontCalculator{font: font}
}

func (fc *FontCalculator) TextWidth(fontSize float64, text string) (int, error) {
	scale := fontSize / float64(fc.font.FUnitsPerEm())

	width := 0
	prev, hasPrev := truetype.Index(0), false
	for _, r := range text {
		fUnitsPerEm := fixed.Int26_6(fc.font.FUnitsPerEm())
		index := fc.font.Index(r)
		if hasPrev {
			width += int(fc.font.Kern(fUnitsPerEm, prev, index))
		}
		width += int(fc.font.HMetric(fUnitsPerEm, index).AdvanceWidth)
		prev, hasPrev = index, true
	}

	return int(float64(width) * scale), nil
}
