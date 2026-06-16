package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"log/slog"
	"math"
	"net/http"
	"sort"
)

func compressImage(data []byte, maxSize int) ([]byte, error) {
	contentType := http.DetectContentType(data)

	if contentType == "image/gif" {
		return compressGIF(data, maxSize)
	}

	return compressStatic(data, maxSize)
}

func compressStatic(data []byte, maxSize int) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("解码图片失败: %w", err)
	}

	for quality := 85; quality >= 10; quality -= 10 {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, fmt.Errorf("JPEG 编码失败: %w", err)
		}
		if buf.Len() <= maxSize {
			return buf.Bytes(), nil
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 5}); err != nil {
		return nil, fmt.Errorf("JPEG 编码失败: %w", err)
	}
	if buf.Len() <= maxSize {
		return buf.Bytes(), nil
	}

	return nil, fmt.Errorf("压缩后仍超过限制 (%d > %d)", buf.Len(), maxSize)
}

func compressGIF(data []byte, maxSize int) ([]byte, error) {
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("解码 GIF 失败: %w", err)
	}

	for _, skip := range []int{2, 3, 4, 5} {
		scaled := scaleGIF(g, 0.8)
		reduced := skipFramesGIF(scaled, skip)
		if out, ok := tryEncodeGIF(reduced, maxSize); ok {
			slog.Info("GIF 压缩: 缩放+抽帧成功", "scale", 0.8, "skip", skip)
			return out, nil
		}
	}

	for _, scale := range []float64{0.8, 0.7, 0.6, 0.5} {
		result := scaleGIF(g, scale)
		if out, ok := tryEncodeGIF(result, maxSize); ok {
			slog.Info("GIF 压缩: 缩放成功", "scale", scale)
			return out, nil
		}
	}

	for _, scale := range []float64{0.6, 0.5} {
		for _, nColor := range []int{128, 64, 32} {
			result := scaleAndReduceColorsGIF(g, scale, nColor)
			if out, ok := tryEncodeGIF(result, maxSize); ok {
				slog.Info("GIF 压缩: 缩放+减色成功", "scale", scale, "colors", nColor)
				return out, nil
			}
		}
	}

	for _, skip := range []int{2, 3, 4} {
		scaled := scaleGIF(g, 0.4)
		reduced := skipFramesGIF(scaled, skip)
		if out, ok := tryEncodeGIF(reduced, maxSize); ok {
			slog.Info("GIF 压缩: 极限缩放+抽帧成功", "scale", 0.4, "skip", skip)
			return out, nil
		}
	}

	return nil, fmt.Errorf("GIF 压缩后仍超过限制")
}

func tryEncodeGIF(g *gif.GIF, maxSize int) ([]byte, bool) {
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		return nil, false
	}
	if buf.Len() <= maxSize {
		return buf.Bytes(), true
	}
	return nil, false
}

func scaleGIF(g *gif.GIF, scale float64) *gif.GIF {
	origW, origH := g.Config.Width, g.Config.Height
	newW := int(math.Round(float64(origW) * scale))
	newH := int(math.Round(float64(origH) * scale))
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	out := &gif.GIF{
		LoopCount:       g.LoopCount,
		BackgroundIndex: g.BackgroundIndex,
		Config:          image.Config{Width: newW, Height: newH},
	}

	canvas := image.NewRGBA(image.Rect(0, 0, origW, origH))

	for i, frame := range g.Image {
		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)

		scaled := scaleRGBA(canvas, newW, newH)

		paletted := image.NewPaletted(image.Rect(0, 0, newW, newH), frame.Palette)
		draw.Draw(paletted, paletted.Bounds(), scaled, image.Point{}, draw.Src)

		out.Image = append(out.Image, paletted)
		out.Disposal = append(out.Disposal, gif.DisposalBackground)
		if i < len(g.Delay) {
			out.Delay = append(out.Delay, g.Delay[i])
		} else {
			out.Delay = append(out.Delay, 10)
		}

		if i < len(g.Disposal) {
			switch g.Disposal[i] {
			case gif.DisposalBackground, gif.DisposalPrevious:
				draw.Draw(canvas, frame.Bounds(), image.Transparent, image.Point{}, draw.Src)
			}
		}
	}

	return out
}

func scaleAndReduceColorsGIF(g *gif.GIF, scale float64, nColor int) *gif.GIF {
	origW, origH := g.Config.Width, g.Config.Height
	newW := int(math.Round(float64(origW) * scale))
	newH := int(math.Round(float64(origH) * scale))
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	pal := extractPalette(g, nColor)

	out := &gif.GIF{
		LoopCount:       g.LoopCount,
		BackgroundIndex: g.BackgroundIndex,
		Config:          image.Config{Width: newW, Height: newH},
	}

	canvas := image.NewRGBA(image.Rect(0, 0, origW, origH))

	for i, frame := range g.Image {
		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)

		scaled := scaleRGBA(canvas, newW, newH)

		paletted := image.NewPaletted(image.Rect(0, 0, newW, newH), pal)
		draw.Draw(paletted, paletted.Bounds(), scaled, image.Point{}, draw.Src)

		out.Image = append(out.Image, paletted)
		out.Disposal = append(out.Disposal, gif.DisposalBackground)
		if i < len(g.Delay) {
			out.Delay = append(out.Delay, g.Delay[i])
		} else {
			out.Delay = append(out.Delay, 10)
		}

		if i < len(g.Disposal) {
			switch g.Disposal[i] {
			case gif.DisposalBackground, gif.DisposalPrevious:
				draw.Draw(canvas, frame.Bounds(), image.Transparent, image.Point{}, draw.Src)
			}
		}
	}

	return out
}

