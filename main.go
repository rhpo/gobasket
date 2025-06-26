package main

import (
	"boughtnine/entities"
	"boughtnine/levels"
	"boughtnine/life"
)

func main() {
	world := entities.NewWorld()
	game := life.NewGame(world)

	world.Levels = []life.Level{
		levels.One,
		levels.Two,
	}

	game.Run()
}
