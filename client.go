// Copyright 2017 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"

	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"

	pikachuPicture "github.com/pikachu0310/kiichigojam_2023-04-22/pictures"
)

const (
	lineHeight   = 16
	padding      = 20
	screenWidth  = 1105 // 640 * 2
	screenHeight = 830  // 480 * 1.5
)

var (
	uiImage       *ebiten.Image
	uiFont        font.Face
	uiFontMHeight int
	bgImage       *ebiten.Image
	// players      = make(map[int]*Player)
	playersMutex = sync.Mutex{}
)

func init() {
	// Decode an image from the image file's byte slice.
	img, _, err := image.Decode(bytes.NewReader(images.UI_png))
	if err != nil {
		log.Fatal(err)
	}
	uiImage = ebiten.NewImageFromImage(img)

	tt, err := opentype.Parse(goregular.TTF)
	if err != nil {
		log.Fatal(err)
	}
	uiFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    12,
		DPI:     72,
		Hinting: font.HintingVertical,
	})
	if err != nil {
		log.Fatal(err)
	}
	b, _, _ := uiFont.GlyphBounds('M')
	uiFontMHeight = (b.Max.Y - b.Min.Y).Ceil()

	img, _, err = image.Decode(bytes.NewReader(pikachuPicture.HoshizoraPng))
	if err != nil {
		log.Fatal(err)
	}
	bgImage = ebiten.NewImageFromImage(img)
}

type imageType int

const (
	imageTypeButton imageType = iota
	imageTypeButtonPressed
	imageTypeTextBox
	imageTypeVScrollBarBack
	imageTypeVScrollBarFront
	imageTypeCheckBox
	imageTypeCheckBoxPressed
	imageTypeCheckBoxMark
)

var imageSrcRects = map[imageType]image.Rectangle{
	imageTypeButton:          image.Rect(0, 0, 16, 16),
	imageTypeButtonPressed:   image.Rect(16, 0, 32, 16),
	imageTypeTextBox:         image.Rect(0, 16, 16, 32),
	imageTypeVScrollBarBack:  image.Rect(16, 16, 24, 32),
	imageTypeVScrollBarFront: image.Rect(24, 16, 32, 32),
	imageTypeCheckBox:        image.Rect(0, 32, 16, 48),
	imageTypeCheckBoxPressed: image.Rect(16, 32, 32, 48),
	imageTypeCheckBoxMark:    image.Rect(32, 32, 48, 48),
}

type Input struct {
	mouseButtonState int
}

func drawNinePatches(dst *ebiten.Image, dstRect image.Rectangle, srcRect image.Rectangle) {
	srcX := srcRect.Min.X
	srcY := srcRect.Min.Y
	srcW := srcRect.Dx()
	srcH := srcRect.Dy()

	dstX := dstRect.Min.X
	dstY := dstRect.Min.Y
	dstW := dstRect.Dx()
	dstH := dstRect.Dy()

	op := &ebiten.DrawImageOptions{}
	for j := 0; j < 3; j++ {
		for i := 0; i < 3; i++ {
			op.GeoM.Reset()

			sx := srcX
			sy := srcY
			sw := srcW / 4
			sh := srcH / 4
			dx := 0
			dy := 0
			dw := sw
			dh := sh
			switch i {
			case 1:
				sx = srcX + srcW/4
				sw = srcW / 2
				dx = srcW / 4
				dw = dstW - 2*srcW/4
			case 2:
				sx = srcX + 3*srcW/4
				dx = dstW - srcW/4
			}
			switch j {
			case 1:
				sy = srcY + srcH/4
				sh = srcH / 2
				dy = srcH / 4
				dh = dstH - 2*srcH/4
			case 2:
				sy = srcY + 3*srcH/4
				dy = dstH - srcH/4
			}

			op.GeoM.Scale(float64(dw)/float64(sw), float64(dh)/float64(sh))
			op.GeoM.Translate(float64(dx), float64(dy))
			op.GeoM.Translate(float64(dstX), float64(dstY))
			dst.DrawImage(uiImage.SubImage(image.Rect(sx, sy, sx+sw, sy+sh)).(*ebiten.Image), op)
		}
	}
}

