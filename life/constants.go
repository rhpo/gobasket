package life

import (
	"math"
	"math/rand"
	"time"
)

type AxisX string
type AxisY string

const (
	EPSILON_DIRECTION_CHANGED = 0
	EPSILON_STABILISATION     = 1

	DirectionUp    AxisY = "up"
	DirectionDown  AxisY = "down"
	DirectionLeft  AxisX = "left"
	DirectionRight AxisX = "right"
)

type Axis struct {
	X *AxisX
	Y *AxisY
}

const (
	Deg = math.Pi / 180
	PTM = 8.0
)

func PixelsToMeters(p float64) float64 {
	return p / PTM
}

func MetersToPixels(m float64) float64 {
	return m * PTM
}

type ShapeType string

const (
	ShapeCircle    ShapeType = "circle"
	ShapeSquare    ShapeType = "square"
	ShapeRectangle ShapeType = "rectangle"
	ShapeRect      ShapeType = "rectangle"
	ShapeLine      ShapeType = "line"
	ShapeDot       ShapeType = "dot"
)

type PatternType string

const (
	PatternImage      PatternType = "image"
	PatternColor      PatternType = "color"
	PatternSolidColor PatternType = "color"
	PatternGradient   PatternType = "gradient"
)

type CursorType string

const (
	CursorDefault   CursorType = "default"
	CursorPointer   CursorType = "pointer"
	CursorCrosshair CursorType = "crosshair"
	CursorMove      CursorType = "move"
	CursorText      CursorType = "text"
)

func ID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 7)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func RandName() string {
	names := []string{"James", "Robert", "John", "Michael", "David", "William", "Richard", "Thomas", "Charles", "Islam", "Mohammed", "Ramy"}
	return names[rand.Intn(len(names))]
}

func Defined(v interface{}) bool {
	return v != nil
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
