package life

import (
	"image/color"
	"math"

	"github.com/ByteArena/box2d"
	"github.com/hajimehoshi/ebiten/v2"
)

// Border represents border properties
type Border struct {
	Width      float64
	Background color.Color
	Pattern    PatternType
}

// Helper function to create Box2D vector
func Box2dVec2(x, y float64) box2d.B2Vec2 {
	return box2d.MakeB2Vec2(x, y)
}

// Shape represents a game object
type Shape struct {
	*EventEmitter

	// Basic properties
	ID            string
	Name          string
	Tag           string
	Type          ShapeType
	X, Y          float64
	Width         float64
	Height        float64
	Radius        float64
	RotationAngle float64
	RotationSpeed float64
	RotationLock  bool
	Mass          float64
	Density       float64
	ZIndex        int
	Scale         float64
	Opacity       float64

	// Visual properties
	Pattern    PatternType
	Background color.Color
	Image      *ebiten.Image
	Border     *Border
	Flip       struct{ X, Y bool }

	// Physics properties
	IsBody   bool
	Physics  bool
	Velocity Vector2
	Speed    float64
	Rebound  float64
	Friction float64
	Body     *box2d.B2Body

	// Collision properties
	CollisionObjects []*Shape
	CacheDirection   string

	// State
	Hovered bool
	Clicked bool

	// Line coordinates (for line shapes)
	LineCoordinates struct{ X1, Y1, X2, Y2 float64 }

	// Callbacks
	OnCollisionFunc       func(*Shape)
	OnFinishCollisionFunc func(*Shape)

	// Reference to world
	world *World

	// Cached images to prevent color bleeding
	cachedColorImage *ebiten.Image
	lastBackground   color.Color

	directions *Axis
	Ghost      bool // Indicates if the shape is a ghost (not colliding with other shapes)

	// Collision filtering - simple approach
	noCollideWith map[string]bool // Track shapes this shouldn't collide with
}

// NewShape creates a new shape
func NewShape(props *ShapeProps) *Shape {
	if props == nil {
		props = &ShapeProps{}
	}

	// Set defaults
	if props.Type == "" {
		props.Type = ShapeRectangle
	}
	if props.Name == "" {
		props.Name = RandName()
	}
	if props.Tag == "" {
		props.Tag = "unknown"
	}
	if props.Width == 0 {
		props.Width = 10
	}
	if props.Height == 0 {
		props.Height = 10
	}
	if props.Speed == 0 {
		props.Speed = 3
	}
	if props.Scale == 0 {
		props.Scale = 1
	}
	if props.Opacity == 0 {
		props.Opacity = 1
	}
	if props.Background == nil {
		props.Background = color.RGBA{0, 0, 0, 255}
	}
	if props.Pattern == "" {
		props.Pattern = PatternColor
	}
	if props.Radius == 0 && props.Type == ShapeCircle {
		if props.Width != 0 {
			props.Radius = props.Width / 2
		} else if props.Height != 0 {
			props.Radius = props.Height / 2
		} else {
			props.Radius = 20
		}
	}

	shape := &Shape{
		EventEmitter:          NewEventEmitter(),
		ID:                    ID(),
		Name:                  props.Name,
		Tag:                   props.Tag,
		Type:                  props.Type,
		X:                     props.X,
		Y:                     props.Y,
		Width:                 props.Width,
		Height:                props.Height,
		Radius:                props.Radius,
		RotationAngle:         props.Rotation,
		RotationLock:          props.RotationLock,
		ZIndex:                props.ZIndex,
		Scale:                 props.Scale,
		Opacity:               props.Opacity,
		Pattern:               props.Pattern,
		Background:            props.Background,
		Image:                 props.Image,
		Border:                props.Border,
		IsBody:                props.IsBody,
		Physics:               props.Physics,
		Velocity:              props.Velocity,
		Speed:                 props.Speed,
		Rebound:               props.Rebound,
		Friction:              props.Friction,
		OnCollisionFunc:       props.OnCollisionFunc,
		OnFinishCollisionFunc: props.OnFinishCollisionFunc,
		Flip:                  props.Flip,
		directions:            &Axis{},
		Ghost:                 props.Ghost,
		noCollideWith:         make(map[string]bool),
	}

	// Fix: For circles, width and height should be diameter (2 * radius)
	if props.Radius > 0 && props.Type == ShapeCircle {
		shape.Width = props.Radius * 2
		shape.Height = props.Radius * 2
	}

	return shape
}

