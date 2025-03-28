package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/draw"
	"image/png"   // Import for PNG encoding
	_ "image/png" // Import for PNG decoding
	"log"
	"os"
	"strings"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

//go:embed template.png
var templateImageBytes []byte

//go:embed font.ttf
var fontBytes []byte

const (
	dpi      = 72.0  // Screen DPI
	fontSize = 144.0 // Font size in points (was 48.0, now tripled)
	paddingY = 20    // Padding from the top/bottom edge
	// --- New constant for outline ---
	outlineThickness = 2 // Outline width in pixels
)

// --- Define colors ---
var (
	fillColor    = image.White // Or color.White
	outlineColor = image.Black // Or color.Black
)

func main() {
	// --- 1. Argument Handling ---
	if len(os.Args) < 2 || os.Args[1] == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s \"<text>\" [output.png]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  <text>: The text to draw on the meme.\n")
		fmt.Fprintf(os.Stderr, "  [output.png]: Optional output PNG filename. If omitted, writes to stdout.\n")
		os.Exit(1)
	}
	memeText := strings.ToUpper(os.Args[1])
	outputDest := os.Stdout

	if len(os.Args) > 2 {
		filePath := os.Args[2]
		if !strings.HasSuffix(strings.ToLower(filePath), ".png") {
			log.Printf("Warning: Output filename '%s' does not end with .png. Appending .png", filePath)
			filePath += ".png"
		}
		outFile, err := os.Create(filePath)
		if err != nil {
			log.Fatalf("❌ Error creating output file '%s': %v", filePath, err)
		}
		defer outFile.Close()
		outputDest = outFile
		log.Printf("✅ Output will be saved to: %s", filePath)
	} else {
		log.Println("✅ No output filename specified. Writing PNG data to stdout.")
	}

	// --- 2. Load Template Image ---
	imgReader := bytes.NewReader(templateImageBytes)
	baseImg, format, err := image.Decode(imgReader) // image.Decode handles various formats including JPEG
	if err != nil {
		log.Fatalf("❌ Error decoding template image: %v", err)
	}
	// Check if it's JPEG and log a warning if embedding JPEG directly (can sometimes cause issues)
	if format == "jpeg" {
		log.Printf("🖼️ Template image loaded (format: %s). Note: Embedded JPEGs might differ slightly after decode/encode.", format)
	} else {
		log.Printf("🖼️ Template image loaded (format: %s)", format)
	}

	// --- 3. Load Font ---
	ttFont, err := freetype.ParseFont(fontBytes)
	if err != nil {
		log.Fatalf("❌ Error parsing font: %v", err)
	}
	log.Printf("🔤 Font loaded successfully")

	// --- 4. Prepare Drawing Canvas ---
	bounds := baseImg.Bounds()
	rgbaImg := image.NewRGBA(bounds)
	draw.Draw(rgbaImg, bounds, baseImg, image.Point{}, draw.Src)

	// --- 5. Setup Text Drawing Context ---
	c := freetype.NewContext()
	c.SetDPI(dpi)
	c.SetFont(ttFont)
	c.SetFontSize(fontSize) // Use the new larger font size
	c.SetClip(rgbaImg.Bounds())
	c.SetDst(rgbaImg)
	// SetHinting - Note: We will set the color (Src) just before drawing.
	c.SetHinting(font.HintingFull)

	// --- 6. Calculate Text Position (Centered, near TOP) ---
	// Measure the text width
	textWidth, err := measureString(ttFont, fontSize, dpi, font.HintingFull, memeText)
	if err != nil {
		log.Fatalf("❌ Error measuring text: %v", err)
	}

	imageWidth := bounds.Dx()
	// imageHeight := bounds.Dy() // Not directly needed for top alignment calculation

	// Calculate starting X for centered text
	startX := (imageWidth - textWidth) / 2

	// Calculate starting Y (baseline position) near the TOP
	// Baseline = top padding + approximate font height (ascender)
	// Using fontSize as approximation for height works reasonably well.
	// Adjust paddingY or this calculation slightly if needed for exact placement.
	startY := paddingY + int(fontSize*dpi/72.0) // Convert fontSize points to pixel height roughly

	pt := freetype.Pt(startX, startY) // Baseline point for the main text

	// --- 7. Draw the Text with Outline ---
	log.Printf("✏️ Drawing text outline...")

	// Offsets for 8 directions for a smoother outline
	offsets := []image.Point{
		{-outlineThickness, -outlineThickness}, {0, -outlineThickness}, {outlineThickness, -outlineThickness},
		{-outlineThickness, 0} /* {0, 0} */, {outlineThickness, 0},
		{-outlineThickness, outlineThickness}, {0, outlineThickness}, {outlineThickness, outlineThickness},
	}

	// Draw outline parts first
	c.SetSrc(outlineColor) // Set color to black for outline
	for _, offset := range offsets {
		// Calculate the target integer coordinates for the outline point
		targetX := startX + offset.X
		targetY := startY + offset.Y
		// Convert the target integer coordinates to the fixed.Point26_6 needed by DrawString
		offsetPt := freetype.Pt(targetX, targetY)
		_, err = c.DrawString(memeText, offsetPt)
		if err != nil {
			// Log error but continue trying to draw other parts and the main text
			log.Printf("⚠️ Error drawing outline part at offset %v: %v", offset, err)
		}
	}

	// Draw main text (fill) on top
	log.Printf("✏️ Drawing text fill...")
	c.SetSrc(fillColor)                 // Set color to white for fill
	_, err = c.DrawString(memeText, pt) // Draw at the final position
	if err != nil {
		log.Fatalf("❌ Error drawing main text: %v", err) // Fail if the main text can't be drawn
	}

	log.Printf("✏️ Text '%s' drawn on image with outline", memeText)

	// --- 8. Encode and Output PNG ---
	err = png.Encode(outputDest, rgbaImg) // Always encode as PNG
	if err != nil {
		if pe, ok := err.(*os.PathError); ok && pe.Op == "write" && pe.Path == "|1" {
			log.Println("⚠️ Broken pipe writing to stdout (likely normal if piping).")
		} else {
			log.Fatalf("❌ Error encoding PNG: %v", err)
		}
	}

	if outputDest != os.Stdout {
		log.Println("🎉 Meme generated successfully!")
	}
}

// Helper function to measure string width using font settings.
// Returns width in pixels.
func measureString(fnt *truetype.Font, size, dpi float64, hinting font.Hinting, text string) (int, error) {
	face := truetype.NewFace(fnt, &truetype.Options{
		Size:    size,
		DPI:     dpi,
		Hinting: hinting,
	})
	widthInFixedPoint := font.MeasureString(face, text)
	widthInPixels := int(widthInFixedPoint >> 6)
	return widthInPixels, nil
}
