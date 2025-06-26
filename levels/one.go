package levels

import (
	"boughtnine/entities"
	"boughtnine/life"
	"embed"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed assets/*
var assets embed.FS

const (
	PlayerBallGap = 0
)

var (
	player       *life.Shape
	playerEntity *entities.PlayerEntity
	ball         *life.Shape
	ballEntity   *entities.BallEntity
	world        *life.World
)

var One life.Level = life.Level{

	Map: life.Map{
		"#############################",
		"#                           #",
		"#                        $  #",
		"#   @                    $PP#",
		"''''''                      #",
		"#                           #",
		"#                           #",
		"#                           #",
		"'''''''''''''''''''''''''''''",
	},

	MapItems: life.MapItems{
		"#": func(position life.Vector2, width float64, height float64) {
			s := life.NewShape(&life.ShapeProps{
				Tag:          "wall",
				Type:         life.ShapeRectangle,
				Pattern:      life.PatternColor,
				Physics:      false,
				IsBody:       false,
				Background:   color.Black,
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

		"$": func(position life.Vector2, width float64, height float64) {
			s := life.NewShape(&life.ShapeProps{
				Name:         "wall",
				Type:         life.ShapeRectangle,
				Pattern:      life.PatternColor,
				Physics:      false,
				IsBody:       false,
				Background:   color.RGBA{R: 255, G: 165, B: 0, A: 255},
				X:            position.X,
				Y:            position.Y,
				Width:        width,
				Height:       height,
				Friction:     0,
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
				Background:   color.Black,
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

		"P": func(position life.Vector2, width float64, height float64) {
			s := life.NewShape(&life.ShapeProps{
				Type:         life.ShapeRectangle,
				Pattern:      life.PatternColor,
				Background:   color.RGBA{R: 255, G: 165, B: 0, A: 255},
				X:            position.X,
				Y:            position.Y,
				Width:        width,
				Height:       height,
				Physics:      false,
				IsBody:       false,
				Friction:     0,
				Mass:         100,
				Rebound:      0.5,
				RotationLock: false,

				OnCollisionFunc: func(who *life.Shape) {
					if who == ball {
						panic("You won!")
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

		// Load audio files
		world.LoadSound("jump", assets, "assets/sounds/jump.wav")
		world.LoadSound("level_complete", assets, "assets/sounds/complete.wav")
		world.LoadMusic("background", assets, "assets/sounds/background.mp3")

		playerEntity = entities.NewPlayerEntity(world, assets)
		player = playerEntity.Shape

		ballEntity = entities.NewBallEntity(world, assets)
		ball = ballEntity.Shape

		// This should now work correctly - only disable collision between player and ball
		// player.NotCollideWith(ball)

		// Play background music
		// world.PlayMusic("background")

		world.OnMouseDown = func(x, y float64) {
			ball.SetX(x)
			ball.SetY(y)
		}
	},

	Tick: func(ld life.LoopData) {
		playerEntity.Update(ld)
		ballEntity.Update(ld)
	},

	OnMount: func() {
		ballPosX := player.X + player.Width + PlayerBallGap
		ballPosY := player.Y + player.Height/2 - ball.Height/2

		if player.Flip.X {
			ballPosX = player.X - ball.Width - PlayerBallGap
		}

		ball.SetX(ballPosX)
		ball.SetY(float64(ballPosY))

		world.On(life.EventClick, func(data any) {
			pos := data.(life.Vector2)

			ball.SetX(pos.X)
			ball.SetY(pos.Y)
		})

	},

	Render: func(screen *ebiten.Image) {
		cycle := 1.0
		show := 0.5
		if float64(time.Now().UnixNano()%int64(cycle*1e9))/1e9 < show {
			life.DrawText(screen, &life.TextProps{
				Text:  player.Name,
				X:     player.X - float64(len(player.Name))*1.48/2,
				Y:     player.Y - 5,
				Color: color.White,
			})
		}
	},

	OnDestroy: func(world *life.World) {
		world.StopMusic()
	},
}