// ShapeProps contains properties for creating a shape
type ShapeProps struct {
	Type                  ShapeType
	X, Y                  float64
	Width, Height         float64
	Radius                float64
	ZIndex                int
	IsBody                bool
	Pattern               PatternType
	Background            color.Color
	Image                 *ebiten.Image
	Name                  string
	Rotation              float64
	RotationLock          bool
	Tag                   string
	OnCollisionFunc       func(*Shape)
	OnFinishCollisionFunc func(*Shape)
	Physics               bool
	Rebound               float64
	Friction              float64
	Mass                  float64
	Speed                 float64
	Velocity              Vector2
	Border                *Border
	Flip                  struct{ X, Y bool }
	Opacity               float64
	LineCoordinates       struct {
		A Vector2
		B Vector2
	}
	Ghost bool // Indicates if the shape is a ghost (not colliding with other shapes)
	Scale float64
}

func (s *Shape) Update() {
	s.updateDirection()
	s.updatePhysicsInfo()
}

func (obj *Shape) updateDirection() {
	if math.Abs(obj.Velocity.X) > EPSILON_DIRECTION_CHANGED {
		var newDirection AxisX
		if obj.directions.X != nil {
			newDirection = *obj.directions.X
		}

		if obj.Velocity.X < 0 && (obj.directions.X == nil || *obj.directions.X != DirectionLeft) {
			newDirection = DirectionLeft
		} else if obj.Velocity.X > 0 && (obj.directions.X == nil || *obj.directions.X != DirectionRight) {
			newDirection = DirectionRight
		}

		obj.directions.X = &newDirection
		if obj.directions.X != nil && obj.directions.Y != nil {
			obj.Emit(EventDirectionChange, EventDirectionChangeData{
				Direction: obj.directions,
			})
		}
	} else if obj.Velocity.X != 0 {
		obj.SetXVelocity(0)
	}

	if math.Abs(obj.Velocity.Y) > EPSILON_DIRECTION_CHANGED {
		var newDirection AxisY
		if obj.directions.Y != nil {
			newDirection = *obj.directions.Y
		}

		if obj.Velocity.Y < 0 && (obj.directions.Y == nil || *obj.directions.Y != DirectionUp) {
			newDirection = DirectionUp
		} else if obj.Velocity.Y > 0 && (obj.directions.Y == nil || *obj.directions.Y != DirectionDown) {
			newDirection = DirectionDown
		}

		obj.directions.Y = &newDirection
		if obj.directions.Y != nil && obj.directions.X != nil {
			obj.Emit(EventDirectionChange, EventDirectionChangeData{
				Direction: obj.directions,
			})
		}
	} else if obj.Velocity.Y != 0 {
		obj.SetXVelocity(0)
	}
}

func (s *Shape) updatePhysicsInfo() {
	center := s.Body.GetPosition()
	s.X = MetersToPixels(center.X) - s.Width/2
	s.Y = MetersToPixels(center.Y) - s.Height/2
	s.RotationAngle = s.Body.GetAngle()
	s.RotationSpeed = s.Body.GetAngularVelocity()
	s.Velocity = NewVector2(float64(int((MetersToPixels(s.Body.GetLinearVelocity().X)))), float64(int(MetersToPixels(s.Body.GetLinearVelocity().Y))))
	if s.Body.GetMass() > 0 {
		s.Mass = s.Body.GetMass()
	} else {
		s.Mass = 1
	}
}

func (s *Shape) requireInit() {
	if s.Body == nil {
		panic("Required: (*World).Register(Shape) pre-op.")
	}
}

func (s *Shape) LockRotation(lock bool) {
	s.requireInit()
	s.RotationLock = lock

	s.Body.SetFixedRotation(lock)
}

func (s *Shape) SetX(x float64) {
	s.X = x
	centerX := x + s.Width/2
	s.Body.SetTransform(box2d.MakeB2Vec2(PixelsToMeters(centerX), PixelsToMeters(s.Y+s.Height/2)), s.RotationAngle)
}

