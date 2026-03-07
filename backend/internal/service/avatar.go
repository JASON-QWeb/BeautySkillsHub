package service

import (
	"crypto/md5"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// GenerateAvatar creates a unique 5x5 symmetric pixel identicon for a username.
// Returns the filename of the generated avatar PNG.
func GenerateAvatar(username, avatarDir string) (string, error) {
	const (
		gridSize  = 5
		cellSize  = 10
		padding   = 5
		imageSize = gridSize*cellSize + padding*2 // 60x60
	)

	// Hash username for deterministic output
	hash := md5.Sum([]byte(username))

	// Derive foreground color from hash using HSL for vibrant colors
	hue := float64(hash[0]) + float64(hash[1])/256.0
	hue = math.Mod(hue, 360.0)
	fg := hslToRGB(hue, 0.65, 0.45)
	bg := color.RGBA{240, 240, 240, 255}

	// Build 5x5 grid with vertical symmetry (only need 3 columns)
	var grid [gridSize][gridSize]bool
	bitIndex := 0
	for col := 0; col < 3; col++ {
		for row := 0; row < gridSize; row++ {
			byteIdx := (bitIndex / 8) + 2 // offset by 2 to skip color bytes
			bitIdx := bitIndex % 8
			if byteIdx < len(hash) {
				grid[row][col] = hash[byteIdx]&(1<<uint(bitIdx)) != 0
			}
			bitIndex++
			// Mirror horizontally
			grid[row][gridSize-1-col] = grid[row][col]
		}
	}

	// Create image
	img := image.NewRGBA(image.Rect(0, 0, imageSize, imageSize))

	// Fill background
	for y := 0; y < imageSize; y++ {
		for x := 0; x < imageSize; x++ {
			img.SetRGBA(x, y, bg)
		}
	}

	// Draw grid cells
	for row := 0; row < gridSize; row++ {
		for col := 0; col < gridSize; col++ {
			if grid[row][col] {
				x0 := padding + col*cellSize
				y0 := padding + row*cellSize
				for dy := 0; dy < cellSize; dy++ {
					for dx := 0; dx < cellSize; dx++ {
						img.SetRGBA(x0+dx, y0+dy, fg)
					}
				}
			}
		}
	}

	// Ensure directory exists
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		return "", err
	}

	// Safe filename
	safeName := strings.ToLower(strings.TrimSpace(username))
	var sb strings.Builder
	for _, r := range safeName {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			sb.WriteRune(r)
		}
	}
	base := sb.String()
	if base == "" {
		base = "user"
	}
	fileName := base + "_avatar.png"
	filePath := filepath.Join(avatarDir, fileName)

	f, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return "", err
	}

	return fileName, nil
}

// hslToRGB converts HSL values to an RGBA color.
func hslToRGB(h, s, l float64) color.RGBA {
	h = math.Mod(h, 360.0)
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60.0, 2)-1))
	m := l - c/2

	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	return color.RGBA{
		R: uint8((r + m) * 255),
		G: uint8((g + m) * 255),
		B: uint8((b + m) * 255),
		A: 255,
	}
}
