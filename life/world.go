package life

import (
	"embed"
	"image/color"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/ByteArena/box2d"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type ContactListener struct {
	box2d.B2ContactListenerInterface
	world *World
}

func (cl *ContactListener) PreSolve(contact box2d.B2ContactInterface, oldManifold box2d.B2Manifold) {

	fixtureA := contact.GetFixtureA()
	fixtureB := contact.GetFixtureB()

	if fixtureA == nil || fixtureB == nil {
		return
	}

	bodyA := fixtureA.GetBody()
	bodyB := fixtureB.GetBody()

	var shapeA, shapeB *Shape
	for _, obj := range cl.world.Objects {
		if obj.Body == bodyA {
			shapeA = obj
		} else if obj.Body == bodyB {
			shapeB = obj
		}
	}

	if shapeA == nil || shapeB == nil {
		return
	}

	if !shapeA.ShouldCollideWith(shapeB) || !shapeB.ShouldCollideWith(shapeA) {

		contact.SetEnabled(false)
	}
}

func (cl *ContactListener) PostSolve(contact box2d.B2ContactInterface, impulse *box2d.B2ContactImpulse) {
	bodyA := contact.GetFixtureA().GetBody()
	bodyB := contact.GetFixtureB().GetBody()

	var shapeA, shapeB *Shape
	for _, obj := range cl.world.Objects {
		if obj.Body == bodyA {
			shapeA = obj
		} else if obj.Body == bodyB {
			shapeB = obj
		}
	}

	if shapeA == nil || shapeB == nil {
		return
	}

	// Get collision normal to determine direction
	var worldManifold box2d.B2WorldManifold
	contact.GetWorldManifold(&worldManifold)

	normal := worldManifold.Normal

	// We care only about mostly vertical collisions
	if math.Abs(normal.Y) > 0.7 {
		imp := totalImpulse(*impulse)

		if imp > 0.5 { // ignore soft impacts
			shapeA.LastCollisionImpulse = imp
			shapeB.LastCollisionImpulse = imp
		}
	}
}

func totalImpulse(imp box2d.B2ContactImpulse) float64 {
	n0 := imp.NormalImpulses[0]
	n1 := imp.NormalImpulses[1]
	t0 := imp.TangentImpulses[0]
	t1 := imp.TangentImpulses[1]
	return n0 + n1 + t0 + t1
}

func (cl *ContactListener) BeginContact(contact box2d.B2ContactInterface) {

	fixtureA := contact.GetFixtureA()
	fixtureB := contact.GetFixtureB()

	if fixtureA == nil || fixtureB == nil {
		return
	}

	bodyA := fixtureA.GetBody()
	bodyB := fixtureB.GetBody()

	var shapeA, shapeB *Shape
	for _, obj := range cl.world.Objects {
		if obj.Body == bodyA {
			shapeA = obj
		} else if obj.Body == bodyB {
			shapeB = obj
		}
	}

	if shapeA == nil || shapeB == nil {
		return
	}

	cl.world.queueCollision(shapeA, shapeB)
}

func (cl *ContactListener) EndContact(contact box2d.B2ContactInterface) {

	fixtureA := contact.GetFixtureA()
	fixtureB := contact.GetFixtureB()

	if fixtureA == nil || fixtureB == nil {
		return
	}

	bodyA := fixtureA.GetBody()
	bodyB := fixtureB.GetBody()

	var shapeA, shapeB *Shape
	for _, obj := range cl.world.Objects {
		if obj.Body == bodyA {
			shapeA = obj
		} else if obj.Body == bodyB {
			shapeB = obj
		}
	}

	if shapeA == nil || shapeB == nil {
		return
	}

	shapeA.FinishCollideWith(shapeB)
	shapeB.FinishCollideWith(shapeA)
}

type CollisionEvent struct {
	ShapeA *Shape
	ShapeB *Shape
}

type DrawCommand struct {
	Type  ShapeType
	Props *ShapeProps
}

