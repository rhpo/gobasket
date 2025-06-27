package life

import "math"

type Vector2 struct {
	X, Y float64
}

func NewVector2(x, y float64) Vector2 {
	return Vector2{X: x, Y: y}
}

func (v Vector2) Add(other Vector2) Vector2 {
	return Vector2{X: v.X + other.X, Y: v.Y + other.Y}
}

func (v Vector2) Sub(other Vector2) Vector2 {
	return Vector2{X: v.X - other.X, Y: v.Y - other.Y}
}

func (v Vector2) Mul(scalar float64) Vector2 {
	return Vector2{X: v.X * scalar, Y: v.Y * scalar}
}

func (v Vector2) Length() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y)
}

func (v Vector2) Normalize() Vector2 {
	length := v.Length()
	if length == 0 {
		return Vector2{0, 0}
	}
	return Vector2{X: v.X / length, Y: v.Y / length}
}

func (v Vector2) Angle(other Vector2) float64 {
	return math.Atan2(other.Y-v.Y, other.X-v.X) * 180 / math.Pi
}
