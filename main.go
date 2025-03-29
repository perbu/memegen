package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
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
	dpi              = 72.0  // Screen DPI
	fontSize         = 144.0 // Font size in points
	paddingY         = 20    // Padding from the top edge
	outlineThickness = 2     // Outline width in pixels
)

// Define colors
var (
	fillColor    = image.White // Color for the text fill
	outlineColor = image.Black // Color for the text outline
)

// main handles command-line argument parsing, calls the core run function,
// and manages program exit status based on errors.
func main() {
	if len(os.Args) < 2 || os.Args[1] == "" {
		// Print usage instructions to standard error
		fmt.Fprintf(os.Stderr, "Usage: %s \"<text>\" [output.png]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  <text>: The text to draw on the image.\n")
		fmt.Fprintf(os.Stderr, "  [output.png]: Optional output PNG filename. If omitted, writes PNG to stdout.\n")
		os.Exit(1) // Exit with error status 1
	}

	memeText := strings.ToUpper(os.Args[1])
	outputFilename := ""
	if len(os.Args) > 2 {
		outputFilename = os.Args[2]
		// Simple check and warning for non-PNG extension
		if !strings.HasSuffix(strings.ToLower(outputFilename), ".png") {
			fmt.Fprintf(os.Stderr, "Warning: Output filename '%s' does not end with .png. Appending .png\n", outputFilename)
			outputFilename += ".png"
		}
	}

	// Execute the main application logic
	err := run(memeText, outputFilename)
	if err != nil {
		// Print any error returned by run() to standard error
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1) // Exit with error status 1
	}

	// If writing to a file and successful, print a confirmation message
	if outputFilename != "" {
		fmt.Printf("Successfully generated meme to %s\n", outputFilename)
	}
	// If writing to stdout, no explicit success message is printed here.
	// The PNG data will be on stdout.
}

// run encapsulates the core logic of loading resources, generating the image,
// and writing the output. It returns an error if any step fails.
func run(memeText, outputFilename string) error {
	// --- 1. Determine Output Destination ---
	var destWriter io.Writer = os.Stdout // Default to standard output
	if outputFilename != "" {
		outFile, err := os.Create(outputFilename)
		if err != nil {
			return fmt.Errorf("creating output file '%s': %w", outputFilename, err)
		}
		defer outFile.Close() // Ensure file is closed when function returns
		destWriter = outFile
	}

	// --- 2. Load Template Image ---
	imgReader := bytes.NewReader(templateImageBytes)
	baseImg, _, err := image.Decode(imgReader) // Format is not used, ignore it
	if err != nil {
		return fmt.Errorf("decoding template image: %w", err)
	}

	// --- 3. Load Font ---
	ttFont, err := freetype.ParseFont(fontBytes)
	if err != nil {
		return fmt.Errorf("parsing font: %w", err)
	}

	// --- 4. Prepare Drawing Canvas ---
	bounds := baseImg.Bounds()
	// Create a new RGBA image to draw on. This ensures we have an image
	// type that supports setting individual pixel colors.
	rgbaImg := image.NewRGBA(bounds)
	draw.Draw(rgbaImg, bounds, baseImg, image.Point{}, draw.Src)

	// --- 5. Setup Text Drawing Context ---
	c := freetype.NewContext()
	c.SetDPI(dpi)
	c.SetFont(ttFont)
	c.SetFontSize(fontSize)
	c.SetClip(rgbaImg.Bounds())
	c.SetDst(rgbaImg)
	c.SetHinting(font.HintingFull) // Improve font rendering quality

	// --- 6. Calculate Text Position (Centered, near TOP) ---
	textWidth, err := measureString(ttFont, fontSize, dpi, font.HintingFull, memeText)
	if err != nil {
		return fmt.Errorf("measuring text width: %w", err)
	}

	imageWidth := bounds.Dx()
	// Calculate starting X for centered text
	startX := (imageWidth - textWidth) / 2
	if startX < 0 {
		startX = 0 // Prevent negative start X if text is wider than image
	}

	// Calculate starting Y (baseline position) near the TOP
	// baseline = top padding + approximate font ascent
	// Using fontSize * dpi / 72.0 provides a reasonable pixel height estimate.
	startY := paddingY + int(c.PointToFixed(fontSize)>>6) // More accurate ascent calculation

	pt := freetype.Pt(startX, startY) // Baseline point for the main text

	// --- 7. Draw the Text with Outline ---

	// Define offsets for the 8 directions around the center for the outline
	offsets := []image.Point{
		{-outlineThickness, -outlineThickness}, {0, -outlineThickness}, {outlineThickness, -outlineThickness},
		{-outlineThickness, 0} /* {0, 0} is the center, skip */, {outlineThickness, 0},
		{-outlineThickness, outlineThickness}, {0, outlineThickness}, {outlineThickness, outlineThickness},
	}

	// Draw outline parts first
	c.SetSrc(outlineColor) // Set color to black for outline
	for _, offset := range offsets {
		offsetPt := freetype.Pt(startX+offset.X, startY+offset.Y)
		_, err = c.DrawString(memeText, offsetPt)
		if err != nil {
			// Return error if any part of the outline fails to draw
			return fmt.Errorf("drawing outline part at offset %v: %w", offset, err)
		}
	}

	// Draw main text (fill) on top
	c.SetSrc(fillColor)                 // Set color to white for fill
	_, err = c.DrawString(memeText, pt) // Draw at the final baseline position
	if err != nil {
		// Return error if the main text fill fails to draw
		return fmt.Errorf("drawing main text fill: %w", err)
	}

	// --- 8. Encode and Output PNG ---
	// Use the determined destination writer (stdout or file)
	err = png.Encode(destWriter, rgbaImg)
	if err != nil {
		// Check specifically for broken pipe when writing to stdout, which can be normal
		// if the reading end of the pipe closes early.
		if opErr, ok := err.(*os.PathError); ok && opErr.Op == "write" && destWriter == os.Stdout {
			// Suppress broken pipe errors when writing to stdout, but still return it
			// as something did technically go wrong. Caller (main) ignores it if needed.
			// Or we could return nil here if we consider broken pipe non-fatal.
			// Let's return the error for now, main will exit non-zero.
			return fmt.Errorf("writing PNG to stdout: %w", err)
		}
		// For other file write errors or general encoding errors
		return fmt.Errorf("encoding or writing PNG: %w", err)
	}

	// If we reached here, all steps were successful
	return nil
}

// measureString calculates the width of a string in pixels when rendered
// with the specified font properties.
func measureString(fnt *truetype.Font, size, dpi float64, hinting font.Hinting, text string) (int, error) {
	// Create a font face with the given options
	face := truetype.NewFace(fnt, &truetype.Options{
		Size:    size,
		DPI:     dpi,
		Hinting: hinting,
	})

	// Measure the string using the font face
	// font.MeasureString returns width in 26.6 fixed-point units
	widthInFixedPoint := font.MeasureString(face, text)

	// Convert fixed-point to integer pixels (shift right by 6 bits)
	widthInPixels := int(widthInFixedPoint >> 6)

	// face.Close() // truetype.NewFace doesn't require explicit closing

	return widthInPixels, nil
}
