package entities

import (
	"boughtnine/life"
	"image/color"
)

func NewGround(world *life.World) *life.Shape {
	ground := life.NewShape(&life.ShapeProps{
		Type:       life.ShapeRectangle,
		Background: color.RGBA{R: 255, G: 23, B: 0},
		Physics:    false,
		X:          0,
		Y:          500 - 30,
		Width:      500,
		Height:     30,
		IsBody:     false,
		Rebound:    0,
		Friction:   0,
	})

	world.Register(ground)

	return ground
}
