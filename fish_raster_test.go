package main

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"mygo-desktop-fish/tank"
)

// ui.Tester renders entirely in memory using MyGo's software renderer and the
// host text engine. This compares actual raster output after scale changes,
// not just Find()'s geometry; it still is not native-window/owner acceptance.
func TestFishHeadlessRasterResizeAndDPIMatchFresh(t *testing.T) {
	a := newApp()
	a.tank.Foods, a.tank.Bubbles, a.tank.Chomps, a.tank.Plants = nil, nil, nil, nil
	a.tank.Fishes = a.tank.Fishes[:2]
	f := &a.tank.Fishes[0]
	f.Size, f.Swell, f.State, f.Facing = tank.MaxSize, tank.SwellPeak, tank.LifeBirthSwell, -1
	f.X, f.Y = a.tank.Panel.ClampInside(1000, 1000, f.W(), f.H())
	baby := &a.tank.Fishes[1]
	baby.Size, baby.Swell, baby.State, baby.Facing = .53, 1, tank.LifeSwim, 1
	baby.X, baby.Y = 220, 80
	// Each Tester starts at a different real instant. Freeze simulation
	// explicitly: otherwise constructing "fresh" advances the shared tank,
	// and a legitimate animation difference masquerades as a cache failure.
	view := func(c *ui.Context) { a.last = c.Now(); a.view(c) }
	w, h := WindowSize(a.settings.Scale)
	tt := ui.NewTester(view, w, h)
	for _, dpi := range []float32{1, 1.25, 2, 3} {
		tt.SetScale(dpi)
		for _, scale := range []float64{1.25, .75, .35, .55, .35, .75, 1.25} {
			a.settings.Scale = scale
			w, h := WindowSize(scale)
			tt.SetSize(w, h)
			tt.Frame()
			got := tt.Image()
			fresh := ui.NewTester(view, w, h)
			fresh.SetScale(dpi)
			want := fresh.Image()
			// Compare all interior pixels (including both complete fish), not
			// chassis edge antialiasing: independent software renderers can
			// differ by a few channel values at a fractional vector corner.
			p := a.tank.Panel
			region := image.Rect(int(math.Ceil(dipX(p.IX, scale)*float64(dpi))), int(math.Ceil(dipY(p.IY, scale)*float64(dpi))), int(math.Floor(dipX(p.IX+p.IW, scale)*float64(dpi))), int(math.Floor(dipY(p.FloorY(), scale)*float64(dpi))))
			equal := got.Rect == want.Rect && region.In(got.Rect) && region.In(want.Rect)
			if equal {
				for y := region.Min.Y; y < region.Max.Y && equal; y++ {
					for x := region.Min.X; x < region.Max.X; x++ {
						if got.RGBAAt(x, y) != want.RGBAAt(x, y) {
							equal = false
							break
						}
					}
				}
			}
			if !equal {
				if dir := os.Getenv("MYFISH_HEADLESS_EVIDENCE_DIR"); dir != "" {
					for name, img := range map[string]image.Image{"mismatch-reused": got, "mismatch-fresh": want} {
						out, err := os.Create(filepath.Join(dir, name+".png"))
						if err != nil {
							t.Fatal(err)
						}
						err = png.Encode(out, img)
						out.Close()
						if err != nil {
							t.Fatal(err)
						}
					}
				}
				t.Fatalf("raster mismatch after live resize/DPI: scale=%v dpi=%v reused=%v fresh=%v", scale, dpi, got.Rect, want.Rect)
			}
			if dir := os.Getenv("MYFISH_HEADLESS_EVIDENCE_DIR"); dir != "" {
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				name := filepath.Join(dir, fmt.Sprintf("linux-headless-scale-%.2f-dpi-%.2f.png", scale, dpi))
				out, err := os.Create(name)
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(out, got)
				closeErr := out.Close()
				if err != nil {
					t.Fatal(err)
				}
				if closeErr != nil {
					t.Fatal(closeErr)
				}
			}
		}
	}
}