type World struct {
	*EventEmitter

	Width  int
	Height int

	PhysicsWorld    *box2d.B2World
	contactListener *ContactListener
	G               Vector2
	AirResistance   float64

	Screen *ebiten.Image

	drawCommands []DrawCommand
	drawMutex    sync.Mutex

	Tick       GameLoop
	Init       func()
	Render     func(screen *ebiten.Image)
	Title      string
	lastUpdate time.Time

	Pattern    PatternType
	Background color.Color
	Border     *Border

	Objects []*Shape
	mutex   sync.RWMutex

	AudioManager *AudioManager

	Mouse struct {
		X, Y                          float64
		IsLeftClicked, IsRightClicked bool
		IsMiddleClicked               bool
	}
	Keys      map[ebiten.Key]bool
	keysMutex sync.RWMutex

	HasLimits bool
	Paused    bool
	Cursor    CursorType

	OnMouseDown func(x, y float64)
	OnMouseUp   func(x, y float64)
	OnMouseMove func(x, y float64)

	Levels       []Level
	CurrentLevel int

	pendingLevelSwitch *int
	collisionQueue     []CollisionEvent
	collisionMutex     sync.Mutex
}

func (w *World) SetTagCollisionFilter(tag string, collidesWith []string) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	taggedShapes := w.GetElementsByTagName(tag)
	if len(taggedShapes) == 0 {
		return
	}

	groupIndex := int16(w.getTagGroupIndex(tag))

	if len(collidesWith) == 0 {
		groupIndex = -groupIndex
	}

	for _, shape := range taggedShapes {
		if shape.Body != nil {
			fixture := shape.Body.GetFixtureList()
			if fixture != nil {
				filter := fixture.GetFilterData()
				filter.GroupIndex = groupIndex
				fixture.SetFilterData(filter)
			}
		}
	}

	if len(collidesWith) > 0 {
		w.setupCategoryMaskFiltering(tag, collidesWith)
	}
}

func (w *World) setupCategoryMaskFiltering(tag string, collidesWith []string) {

	taggedShapes := w.GetElementsByTagName(tag)

	categoryBit := uint16(1 << w.getTagCategoryBit(tag))

	maskBits := uint16(0)
	for _, collidesWithTag := range collidesWith {
		maskBits |= uint16(1 << w.getTagCategoryBit(collidesWithTag))
	}

	for _, shape := range taggedShapes {
		if shape.Body != nil {
			fixture := shape.Body.GetFixtureList()
			if fixture != nil {
				filter := fixture.GetFilterData()
				filter.CategoryBits = categoryBit
				filter.MaskBits = maskBits
				fixture.SetFilterData(filter)
			}
		}
	}
}

func (w *World) getTagGroupIndex(tag string) int {
	hash := 0
	for _, char := range tag {
		hash = hash*31 + int(char)
	}
	return (hash % 32000) + 1
}

func (w *World) getTagCategoryBit(tag string) uint16 {
	hash := 0
	for _, char := range tag {
		hash = hash*31 + int(char)
	}

	bitPosition := (hash % 15) + 1
	return uint16(1 << bitPosition)
}

func (w *World) DisableCollisionBetweenTags(tagA, tagB string) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	shapesA := w.GetElementsByTagName(tagA)
	shapesB := w.GetElementsByTagName(tagB)

	for _, shapeA := range shapesA {
		for _, shapeB := range shapesB {
			shapeA.NotCollideWith(shapeB)
		}
	}
}

func (w *World) EnableCollisionBetweenTags(tagA, tagB string) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	shapesA := w.GetElementsByTagName(tagA)
	shapesB := w.GetElementsByTagName(tagB)

	for _, shapeA := range shapesA {
		for _, shapeB := range shapesB {
			shapeA.RestoreCollisionWith(shapeB)
		}
	}
}

type WorldProps struct {
	Width         int
	Height        int
	G             Vector2
	Pattern       PatternType
	Background    color.Color
	HasLimits     bool
	Border        *Border
	Paused        bool
	Cursor        CursorType
	Title         string
	AirResistance float64
	AudioProps    *AudioProps

	Levels       []Level
	CurrentLevel int
}