func (s *Shape) SetY(y float64) {
	s.Y = y
	centerY := y + s.Height/2
	s.Body.SetTransform(box2d.MakeB2Vec2(PixelsToMeters(s.X+s.Width/2), PixelsToMeters(centerY)), s.RotationAngle)
}

func (s *Shape) SetPosition(x, y float64) {
	s.X = x
	s.Y = y
	centerX := x + s.Width/2
	centerY := y + s.Height/2
	s.Body.SetTransform(box2d.MakeB2Vec2(PixelsToMeters(centerX), PixelsToMeters(centerY)), s.RotationAngle)
}

// SetRotation sets the rotation angle of the shape
func (s *Shape) SetRotation(angle float64) {
	s.RotationAngle = angle

	s.Body.SetTransform(s.Body.GetPosition(), angle*Deg)
}

// SetScale sets the scale of the shape
func (s *Shape) SetScale(scale float64) {
	s.Scale = scale

	// Box2D does not support scaling directly, so we need to recreate the body with the new scale
	// This is a simplified approach; in a real scenario, you would need to destroy the old body and create a new one
	s.Body.SetTransform(box2d.MakeB2Vec2(PixelsToMeters(s.X), PixelsToMeters(s.Y)), s.RotationAngle*Deg)
}

// SetBackground updates the background color and invalidates cached images
func (s *Shape) SetBackground(bg color.Color) {
	s.Background = bg
	// Invalidate cached image to force recreation with new color
	s.cachedColorImage = nil
}

// getColorImage returns a cached color image or creates a new one
func (s *Shape) getColorImage(width, height int) *ebiten.Image {
	// Check if we need to create/recreate the cached image
	if s.cachedColorImage == nil || s.lastBackground != s.Background {
		s.cachedColorImage = ebiten.NewImage(width, height)
		s.cachedColorImage.Fill(s.Background)
		s.lastBackground = s.Background
	}
	return s.cachedColorImage
}

// Draw renders the shape
func (s *Shape) Draw(screen *ebiten.Image) {

	if s.Opacity <= 0 {
		return
	}

	if s.Border != nil && s.Border.Width > 0 {
		s.drawBorder(screen)
	}

	switch s.Type {
	case ShapeRectangle:
		s.drawRectangle(screen)
	case ShapeSquare:
		s.drawSquare(screen)
	case ShapeCircle:
		s.drawCircle(screen)
	case ShapeDot:
		s.drawDot(screen)
	case ShapeLine:
		s.drawLine(screen)
	}
}

func (s *Shape) drawRectangle(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}

	switch s.Pattern {
	case PatternColor:
		// Pretty print s
		// Use cached color image to prevent color bleeding
		img := s.getColorImage(int(s.Width), int(s.Height))

		// Apply transformations for colored rectangle
		s.applyTransformations(op, s.Width, s.Height)
		screen.DrawImage(img, op)

	case PatternImage:
		if s.Image != nil {
			// Get original image dimensions
			imgBounds := s.Image.Bounds()
			imgWidth := float64(imgBounds.Dx())
			imgHeight := float64(imgBounds.Dy())

			// Apply transformations for image
			s.applyTransformations(op, imgWidth, imgHeight)
			screen.DrawImage(s.Image, op)
		}
	}
}

func (s *Shape) drawSquare(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}

	// For squares, ensure width and height are equal
	size := s.Width
	if s.Height > s.Width {
		size = s.Height
	}

	switch s.Pattern {
	case PatternColor:
		// Use cached color image to prevent color bleeding
		img := s.getColorImage(int(size), int(size))

		s.applyTransformations(op, size, size)
		screen.DrawImage(img, op)

	case PatternImage:
		if s.Image != nil {
			imgBounds := s.Image.Bounds()
			imgWidth := float64(imgBounds.Dx())
			imgHeight := float64(imgBounds.Dy())

			s.applyTransformations(op, imgWidth, imgHeight)
			screen.DrawImage(s.Image, op)
		}
	}
}

