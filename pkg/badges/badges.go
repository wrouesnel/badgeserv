package badges

import (
	"github.com/flosch/pongo2/v6"
	"github.com/golang/freetype/truetype"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/wrouesnel/badgeserv/assets"
	"sort"
	"strings"
)

// ColorMapping provides structure for returning and sorting color mappings
type ColorMapping struct {
	Name  string
	Color string
}

// BadgeConfig provides configuration for a badge service
type BadgeConfig struct {
	FontSize     float64           `help:"Font size of badges" default:"11"`
	XSpacing     int               `help:"X Spacing of Badge Elements" default:"8"`
	DefaultColor string            `help:"Default badge color" default:"4c1"`
	ColorList    map[string]string `help:"Plaintext badge colors" default:"brightgreen=4c1;green=97CA00;yellow=dfb317;yellowgreen=a4a61d;orange=fe7d37;red=e05d44;blue=007ec6;grey=555;gray=555;lightgrey=9f9f9f;lightgray=9f9f9f"`
}

type BadgeService interface {
	CreateBadge(title string, text string, color string) (string, error)
	Colors() []ColorMapping
}

// BadgeService implements the actual badge generator
type badgeService struct {
	config        *BadgeConfig
	badgeTemplate *pongo2.Template
	fontCalc      *FontCalculator
}

func NewBadgeService(config *BadgeConfig) BadgeService {
	font := lo.Must(truetype.Parse(lo.Must(assets.ReadFile("fonts/DejaVuSans.ttf"))))
	fontCalc := NewFontCalculator(font)

	return &badgeService{
		config:        config,
		badgeTemplate: lo.Must(pongo2.FromBytes(lo.Must(assets.ReadFile("badges/badge.svg.p2")))),
		fontCalc:      fontCalc,
	}

}

// Colors returns the current configured color mappings
func (bs *badgeService) Colors() []ColorMapping {
	colors := lo.MapToSlice(bs.config.ColorList, func(name string, color string) ColorMapping {
		return ColorMapping{
			Name:  name,
			Color: color,
		}
	})

	sort.Slice(colors, func(i, j int) bool {
		return strings.Compare(colors[i].Name, colors[j].Name) < 0
	})

	return colors
}

func (bs *badgeService) CreateBadge(title string, text string, color string) (string, error) {

	titleW, _ := bs.fontCalc.TextWidth(bs.config.FontSize, title)
	textW, _ := bs.fontCalc.TextWidth(bs.config.FontSize, text)

	width := titleW + textW + 4*bs.config.XSpacing

	if c, ok := bs.config.ColorList[color]; ok {
		color = c
	}

	result, err := bs.badgeTemplate.Execute(map[string]interface{}{
		"Width":       width,
		"TitleWidth":  titleW + 2*bs.config.XSpacing,
		"Title":       title,
		"Text":        text,
		"TitleAnchor": titleW/2 + bs.config.XSpacing,
		"TextAnchor":  titleW + textW/2 + 3*bs.config.XSpacing,
		"Color":       color,
	})

	if err != nil {
		return result, errors.Wrap(err, "CreateBadge: error templating")
	}
	return result, nil
}
