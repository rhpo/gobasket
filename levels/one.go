package levels

import (
	"boughtnine/entities"
	"boughtnine/life"
	"embed"
	"fmt"
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

	enemy       *life.Shape
	enemyEntity *entities.PlayerEntity

	ball       *life.Shape
	ballEntity *entities.BallEntity
	world      *life.World

	attached bool = true
	pressed  bool = false
	launched bool = false

	background, floor *ebiten.Image

	ld life.LoopData
)

var One life.Level = life.Level{

	Init: func(w *life.World) {
		world = w

		LoadResources()

		playerEntity = entities.NewPlayerEntity(world, assets)
		player = playerEntity.Shape

		ballEntity = entities.NewBallEntity(world, assets)
		ball = ballEntity.Shape

		player.NotCollideWith(ball)

		world.PlayMusic("background")

	},

	Map: life.Map{
		"#############################",
		"#                           #",
		"#                        $  #",
		"#   @                    $PP#",
		"''''''                      #",
		"#                           #",
		"#                           #",
		"#        !                  #",
		"'''''''''''''''''''''''''''''",
	},

	MapItems: life.MapItems{
		"#": func(position life.Vector2, width float64, height float64) {
			s := life.NewShape(&life.ShapeProps{
				Tag:          "wall",
				Type:         life.ShapeRectangle,
				Pattern:      life.PatternImage,
				Physics:      false,
				IsBody:       false,
				Image:        floor,
				X:            position.X,
				Y:            position.Y,
				Width:        height,
				Height:       height,
				Friction:     0.6,
				Rebound:      0.3,
				RotationLock: false,
			})

			world.Register(s)
		},

		"$": func(position life.Vector2, width float64, height float64) {
			s := life.NewShape(&life.ShapeProps{
				Name:         "wall",
				Type:         life.ShapeRectangle,
				Pattern:      life.PatternImage,
				Physics:      false,
				IsBody:       false,
				Image:        floor,
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
				Pattern:      life.PatternImage,
				Physics:      false,
				IsBody:       false,
				Image:        floor,
				X:            position.X,
				Y:            position.Y,
				Width:        height,
				Height:       height,
				Friction:     0.5,
				Rebound:      0,
				RotationLock: true,
			})

			world.Register(s)
		},

		"!": func(position life.Vector2, width float64, height float64) {
			enemyEntity = entities.NewPlayerEntity(world, assets)
			enemy = enemyEntity.Shape

			enemy.SetX(position.X)
			enemy.SetY(position.Y)

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
						world.PlaySound("level_complete")
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

	Tick: func(ld_ life.LoopData) {
		ld = ld_

		playerEntity.Update(ld)
		ballEntity.Update(ld)

		enemy.Follow(player)

		if pressed {
			if player.Flip.X {
				player.Flip.X = false
			}
		}

		if attached && !pressed {
			ball.SetX(player.X + player.Width + PlayerBallGap)
			ball.SetY(player.Y + player.Height/2 - ball.Height/2)

			if player.Flip.X {
				ball.SetX(player.X - ball.Width - PlayerBallGap)
			}
		} else if attached && pressed {

			ball.SetX(player.X + player.Width + PlayerBallGap)
			ball.SetY(player.Y - ball.Height/2)

			if player.Flip.X {
				ball.SetX(player.X - ball.Width - PlayerBallGap)
			}
		}

		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			if !pressed {

				pressed = true
			}
		} else if pressed {
			attached = false
			pressed = false
		}

		toLaunch := !attached && !launched
		if toLaunch {
			x := world.Mouse.X - player.X
			y := world.Mouse.Y - player.Y
			ball.SetVelocity(x*ball.Speed, y*ball.Speed)
			launched = true
		}

		if ebiten.IsKeyPressed(ebiten.KeyE) && !attached {
			// if ball is close, AABB collision
			if ball.X+ball.Width > player.X && ball.X < player.X+player.Width &&
				ball.Y+ball.Height > player.Y && ball.Y < player.Y+player.Height {
				attached = true
				launched = false
				pressed = false

				ball.Body.SetAngularVelocity(0)
			}
		}
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

		life.DrawText(screen, &life.TextProps{
			Text:  fmt.Sprint("FPS: ", int(1/ld.Delta)),
			X:     0,
			Y:     0,
			Color: color.White,
		})

		world.Pen(life.ShapeRectangle, &life.ShapeProps{
			X:       0,
			Y:       0,
			Width:   float64(world.Width),
			Height:  float64(world.Height),
			Pattern: life.PatternImage,
			Image:   background,
			ZIndex:  -1000,
		})

		if ball.X+ball.Width > player.X && ball.X < player.X+player.Width &&
			ball.Y+ball.Height > player.Y && ball.Y < player.Y+player.Height &&
			!attached {
			life.DrawText(screen, &life.TextProps{
				Text:  "Press E to pick up the ball",
				X:     ball.X - float64(len("Press E to pick up the ball"))/2,
				Y:     ball.Y - 5,
				Color: color.White,
			})
		}

		if pressed && !launched {
			world.Line(ball.X+ball.Width/2, ball.Y+ball.Height/2, world.Mouse.X, world.Mouse.Y, color.RGBA{R: 255}, 1.0)
		}
	},

	OnDestroy: func(world *life.World) {
		world.StopMusic()
	},
}
