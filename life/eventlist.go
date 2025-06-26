package life

// Event types
type EventType string

const (
	EventMouseDown       EventType = "mousedown"
	EventMouseUp         EventType = "mouseup"
	EventMouseMove       EventType = "mousemove"
	EventMouseEnter      EventType = "mouseenter"
	EventMouseLeave      EventType = "mouseleave"
	EventHover           EventType = "hover"
	EventUnHover         EventType = "unhover"
	EventClick           EventType = "click"
	EventCollision       EventType = "collision"
	EventDirectionChange EventType = "event-direction-change"
)

type EventDirectionChangeData struct {
	Direction *Axis
}

type EventMouseEnterData struct {
	Shape *Shape
}

type EventCollisionData struct {
	ShapeA *Shape
	ShapeB *Shape
}
