package entities

import (
	"boughtnine/life"
	"image/color"
)

func NewWorld() *life.World {
	world := life.NewWorld(&life.WorldProps{
		Width:         800,
		Height:        600,
		G:             life.NewVector2(0, 0),
		Background:    color.RGBA{R: 65, G: 105, B: 225, A: 255},
		Paused:        false,
		HasLimits:     false,
		AirResistance: 1,
		Title:         "Basketball Game",
	})

	world.CreateBorders()
	return world
}
