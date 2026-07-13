package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"math/rand"
	"unicode"

	"github.com/gogpu/gg/text"
)

const (
	wordPadding    = 3    // 词与词之间的最小间距（像素）
	canvasMargin   = 4    // 画布四周留白（像素）
	angleStep      = 0.4  // 螺旋线每步转过的弧度
	radiusPerAngle = 1.1  // 螺旋线半径随弧度增长的系数
	shrinkFactor   = 0.82 // 放不下时字号的缩小系数
	verticalRatio  = 0.25 // 竖排词的比例（仅 2~4 字纯中文词参与竖排）
	footerFontSize = 14   // 脚注字号
)

// renderOptions 词云渲染参数
type renderOptions struct {
	width   int
	height  int
	minFont float64
	maxFont float64
}

// renderWordCloud 把词频列表渲染成词云图片。
// words 需按频次降序排列；footer 非空时绘制在右下角。
func renderWordCloud(font *text.FontSource, opts renderOptions, words []wordCount, footer string) (image.Image, error) {
	if len(words) == 0 {
		return nil, fmt.Errorf("没有可渲染的词")
	}
	if opts.width < 200 || opts.height < 200 {
		return nil, fmt.Errorf("画布尺寸过小: %dx%d", opts.width, opts.height)
	}
	if opts.minFont <= 0 {
		opts.minFont = 12
	}
	if opts.maxFont < opts.minFont {
		opts.maxFont = opts.minFont
	}

	canvas := image.NewRGBA(image.Rect(0, 0, opts.width, opts.height))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)

	l := &cloudLayout{
		font:   font,
		opts:   opts,
		canvas: canvas,
		faces:  make(map[float64]text.Face),
	}
	l.reserveFooter(footer)

	placed := 0
	maxCount, minCount := words[0].count, words[len(words)-1].count
	for _, wc := range words {
		size := fontSizeFor(wc.count, minCount, maxCount, opts.minFont, opts.maxFont)
		if l.placeWord(wc.word, size) {
			placed++
		}
	}
	if placed == 0 {
		return nil, fmt.Errorf("画布空间不足，未能放置任何词")
	}
	l.drawFooter(footer)
	return canvas, nil
}

// fontSizeFor 按词频在 [minFont, maxFont] 间插值计算字号。
// 使用平方根缩放：头部词被压缩、长尾词被抬升，避免一家独大时其余词都挤在最小字号。
func fontSizeFor(count, minCount, maxCount int, minFont, maxFont float64) float64 {
	if maxCount <= minCount {
		return (minFont + maxFont) / 2
	}
	ratio := math.Sqrt(float64(count-minCount) / float64(maxCount-minCount))
	return minFont + ratio*(maxFont-minFont)
}

// placedBox 已放置词的包围盒
type placedBox struct {
	x, y, w, h float64
}

// cloudLayout 单次词云渲染的布局状态
type cloudLayout struct {
	font   *text.FontSource
	opts   renderOptions
	canvas *image.RGBA
	faces  map[float64]text.Face // 字号 → Face 缓存
	placed []placedBox
}

// face 取指定字号的 Face（按整数字号缓存）
func (l *cloudLayout) face(size float64) text.Face {
	size = math.Round(size)
	if f, ok := l.faces[size]; ok {
		return f
	}
	f := l.font.Face(size)
	l.faces[size] = f
	return f
}

// placeWord 尝试放置一个词：放不下时逐步缩小字号重试，缩到下限仍放不下则放弃该词
func (l *cloudLayout) placeWord(word string, size float64) bool {
	for ; size >= l.opts.minFont*0.8; size *= shrinkFactor {
		if l.tryPlace(word, size) {
			return true
		}
	}
	return false
}

// tryPlace 以指定字号沿椭圆螺旋线为词寻找空位，找到即绘制并记录包围盒。
// 螺旋线从画布中心出发、随机起始角，x 方向按宽高比拉伸以铺满整个画布。
func (l *cloudLayout) tryPlace(word string, size float64) bool {
	face := l.face(size)
	vertical := shouldVertical(word, size, l.opts.maxFont)

	m := face.Metrics()
	charH := m.Ascent + m.Descent
	var w, h float64
	if vertical {
		w = maxRuneAdvance(face, word)
		h = charH * float64(len([]rune(word)))
	} else {
		w = face.Advance(word)
		h = charH
	}
	if w <= 0 || h <= 0 {
		return false
	}

	cx := float64(l.opts.width) / 2
	cy := float64(l.opts.height) / 2
	aspect := float64(l.opts.width) / float64(l.opts.height)
	maxRadius := math.Hypot(cx, cy)
	start := rand.Float64() * 2 * math.Pi

	for t := 0.0; ; t += angleStep {
		radius := radiusPerAngle * t
		if radius > maxRadius {
			return false
		}
		a := start + t
		x := cx + radius*math.Cos(a)*aspect - w/2
		y := cy + radius*math.Sin(a) - h/2
		if l.fits(x, y, w, h) {
			l.draw(word, face, x, y, w, vertical)
			l.placed = append(l.placed, placedBox{x: x, y: y, w: w, h: h})
			return true
		}
	}
}

