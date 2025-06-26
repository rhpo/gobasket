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
	ballRadius = 25 / 2 // Slightly bigger for more basketball feel
)

// NewBall creates a new ball instance
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
		Friction:     0.8,   // Much higher friction - basketballs have good grip
		Rebound:      0.6,   // Lower bounce - not a super ball!
		RotationLock: false, // Allow spinning
		Mass:         8.0,   // Much heavier - real basketball weighs ~600g
		ZIndex:       1000,  // Ensure the ball is drawn above other shapes
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
	// Add realistic rolling physics
	ballEntity.updateRollingPhysics()

	// Add air resistance to make it feel more realistic
	ballEntity.applyAirResistance()
}

// updateRollingPhysics applies realistic rolling rotation to the ball
func (ballEntity *BallEntity) updateRollingPhysics() {
	ball := ballEntity.Shape

	if ball.Body == nil {
		return
	}

	// Get linear velocity
	velocity := ball.Body.GetLinearVelocity()

	// Only apply rolling if the ball is moving and touching something
	if abs(float64(velocity.X)) > 0.1 && ballEntity.isTouchingGround() {
		// Calculate angular velocity for realistic rolling
		// ω = v / r (angular velocity = linear velocity / radius)
		// Moving right (positive X) = clockwise rotation (positive angular velocity)
		// Moving left (negative X) = counter-clockwise rotation (negative angular velocity)
		targetAngularVelocity := float64(velocity.X) / life.PixelsToMeters(ball.Radius)

		// Apply smoothing
		currentAngularVelocity := ball.Body.GetAngularVelocity()
		smoothingFactor := 0.3 // More responsive for basketball feel
		newAngularVelocity := currentAngularVelocity + (targetAngularVelocity-currentAngularVelocity)*smoothingFactor

		ball.Body.SetAngularVelocity(newAngularVelocity)
	}
}

// applyAirResistance makes the ball slow down more realistically
func (ballEntity *BallEntity) applyAirResistance() {
	ball := ballEntity.Shape

	if ball.Body == nil {
		return
	}

	velocity := ball.Body.GetLinearVelocity()

	// Apply air resistance (drag force proportional to velocity squared)
	dragCoefficient := 0.00 // Adjust this to control air resistance

	// Calculate drag force
	speed := velocity.Length()
	if speed > 0.1 {
		// Drag force opposes motion
		dragForceX := -dragCoefficient * velocity.X * speed
		dragForceY := -dragCoefficient * velocity.Y * speed

		// Apply the drag force
		ball.Body.ApplyForce(
			life.Box2dVec2(dragForceX, dragForceY),
			ball.Body.GetWorldCenter(),
			true,
		)
	}

	// Also apply angular drag (rotation slows down)
	angularVelocity := ball.Body.GetAngularVelocity()
	if abs(float64(angularVelocity)) > 0.1 {
		angularDrag := -0.05 * angularVelocity // Small angular drag
		ball.Body.SetAngularVelocity(angularVelocity + angularDrag)
	}
}

// isTouchingGround checks if the ball is touching the ground or any surface
func (ballEntity *BallEntity) isTouchingGround() bool {
	return len(ballEntity.Shape.CollisionObjects) > 0
}

// Helper function for absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