type Button struct {
	Rect image.Rectangle
	Text string

	mouseDown bool

	onPressed func(b *Button)
}

func (b *Button) Update() {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if b.Rect.Min.X <= x && x < b.Rect.Max.X && b.Rect.Min.Y <= y && y < b.Rect.Max.Y {
			b.mouseDown = true
		} else {
			b.mouseDown = false
		}
	} else {
		if b.mouseDown {
			if b.onPressed != nil {
				b.onPressed(b)
			}
		}
		b.mouseDown = false
	}
}

func (b *Button) Draw(dst *ebiten.Image) {
	t := imageTypeButton
	if b.mouseDown {
		t = imageTypeButtonPressed
	}
	drawNinePatches(dst, b.Rect, imageSrcRects[t])

	bounds, _ := font.BoundString(uiFont, b.Text)
	w := (bounds.Max.X - bounds.Min.X).Ceil()
	x := b.Rect.Min.X + (b.Rect.Dx()-w)/2
	y := b.Rect.Max.Y - (b.Rect.Dy()-uiFontMHeight)/2
	text.Draw(dst, b.Text, uiFont, x, y, color.Black)
}

func (b *Button) SetOnPressed(f func(b *Button)) {
	b.onPressed = f
}

const VScrollBarWidth = 16

type VScrollBar struct {
	X      int
	Y      int
	Height int

	thumbRate           float64
	thumbOffset         int
	dragging            bool
	draggingStartOffset int
	draggingStartY      int
	contentOffset       int
}

func (v *VScrollBar) thumbSize() int {
	const minThumbSize = VScrollBarWidth

	r := v.thumbRate
	if r > 1 {
		r = 1
	}
	s := int(float64(v.Height) * r)
	if s < minThumbSize {
		return minThumbSize
	}
	return s
}

func (v *VScrollBar) thumbRect() image.Rectangle {
	if v.thumbRate >= 1 {
		return image.Rectangle{}
	}

	s := v.thumbSize()
	return image.Rect(v.X, v.Y+v.thumbOffset, v.X+VScrollBarWidth, v.Y+v.thumbOffset+s)
}

func (v *VScrollBar) maxThumbOffset() int {
	return v.Height - v.thumbSize()
}

func (v *VScrollBar) ContentOffset() int {
	return v.contentOffset
}

func (v *VScrollBar) Update(contentHeight int) {
	v.thumbRate = float64(v.Height) / float64(contentHeight)

	if !v.dragging && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		tr := v.thumbRect()
		if tr.Min.X <= x && x < tr.Max.X && tr.Min.Y <= y && y < tr.Max.Y {
			v.dragging = true
			v.draggingStartOffset = v.thumbOffset
			v.draggingStartY = y
		}
	}
	if v.dragging {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			_, y := ebiten.CursorPosition()
			v.thumbOffset = v.draggingStartOffset + (y - v.draggingStartY)
			if v.thumbOffset < 0 {
				v.thumbOffset = 0
			}
			if v.thumbOffset > v.maxThumbOffset() {
				v.thumbOffset = v.maxThumbOffset()
			}
		} else {
			v.dragging = false
		}
	}

	v.contentOffset = 0
	if v.thumbRate < 1 {
		v.contentOffset = int(float64(contentHeight) * float64(v.thumbOffset) / float64(v.Height))
	}
}

func (v *VScrollBar) Draw(dst *ebiten.Image) {
	sd := image.Rect(v.X, v.Y, v.X+VScrollBarWidth, v.Y+v.Height)
	drawNinePatches(dst, sd, imageSrcRects[imageTypeVScrollBarBack])

	if v.thumbRate < 1 {
		drawNinePatches(dst, v.thumbRect(), imageSrcRects[imageTypeVScrollBarFront])
	}
}

const (
	textBoxPaddingLeft = 8
)

type TextBox struct {
	Rect image.Rectangle
	Text string

	contentBuf *ebiten.Image
	vScrollBar *VScrollBar
	offsetX    int
	offsetY    int
}

func (t *TextBox) AppendLine(line string) {
	if t.Text == "" {
		t.Text = line
	} else {
		t.Text += "\n" + line
	}
}

