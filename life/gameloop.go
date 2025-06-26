package life

import "time"

// "sync"
// "time"

type GameLoop = func(ld LoopData)

// // GameLoop manages the game loop with FPS control
//
//	type GameLoop struct {
//		mainFunc  func(LoopData)
//		fps       int
//		isPlaying bool
//		ticker    *time.Ticker
//		stopCh    chan bool
//		mutex     sync.RWMutex
//	}
//
// LoopData contains information passed to the main loop function
type LoopData struct {
	Time  time.Time
	Frame int64
	Delta float64
}

//
// // NewGameLoop creates a new game loop
// func NewGameLoop(mainFunc func(LoopData), fps int) *GameLoop {
// 	if fps <= 0 {
// 		fps = 60
// 	}
//
// 	return &GameLoop{
// 		mainFunc:  mainFunc,
// 		fps:       fps,
// 		isPlaying: false,
// 		stopCh:    make(chan bool),
// 	}
// }
//
// // Start begins the game loop
// func (gl *GameLoop) Start() {
// 	gl.mutex.Lock()
// 	defer gl.mutex.Unlock()
//
// 	if gl.isPlaying {
// 		return
// 	}
//
// 	gl.isPlaying = true
// 	gl.ticker = time.NewTicker(time.Duration(1000/gl.fps) * time.Millisecond)
//
// 	go func() {
// 		frame := int64(0)
// 		startTime := time.Now()
// 		lastTime := startTime
//
// 		for {
// 			select {
// 			case <-gl.ticker.C:
// 				currentTime := time.Now()
// 				delta := currentTime.Sub(lastTime).Seconds()
//
// 				gl.mainFunc(LoopData{
// 					Time:  currentTime,
// 					Frame: frame,
// 					Delta: delta,
// 				})
//
// 				frame++
// 				lastTime = currentTime
//
// 			case <-gl.stopCh:
// 				return
// 			}
// 		}
// 	}()
// }
//
// // Pause stops the game loop
// func (gl *GameLoop) Pause() {
// 	gl.mutex.Lock()
// 	defer gl.mutex.Unlock()
//
// 	if !gl.isPlaying {
// 		return
// 	}
//
// 	gl.isPlaying = false
// 	if gl.ticker != nil {
// 		gl.ticker.Stop()
// 	}
// 	gl.stopCh <- true
// }
//
// // SetFPS changes the FPS of the game loop
// func (gl *GameLoop) SetFPS(fps int) {
// 	gl.mutex.Lock()
// 	defer gl.mutex.Unlock()
//
// 	if fps <= 0 {
// 		fps = 60
// 	}
//
// 	gl.fps = fps
//
// 	if gl.isPlaying {
// 		gl.Pause()
// 		gl.Start()
// 	}
// }
//
// // IsPlaying returns whether the game loop is currently running
// func (gl *GameLoop) IsPlaying() bool {
// 	gl.mutex.RLock()
// 	defer gl.mutex.RUnlock()
// 	return gl.isPlaying
// }