func extractPalette(g *gif.GIF, n int) color.Palette {
	seen := make(map[color.RGBA]struct{})
	for _, frame := range g.Image {
		for _, c := range frame.Palette {
			r, g, b, a := c.RGBA()
			seen[color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}] = struct{}{}
		}
	}

	colors := make([]color.RGBA, 0, len(seen))
	for c := range seen {
		colors = append(colors, c)
	}

	if len(colors) <= n {
		pal := make(color.Palette, len(colors))
		for i, c := range colors {
			pal[i] = c
		}
		return pal
	}

	return medianCut(colors, n)
}

func medianCut(colors []color.RGBA, n int) color.Palette {
	type bucket struct {
		colors []color.RGBA
	}

	buckets := []bucket{{colors: colors}}

	for len(buckets) < n {
		maxIdx := 0
		maxSize := 0
		for i, b := range buckets {
			if len(b.colors) > maxSize {
				maxSize = len(b.colors)
				maxIdx = i
			}
		}

		if maxSize <= 1 {
			break
		}

		target := buckets[maxIdx].colors

		var minR, minG, minB uint8 = 255, 255, 255
		var maxR, maxG, maxB uint8 = 0, 0, 0
		for _, c := range target {
			if c.R < minR {
				minR = c.R
			}
			if c.R > maxR {
				maxR = c.R
			}
			if c.G < minG {
				minG = c.G
			}
			if c.G > maxG {
				maxG = c.G
			}
			if c.B < minB {
				minB = c.B
			}
			if c.B > maxB {
				maxB = c.B
			}
		}

		rangeR := int(maxR) - int(minR)
		rangeG := int(maxG) - int(minG)
		rangeB := int(maxB) - int(minB)

		switch {
		case rangeR >= rangeG && rangeR >= rangeB:
			sort.Slice(target, func(i, j int) bool { return target[i].R < target[j].R })
		case rangeG >= rangeR && rangeG >= rangeB:
			sort.Slice(target, func(i, j int) bool { return target[i].G < target[j].G })
		default:
			sort.Slice(target, func(i, j int) bool { return target[i].B < target[j].B })
		}

		mid := len(target) / 2
		buckets[maxIdx] = bucket{colors: target[:mid]}
		buckets = append(buckets, bucket{colors: target[mid:]})
	}

	pal := make(color.Palette, 0, len(buckets))
	for _, b := range buckets {
		if len(b.colors) == 0 {
			continue
		}
		var sumR, sumG, sumB, sumA int
		for _, c := range b.colors {
			sumR += int(c.R)
			sumG += int(c.G)
			sumB += int(c.B)
			sumA += int(c.A)
		}
		n := len(b.colors)
		pal = append(pal, color.RGBA{
			R: uint8(sumR / n),
			G: uint8(sumG / n),
			B: uint8(sumB / n),
			A: uint8(sumA / n),
		})
	}
	return pal
}

func skipFramesGIF(g *gif.GIF, skip int) *gif.GIF {
	out := &gif.GIF{
		LoopCount:       g.LoopCount,
		BackgroundIndex: g.BackgroundIndex,
		Config:          g.Config,
	}
	for i := 0; i < len(g.Image); i += skip {
		out.Image = append(out.Image, g.Image[i])
		out.Disposal = append(out.Disposal, g.Disposal[i])
		totalDelay := 0
		for j := i; j < i+skip && j < len(g.Delay); j++ {
			totalDelay += g.Delay[j]
		}
		out.Delay = append(out.Delay, totalDelay)
	}
	return out
}

func scaleRGBA(src *image.RGBA, newW, newH int) *image.RGBA {
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	xRatio := float64(srcW) / float64(newW)
	yRatio := float64(srcH) / float64(newH)

	for y := 0; y < newH; y++ {
		srcY := float64(y) * yRatio
		y0 := int(srcY)
		y1 := y0 + 1
		if y1 >= srcH {
			y1 = srcH - 1
		}
		fy := srcY - float64(y0)

		for x := 0; x < newW; x++ {
			srcX := float64(x) * xRatio
			x0 := int(srcX)
			x1 := x0 + 1
			if x1 >= srcW {
				x1 = srcW - 1
			}
			fx := srcX - float64(x0)

			c00 := src.RGBAAt(srcBounds.Min.X+x0, srcBounds.Min.Y+y0)
			c10 := src.RGBAAt(srcBounds.Min.X+x1, srcBounds.Min.Y+y0)
			c01 := src.RGBAAt(srcBounds.Min.X+x0, srcBounds.Min.Y+y1)
			c11 := src.RGBAAt(srcBounds.Min.X+x1, srcBounds.Min.Y+y1)

			r := bilinear(float64(c00.R), float64(c10.R), float64(c01.R), float64(c11.R), fx, fy)
			gr := bilinear(float64(c00.G), float64(c10.G), float64(c01.G), float64(c11.G), fx, fy)
			b := bilinear(float64(c00.B), float64(c10.B), float64(c01.B), float64(c11.B), fx, fy)
			a := bilinear(float64(c00.A), float64(c10.A), float64(c01.A), float64(c11.A), fx, fy)

			dst.SetRGBA(x, y, color.RGBA{R: r, G: gr, B: b, A: a})
		}
	}

	return dst
}

func bilinear(c00, c10, c01, c11, fx, fy float64) uint8 {
	v := c00*(1-fx)*(1-fy) + c10*fx*(1-fy) + c01*(1-fx)*fy + c11*fx*fy
	return uint8(math.Max(0, math.Min(255, math.Round(v))))
}