func NewWorld(props *WorldProps) *World {
	if props == nil {
		props = &WorldProps{}
	}

	if props.Width == 0 {
		props.Width = 800
	}
	if props.Height == 0 {
		props.Height = 600
	}
	if props.Background == nil {
		props.Background = color.RGBA{0, 0, 0, 255}
	}
	if props.Pattern == "" {
		props.Pattern = PatternColor
	}
	if props.Cursor == "" {
		props.Cursor = CursorDefault
	}
	if props.Title == "" {
		props.Title = "Life Game"
	}

	contactListener := ContactListener{}

	gravity := box2d.MakeB2Vec2(MetersToPixels(props.G.X), MetersToPixels(props.G.Y))

	physicsWorld := box2d.MakeB2World(gravity)
	physicsWorld.SetAllowSleeping(true)

	physicsWorld.SetContactListener(&contactListener)

	world := &World{
		EventEmitter:       NewEventEmitter(),
		contactListener:    &contactListener,
		PhysicsWorld:       &physicsWorld,
		Width:              props.Width,
		Height:             props.Height,
		G:                  props.G,
		Tick:               nil,
		Pattern:            props.Pattern,
		Background:         props.Background,
		Border:             props.Border,
		Paused:             props.Paused,
		Cursor:             props.Cursor,
		Keys:               make(map[ebiten.Key]bool),
		lastUpdate:         time.Now(),
		Title:              props.Title,
		AirResistance:      props.AirResistance,
		AudioManager:       NewAudioManager(props.AudioProps),
		Levels:             props.Levels,
		CurrentLevel:       0,
		pendingLevelSwitch: nil,
		collisionQueue:     make([]CollisionEvent, 0),
		drawCommands:       make([]DrawCommand, 0),
	}

	if len(world.Levels) == 0 {
		world.Levels = []Level{
			{
				Map:      Map{},
				MapItems: MapItems{},
				Init: func(world *World) {
					world.Levels[world.CurrentLevel].Init(world)
				},
				Tick:   world.Tick,
				Render: world.Render,
			},
		}
	}

	contactListener.world = world

	if world.Render == nil {
		world.Render = func(screen *ebiten.Image) {
			if world.CurrentLevel < len(world.Levels) && world.Levels[world.CurrentLevel].Render != nil {
				world.Levels[world.CurrentLevel].Render(screen)
			}
		}
	}

	if world.Init == nil {
		world.Init = func() {
			if world.CurrentLevel < len(world.Levels) && world.Levels[world.CurrentLevel].Init != nil {
				world.Levels[world.CurrentLevel].Init(world)
			}
		}
	}

	if world.Tick == nil {
		world.Tick = func(ld LoopData) {
			if world.CurrentLevel < len(world.Levels) && world.Levels[world.CurrentLevel].Tick != nil {
				world.Levels[world.CurrentLevel].Tick(ld)
			}
		}
	}

	return world
}

func (w *World) Pen(shapeType ShapeType, props *ShapeProps) {
	w.drawMutex.Lock()
	defer w.drawMutex.Unlock()

	if props == nil {
		props = &ShapeProps{}
	}

	drawProps := *props
	drawProps.Type = shapeType

	if drawProps.Pattern == "" {
		drawProps.Pattern = PatternColor
	}
	if drawProps.Background == nil {
		drawProps.Background = color.RGBA{255, 255, 255, 255}
	}
	if drawProps.Width == 0 {
		drawProps.Width = 10
	}
	if drawProps.Height == 0 {
		drawProps.Height = 10
	}
	if drawProps.Radius == 0 && shapeType == ShapeCircle {
		drawProps.Radius = 10
	}

	w.drawCommands = append(w.drawCommands, DrawCommand{
		Type:  shapeType,
		Props: &drawProps,
	})
}

func (w *World) Line(x1, y1, x2, y2 float64, lineColor color.Color, thickness float64) {
	if thickness <= 0 {
		thickness = 1
	}

	dx := x2 - x1
	dy := y2 - y1
	length := math.Sqrt(dx*dx + dy*dy)
	angle := math.Atan2(dy, dx)

	centerX := x1 + dx/2
	centerY := y1 + dy/2

	w.Pen(ShapeRectangle, &ShapeProps{
		X:          centerX - length/2,
		Y:          centerY - thickness/2,
		Width:      length,
		Height:     thickness,
		Background: lineColor,
		Pattern:    PatternColor,
		Rotation:   angle,
		ZIndex:     1000,
	})
}

