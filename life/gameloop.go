package life

import "time"

type GameLoop = func(ld LoopData)

type LoopData struct {
	Time  time.Time
	Frame int64
	Delta float64
}
