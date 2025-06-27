package levels

import "boughtnine/life"

func LoadResources() {
	world.LoadSound("jump", assets, "assets/sounds/jump.wav")
	world.LoadSound("level_complete", assets, "assets/sounds/tada.mp3")
	world.LoadSound("ball_hit", assets, "assets/sounds/ballhit.mp3")
	world.LoadMusic("background", assets, "assets/sounds/background.mp3")

	imageBack, err := life.LoadImageFromFS(assets, "assets/background.png")
	if err != nil {
		panic(err)
	}

	background = imageBack

	imageFloor, err := life.LoadImageFromFS(assets, "assets/floor.png")
	if err != nil {
		panic(err)
	}
	floor = imageFloor
}