func (w *World) Circle(x, y, radius float64, circleColor color.Color) {
	w.Pen(ShapeCircle, &ShapeProps{
		X:          x - radius,
		Y:          y - radius,
		Radius:     radius,
		Background: circleColor,
		Pattern:    PatternColor,
		ZIndex:     1000,
	})
}

func (w *World) Rect(x, y, width, height float64, rectColor color.Color) {
	w.Pen(ShapeRectangle, &ShapeProps{
		X:          x,
		Y:          y,
		Width:      width,
		Height:     height,
		Background: rectColor,
		Pattern:    PatternColor,
		ZIndex:     1000,
	})
}

func (w *World) queueCollision(shapeA, shapeB *Shape) {
	w.collisionMutex.Lock()
	defer w.collisionMutex.Unlock()

	w.collisionQueue = append(w.collisionQueue, CollisionEvent{
		ShapeA: shapeA,
		ShapeB: shapeB,
	})
}

func (w *World) processCollisions() {
	w.collisionMutex.Lock()
	collisions := make([]CollisionEvent, len(w.collisionQueue))
	copy(collisions, w.collisionQueue)
	w.collisionQueue = w.collisionQueue[:0]
	w.collisionMutex.Unlock()

	for _, collision := range collisions {
		shapeA := collision.ShapeA
		shapeB := collision.ShapeB

		w.Emit(EventCollision, EventCollisionData{
			ShapeA: shapeA,
			ShapeB: shapeB,
		})

		shapeA.CollideWith(shapeB)
		shapeB.CollideWith(shapeA)

		if shapeA.OnCollisionFunc != nil {
			shapeA.OnCollisionFunc(shapeB)
		}

		if shapeB.OnCollisionFunc != nil {
			shapeB.OnCollisionFunc(shapeA)
		}
	}
}

func (w *World) Destroy() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	for _, obj := range w.Objects {
		if obj.Body != nil {
			w.PhysicsWorld.DestroyBody(obj.Body)
		}
	}

	w.Objects = nil
	w.contactListener.world = nil
	w.contactListener = nil
	w.PhysicsWorld = nil

	if w.AudioManager != nil {
		w.AudioManager.Cleanup()
	}
}

func (w *World) NextLevel() {
	if w.CurrentLevel+1 >= len(w.Levels) {
		return
	}

	w.SwitchToLevel(w.CurrentLevel + 1)
}

func (w *World) SwitchToLevel(levelIndex int) {
	if levelIndex < 0 || levelIndex >= len(w.Levels) {
		return
	}

	w.pendingLevelSwitch = &levelIndex
}

func (w *World) SelectLevel(index int) {
	if index < 0 || index >= len(w.Levels) {
		return
	}

	w.CurrentLevel = index
	level := w.Levels[index]

	if level.OnDestroy != nil {

		level.OnDestroy(w)
	}

	w.mutex.Lock()

	for _, obj := range w.Objects {
		if obj.Body != nil {
			w.PhysicsWorld.DestroyBody(obj.Body)
			obj.Body = nil
		}
	}
	w.Objects = make([]*Shape, 0)
	w.mutex.Unlock()

	if level.Tick != nil {
		w.Tick = level.Tick
	} else {
		w.Tick = func(ld LoopData) {}
	}

	if level.Render != nil {
		w.Render = level.Render
	} else {
		w.Render = func(screen *ebiten.Image) {}
	}

	if level.Init != nil {
		level.Init(w)
	}

	w.GenerateLevelFromMap(level.Map, level.MapItems)

	if level.OnMount != nil {
		level.OnMount()
	}

}

