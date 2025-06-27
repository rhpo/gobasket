package life

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

type TextProps struct {
	Text    string
	X, Y    float64
	Color   color.Color
	Font    font.Face
	Size    float64
	FromEnd bool
	Type    string
}

func DrawText(screen *ebiten.Image, props *TextProps) {
	if props == nil {
		return
	}

	if props.Font == nil {
		props.Font = basicfont.Face7x13
	}
	if props.Color == nil {
		props.Color = color.RGBA{255, 255, 255, 255}
	}
	if props.Text == "" {
		return
	}

	x := int(props.X)
	y := int(props.Y)

	if props.FromEnd {

		bounds, _ := font.BoundString(props.Font, props.Text)
		x = int(float64(screen.Bounds().Dx()) - float64(bounds.Max.X) - props.X)
	}

	text.Draw(screen, props.Text, props.Font, x, y, props.Color)
}

func LoadFont(path string, size float64) (font.Face, error) {

	return basicfont.Face7x13, nil
}
