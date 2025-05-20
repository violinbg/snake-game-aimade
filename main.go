package main

import (
	"bytes"
	_ "embed"
	"image/color"
	"image/png"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

//go:embed head.png
var headPng []byte

//go:embed body.png
var bodyPng []byte

//go:embed apple.png
var applePng []byte

const (
	screenWidth  = 320
	screenHeight = 240
	gridSize     = 24 // tile size in pixels
	initialLen   = 3
)

type Point struct {
	X, Y int
}

type Food struct {
	Pos      Point
	Spawned  time.Time
	Lifetime time.Duration
}

type Game struct {
	snake        []Point
	dir          Point
	foods        []Food
	grow         bool
	gameOver     bool
	lastMove     time.Time
	score        int
	lives        int
	nextFoodTime time.Time
	// Images
	headImg *ebiten.Image
	bodyImg *ebiten.Image
	foodImg *ebiten.Image
}

func (g *Game) Update() error {
	if g.gameOver {
		if ebiten.IsKeyPressed(ebiten.KeySpace) {
			g.lives = 3
			g.score = 0
			g.init()
		}
		return nil
	}

	// Direction input
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) && g.dir.Y != 1 {
		g.dir = Point{0, -1}
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowDown) && g.dir.Y != -1 {
		g.dir = Point{0, 1}
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) && g.dir.X != 1 {
		g.dir = Point{-1, 0}
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowRight) && g.dir.X != -1 {
		g.dir = Point{1, 0}
	}

	// Move snake every 100ms
	if time.Since(g.lastMove) < 100*time.Millisecond {
		return nil
	}
	g.lastMove = time.Now()

	head := g.snake[0]
	newHead := Point{head.X + g.dir.X, head.Y + g.dir.Y}

	// Check collision with walls
	if newHead.X < 0 || newHead.Y < 0 || newHead.X >= screenWidth/gridSize || newHead.Y >= screenHeight/gridSize {
		g.lives--
		if g.lives > 0 {
			g.init()
		} else {
			g.gameOver = true
		}
		return nil
	}
	// Check collision with self
	for _, s := range g.snake {
		if s == newHead {
			g.lives--
			if g.lives > 0 {
				g.init()
			} else {
				g.gameOver = true
			}
			return nil
		}
	}

	g.snake = append([]Point{newHead}, g.snake...)

	// Check collision with any food
	ateFood := -1
	for i, food := range g.foods {
		if newHead == food.Pos {
			ateFood = i
			break
		}
	}
	if ateFood != -1 {
		g.grow = true
		g.score += 10
		// Remove the eaten food
		g.foods = append(g.foods[:ateFood], g.foods[ateFood+1:]...)
	}
	if !g.grow {
		g.snake = g.snake[:len(g.snake)-1]
	} else {
		g.grow = false
	}

	// Remove expired food
	now := time.Now()
	filtered := g.foods[:0]
	for _, food := range g.foods {
		if now.Sub(food.Spawned) < food.Lifetime {
			filtered = append(filtered, food)
		}
	}
	g.foods = filtered

	// Food spawn logic: spawn one at a time with delay, but if less than 4, spawn immediately
	if len(g.foods) < 4 {
		if g.nextFoodTime.IsZero() || now.After(g.nextFoodTime) {
			g.spawnFood()
			if len(g.foods) < 4 {
				g.nextFoodTime = now // spawn next immediately
			} else {
				g.nextFoodTime = now.Add(1 * time.Second) // delay for next spawn
			}
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 0xff})
	// Draw snake
	for i, s := range g.snake {
		op := &ebiten.DrawImageOptions{}
		// For the head, rotate based on direction
		if i == 0 {
			// Default: head.png faces down (0,1)
			// To rotate around the center, translate to center, rotate, then translate back
			cx := float64(gridSize) / 2
			cy := float64(gridSize) / 2
			op.GeoM.Translate(-cx, -cy)
			switch g.dir {
			case Point{0, -1}: // Up
				op.GeoM.Rotate(-3.14159265) // 180 deg
			case Point{-1, 0}: // Left
				op.GeoM.Rotate(1.57079633) // +90 deg (was -90)
			case Point{1, 0}: // Right
				op.GeoM.Rotate(-1.57079633) // -90 deg (was +90)
			}
			op.GeoM.Translate(cx+float64(s.X*gridSize), cy+float64(s.Y*gridSize))
			if g.headImg != nil {
				screen.DrawImage(g.headImg, op)
			}
		} else {
			// Determine direction for this body segment
			var angle float64 = 0
			cx := float64(gridSize) / 2
			cy := float64(gridSize) / 2
			op.GeoM.Translate(-cx, -cy)
			if i < len(g.snake)-1 {
				prev := g.snake[i-1]
				curr := g.snake[i]
				// next := g.snake[i+1] // not used
				// Prefer direction from previous to current (head to tail)
				dx := prev.X - curr.X
				dy := prev.Y - curr.Y
				switch {
				case dx == 0 && dy == 1: // Down
					angle = 0
				case dx == 0 && dy == -1: // Up
					angle = -3.14159265
				case dx == 1 && dy == 0: // Right
					angle = -1.57079633
				case dx == -1 && dy == 0: // Left
					angle = 1.57079633
				}
			} else if i > 0 { // tail: use previous segment direction
				prev := g.snake[i-1]
				curr := g.snake[i]
				dx := prev.X - curr.X
				dy := prev.Y - curr.Y
				switch {
				case dx == 0 && dy == 1: // Down
					angle = 0
				case dx == 0 && dy == -1: // Up
					angle = -3.14159265
				case dx == 1 && dy == 0: // Right
					angle = -1.57079633
				case dx == -1 && dy == 0: // Left
					angle = 1.57079633
				}
			}
			op.GeoM.Rotate(angle)
			op.GeoM.Translate(cx+float64(s.X*gridSize), cy+float64(s.Y*gridSize))
			if g.bodyImg != nil {
				screen.DrawImage(g.bodyImg, op)
			}
		}
	}
	// Draw food items
	now := time.Now()
	for _, food := range g.foods {
		elapsed := now.Sub(food.Spawned)
		// Flashing effect in last 1s
		visible := true
		if food.Lifetime-elapsed < time.Second {
			// Flash every 100ms
			visible = (elapsed.Milliseconds()/100)%2 == 0
		}
		if visible && g.foodImg != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(food.Pos.X*gridSize), float64(food.Pos.Y*gridSize))
			screen.DrawImage(g.foodImg, op)
		}
	}
	// Draw score and lives
	ebitenutil.DebugPrintAt(screen, "Score: "+itoa(g.score)+"  Lives: "+itoa(g.lives), 4, 4)
	// Game over message
	if g.gameOver {
		ebitenutil.DebugPrintAt(screen, "Game Over! Press Space to restart.", 60, screenHeight/2-8)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func (g *Game) init() {
	g.snake = make([]Point, initialLen)
	for i := 0; i < initialLen; i++ {
		g.snake[i] = Point{X: screenWidth/gridSize/2 - i, Y: screenHeight / gridSize / 2}
	}
	g.dir = Point{1, 0}
	g.grow = false
	g.gameOver = false
	g.foods = nil
	g.lastMove = time.Now()
	g.nextFoodTime = time.Now()
}

func (g *Game) spawnFood() {
	// Place food at a random position not occupied by the snake or other food
	for {
		fx := rand.Intn(screenWidth / gridSize)
		fy := rand.Intn(screenHeight / gridSize)
		pos := Point{fx, fy}
		overlap := false
		for _, s := range g.snake {
			if s == pos {
				overlap = true
				break
			}
		}
		for _, f := range g.foods {
			if f.Pos == pos {
				overlap = true
				break
			}
		}
		if !overlap {
			g.foods = append(g.foods, Food{
				Pos:      pos,
				Spawned:  time.Now(),
				Lifetime: 4 * time.Second,
			})
			return
		}
	}
}

// Helper function for int to string (no strconv needed for this simple case)
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var b [20]byte
	bp := len(b)
	for i > 0 {
		bp--
		b[bp] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		bp--
		b[bp] = '-'
	}
	return string(b[bp:])
}

func loadImageFromBytes(data []byte, name string) *ebiten.Image {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		log.Fatalf("failed to decode %s: %v", name, err)
	}
	eimg := ebiten.NewImageFromImage(img)
	return eimg
}

func main() {
	game := &Game{}
	game.lives = 3
	game.score = 0
	// Load images from embedded data
	game.headImg = loadImageFromBytes(headPng, "head.png")
	game.bodyImg = loadImageFromBytes(bodyPng, "body.png")
	game.foodImg = loadImageFromBytes(applePng, "apple.png")
	game.init()
	ebiten.SetWindowTitle("Snake Game")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
