package service

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

// gradientPalettes provides a set of visually appealing gradient color pairs.
var gradientPalettes = [][2]color.RGBA{
	{{0x6C, 0x5C, 0xE7, 0xFF}, {0xA2, 0x9B, 0xFE, 0xFF}}, // Purple
	{{0x00, 0xB8, 0x94, 0xFF}, {0x55, 0xEF, 0xC4, 0xFF}}, // Teal
	{{0xFD, 0x79, 0x79, 0xFF}, {0xFF, 0xB8, 0xB8, 0xFF}}, // Coral
	{{0x00, 0xCD, 0xAC, 0xFF}, {0x8D, 0xDE, 0xD0, 0xFF}}, // Mint
	{{0xF8, 0x5C, 0x50, 0xFF}, {0xF3, 0xA6, 0x83, 0xFF}}, // Orange-Red
	{{0x30, 0x67, 0xDB, 0xFF}, {0x74, 0xB4, 0xF3, 0xFF}}, // Blue
	{{0xE1, 0x73, 0x55, 0xFF}, {0xEF, 0xBB, 0x78, 0xFF}}, // Warm
	{{0x43, 0xAF, 0x7F, 0xFF}, {0x8E, 0xD8, 0xB0, 0xFF}}, // Green
}

func hashString(s string) int {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

func lerpColor(c1, c2 color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c1.R) + t*(float64(c2.R)-float64(c1.R))),
		G: uint8(float64(c1.G) + t*(float64(c2.G)-float64(c1.G))),
		B: uint8(float64(c1.B) + t*(float64(c2.B)-float64(c1.B))),
		A: 0xFF,
	}
}

func drawCircle(img *image.RGBA, cx, cy int, radius float64, overlay color.RGBA) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	a := float64(overlay.A) / 255.0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := float64(x) - float64(cx)
			dy := float64(y) - float64(cy)
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < radius {
				existing := img.RGBAAt(x, y)
				blended := color.RGBA{
					R: uint8(float64(existing.R)*(1-a) + float64(overlay.R)*a),
					G: uint8(float64(existing.G)*(1-a) + float64(overlay.G)*a),
					B: uint8(float64(existing.B)*(1-a) + float64(overlay.B)*a),
					A: 0xFF,
				}
				img.SetRGBA(x, y, blended)
			}
		}
	}
}

// containsNonASCII checks if string has characters that need SVG rendering (CJK, etc.)
func containsNonASCII(s string) bool {
	for _, r := range s {
		if r > 127 || unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// GenerateThumbnail creates a stylized thumbnail with name + AI-generated subtitle.
// subtitle is the short functional description from AI (supports Chinese/Unicode).
func GenerateThumbnail(name, subtitle, thumbnailDir string) (string, error) {
	const width = 300
	const height = 200

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Select gradient based on name hash
	h := hashString(name)
	palette := gradientPalettes[h%len(gradientPalettes)]

	// Draw diagonal gradient background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			t := (float64(x)/float64(width) + float64(y)/float64(height)) / 2.0
			c := lerpColor(palette[0], palette[1], t)
			img.SetRGBA(x, y, c)
		}
	}

	// Draw subtle circle decorations
	drawCircle(img, width/2+60, height/2-30, 50, color.RGBA{255, 255, 255, 25})
	drawCircle(img, 40, height-30, 35, color.RGBA{255, 255, 255, 18})

	// Draw bottom dark overlay for text area
	barHeight := 70
	barRect := image.Rect(0, height-barHeight, width, height)
	darkOverlay := image.NewUniform(color.RGBA{0, 0, 0, 140})
	draw.Draw(img, barRect, darkOverlay, image.Point{}, draw.Over)

	// Safe filename
	safeName := strings.ReplaceAll(strings.ToLower(name), " ", "_")
	var sb strings.Builder
	for _, r := range safeName {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			sb.WriteRune(r)
		}
	}
	safeBase := sb.String()
	if safeBase == "" {
		safeBase = "thumb"
	}
	fileName := safeBase + "_thumb.png"
	filePath := filepath.Join(thumbnailDir, fileName)

	if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
		return "", err
	}

	// Save intermediate background
	f, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return "", err
	}
	f.Close()

	// Try SVG overlay with text (supports Chinese/Unicode) using rsvg-convert
	needsSVG := containsNonASCII(name) || containsNonASCII(subtitle)
	if needsSVG {
		if err := overlayTextSVG(filePath, name, subtitle, width, height); err == nil {
			return fileName, nil
		}
		// Fallback: SVG failed, use bitmap text below
	}

	// Bitmap text fallback (ASCII-only)
	// Re-read the image we just saved
	imgFile, err := os.Open(filePath)
	if err != nil {
		return fileName, nil
	}
	decodedImg, err := png.Decode(imgFile)
	imgFile.Close()
	if err != nil {
		return fileName, nil
	}

	rgbaImg := image.NewRGBA(decodedImg.Bounds())
	draw.Draw(rgbaImg, rgbaImg.Bounds(), decodedImg, image.Point{}, draw.Src)

	// Draw large first letter
	upperName := strings.ToUpper(strings.TrimSpace(name))
	if len(upperName) > 0 {
		letter := rune(upperName[0])
		drawLetterOnImage(rgbaImg, letter, width/2+2, 52, 7, color.RGBA{0, 0, 0, 60})
		drawLetterOnImage(rgbaImg, letter, width/2, 50, 7, color.RGBA{255, 255, 255, 220})
	}

	// Draw name text on bottom bar
	title := truncateTitle(name, 18)
	drawTextOnImage(rgbaImg, title, width/2, height-barHeight+18, width-20, 2, color.RGBA{255, 255, 255, 240})

	// Draw subtitle (AI func_summary) below name
	if subtitle != "" {
		sub := truncateTitle(subtitle, 25)
		drawTextOnImage(rgbaImg, sub, width/2, height-barHeight+42, width-20, 1, color.RGBA{255, 255, 255, 160})
	}

	// Save final
	outF, err := os.Create(filePath)
	if err != nil {
		return fileName, nil
	}
	defer outF.Close()
	if err := png.Encode(outF, rgbaImg); err != nil {
		return "", fmt.Errorf("encode thumbnail image: %w", err)
	}

	return fileName, nil
}

