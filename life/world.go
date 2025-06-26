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
	// No-op, required by Box2D interface (No-op = no-operation)
}

func (cl *ContactListener) PostSolve(contact box2d.B2ContactInterface, impulse *box2d.B2ContactImpulse) {
	// No-op, required by Box2D interface
}

func (cl *ContactListener) BeginContact(contact box2d.B2ContactInterface) {
	// NO MUTEX LOCKING HERE - just queue the collision for processing
	fixtureA := contact.GetFixtureA()
	fixtureB := contact.GetFixtureB()

	if fixtureA == nil || fixtureB == nil {
		return
	}

	bodyA := fixtureA.GetBody()
	bodyB := fixtureB.GetBody()

	// Find shapes without locking
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

	// Queue collision for safe processing later
	cl.world.queueCollision(shapeA, shapeB)
}

func (cl *ContactListener) EndContact(contact box2d.B2ContactInterface) {
	// Similar approach for end contact - no mutex locking
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

// Collision event for queuing
type CollisionEvent struct {
	ShapeA *Shape
	ShapeB *Shape
}

// World represents the game world
type World struct {
	*EventEmitter

	// Display properties
	Width  int
	Height int

	// Physics
	PhysicsWorld    *box2d.B2World
	contactListener *ContactListener
	G               Vector2 // Gravity
	AirResistance   float64 // Air resistance for physics bodies

	Screen *ebiten.Image // Screen to draw on

	Tick       GameLoop
	Init       func()
	Render     func(screen *ebiten.Image)
	Title      string
	lastUpdate time.Time

	// Visual properties
	Pattern    PatternType
	Background color.Color
	Border     *Border

	// Game objects
	Objects []*Shape
	mutex   sync.RWMutex

	// Audio
	AudioManager *AudioManager

	// Input
	Mouse struct {
		X, Y                          float64
		IsLeftClicked, IsRightClicked bool
		IsMiddleClicked               bool
	}
	Keys      map[ebiten.Key]bool
	keysMutex sync.RWMutex

	// State
	HasLimits bool
	Paused    bool
	Cursor    CursorType

	// Callbacks
	OnMouseDown func(x, y float64)
	OnMouseUp   func(x, y float64)
	OnMouseMove func(x, y float64)

	Levels       []Level
	CurrentLevel int

	// Deferred operations
	pendingLevelSwitch *int
	collisionQueue     []CollisionEvent
	collisionMutex     sync.Mutex
}

// WorldProps contains properties for creating a world
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

// NewWorld creates a new world
func NewWorld(props *WorldProps) *World {
	if props == nil {
		props = &WorldProps{}
	}

	// Set defaults
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

	// Create Box2D world
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

// queueCollision adds a collision to the processing queue
func (w *World) queueCollision(shapeA, shapeB *Shape) {
	w.collisionMutex.Lock()
	defer w.collisionMutex.Unlock()

	w.collisionQueue = append(w.collisionQueue, CollisionEvent{
		ShapeA: shapeA,
		ShapeB: shapeB,
	})
}

// processCollisions handles all queued collisions safely
func (w *World) processCollisions() {
	w.collisionMutex.Lock()
	collisions := make([]CollisionEvent, len(w.collisionQueue))
	copy(collisions, w.collisionQueue)
	w.collisionQueue = w.collisionQueue[:0] // Clear the queue
	w.collisionMutex.Unlock()

	// Process collisions without any mutex locks
	for _, collision := range collisions {
		shapeA := collision.ShapeA
		shapeB := collision.ShapeB

		// Emit collision event
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

	// Cleanup audio
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

	// Schedule level switch for after physics step
	w.pendingLevelSwitch = &levelIndex
}

func (w *World) SelectLevel(index int) {
	if index < 0 || index >= len(w.Levels) {
		return
	}

	// Clear all existing objects
	w.mutex.Lock()

	for _, obj := range w.Objects {
		if obj.Body != nil {
			w.PhysicsWorld.DestroyBody(obj.Body)
			obj.Body = nil
		}
	}
	w.Objects = make([]*Shape, 0)
	w.mutex.Unlock()

	// Set current level
	w.CurrentLevel = index
	level := w.Levels[index]

	// Set up level functions
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

	// Initialize the level
	if level.Init != nil {
		level.Init(w)
	}

	// Generate level from map
	w.GenerateLevelFromMap(level.Map, level.MapItems)

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

	if !object.IsBody || object.Tag == "border" {
		bodyDef.Type = box2d.B2BodyType.B2_staticBody
	} else {
		bodyDef.Type = box2d.B2BodyType.B2_dynamicBody
	}

	if !object.Physics {
		bodyDef.GravityScale = 0
	}

	bodyDef.FixedRotation = object.RotationLock

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
		density = 1.0
	}

	fixture := body.CreateFixture(shape, density)

	if object.Ghost {
		fixture.SetSensor(true)
	}

	fixture.SetFriction(object.Friction)
	fixture.SetRestitution(object.Rebound)

	object.Body = body
}

func (w *World) GenerateLevelFromMap(levelMap Map, objects map[string]func(position Vector2, width, height float64)) {
	if len(levelMap) == 0 {
		return
	}

	rows := len(levelMap)
	cols := len(levelMap[0])
	tileWidth := float64(w.Width / cols)
	tileHeight := float64(w.Height / rows)

	for y, row := range levelMap {
		for x, ch := range row {
			fn, ok := objects[string(ch)]
			if !ok {
				continue
			}
			pos := Vector2{
				X: float64(x) * tileWidth,
				Y: float64(y) * tileHeight,
			}
			fn(pos, tileWidth, tileHeight)
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

	// Step physics simulation
	velocityIterations := 6
	positionIterations := 3
	w.PhysicsWorld.Step(deltaTime, velocityIterations, positionIterations)

	// Update audio system
	if w.AudioManager != nil {
		w.AudioManager.Update()
	}

	// Process queued collisions AFTER physics step
	w.processCollisions()

	// Handle pending level switch AFTER collision processing
	if w.pendingLevelSwitch != nil {
		levelIndex := *w.pendingLevelSwitch
		w.pendingLevelSwitch = nil
		w.SelectLevel(levelIndex)
		return nil // Return early after level switch
	}

	// Update objects
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
			obj.Emit(EventMouseDown, map[string]float64{"x": x, "y": y})
		}
	}

	if w.OnMouseDown != nil {
		w.OnMouseDown(x, y)
	}
}

func (w *World) handleMouseUp(x, y float64) {
	hoveredObjects := w.HoveredObjects()
	for _, obj := range hoveredObjects {
		obj.Emit(EventMouseUp, map[string]float64{"x": x, "y": y})
		obj.Emit(EventClick, map[string]float64{"x": x, "y": y})
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

	sort.Slice(objects, func(i, j int) bool {
		if objects[i].Tag == "border" && objects[j].Tag != "border" {
			return true
		}
		if objects[i].Tag != "border" && objects[j].Tag == "border" {
			return false
		}
		return objects[i].ZIndex < objects[j].ZIndex
	})

	for _, obj := range objects {
		obj.Draw(screen)
	}
}

// Audio convenience methods for World
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

// Utility methods
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
	// Placeholder implementation
}
