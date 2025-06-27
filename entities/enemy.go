package entities

import (
	"boughtnine/life"
)

func NewEnemy(world *life.World, optionalProps ...*life.ShapeProps) *life.Shape {
	imgWidth := 384
	imgHeight := 832
	spriteSheet, _ := life.LoadImage("assets/walk.png")
	sprites := life.ExtractSprites(spriteSheet, float32(imgWidth)/6, float32(imgHeight)/13, 0, 0, 0, 0)

	defaultProps := &life.ShapeProps{
		Type:     life.ShapeRectangle,
		Pattern:  life.PatternImage,
		Physics:  true,
		IsBody:   true,
		Image:    sprites[0],
		Width:    25,
		Height:   34,
		Friction: 1,
		Rebound:  0,
	}

	if len(optionalProps) > 0 && optionalProps[0] != nil {

		if optionalProps[0].Type != "" {
			defaultProps.Type = optionalProps[0].Type
		}
		if optionalProps[0].Pattern != "" {
			defaultProps.Pattern = optionalProps[0].Pattern
		}
		if optionalProps[0].Image != nil {
			defaultProps.Image = optionalProps[0].Image
		}
		defaultProps.Physics = optionalProps[0].Physics
		defaultProps.IsBody = optionalProps[0].IsBody
		if optionalProps[0].Width != 0 {
			defaultProps.Width = optionalProps[0].Width
		}
		if optionalProps[0].Height != 0 {
			defaultProps.Height = optionalProps[0].Height
		}
		if optionalProps[0].Friction != 0 {
			defaultProps.Friction = optionalProps[0].Friction
		}
		if optionalProps[0].Rebound != 0 {
			defaultProps.Rebound = optionalProps[0].Rebound
		}
	}

	enemy := life.NewShape(defaultProps)

	enemy.Friction = 0
	enemy.Rebound = 1

	world.Register(enemy)

	return enemy
}