func (t *TextBox) Update() {
	if t.vScrollBar == nil {
		t.vScrollBar = &VScrollBar{}
	}
	t.vScrollBar.X = t.Rect.Max.X - VScrollBarWidth
	t.vScrollBar.Y = t.Rect.Min.Y
	t.vScrollBar.Height = t.Rect.Dy()

	_, h := t.contentSize()
	t.vScrollBar.Update(h)

	t.offsetX = 0
	t.offsetY = t.vScrollBar.ContentOffset()
}

func (t *TextBox) contentSize() (int, int) {
	h := len(strings.Split(t.Text, "\n")) * lineHeight
	return t.Rect.Dx(), h
}

func (t *TextBox) viewSize() (int, int) {
	return t.Rect.Dx() - VScrollBarWidth - textBoxPaddingLeft, t.Rect.Dy()
}

func (t *TextBox) contentOffset() (int, int) {
	return t.offsetX, t.offsetY
}

func (t *TextBox) Draw(dst *ebiten.Image) {
	drawNinePatches(dst, t.Rect, imageSrcRects[imageTypeTextBox])

	if t.contentBuf != nil {
		vw, vh := t.viewSize()
		w, h := t.contentBuf.Bounds().Dx(), t.contentBuf.Bounds().Dy()
		if vw > w || vh > h {
			t.contentBuf.Dispose()
			t.contentBuf = nil
		}
	}
	if t.contentBuf == nil {
		w, h := t.viewSize()
		t.contentBuf = ebiten.NewImage(w, h)
	}

	t.contentBuf.Clear()
	for i, line := range strings.Split(t.Text, "\n") {
		x := -t.offsetX + textBoxPaddingLeft
		y := -t.offsetY + i*lineHeight + lineHeight - (lineHeight-uiFontMHeight)/2
		if y < -lineHeight {
			continue
		}
		if _, h := t.viewSize(); y >= h+lineHeight {
			continue
		}
		text.Draw(t.contentBuf, line, uiFont, x, y, color.Black)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(t.Rect.Min.X), float64(t.Rect.Min.Y))
	dst.DrawImage(t.contentBuf, op)

	t.vScrollBar.Draw(dst)
}

const (
	checkboxWidth       = 16
	checkboxHeight      = 16
	checkboxPaddingLeft = 8
)

type CheckBox struct {
	X    int
	Y    int
	Text string

	checked   bool
	mouseDown bool

	onCheckChanged func(c *CheckBox)
}

func (c *CheckBox) width() int {
	b, _ := font.BoundString(uiFont, c.Text)
	w := (b.Max.X - b.Min.X).Ceil()
	return checkboxWidth + checkboxPaddingLeft + w
}

func (c *CheckBox) Update() {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if c.X <= x && x < c.X+c.width() && c.Y <= y && y < c.Y+checkboxHeight {
			c.mouseDown = true
		} else {
			c.mouseDown = false
		}
	} else {
		if c.mouseDown {
			c.checked = !c.checked
			if c.onCheckChanged != nil {
				c.onCheckChanged(c)
			}
		}
		c.mouseDown = false
	}
}

func (c *CheckBox) Draw(dst *ebiten.Image) {
	t := imageTypeCheckBox
	if c.mouseDown {
		t = imageTypeCheckBoxPressed
	}
	r := image.Rect(c.X, c.Y, c.X+checkboxWidth, c.Y+checkboxHeight)
	drawNinePatches(dst, r, imageSrcRects[t])
	if c.checked {
		drawNinePatches(dst, r, imageSrcRects[imageTypeCheckBoxMark])
	}

	x := c.X + checkboxWidth + checkboxPaddingLeft
	y := (c.Y + 16) - (16-uiFontMHeight)/2
	text.Draw(dst, c.Text, uiFont, x, y, color.Black)
}

func (c *CheckBox) Checked() bool {
	return c.checked
}

func (c *CheckBox) SetOnCheckChanged(f func(c *CheckBox)) {
	c.onCheckChanged = f
}

/*
GameMainFunc Start
*/

type Game struct {
	button1    *Button
	button2    *Button
	checkBox   *CheckBox
	textBoxLog *TextBox
	keys       []ebiten.Key
	me         *Player
	players    map[int]*Player
}