func (w *World) CreateBorders() {
	borderWidth := 10.0
	if w.Border != nil && w.Border.Width > 0 {
		borderWidth = w.Border.Width
	}

	var borderColor color.Color = color.RGBA{0, 0, 0, 255}
	if w.Border != nil && w.Border.Background != nil {
		borderColor = w.Border.Background
	}

	borders := []*Shape{
		NewShape(&ShapeProps{
			Type:       ShapeRectangle,
			X:          0,
			Y:          0,
			Width:      float64(w.Width),
			Height:     borderWidth,
			Background: borderColor,
			Tag:        "border",
			Name:       "borderTop",
			Physics:    true,
			IsBody:     true,
		}),
		NewShape(&ShapeProps{
			Type:       ShapeRectangle,
			X:          0,
			Y:          float64(w.Height) - borderWidth,
			Width:      float64(w.Width),
			Height:     borderWidth,
			Background: borderColor,
			Tag:        "border",
			Name:       "borderBottom",
			Physics:    true,
			IsBody:     true,
		}),
		NewShape(&ShapeProps{
			Type:       ShapeRectangle,
			X:          0,
			Y:          0,
			Width:      borderWidth,
			Height:     float64(w.Height),
			Background: borderColor,
			Tag:        "border",
			Name:       "borderLeft",
			Physics:    true,
			IsBody:     true,
		}),
		NewShape(&ShapeProps{
			Type:       ShapeRectangle,
			X:          float64(w.Width) - borderWidth,
			Y:          0,
			Width:      borderWidth,
			Height:     float64(w.Height),
			Background: borderColor,
			Tag:        "border",
			Name:       "borderRight",
			Physics:    true,
			IsBody:     true,
		}),
	}

	for _, border := range borders {
		w.Register(border)
	}
}

func (w *World) Register(object *Shape) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	object.world = w
	w.Objects = append(w.Objects, object)
	w.createPhysicsBody(object)
}

func (w *World) Unregister(object *Shape) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	for i, obj := range w.Objects {
		if obj.ID == object.ID {
			if obj.Body != nil {
				w.PhysicsWorld.DestroyBody(obj.Body)
			}
			w.Objects = append(w.Objects[:i], w.Objects[i+1:]...)
			break
		}
	}
}

func (w *World) createPhysicsBody(object *Shape) {
	bodyDef := box2d.MakeB2BodyDef()
	bodyDef.AllowSleep = true

	if !object.IsBody || object.Tag == "border" {
		bodyDef.Type = box2d.B2BodyType.B2_staticBody
	} else {
		bodyDef.Type = box2d.B2BodyType.B2_dynamicBody
	}

	if !object.Physics {
		bodyDef.GravityScale = 0
	}

	bodyDef.FixedRotation = false

	centerX := object.X + object.Width/2
	centerY := object.Y + object.Height/2
	bodyDef.Position.Set(PixelsToMeters(centerX), PixelsToMeters(centerY))

	body := w.PhysicsWorld.CreateBody(&bodyDef)

	body.SetMassData(&box2d.B2MassData{
		Mass: object.Mass,
	})

	var shape box2d.B2ShapeInterface
	switch object.Type {
	case ShapeCircle:
		circleShape := box2d.MakeB2CircleShape()
		circleShape.SetRadius(PixelsToMeters(object.Radius))
		shape = &circleShape
	default:
		boxShape := box2d.MakeB2PolygonShape()
		if object.Width <= 0 || object.Height <= 0 {
			panic("Width and Height must be greater than 0 for rectangle shapes")
		}
		boxShape.SetAsBox(PixelsToMeters(object.Width/2), PixelsToMeters(object.Height/2))
		shape = &boxShape
	}

	density := 0.0
	if !object.Physics {
		density = 0.0
	}

	fixture := body.CreateFixture(shape, density)

	if object.Ghost {
		fixture.SetSensor(true)
	}

	fixture.SetFriction(object.Friction)
	fixture.SetRestitution(object.Rebound)

	filter := fixture.GetFilterData()
	filter.CategoryBits = 1
	filter.MaskBits = 0xFFFF
	filter.GroupIndex = 0

	fixture.SetFilterData(filter)

	object.Body = body
}

func (w *World) GenerateLevelFromMap(levelMap Map, objects map[string]func(position Vector2, width, height float64)) {
	if len(levelMap) == 0 {
		return
	}

	rows := len(levelMap)
	cols := len(levelMap[0])
	tileWidth := float64(w.Width) / float64(cols)
	tileHeight := float64(w.Height) / float64(rows)

	const overlap = 0.5

	for y, row := range levelMap {
		for x, ch := range row {
			fn, ok := objects[string(ch)]
			if !ok {
				continue
			}

			pos := Vector2{
				X: float64(x)*tileWidth - overlap/2,
				Y: float64(y)*tileHeight - overlap/2,
			}
			fn(pos, tileWidth+overlap, tileHeight+overlap)
		}
	}
}