func (s *Shape) drawCircle(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}

	switch s.Pattern {
	case PatternColor:
		// Create a fresh circle image each time to prevent color bleeding
		size := int(s.Radius * 2)
		img := ebiten.NewImage(size, size)

		// Simple circle drawing (can be optimized)
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				dx := float64(x) - s.Radius
				dy := float64(y) - s.Radius
				if dx*dx+dy*dy <= s.Radius*s.Radius {
					img.Set(x, y, s.Background)
				}
			}
		}

		s.applyTransformations(op, s.Radius*2, s.Radius*2)
		screen.DrawImage(img, op)

	case PatternImage:
		if s.Image != nil {
			imgBounds := s.Image.Bounds()
			imgWidth := float64(imgBounds.Dx())
			imgHeight := float64(imgBounds.Dy())

			// For circles with images, use the circle's diameter as the target size
			s.applyTransformations(op, imgWidth, imgHeight)
			screen.DrawImage(s.Image, op)
		}
	}
}

func (s *Shape) drawLine(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}

	// Create a fresh line image each time
	img := ebiten.NewImage(int(s.Width), int(s.Height))
	img.Fill(s.Background)

	s.applyTransformations(op, s.Width, s.Height)
	screen.DrawImage(img, op)
}

func (s *Shape) drawDot(screen *ebiten.Image) {
	s.drawCircle(screen)
}

// applyTransformations applies all necessary transformations to the DrawImageOptions
func (s *Shape) applyTransformations(op *ebiten.DrawImageOptions, originalWidth, originalHeight float64) {
	// Step 1: Move origin to center of the original image
	op.GeoM.Translate(-originalWidth/2, -originalHeight/2)

	// Step 2: Apply flipping
	scaleX, scaleY := 1.0, 1.0
	if s.Flip.X {
		scaleX = -1.0
	}
	if s.Flip.Y {
		scaleY = -1.0
	}

	// Step 3: Scale to match shape dimensions
	if s.Pattern == PatternImage && s.Image != nil {
		// Scale image to fit shape dimensions
		scaleX *= s.Width / originalWidth
		scaleY *= s.Height / originalHeight
	}

	// Step 4: Apply shape scale
	scaleX *= s.Scale
	scaleY *= s.Scale

	op.GeoM.Scale(scaleX, scaleY)

	// Step 5: Apply rotation
	op.GeoM.Rotate(s.RotationAngle)

	// Step 6: Translate to final position (center of shape)
	op.GeoM.Translate(s.X+s.Width/2, s.Y+s.Height/2)

	// Step 7: Apply opacity
	if s.Opacity < 1.0 {
		op.ColorScale.Scale(1, 1, 1, float32(s.Opacity))
	}
}

func (s *Shape) drawBorder(screen *ebiten.Image) {
	// Border drawing implementation
	borderImg := ebiten.NewImage(int(s.Width+s.Border.Width*2), int(s.Height+s.Border.Width*2))
	borderImg.Fill(s.Border.Background)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(s.X-s.Border.Width, s.Y-s.Border.Width)
	screen.DrawImage(borderImg, op)
}

func (s *Shape) MoveTheta(angle float64, optionalSpeed ...float64) {
	speed := s.Speed
	if len(optionalSpeed) > 0 {
		speed = optionalSpeed[0]
	}

	// Apply force in the specified direction
	impulse := box2d.MakeB2Vec2(
		math.Cos(angle*Deg)*speed,
		math.Sin(angle*Deg)*speed,
	)
	s.Body.ApplyLinearImpulse(impulse, s.Body.GetWorldCenter(), true)

	s.X += math.Cos(angle*Deg) * speed
	s.Y += math.Sin(angle*Deg) * speed
}

func (s *Shape) Follow(target *Shape) {
	dx := target.X - s.X
	dy := target.Y - s.Y
	angle := math.Atan2(dy, dx)

	// Set velocity towards the target
	s.SetVelocity(math.Cos(angle)*s.Speed, math.Sin(angle)*s.Speed)
}

// Physics methods
func (s *Shape) SetVelocity(x, y float64) {
	s.Body.SetLinearVelocity(box2d.MakeB2Vec2(x, y))
}

