// Package mapbuilder renders a vacuum route on top of a PPM map.
package mapbuilder

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/spakin/netpbm"
)

const (
	coordinateScale = 80  // Rendered pixels per SLAM coordinate unit.
	cropMargin      = 20  // Rendered pixels retained around the content bounding box.
	estimateFields  = 4   // Fields in an estimate entry: keyword, y, x, and z.
	fullChannel     = 255 // Maximum value of an 8-bit color channel.
	lightGray       = 250 // 8-bit channel value of the source map's light gray.
	markerRadius    = 4   // Rendered pixels from the marker center to each edge.
	resizeFactor    = 4   // Dimensionless source-map enlargement factor.
	yellowGreen     = 204 // Green-channel value of the source map's yellow.
)

type (
	position struct {
		x float64
		y float64
	}
)

var (
	backgroundColor  = color.NRGBA{R: 24, G: 102, B: 181, A: 255}  // Dark blue.
	currentSpotColor = color.NRGBA{R: 248, G: 205, B: 71, A: 255}  // Yellow.
	floorColor       = color.NRGBA{R: 33, G: 117, B: 197, A: 255}  // Blue.
	originalGray     = color.NRGBA{R: 125, G: 125, B: 125, A: 255} // Gray.
	traceColor       = color.NRGBA{R: 127, G: 179, B: 224, A: 255} // Light blue.
	wallColor        = color.NRGBA{R: 98, G: 202, B: 255, A: 255}  // Cyan.
)

// Build renders the positions in slamLog onto the PPM image in mapData and
// returns the resulting PNG.
func Build(slamLog, mapData []byte) ([]byte, error) {
	positions, err := parseSLAMLog(slamLog)
	if err != nil {
		return nil, fmt.Errorf("parsing SLAM log: %w", err)
	}

	decoded, err := netpbm.Decode(bytes.NewReader(mapData), &netpbm.DecodeOptions{
		Target: netpbm.PPM,
		Exact:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("decoding PPM map: %w", err)
	}

	result, err := render(decoded, positions)
	if err != nil {
		return nil, fmt.Errorf("rendering map: %w", err)
	}

	var output bytes.Buffer
	if err = png.Encode(&output, result); err != nil {
		return nil, fmt.Errorf("encoding PNG map: %w", err)
	}

	return output.Bytes(), nil
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func boundingBox(img *image.NRGBA) (image.Rectangle, bool) {
	bounds := img.Bounds()
	box := image.Rectangle{Min: bounds.Max, Max: bounds.Min}
	found := false

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if img.NRGBAAt(x, y) == originalGray {
				continue
			}

			found = true
			if x < box.Min.X {
				box.Min.X = x
			}
			if y < box.Min.Y {
				box.Min.Y = y
			}
			if x >= box.Max.X {
				box.Max.X = x + 1
			}
			if y >= box.Max.Y {
				box.Max.Y = y + 1
			}
		}
	}

	return box, found
}

func drawLine(img *image.NRGBA, from, to image.Point) {
	dx := abs(to.X - from.X)
	dy := -abs(to.Y - from.Y)
	sx, sy := -1, -1
	if from.X < to.X {
		sx = 1
	}
	if from.Y < to.Y {
		sy = 1
	}
	err := dx + dy

	for {
		if from.In(img.Bounds()) {
			img.SetNRGBA(from.X, from.Y, traceColor)
		}
		if from == to {
			return
		}

		e2 := 2 * err
		if e2 >= dy {
			err += dy
			from.X += sx
		}
		if e2 <= dx {
			err += dx
			from.Y += sy
		}
	}
}

func estimateIndex(fields []string) int {
	for i, field := range fields {
		if field == "estimate" {
			return i
		}
	}
	return -1
}

