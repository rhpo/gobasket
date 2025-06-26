package life

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

// TextProps contains properties for rendering text
type TextProps struct {
	Text    string
	X, Y    float64
	Color   color.Color
	Font    font.Face
	Size    float64
	FromEnd bool
	Type    string // "fill" or "stroke"
}

// DrawText renders text on the screen
func DrawText(screen *ebiten.Image, props *TextProps) {
	if props == nil {
		return
	}

	// Set defaults
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
		// Calculate text width and adjust position
		bounds, _ := font.BoundString(props.Font, props.Text)
		x = int(float64(screen.Bounds().Dx()) - float64(bounds.Max.X) - props.X)
	}

	text.Draw(screen, props.Text, props.Font, x, y, props.Color)
}

// LoadFont loads a font from a file (placeholder implementation)
func LoadFont(path string, size float64) (font.Face, error) {
	// This would load a font from a file
	// For now, return the basic font
	return basicfont.Face7x13, nil
}
