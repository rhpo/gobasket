package entities

import (
	"boughtnine/life"
)

// NewEnemy creates a new enemy instance
func NewEnemy(world *life.World, optionalProps ...*life.ShapeProps) *life.Shape {
	imgWidth := 384
	imgHeight := 832
	spriteSheet, _ := life.LoadImage("assets/walk.png") // replace with your file path
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

	// Merge optional properties with default properties
	if len(optionalProps) > 0 && optionalProps[0] != nil {
		// Manually merge fields from optionalProps[0] into defaultProps if they are non-zero values
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

	// Set default properties for the enemy
	enemy.Friction = 0
	enemy.Rebound = 1

	// Register the enemy with the world
	world.Register(enemy)

	return enemy
}
