package entities

import (
	"boughtnine/life"
	"embed"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type PlayerEntity struct {
	Shape *life.Shape
	World *life.World

	Animation *life.Animation
	States    map[string]*life.Animation
}

const (
	playerScale = 1.5
	playerSpeed = 20
)

// NewPlayer creates a new player instance
func NewPlayerEntity(world *life.World, assets embed.FS) *PlayerEntity {
	imgWidth := 384
	imgHeight := 832

	spriteSheet, _ := life.LoadImageFromFS(assets, "assets/walk.png") // replace with your file path
	sprites := life.ExtractSprites(spriteSheet, float32(imgWidth)/6, float32(imgHeight)/13, 0, 0, 0, 0)

	defaultProps := &life.ShapeProps{
		Type:         life.ShapeRectangle,
		Pattern:      life.PatternImage,
		Physics:      true,
		IsBody:       true,
		Image:        sprites[0],
		Width:        25 * playerScale,
		Height:       35 * playerScale,
		Friction:     1,
		Rebound:      0.2,
		RotationLock: false,
		Mass:         100,
	}

	player := life.NewShape(defaultProps)
	world.Register(player)

	playerEntity := PlayerEntity{
		Shape:  player,
		World:  world,
		States: make(map[string]*life.Animation),
	}

	animationWalk := life.NewAnimation(player, 100*time.Millisecond, true, sprites[13:16]...)
	animationIdle := life.NewAnimation(player, 100*time.Millisecond, true, sprites[13:16]...)

	playerEntity.States["walk"] = animationWalk
	playerEntity.States["idle"] = animationIdle

	playerEntity.Animation = playerEntity.States["walk"]

	playerEntity.Initialize()

	return &playerEntity
}

func (playerEntity *PlayerEntity) SetAnimation(name string) {
	if animation, exists := playerEntity.States[name]; exists {

		if playerEntity.Animation != nil && playerEntity.Animation.IsPlaying() {
			playerEntity.Animation.Stop()
		}

		playerEntity.Animation = animation

		if !animation.IsPlaying() {
			animation.Start()
		}
	} else {
		playerEntity.Animation = nil
	}
}

func (playerEntity *PlayerEntity) Initialize() {
	player := playerEntity.Shape

	player.On(life.EventDirectionChange, func(Data any) {
		direction := Data.(life.EventDirectionChangeData).Direction
		player.Flip.X = *direction.X == life.DirectionLeft
	})

}

func isCollidingWithTag(playerEntity *PlayerEntity, tag string) bool {
	objects := playerEntity.World.GetElementsByTagName(tag)
	for _, element := range objects {
		if playerEntity.Shape.IsCollidingWith(element) {
			return true
		}
	}
	return false
}

func (playerEntity *PlayerEntity) Update(ld life.LoopData) {

	player := playerEntity.Shape

	if playerEntity.World.IsKeyPressed(ebiten.KeyLeft) {
		player.SetXVelocity(-playerSpeed * 100 * ld.Delta)
	} else if playerEntity.World.IsKeyPressed(ebiten.KeyRight) {
		player.SetXVelocity(playerSpeed * 100 * ld.Delta)
	} else {
		playerEntity.SetAnimation("idle")
	}

	if playerEntity.World.IsKeyPressed(ebiten.KeyUp) && isCollidingWithTag(playerEntity, "ground") {
		player.Jump(playerSpeed * 80 * ld.Delta)
		playerEntity.World.PlaySound("jump")
	}
}
