package levels

import (
	"boughtnine/entities"
	"boughtnine/life"
	"image/color"
)

var Two life.Level = life.Level{

	MapItems: life.MapItems{
		"#": func(position life.Vector2, width float64, height float64) {
			s := life.NewShape(&life.ShapeProps{
				Name:         "wall",
				Type:         life.ShapeRectangle,
				Pattern:      life.PatternColor,
				Physics:      false,
				IsBody:       false,
				Background:   color.Opaque,
				X:            position.X,
				Y:            position.Y,
				Width:        width,
				Height:       height,
				Friction:     0.5,
				Rebound:      0,
				RotationLock: true,
			})

			world.Register(s)
		},

		"'": func(position life.Vector2, width float64, height float64) {
			s := life.NewShape(&life.ShapeProps{
				Tag:          "ground",
				Type:         life.ShapeRectangle,
				Pattern:      life.PatternColor,
				Physics:      false,
				IsBody:       false,
				Background:   color.Opaque,
				X:            position.X,
				Y:            position.Y,
				Width:        width,
				Height:       height,
				Friction:     0.5,
				Rebound:      0,
				RotationLock: true,
			})

			world.Register(s)
		},

		"F": func(position life.Vector2, width float64, height float64) {
			s := life.NewShape(&life.ShapeProps{
				Type:       life.ShapeRectangle,
				Pattern:    life.PatternColor,
				Background: color.RGBA{R: 255, G: 0, B: 0, A: 255}, // Red color for finish line
				X:          position.X,
				Y:          position.Y,
				Width:      width,
				Height:     height,
				OnCollisionFunc: func(who *life.Shape) {
					if who == player {
						world.NextLevel()
					}
				},
			})

			world.Register(s)
		},

		"@": func(position life.Vector2, width float64, height float64) {
			player.SetX(position.X)
			player.SetY(position.Y)
		},
	},

	Init: func(world_ *life.World) {
		world = world_

		playerEntity = entities.NewPlayerEntity(world, assets)
		player = playerEntity.Shape
	},

	Tick: func(ld life.LoopData) {
		playerEntity.Update(ld)
	},

	Map: life.Map{
		"#############################",
		"#      @                    #",
		"#      '''                  #",
		"#                           #",
		"#  '''                      #",
		"#          '''              #",
		"#                           #",
		"#             FFF           #",
		"'''''''''''''''''''''''''''''",
	},
}
