package life

import "github.com/hajimehoshi/ebiten/v2"

type Map []string
type MapItems map[string]func(position Vector2, width float64, height float64)

type Level struct {
	Map      Map
	MapItems MapItems

	Init      func(world *World)
	Tick      func(ld LoopData)
	Render    func(screen *ebiten.Image)
	OnMount   func()
	OnDestroy func(world *World)
}
