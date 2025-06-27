package life

import (
	"image/color"
	"math"

	"github.com/ByteArena/box2d"
	"github.com/hajimehoshi/ebiten/v2"
)

type Border struct {
	Width      float64
	Background color.Color
	Pattern    PatternType
}

func Box2dVec2(x, y float64) box2d.B2Vec2 {
	return box2d.MakeB2Vec2(x, y)
}

type Shape struct {
	*EventEmitter

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

	Pattern    PatternType
	Background color.Color
	Image      *ebiten.Image
	Border     *Border
	Flip       struct{ X, Y bool }

	IsBody   bool
	Physics  bool
	Velocity Vector2
	Speed    float64
	Rebound  float64
	Friction float64
	Body     *box2d.B2Body

	CollisionObjects []*Shape
	CacheDirection   string

	Hovered bool
	Clicked bool

	LineCoordinates struct{ X1, Y1, X2, Y2 float64 }

	OnCollisionFunc       func(*Shape)
	OnFinishCollisionFunc func(*Shape)

	world *World

	cachedColorImage *ebiten.Image
	lastBackground   color.Color

	directions *Axis
	Ghost      bool

	noCollideWith        map[string]bool
	LastCollisionImpulse float64
}

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
	Ghost                bool
	Scale                float64
	LastCollisionImpulse float64
}

func NewShape(props *ShapeProps) *Shape {
	if props == nil {
		props = &ShapeProps{}
	}

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
		LastCollisionImpulse:  props.LastCollisionImpulse,
	}

	if props.Radius > 0 && props.Type == ShapeCircle {
		shape.Width = props.Radius * 2
		shape.Height = props.Radius * 2
	}

	return shape
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

func (s *Shape) SetRotation(angle float64) {
	s.RotationAngle = angle

	s.Body.SetTransform(s.Body.GetPosition(), angle*Deg)
}

func (s *Shape) SetScale(scale float64) {
	s.Scale = scale

	s.Body.SetTransform(box2d.MakeB2Vec2(PixelsToMeters(s.X), PixelsToMeters(s.Y)), s.RotationAngle*Deg)
}

func (s *Shape) SetBackground(bg color.Color) {
	s.Background = bg

	s.cachedColorImage = nil
}

func (s *Shape) getColorImage(width, height int) *ebiten.Image {

	if s.cachedColorImage == nil || s.lastBackground != s.Background {
		s.cachedColorImage = ebiten.NewImage(width, height)
		s.cachedColorImage.Fill(s.Background)
		s.lastBackground = s.Background
	}
	return s.cachedColorImage
}

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

		img := s.getColorImage(int(s.Width), int(s.Height))

		s.applyTransformations(op, s.Width, s.Height)
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

func (s *Shape) drawSquare(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}

	size := s.Width
	if s.Height > s.Width {
		size = s.Height
	}

	switch s.Pattern {
	case PatternColor:

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

		size := int(s.Radius * 2)
		img := ebiten.NewImage(size, size)

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

			s.applyTransformations(op, imgWidth, imgHeight)
			screen.DrawImage(s.Image, op)
		}
	}
}

func (s *Shape) drawLine(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}

	img := ebiten.NewImage(int(s.Width), int(s.Height))
	img.Fill(s.Background)

	s.applyTransformations(op, s.Width, s.Height)
	screen.DrawImage(img, op)
}

func (s *Shape) drawDot(screen *ebiten.Image) {
	s.drawCircle(screen)
}

func (s *Shape) applyTransformations(op *ebiten.DrawImageOptions, originalWidth, originalHeight float64) {

	op.GeoM.Translate(-originalWidth/2, -originalHeight/2)

	// scale := ebiten.Monitor().DeviceScaleFactor()
	// op.GeoM.Scale(scale, scale)

	op.Filter = ebiten.FilterLinear

	scaleX, scaleY := 1.0, 1.0
	if s.Flip.X {
		scaleX = -1.0
	}
	if s.Flip.Y {
		scaleY = -1.0
	}

	if s.Pattern == PatternImage && s.Image != nil {

		scaleX *= s.Width / originalWidth
		scaleY *= s.Height / originalHeight
	}

	scaleX *= s.Scale
	scaleY *= s.Scale

	op.GeoM.Scale(scaleX, scaleY)

	op.GeoM.Rotate(s.RotationAngle)

	op.GeoM.Translate(s.X+s.Width/2, s.Y+s.Height/2)

	if s.Opacity < 1.0 {
		op.ColorScale.Scale(1, 1, 1, float32(s.Opacity))
	}
}

func (s *Shape) drawBorder(screen *ebiten.Image) {

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

	s.SetVelocity(math.Cos(angle)*s.Speed, math.Sin(angle)*s.Speed)
}

func (s *Shape) SetVelocity(x, y float64) {
	s.Body.SetLinearVelocity(box2d.MakeB2Vec2(x, y))
}

func (s *Shape) SetXVelocity(x float64) {

	vel := s.Body.GetLinearVelocity()
	s.Body.SetLinearVelocity(box2d.MakeB2Vec2(x, vel.Y))
	s.Velocity.X = x

}

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

}

func (s *Shape) Get(property string) interface{} {

	return nil
}

func (s *Shape) Set(property string, value interface{}) {

}

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

func (s *Shape) NotCollideWith(other *Shape) {
	s.requireInit()
	other.requireInit()

	s.noCollideWith[other.ID] = true
	other.noCollideWith[s.ID] = true
}

func (s *Shape) RestoreCollisionWith(other *Shape) {
	s.requireInit()
	other.requireInit()

	delete(s.noCollideWith, other.ID)
	delete(other.noCollideWith, s.ID)
}

func (s *Shape) NotCollideWithTag(tag string) {
	s.requireInit()

	if s.world == nil {
		return
	}

	taggedShapes := s.world.GetElementsByTagName(tag)

	for _, taggedShape := range taggedShapes {
		s.NotCollideWith(taggedShape)
	}
}

func (s *Shape) RestoreCollisionWithTag(tag string) {
	s.requireInit()

	if s.world == nil {
		return
	}

	taggedShapes := s.world.GetElementsByTagName(tag)

	for _, taggedShape := range taggedShapes {
		s.RestoreCollisionWith(taggedShape)
	}
}

func (s *Shape) ShouldCollideWith(other *Shape) bool {
	return !s.noCollideWith[other.ID]
}
