package badges

import (
	"sort"
	"strings"

	"github.com/flosch/pongo2/v6"
	"github.com/golang/freetype/truetype"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/wrouesnel/badgeserv/assets"
)

// ColorMapping provides structure for returning and sorting color mappings.
type ColorMapping struct {
	Name  string
	Color string
}

// BadgeConfig provides configuration for a badge service.
type BadgeConfig struct {
	FontSize     float64           `help:"Font size of badges" default:"11"`
	XSpacing     int               `help:"X Spacing of Badge Elements" default:"8"`
	DefaultColor string            `help:"Default badge color" default:"4c1"`
	ColorList    map[string]string `help:"Plaintext badge colors" default:"brightgreen=4c1;green=97CA00;yellow=dfb317;yellowgreen=a4a61d;orange=fe7d37;red=e05d44;blue=007ec6;grey=555;gray=555;lightgrey=9f9f9f;lightgray=9f9f9f"`
}

// BadgeDesc is all the data to generate a badge.
type BadgeDesc struct {
	Title string
	Text  string
	Color string
}

// BadgeService implements generating badge SVGs.
type BadgeService interface {
	CreateBadge(desc BadgeDesc) (string, error)
	Colors() []ColorMapping
}

// badgeService implements the actual badge generator.
type badgeService struct {
	config        *BadgeConfig
	badgeTemplate *pongo2.Template
	fontCalc      *FontCalculator
}

// NewBadgeService initializes a new BadgeService interface.
func NewBadgeService(config *BadgeConfig) BadgeService {
	font := lo.Must(truetype.Parse(lo.Must(assets.ReadFile("fonts/DejaVuSans.ttf"))))
	fontCalc := NewFontCalculator(font)

	return &badgeService{
		config:        config,
		badgeTemplate: lo.Must(pongo2.FromBytes(lo.Must(assets.ReadFile("badges/badge.svg.p2")))),
		fontCalc:      fontCalc,
	}
}

// Colors returns the current configured color mappings.
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

// CreateBadge takes the given parameters and generates an SVG for the badge
//nolint:gomnd
func (bs *badgeService) CreateBadge(desc BadgeDesc) (string, error) {
	titleW, _ := bs.fontCalc.TextWidth(bs.config.FontSize, desc.Title)
	textW, _ := bs.fontCalc.TextWidth(bs.config.FontSize, desc.Text)

	width := titleW + textW + 4*bs.config.XSpacing

	if c, ok := bs.config.ColorList[desc.Color]; ok {
		desc.Color = c
	}

	result, err := bs.badgeTemplate.Execute(map[string]interface{}{
		"Width":       width,
		"TitleWidth":  titleW + 2*bs.config.XSpacing,
		"Title":       desc.Title,
		"Text":        desc.Text,
		"TitleAnchor": titleW/2 + bs.config.XSpacing,
		"TextAnchor":  titleW + textW/2 + 3*bs.config.XSpacing,
		"Color":       desc.Color,
	})

	if err != nil {
		return result, errors.Wrap(err, "CreateBadge: error templating")
	}
	return result, nil
}
