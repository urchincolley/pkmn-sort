package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/nfnt/resize"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"math"
	"os"
	"regexp"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func readFileToImg(fname string) image.Image {
	data, err := ioutil.ReadFile(fname)
	check(err)
	r := bytes.NewReader(data)
	img, err := png.Decode(r)
	check(err)
	return img
}

func getAverageColor(img image.Image) color.RGBA {
	rBucket, gBucket, bBucket, pxCount := 0, 0, 0, 0
	width, height := img.Bounds().Dx(), img.Bounds().Dy()
	for i := 0; i < width; i++ {
		for j := 0; j < height; j++ {
			r, g, b, a := img.At(i, j).RGBA()
			// Only count fully opaque pixels for average color
			if a == 65535 {
				rBucket += int(r)
				gBucket += int(g)
				bBucket += int(b)
				pxCount++
			}
		}
	}
	avg := color.RGBA{
		uint8(rBucket / pxCount >> 8),
		uint8(gBucket / pxCount >> 8),
		uint8(bBucket / pxCount >> 8),
		255,
	}
	return avg
}

func Round(f float64) int {
	return int(math.Floor(f + .5))
}

func RGBAtoHSV(c color.RGBA) (int, float64, float64) {
	fR, fG, fB := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	min, max := math.Min(math.Min(fR, fG), fB), math.Max(math.Max(fR, fG), fB)
	h, s, v := 0, 0.0, max
	d := max - min

	if d != 0 {
		switch max {
		case fR:
			h = Round(60*((fG-fB)/d+6)) % 360
			break
		case fG:
			h = Round(60 * ((fB-fR)/d + 2))
			break
		case fB:
			h = Round(60 * ((fR-fG)/d + 4))
		}
	}

	if max != 0 {
		s = d / max
	}

	return h, s, v
}

func main() {
	// Parse flags
	//   -w: target image width
	//   -h: target image height
	//   -m: maximum width or height of component images (resizing will occur to normalize size)
	//   -s: silhouette mode switch (when true, components images will be represented by silhouettes of the image's average color)
	tw := flag.Int("w", 4000, "-w requires an integer width for the target image")
	th := flag.Int("h", 4000, "-h requires an integer height for the target image")
	maxSize := flag.Int("m", 400, "-m requires an integer for the desired normalized size of source images")
	silhouette := flag.Bool("s", false, "-s requires a boolean to specify silhouette mode")
	flag.Parse()

	// Read source directory
	srcDir := os.Args[len(os.Args)-1]
	files, err := ioutil.ReadDir(srcDir)
	check(err)

	// Initialize target image
	newImg := image.NewRGBA(image.Rect(0, 0, *tw+*maxSize, *th+*maxSize))
	for i := 0; i < newImg.Bounds().Dx(); i++ {
		for j := 0; j < newImg.Bounds().Dy(); j++ {
			newImg.Set(i, j, color.RGBA{255, 255, 255, 255})
		}
	}

	for _, file := range files {
		if match, _ := regexp.MatchString("^.+png$", file.Name()); match {
			// Read component file, compute average color, and convert to HSV
			srcImg := readFileToImg(fmt.Sprintf("%s/%s", srcDir, file.Name()))
			avgRGBA := getAverageColor(srcImg)
			H, _, V := RGBAtoHSV(avgRGBA)

			// Resize component image
			sx, sy := float64(srcImg.Bounds().Dx()), float64(srcImg.Bounds().Dy())
			smax, rmax := math.Max(sx, sy), float64(*maxSize)
			rx, ry := *maxSize, *maxSize
			if smax == sx {
				ry = int((rmax / smax) * sy)
			} else {
				rx = int((rmax / smax) * sx)
			}
			resizedImg := resize.Resize(uint(rx), uint(ry), srcImg, resize.NearestNeighbor)

			// Copy component image to target image
			nx, ny := *maxSize/2+*tw*H/360-rx/2, *maxSize/2+int(float64(*th)*V)-ry/2
			for i := 0; i < rx; i++ {
				for j := 0; j < ry; j++ {
					srcCol := resizedImg.At(i, j)
					_, _, _, a := srcCol.RGBA()
					if *silhouette && a == 65535 {
						newImg.Set(nx+i, ny+j, avgRGBA)
					} else {
						if a > 0 {
							newImg.Set(nx+i, ny+j, srcCol)
						}
					}
				}
			}
		}
	}

	// Write target image
	buf := new(bytes.Buffer)
	err = png.Encode(buf, newImg)
	ioutil.WriteFile("sorted.png", buf.Bytes(), 0644)
}
