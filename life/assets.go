package life

import (
	"embed"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

// LoadImage loads an image from a file
func LoadImage(path string) (*ebiten.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return ebiten.NewImageFromImage(img), nil
}

func LoadImageFromFS(fs embed.FS, path string) (*ebiten.Image, error) {
	file, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return ebiten.NewImageFromImage(img), nil
}

func ExtractSprites(sheet *ebiten.Image, spriteWidth, spriteHeight, marginX, marginY, spacingX, spacingY float32) []*ebiten.Image {
	var frames []*ebiten.Image
	sheetBounds := sheet.Bounds()
	sheetWidth := sheetBounds.Dx()
	sheetHeight := sheetBounds.Dy()

	cols := int((float32(sheetWidth) - marginX*2 + spacingX) / (spriteWidth + spacingX))
	rows := int((float32(sheetHeight) - marginY*2 + spacingY) / (spriteHeight + spacingY))

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			startX := marginX + float32(x)*(spriteWidth+spacingX)
			startY := marginY + float32(y)*(spriteHeight+spacingY)
			r := image.Rect(
				int(startX), int(startY),
				int(startX+spriteWidth), int(startY+spriteHeight),
			)
			sub := sheet.SubImage(r).(*ebiten.Image)
			frames = append(frames, sub)
		}
	}
	return frames
}

// PreSuffixedRange creates a map of strings with prefix and suffix
func PreSuffixedRange(prefix, suffix string, start, end int) map[int]string {
	result := make(map[int]string)
	for i := start; i <= end; i++ {
		result[i] = prefix + string(rune(i)) + suffix
	}
	return result
}

// SpriteSheet represents a sprite sheet
type SpriteSheet struct {
	Image       *ebiten.Image
	FrameWidth  int
	FrameHeight int
	Frames      []*ebiten.Image
}

// NewSpriteSheet creates a new sprite sheet
func NewSpriteSheet(imagePath string, frameWidth, frameHeight int) (*SpriteSheet, error) {
	img, err := LoadImage(imagePath)
	if err != nil {
		return nil, err
	}

	sheet := &SpriteSheet{
		Image:       img,
		FrameWidth:  frameWidth,
		FrameHeight: frameHeight,
	}

	// Extract frames
	bounds := img.Bounds()
	cols := bounds.Dx() / frameWidth
	rows := bounds.Dy() / frameHeight

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			x := col * frameWidth
			y := row * frameHeight

			frame := img.SubImage(image.Rect(x, y, x+frameWidth, y+frameHeight)).(*ebiten.Image)
			sheet.Frames = append(sheet.Frames, frame)
		}
	}

	return sheet, nil
}

// GetFrame returns a specific frame from the sprite sheet
func (s *SpriteSheet) GetFrame(index int) *ebiten.Image {
	if index < 0 || index >= len(s.Frames) {
		return nil
	}
	return s.Frames[index]
}

// GetFrames returns a slice of frames
func (s *SpriteSheet) GetFrames(start, end int) []*ebiten.Image {
	if start < 0 || end >= len(s.Frames) || start > end {
		return nil
	}
	return s.Frames[start : end+1]
}