func (w *World) Update() error {
	if w.Paused {
		return nil
	}

	now := time.Now()
	var deltaTime float64
	if !w.lastUpdate.IsZero() {
		deltaTime = now.Sub(w.lastUpdate).Seconds()
	} else {
		deltaTime = 1.0 / 60.0
	}
	w.lastUpdate = now

	velocityIterations := 6
	positionIterations := 3
	w.PhysicsWorld.Step(deltaTime, velocityIterations, positionIterations)

	if w.AudioManager != nil {
		w.AudioManager.Update()
	}

	w.processCollisions()

	if w.pendingLevelSwitch != nil {
		levelIndex := *w.pendingLevelSwitch
		w.pendingLevelSwitch = nil
		w.SelectLevel(levelIndex)
		return nil
	}

	w.mutex.RLock()
	objects := make([]*Shape, len(w.Objects))
	copy(objects, w.Objects)
	w.mutex.RUnlock()

	for _, obj := range objects {
		obj.Update()
	}

	if w.Tick != nil {
		w.Tick(LoopData{
			Time:  now,
			Delta: deltaTime,
		})
	}

	w.updateInput()
	return nil
}

func (w *World) updateInput() {
	x, y := ebiten.CursorPosition()
	w.Mouse.X = float64(x)
	w.Mouse.Y = float64(y)

	w.Mouse.IsLeftClicked = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	w.Mouse.IsRightClicked = ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	w.Mouse.IsMiddleClicked = ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle)

	w.keysMutex.Lock()
	for key := ebiten.Key(0); key <= ebiten.KeyMax; key++ {
		w.Keys[key] = ebiten.IsKeyPressed(key)
	}
	w.keysMutex.Unlock()

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		w.handleMouseDown(w.Mouse.X, w.Mouse.Y)
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		w.handleMouseUp(w.Mouse.X, w.Mouse.Y)
	}
}

func (w *World) handleMouseDown(x, y float64) {
	hoveredObjects := w.HoveredObjects()
	for _, obj := range hoveredObjects {
		if !obj.Clicked {
			obj.Clicked = true
			obj.Emit(EventMouseDown, Vector2{X: x, Y: y})
		}
	}

	if w.OnMouseDown != nil {
		w.OnMouseDown(x, y)
	}
}

func (w *World) handleMouseUp(x, y float64) {
	hoveredObjects := w.HoveredObjects()
	for _, obj := range hoveredObjects {
		obj.Emit(EventMouseUp, Vector2{X: x, Y: y})
		obj.Emit(EventClick, Vector2{X: x, Y: y})
		if obj.Clicked {
			obj.Clicked = false
		}
	}

	if w.OnMouseUp != nil {
		w.OnMouseUp(x, y)
	}
}

func (w *World) Draw(screen *ebiten.Image) {
	if w.Screen != screen {
		w.Screen = screen
	}

	screen.Fill(w.Background)

	w.mutex.RLock()
	objects := make([]*Shape, len(w.Objects))
	copy(objects, w.Objects)
	w.mutex.RUnlock()

	w.drawMutex.Lock()
	drawCommands := make([]DrawCommand, len(w.drawCommands))
	copy(drawCommands, w.drawCommands)
	w.drawCommands = w.drawCommands[:0]
	w.drawMutex.Unlock()

	var tempShapes []*Shape
	for _, cmd := range drawCommands {
		tempShape := NewShape(cmd.Props)
		tempShape.Type = cmd.Type
		tempShapes = append(tempShapes, tempShape)
	}

	allShapes := append(objects, tempShapes...)

	sort.Slice(allShapes, func(i, j int) bool {
		if allShapes[i].Tag == "border" && allShapes[j].Tag != "border" {
			return true
		}
		if allShapes[i].Tag != "border" && allShapes[j].Tag == "border" {
			return false
		}
		return allShapes[i].ZIndex < allShapes[j].ZIndex
	})

	for _, obj := range allShapes {
		obj.Draw(screen)
	}
}