// setXVelocity sets the X component of the velocity
func (s *Shape) SetXVelocity(x float64) {

	vel := s.Body.GetLinearVelocity()
	s.Body.SetLinearVelocity(box2d.MakeB2Vec2(x, vel.Y))
	s.Velocity.X = x

}

// setYVelocity sets the Y component of the velocity
func (s *Shape) SetYVelocity(y float64) {

	vel := s.Body.GetLinearVelocity()
	s.Body.SetLinearVelocity(box2d.MakeB2Vec2(vel.X, y))
	s.Velocity.Y = y

}

func (s *Shape) Jump(howHigh float64) {
	s.SetYVelocity(-howHigh)

}

func (s *Shape) Rotate(angle float64) {
	s.Body.SetTransform(s.Body.GetPosition(), s.Body.GetAngle()+angle*Deg)
}

func (s *Shape) CollideWith(object *Shape) {
	s.CollisionObjects = append(s.CollisionObjects, object)
}

func (s *Shape) FinishCollideWith(object *Shape) {
	for i, obj := range s.CollisionObjects {
		if obj.ID == object.ID {
			s.CollisionObjects = append(s.CollisionObjects[:i], s.CollisionObjects[i+1:]...)
			break
		}
	}
}

func (s *Shape) IsCollidingWith(target *Shape) bool {
	for _, obj := range s.CollisionObjects {
		if obj.ID == target.ID {
			return true
		}
	}
	return false
}

// Utility methods
func (s *Shape) Remove() {
	if s.world != nil {
		s.world.Unregister(s)
	}
}

func (s *Shape) IsOutOfMap() bool {
	if s.world == nil {
		return false
	}
	return s.X < 0 || s.X > float64(s.world.Width) || s.Y < 0 || s.Y > float64(s.world.Height)
}

func (s *Shape) SetProps(props map[string]interface{}) {
	// Reflection-based property setting could be implemented here
	// For now, this is a placeholder
}

func (s *Shape) Get(property string) interface{} {
	// Property getter implementation
	return nil
}

func (s *Shape) Set(property string, value interface{}) {
	// Property setter implementation
}

// Movement methods
func (s *Shape) Move(direction string) {
	switch direction {
	case "up":
		s.SetYVelocity(-s.Speed)
	case "down":
		s.SetYVelocity(s.Speed)
	case "left":
		s.SetXVelocity(-s.Speed)
	case "right":
		s.SetXVelocity(s.Speed)
	}
}

// NotCollideWith prevents this shape from colliding with another specific shape
// This is the SIMPLE approach - just track which shapes shouldn't collide
func (s *Shape) NotCollideWith(other *Shape) {
	s.requireInit()
	other.requireInit()

	// Add to our no-collide list for tracking
	s.noCollideWith[other.ID] = true
	other.noCollideWith[s.ID] = true
}

// RestoreCollisionWith restores collision between this shape and another specific shape
func (s *Shape) RestoreCollisionWith(other *Shape) {
	s.requireInit()
	other.requireInit()

	// Remove from no-collide list
	delete(s.noCollideWith, other.ID)
	delete(other.noCollideWith, s.ID)
}

// NotCollideWithTag prevents this shape from colliding with all shapes that have the specified tag
func (s *Shape) NotCollideWithTag(tag string) {
	s.requireInit()

	if s.world == nil {
		return
	}

	// Get all shapes with the specified tag
	taggedShapes := s.world.GetElementsByTagName(tag)

	// Apply NotCollideWith to each tagged shape
	for _, taggedShape := range taggedShapes {
		s.NotCollideWith(taggedShape)
	}
}

// RestoreCollisionWithTag restores collision between this shape and all shapes with the specified tag
func (s *Shape) RestoreCollisionWithTag(tag string) {
	s.requireInit()

	if s.world == nil {
		return
	}

	// Get all shapes with the specified tag
	taggedShapes := s.world.GetElementsByTagName(tag)

	// Restore collision with each tagged shape
	for _, taggedShape := range taggedShapes {
		s.RestoreCollisionWith(taggedShape)
	}
}

// ShouldCollideWith checks if this shape should collide with another shape
func (s *Shape) ShouldCollideWith(other *Shape) bool {
	return !s.noCollideWith[other.ID]
}