func NewGame() *Game {
	g := &Game{}
	g.button1 = &Button{
		Rect: image.Rect(16, 16, 144, 48),
		Text: "Button 1",
	}
	g.button2 = &Button{
		Rect: image.Rect(160, 16, 288, 48),
		Text: "Button 2",
	}
	g.checkBox = &CheckBox{
		X:    16,
		Y:    64,
		Text: "Check Box!",
	}
	g.textBoxLog = &TextBox{
		Rect: image.Rect(16, 96, 624, 464),
	}

	g.button1.SetOnPressed(func(b *Button) {
		g.textBoxLog.AppendLine("Button 1 Pressed")
		go g.wsXY()
	})
	g.button2.SetOnPressed(func(b *Button) {
		g.textBoxLog.AppendLine("Button 2 Pressed")
	})
	g.checkBox.SetOnCheckChanged(func(c *CheckBox) {
		msg := "Check box check changed"
		if c.Checked() {
			msg += " (Checked)"
		} else {
			msg += " (Unchecked)"
		}
		g.textBoxLog.AppendLine(msg)
	})

	g.newPlayer()
	g.players = make(map[int]*Player)

	return g
}

func (g *Game) Update() error {
	g.button1.Update()
	g.button2.Update()
	g.checkBox.Update()
	g.textBoxLog.Update()
	g.keys = inpututil.AppendPressedKeys(g.keys[:0])
	g.handleMovement()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0xeb, 0xeb, 0xeb, 0xff})
	screen.DrawImage(bgImage, nil)
	g.button1.Draw(screen)

	g.me.Draw(screen)

	for _, p := range g.PlayersWithoutMe() {
		p.Draw(screen)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

/*
GameMainFunc End
*/

/*
WebSocket Start
*/

type Player struct {
	ID   int
	X    int
	Y    int
	Name string
}

func (g *Game) wsXY() {
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()

	go func() {
		for {
			// var p Player
			p := make(map[int]*Player)
			err := conn.ReadJSON(&p)
			if err != nil {
				fmt.Println("Error reading message from server:", err)
				return
			}
			playersMutex.Lock()
			for k, v := range p {
				g.players[k] = v
			}
			playersMutex.Unlock()
			fmt.Printf("%v\n", g.players)
		}
	}()

	for {
		p := g.me
		data, err := json.Marshal(p)
		if err != nil {
			fmt.Println("Error marshalling JSON:", err)
			break
		}

		err = conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			fmt.Println("Error sending message to server:", err)
			break
		}

		time.Sleep(10 * time.Millisecond)
	}
}

/*
WebSocket End
*/

/*
MyGame Start
*/
func (g *Game) newPlayer() {
	g.me = &Player{
		ID: rand.Int(),
		X:  screenWidth / 2,
		Y:  screenHeight / 2,
	}
}

func (g *Game) PlayersWithoutMe() []*Player {
	playersMutex.Lock()
	defer playersMutex.Unlock()

	players := make([]*Player, 0, len(g.players))
	for _, p := range g.players {
		if p.ID != g.me.ID {
			players = append(players, p)
		}
	}
	return players
}

func (p *Player) Draw(dst *ebiten.Image) {
	vector.DrawFilledRect(dst, float32(p.X)-8, float32(p.Y)-8, 16, 16, color.RGBA{200, 200, 200, 255}, true)
	vector.DrawFilledRect(dst, float32(p.X)-4, float32(p.Y)-4, 8, 8, color.RGBA{255, 100, 100, 255}, true)
}

func (g *Game) handleMovement() {
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		g.me.X += 4
	}

	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		g.me.Y += 4
	}

	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		g.me.X -= 4
	}

	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		g.me.Y -= 4
	}

	// +1/-1 is to stop player before it reaches the border
	if g.me.X >= screenWidth-padding {
		g.me.X = screenWidth - padding - 1
	}

	if g.me.X <= padding {
		g.me.X = padding + 1
	}

	if g.me.Y >= screenHeight-padding {
		g.me.Y = screenHeight - padding - 1
	}

	if g.me.Y <= padding {
		g.me.Y = padding + 1
	}
}

/*
MyGame End
*/

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Mirai Yochi")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