func (w *World) LoadSound(name string, fs embed.FS, filePath string) error {
	return w.AudioManager.LoadSoundFromFS(name, fs, filePath)
}

func (w *World) LoadMusic(name string, fs embed.FS, filePath string) error {
	return w.AudioManager.LoadMusicFromFS(name, fs, filePath)
}

func (w *World) PlaySound(name string) error {
	return w.AudioManager.PlaySound(name)
}

func (w *World) PlaySoundWithVolume(name string, volume float64) error {
	return w.AudioManager.PlaySoundWithVolume(name, volume)
}

func (w *World) PlayMusic(name string) error {
	return w.AudioManager.PlayMusic(name)
}

func (w *World) StopMusic() {
	w.AudioManager.StopMusic()
}

func (w *World) PauseMusic() {
	w.AudioManager.PauseMusic()
}

func (w *World) ResumeMusic() {
	w.AudioManager.ResumeMusic()
}

func (w *World) Center(obj *Shape, resetVelocity bool) {
	obj.SetX(float64(w.Width)/2 - obj.Width/2)
	obj.SetY(float64(w.Height)/2 - obj.Height/2)

	if resetVelocity {
		obj.SetVelocity(0, 0)
	}
}

func (w *World) CenterX(obj *Shape, resetVelocity bool) {
	obj.SetX(float64(w.Width)/2 - obj.Width/2)
	if resetVelocity {
		obj.SetVelocity(0, 0)
	}
}

func (w *World) CenterY(obj *Shape, resetVelocity bool) {
	obj.SetY(float64(w.Height)/2 - obj.Height/2)
	if resetVelocity {
		obj.SetVelocity(0, 0)
	}
}

func (w *World) GetAngleBetween(a, b *Shape) float64 {
	return math.Atan2(b.Y-a.Y, b.X-a.X) * 180 / math.Pi
}

func (w *World) HoveredObjects() []*Shape {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	var hovered []*Shape
	for _, obj := range w.Objects {
		if w.Mouse.X >= obj.X && w.Mouse.X <= obj.X+obj.Width &&
			w.Mouse.Y >= obj.Y && w.Mouse.Y <= obj.Y+obj.Height {
			hovered = append(hovered, obj)
		}
	}
	return hovered
}

func (w *World) UnhoveredObjects() []*Shape {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	var unhovered []*Shape
	for _, obj := range w.Objects {
		if !(w.Mouse.X >= obj.X && w.Mouse.X <= obj.X+obj.Width &&
			w.Mouse.Y >= obj.Y && w.Mouse.Y <= obj.Y+obj.Height) {
			unhovered = append(unhovered, obj)
		}
	}
	return unhovered
}

func (w *World) GetAllElements() []*Shape {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	result := make([]*Shape, len(w.Objects))
	copy(result, w.Objects)
	return result
}

func (w *World) GetElementsByTagName(tag string) []*Shape {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	var result []*Shape
	for _, obj := range w.Objects {
		if obj.Tag == tag {
			result = append(result, obj)
		}
	}
	return result
}

func (w *World) GetElementByName(name string) *Shape {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	for _, obj := range w.Objects {
		if obj.Name == name {
			return obj
		}
	}
	return nil
}

func (w *World) GetElementsByName(name string) []*Shape {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	var result []*Shape
	for _, obj := range w.Objects {
		if obj.Name == name {
			result = append(result, obj)
		}
	}
	return result
}

func (w *World) GetElementsByType(shapeType ShapeType) []*Shape {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	var result []*Shape
	for _, obj := range w.Objects {
		if obj.Type == shapeType {
			result = append(result, obj)
		}
	}
	return result
}

func (w *World) Pause() {
	w.Paused = true
}

func (w *World) Resume() {
	w.Paused = false
}

func (w *World) GetCursorPosition() Vector2 {
	return Vector2{X: w.Mouse.X, Y: w.Mouse.Y}
}

func (w *World) IsKeyPressed(key ebiten.Key) bool {
	w.keysMutex.RLock()
	defer w.keysMutex.RUnlock()
	return w.Keys[key]
}

func (w *World) OncePressed(key ebiten.Key, callback func()) {

}