// fits 判断包围盒是否在画布内且不与已放置的词重叠
func (l *cloudLayout) fits(x, y, w, h float64) bool {
	if x < canvasMargin || y < canvasMargin ||
		x+w > float64(l.opts.width)-canvasMargin || y+h > float64(l.opts.height)-canvasMargin {
		return false
	}
	for _, b := range l.placed {
		if x < b.x+b.w+wordPadding && x+w+wordPadding > b.x &&
			y < b.y+b.h+wordPadding && y+h+wordPadding > b.y {
			return false
		}
	}
	return true
}

// draw 以随机颜色绘制词；vertical 时逐字竖排
func (l *cloudLayout) draw(word string, face text.Face, x, y, w float64, vertical bool) {
	clr := randomColor()
	m := face.Metrics()
	if !vertical {
		text.Draw(l.canvas, word, face, x, y+m.Ascent, clr)
		return
	}
	charH := m.Ascent + m.Descent
	cy := y
	for _, r := range word {
		ch := string(r)
		cx := x + (w-face.Advance(ch))/2
		text.Draw(l.canvas, ch, face, cx, cy+m.Ascent, clr)
		cy += charH
	}
}

// shouldVertical 决定词是否竖排：仅 2~4 字的纯中文词参与，头部大词保持横排更易读
func shouldVertical(word string, size, maxFont float64) bool {
	if size > maxFont*0.7 {
		return false
	}
	runes := []rune(word)
	if len(runes) < 2 || len(runes) > 4 {
		return false
	}
	for _, r := range runes {
		if !unicode.Is(unicode.Han, r) {
			return false
		}
	}
	return rand.Float64() < verticalRatio
}

// maxRuneAdvance 词内最宽字符的宽度（竖排时的包围盒宽度）
func maxRuneAdvance(face text.Face, word string) float64 {
	maxW := 0.0
	for _, r := range word {
		if w := face.Advance(string(r)); w > maxW {
			maxW = w
		}
	}
	return maxW
}

// reserveFooter 预先占住右下角脚注区域，避免词覆盖脚注
func (l *cloudLayout) reserveFooter(footer string) {
	if footer == "" {
		return
	}
	face := l.face(footerFontSize)
	m := face.Metrics()
	w := face.Advance(footer) + 12
	h := m.Ascent + m.Descent + 8
	l.placed = append(l.placed, placedBox{
		x: float64(l.opts.width) - w,
		y: float64(l.opts.height) - h,
		w: w,
		h: h,
	})
}

// drawFooter 在右下角绘制灰色脚注
func (l *cloudLayout) drawFooter(footer string) {
	if footer == "" {
		return
	}
	face := l.face(footerFontSize)
	m := face.Metrics()
	x := float64(l.opts.width) - face.Advance(footer) - 8
	y := float64(l.opts.height) - m.Descent - 6
	text.Draw(l.canvas, footer, face, x, y, color.RGBA{R: 150, G: 150, B: 150, A: 255})
}

// randomColor 生成饱和度、亮度受控的随机颜色，保证在白底上足够清晰
func randomColor() color.Color {
	h := rand.Float64() * 360
	s := 0.55 + rand.Float64()*0.3
	l := 0.35 + rand.Float64()*0.2
	return hslToRGB(h, s, l)
}

// hslToRGB HSL 转 RGB（h∈[0,360) s,l∈[0,1]）
func hslToRGB(h, s, l float64) color.Color {
	if s == 0 {
		v := uint8(math.Round(l * 255))
		return color.RGBA{R: v, G: v, B: v, A: 255}
	}
	q := l + s - l*s
	if l < 0.5 {
		q = l * (1 + s)
	}
	p := 2*l - q
	r := hueToRGB(p, q, h+120)
	g := hueToRGB(p, q, h)
	b := hueToRGB(p, q, h-120)
	return color.RGBA{
		R: uint8(math.Round(r * 255)),
		G: uint8(math.Round(g * 255)),
		B: uint8(math.Round(b * 255)),
		A: 255,
	}
}

// hueToRGB HSL 辅助函数，t 单位为角度
func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t += 360
	}
	if t >= 360 {
		t -= 360
	}
	switch {
	case t < 60:
		return p + (q-p)*t/60
	case t < 180:
		return q
	case t < 240:
		return p + (q-p)*(240-t)/60
	default:
		return p
	}
}
