package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"pelotongrandtour/internal/game"
)

func main() {
	ebiten.SetWindowSize(game.VirtualWidth, game.VirtualHeight)
	ebiten.SetWindowTitle("Peloton Grand Tour: Maillot Jaune")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(60)

	g, err := game.New()
	if err != nil {
		log.Fatalf("failed to initialize Peloton Grand Tour: %v", err)
	}
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
