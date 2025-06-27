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
	ballRadius = 25 / 2
)

func NewBallEntity(world *life.World, assets embed.FS) *BallEntity {

	ballImage, err := life.LoadImageFromFS(assets, "assets/ball.png")

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
		Friction:     0,
		Rebound:      0.8,
		RotationLock: false,
		Mass:         8.0,
		ZIndex:       1000,
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
	ball := ballEntity.Shape

	ball.OnCollisionFunc = func(who *life.Shape) {
		volume := ballEntity.Shape.LastCollisionImpulse / 100

		if volume < 0.08 {
			volume = 0
		}

		ballEntity.World.PlaySoundWithVolume("ball_hit", volume)
	}
}

func (ballEntity *BallEntity) Update(ld life.LoopData) {

	ballEntity.updateRollingPhysics()

	ballEntity.applyAirResistance()

	if ballEntity.Shape.Velocity.X < life.EPSILON_STABILISATION {
		ballEntity.Shape.Velocity.X = 0
	}

	if ballEntity.Shape.Velocity.Y < life.EPSILON_STABILISATION {
		ballEntity.Shape.Velocity.Y = 0
	}
}

func (ballEntity *BallEntity) updateRollingPhysics() {
	ball := ballEntity.Shape

	if ball.Body == nil {
		return
	}

	velocity := ball.Body.GetLinearVelocity()

	if abs(float64(velocity.X)) > 0.1 && ballEntity.isTouchingGround() {
		targetAngularVelocity := float64(velocity.X) / life.PixelsToMeters(ball.Radius)

		currentAngularVelocity := ball.Body.GetAngularVelocity()
		smoothingFactor := 0.3
		newAngularVelocity := currentAngularVelocity + (targetAngularVelocity-currentAngularVelocity)*smoothingFactor

		ball.Body.SetAngularVelocity(newAngularVelocity)
	}
}

func (ballEntity *BallEntity) applyAirResistance() {
	ball := ballEntity.Shape

	if ball.Body == nil {
		return
	}

	velocity := ball.Body.GetLinearVelocity()

	dragCoefficient := 0.00

	speed := velocity.Length()
	if speed > 0.1 {

		dragForceX := -dragCoefficient * velocity.X * speed
		dragForceY := -dragCoefficient * velocity.Y * speed

		ball.Body.ApplyForce(
			life.Box2dVec2(dragForceX, dragForceY),
			ball.Body.GetWorldCenter(),
			true,
		)
	}

	angularVelocity := ball.Body.GetAngularVelocity()
	if abs(float64(angularVelocity)) > 0.1 {
		angularDrag := -0.05 * angularVelocity
		ball.Body.SetAngularVelocity(angularVelocity + angularDrag)
	}
}

func (ballEntity *BallEntity) isTouchingGround() bool {
	return len(ballEntity.Shape.CollisionObjects) > 0
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