func parseEstimate(fields []string, estimate, lineNumber int) (position, error) {
	if len(fields) != estimate+estimateFields {
		return position{}, fmt.Errorf("line %d: estimate requires three coordinates", lineNumber)
	}

	y, err := parseFiniteCoordinate(fields[estimate+1])
	if err != nil {
		return position{}, fmt.Errorf("line %d: invalid y coordinate %q", lineNumber, fields[estimate+1])
	}
	x, err := parseFiniteCoordinate(fields[estimate+2])
	if err != nil {
		return position{}, fmt.Errorf("line %d: invalid x coordinate %q", lineNumber, fields[estimate+2])
	}
	if _, err = parseFiniteCoordinate(fields[estimate+3]); err != nil {
		return position{}, fmt.Errorf("line %d: invalid z coordinate %q", lineNumber, fields[estimate+3])
	}

	return position{x: x, y: y}, nil
}

func parseFiniteCoordinate(value string) (float64, error) {
	coordinate, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsInf(coordinate, 0) || math.IsNaN(coordinate) {
		return 0, fmt.Errorf("coordinate is not finite")
	}
	return coordinate, nil
}

func parseSLAMLog(data []byte) ([]position, error) {
	var positions []position
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		fields := strings.Fields(scanner.Text())
		estimate := estimateIndex(fields)
		if estimate == -1 {
			continue
		}

		pos, err := parseEstimate(fields, estimate, lineNumber)
		if err != nil {
			return nil, err
		}
		positions = append(positions, pos)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading log: %w", err)
	}
	if len(positions) == 0 {
		return nil, fmt.Errorf("no positions found")
	}

	return positions, nil
}

func render(source image.Image, positions []position) (*image.NRGBA, error) {
	bounds := source.Bounds()
	img := imaging.Resize(source, bounds.Dx()*resizeFactor, bounds.Dy()*resizeFactor, imaging.NearestNeighbor)

	centerX := float64(img.Bounds().Dx()) / 2
	centerY := float64(img.Bounds().Dy()) / 2
	points := make([]image.Point, len(positions))
	for i, pos := range positions {
		points[i] = image.Pt(
			int(math.Round(centerX+pos.y*coordinateScale)),
			img.Bounds().Dy()-1-int(math.Round(centerY+pos.x*coordinateScale)),
		)
		if i > 0 {
			drawLine(img, points[i-1], points[i])
		}
	}

	last := points[len(points)-1]
	for y := last.Y - markerRadius; y <= last.Y+markerRadius; y++ {
		for x := last.X - markerRadius; x <= last.X+markerRadius; x++ {
			if image.Pt(x, y).In(img.Bounds()) {
				img.SetNRGBA(x, y, currentSpotColor)
			}
		}
	}

	box, ok := boundingBox(img)
	if !ok {
		return nil, fmt.Errorf("unable to determine image bounding box")
	}
	box = image.Rect(
		max(box.Min.X-cropMargin, img.Bounds().Min.X),
		max(box.Min.Y-cropMargin, img.Bounds().Min.Y),
		min(box.Max.X+cropMargin, img.Bounds().Max.X),
		min(box.Max.Y+cropMargin, img.Bounds().Max.Y),
	)
	img = imaging.Crop(img, box)
	replaceColors(img)

	return img, nil
}

func replaceColors(img *image.NRGBA) {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			current := img.NRGBAAt(x, y)
			switch current {
			case color.NRGBA{A: fullChannel}:
				img.SetNRGBA(x, y, wallColor)
			case color.NRGBA{B: fullChannel, A: fullChannel},
				color.NRGBA{R: lightGray, G: lightGray, B: lightGray, A: fullChannel},
				color.NRGBA{R: fullChannel, B: fullChannel, A: fullChannel},
				color.NRGBA{R: fullChannel, A: fullChannel},
				color.NRGBA{R: fullChannel, G: yellowGreen, A: fullChannel},
				color.NRGBA{R: fullChannel, G: fullChannel, B: fullChannel, A: fullChannel}:
				img.SetNRGBA(x, y, floorColor)
			case originalGray:
				img.SetNRGBA(x, y, backgroundColor)
			}
		}
	}
}
