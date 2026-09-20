package mapbuilder

import (
	"bytes"
	"compress/gzip"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	slamLog, err := os.ReadFile(filepath.Join("..", "..", "test", "slam.log"))
	require.NoError(t, err)

	compressedMap, err := os.Open(filepath.Join("..", "..", "test", "navmap.ppm.gz"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, compressedMap.Close()) })

	gzipReader, err := gzip.NewReader(compressedMap)
	require.NoError(t, err)
	mapData := new(bytes.Buffer)
	_, err = mapData.ReadFrom(gzipReader)
	require.NoError(t, err)
	require.NoError(t, gzipReader.Close())

	actualData, err := Build(slamLog, mapData.Bytes())
	require.NoError(t, err)
	actual, err := png.Decode(bytes.NewReader(actualData))
	require.NoError(t, err)

	goldenFile, err := os.Open(filepath.Join("testdata", "map.png"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, goldenFile.Close()) })
	expected, err := png.Decode(goldenFile)
	require.NoError(t, err)

	assert.Equal(t, expected.Bounds(), actual.Bounds())
	assert.Equal(t, imagePixels(expected), imagePixels(actual))
}

func TestBuildRejectsInvalidPPM(t *testing.T) {
	_, err := Build([]byte("estimate 0 0 0\n"), []byte("not a PPM"))
	require.ErrorContains(t, err, "decoding PPM map")
}

func TestParseSLAMLog(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []position
		err      string
	}{
		{
			name:     "valid",
			input:    "1.0 estimate 2.5 -3.25 0.1\n",
			expected: []position{{x: -3.25, y: 2.5}},
		},
		{
			name:     "irrelevant lines",
			input:    "pause\nanything else\n0 estimate 1 2 3\n",
			expected: []position{{x: 2, y: 1}},
		},
		{
			name:  "invalid coordinate",
			input: "0 estimate one 2 3\n",
			err:   "invalid y coordinate",
		},
		{
			name:  "missing coordinate",
			input: "0 estimate 1 2\n",
			err:   "estimate requires three coordinates",
		},
		{
			name:  "no positions",
			input: "pause\nreset\n",
			err:   "no positions found",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := parseSLAMLog([]byte(test.input))
			if test.err != "" {
				require.ErrorContains(t, err, test.err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func imagePixels(img image.Image) []byte {
	bounds := img.Bounds()
	pixels := make([]byte, 0, bounds.Dx()*bounds.Dy()*4)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			pixels = append(pixels, uint8Channel(r), uint8Channel(g), uint8Channel(b), uint8Channel(a))
		}
	}
	return pixels
}

func uint8Channel(value uint32) uint8 {
	//#nosec:G115 // RGBA channels are bounded to 16 bits by image.Image.
	return uint8(value >> 8)
}
