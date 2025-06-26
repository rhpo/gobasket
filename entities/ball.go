package entities

import (
	"boughtnine/life"
	"embed"
	"image/color"
)

type BallEntity struct {
	Shape *life.Shape
	World *life.World

	Animation *life.Animation
	States    map[string]*life.Animation
}

const (
	ballScale  = 1
	ballSpeed  = 20
	ballRadius = 25
)

// NewBall creates a new ball instance
func NewBallEntity(world *life.World, assets embed.FS) *BallEntity {

	ballImage, err := life.LoadImageFromFS(assets, "assets/ball.png") // replace with your file path

	if err != nil {
		panic(err)
	}

	defaultProps := &life.ShapeProps{
		Type:         life.ShapeCircle,
		Pattern:      life.PatternImage,
		Physics:      true,
		IsBody:       true,
		Background:   color.Opaque,
		Image:        ballImage,
		Radius:       ballRadius * ballScale,
		Friction:     1,
		Rebound:      0.2,
		RotationLock: false,
		Mass:         0.0000001,
		// Ghost:        true,
		ZIndex: 1000, // Ensure the ball is drawn above other shapes
	}

	ball := life.NewShape(defaultProps)
	world.Register(ball)

	ballEntity := BallEntity{
		Shape:  ball,
		World:  world,
		States: make(map[string]*life.Animation),
	}

	ballEntity.Initialize()

	return &ballEntity
}

func (ballEntity *BallEntity) Initialize() {

}

func (ballEntity *BallEntity) Update(ld life.LoopData) {

}