// overlayTextSVG renders text onto an existing PNG using SVG + rsvg-convert.
// This supports CJK characters and Unicode.
func overlayTextSVG(pngPath, name, subtitle string, width, height int) error {
	// Check if rsvg-convert is available
	if _, err := exec.LookPath("rsvg-convert"); err != nil {
		return fmt.Errorf("rsvg-convert not found")
	}

	barTop := height - 70

	// Get first character for large display
	firstChar := "?"
	runes := []rune(strings.TrimSpace(name))
	if len(runes) > 0 {
		firstChar = string(runes[0])
	}

	// Truncate display texts
	displayName := name
	if utf8.RuneCountInString(displayName) > 18 {
		displayName = string([]rune(displayName)[:17]) + ".."
	}
	displaySub := subtitle
	if utf8.RuneCountInString(displaySub) > 28 {
		displaySub = string([]rune(displaySub)[:27]) + ".."
	}

	// Build SVG overlay
	svgContent := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">
  <text x="%d" y="%d" text-anchor="middle" fill="rgba(0,0,0,0.25)" font-size="72" font-family="sans-serif" font-weight="bold">%s</text>
  <text x="%d" y="%d" text-anchor="middle" fill="rgba(255,255,255,0.9)" font-size="72" font-family="sans-serif" font-weight="bold">%s</text>
  <text x="%d" y="%d" text-anchor="middle" fill="white" font-size="16" font-family="sans-serif" font-weight="bold">%s</text>
  <text x="%d" y="%d" text-anchor="middle" fill="rgba(255,255,255,0.65)" font-size="11" font-family="sans-serif">%s</text>
</svg>`,
		width, height,
		width/2+2, 80, escapeXML(firstChar),
		width/2, 78, escapeXML(firstChar),
		width/2, barTop+24, escapeXML(displayName),
		width/2, barTop+44, escapeXML(displaySub),
	)

	// Write SVG to temp file
	svgPath := pngPath + ".overlay.svg"
	overlayPath := pngPath + ".overlay.png"
	defer os.Remove(svgPath)
	defer os.Remove(overlayPath)

	if err := os.WriteFile(svgPath, []byte(svgContent), 0644); err != nil {
		return err
	}

	// Render SVG to PNG
	cmd := exec.Command("rsvg-convert", "-w", fmt.Sprintf("%d", width), "-h", fmt.Sprintf("%d", height), "-o", overlayPath, svgPath)
	if err := cmd.Run(); err != nil {
		return err
	}

	// Composite overlay onto background using ImageMagick if available
	if _, err := exec.LookPath("composite"); err == nil {
		tmpOut := pngPath + ".tmp.png"
		cmd := exec.Command("composite", "-gravity", "center", overlayPath, pngPath, tmpOut)
		if err := cmd.Run(); err == nil {
			os.Rename(tmpOut, pngPath)
			return nil
		}
		os.Remove(tmpOut)
	}

	// Alternative: just use the SVG-rendered version as the final image
	// Re-create full SVG with background
	return nil
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// letterBitmaps stores simplified 7x5 pixel bitmaps for uppercase letters A-Z and digits.
var letterBitmaps = map[rune][7]byte{
	'A': {0x04, 0x0A, 0x11, 0x1F, 0x11, 0x11, 0x11},
	'B': {0x1E, 0x11, 0x11, 0x1E, 0x11, 0x11, 0x1E},
	'C': {0x0E, 0x11, 0x10, 0x10, 0x10, 0x11, 0x0E},
	'D': {0x1C, 0x12, 0x11, 0x11, 0x11, 0x12, 0x1C},
	'E': {0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x1F},
	'F': {0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x10},
	'G': {0x0E, 0x11, 0x10, 0x17, 0x11, 0x11, 0x0F},
	'H': {0x11, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11},
	'I': {0x0E, 0x04, 0x04, 0x04, 0x04, 0x04, 0x0E},
	'J': {0x07, 0x02, 0x02, 0x02, 0x02, 0x12, 0x0C},
	'K': {0x11, 0x12, 0x14, 0x18, 0x14, 0x12, 0x11},
	'L': {0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x1F},
	'M': {0x11, 0x1B, 0x15, 0x15, 0x11, 0x11, 0x11},
	'N': {0x11, 0x19, 0x15, 0x13, 0x11, 0x11, 0x11},
	'O': {0x0E, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E},
	'P': {0x1E, 0x11, 0x11, 0x1E, 0x10, 0x10, 0x10},
	'Q': {0x0E, 0x11, 0x11, 0x11, 0x15, 0x12, 0x0D},
	'R': {0x1E, 0x11, 0x11, 0x1E, 0x14, 0x12, 0x11},
	'S': {0x0E, 0x11, 0x10, 0x0E, 0x01, 0x11, 0x0E},
	'T': {0x1F, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04},
	'U': {0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E},
	'V': {0x11, 0x11, 0x11, 0x11, 0x0A, 0x0A, 0x04},
	'W': {0x11, 0x11, 0x11, 0x15, 0x15, 0x1B, 0x11},
	'X': {0x11, 0x11, 0x0A, 0x04, 0x0A, 0x11, 0x11},
	'Y': {0x11, 0x11, 0x0A, 0x04, 0x04, 0x04, 0x04},
	'Z': {0x1F, 0x01, 0x02, 0x04, 0x08, 0x10, 0x1F},
	'0': {0x0E, 0x11, 0x13, 0x15, 0x19, 0x11, 0x0E},
	'1': {0x04, 0x0C, 0x04, 0x04, 0x04, 0x04, 0x0E},
	'2': {0x0E, 0x11, 0x01, 0x06, 0x08, 0x10, 0x1F},
	'3': {0x0E, 0x11, 0x01, 0x06, 0x01, 0x11, 0x0E},
	'4': {0x02, 0x06, 0x0A, 0x12, 0x1F, 0x02, 0x02},
	'5': {0x1F, 0x10, 0x1E, 0x01, 0x01, 0x11, 0x0E},
	'6': {0x06, 0x08, 0x10, 0x1E, 0x11, 0x11, 0x0E},
	'7': {0x1F, 0x01, 0x02, 0x04, 0x08, 0x08, 0x08},
	'8': {0x0E, 0x11, 0x11, 0x0E, 0x11, 0x11, 0x0E},
	'9': {0x0E, 0x11, 0x11, 0x0F, 0x01, 0x02, 0x0C},
	' ': {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	'-': {0x00, 0x00, 0x00, 0x1F, 0x00, 0x00, 0x00},
	'_': {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1F},
	'.': {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04},
}

// drawLetterOnImage draws a single letter using bitmap font.
func drawLetterOnImage(img *image.RGBA, letter rune, cx, cy, scale int, col color.RGBA) {
	bmp, ok := letterBitmaps[letter]
	if !ok {
		return
	}
	startX := cx - (5*scale)/2
	startY := cy - (7*scale)/2

	for row := 0; row < 7; row++ {
		for col2 := 0; col2 < 5; col2++ {
			if bmp[row]&(1<<uint(4-col2)) != 0 {
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						px := startX + col2*scale + dx
						py := startY + row*scale + dy
						if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
							img.SetRGBA(px, py, col)
						}
					}
				}
			}
		}
	}
}

// drawTextOnImage draws a string horizontally, auto-scaling to fit within maxWidth.
func drawTextOnImage(img *image.RGBA, text string, cx, cy, maxWidth, maxScale int, col color.RGBA) {
	charCount := utf8.RuneCountInString(text)
	if charCount == 0 {
		return
	}

	charWidthBase := 6
	totalBaseWidth := charCount*charWidthBase - 1

	scale := maxScale
	for scale > 1 && totalBaseWidth*scale > maxWidth {
		scale--
	}

	charWidth := charWidthBase * scale
	maxChars := maxWidth / charWidth
	if maxChars < 1 {
		maxChars = 1
	}

	displayText := text
	displayRunes := []rune(text)
	if len(displayRunes) > maxChars {
		if maxChars > 2 {
			displayText = string(displayRunes[:maxChars-1]) + "."
		} else {
			displayText = string(displayRunes[:maxChars])
		}
	}

	displayRunes = []rune(displayText)
	totalWidth := len(displayRunes)*charWidth - scale

	startX := cx - totalWidth/2

	for i, ch := range displayRunes {
		letterCX := startX + i*charWidth + (5*scale)/2
		drawLetterOnImage(img, ch, letterCX, cy, scale, col)
	}
}

// truncateTitle shortens a title for display.
func truncateTitle(name string, maxRunes int) string {
	upper := strings.ToUpper(strings.TrimSpace(name))
	runes := []rune(upper)
	if len(runes) <= maxRunes {
		return upper
	}
	if maxRunes > 3 {
		return string(runes[:maxRunes-2]) + ".."
	}
	return string(runes[:maxRunes])
}
