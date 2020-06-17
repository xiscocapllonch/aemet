package aemet

import (
	"flag"
	"fmt"
	"github.com/golang/freetype"
	"github.com/nfnt/resize"
	"golang.org/x/image/font"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type mapImage struct {
	id    int
	url   string
	label string
	image *image.Paletted
	err   error
}

func newImg(id, delay int, baseTime time.Time, zoneCode string, wind bool) mapImage {
	local := baseTime.Add(time.Duration(delay) * time.Hour).Local()

	return mapImage{
		id:    id,
		label: local.Format("Mon 02 Jan, 15:04"),
		url:   writeImgUrl(baseTime, delay, zoneCode, wind),
	}
}

func (m mapImage) getImg() (image.Image, error) {
	response, err := http.Get(m.url)
	if err != nil {
		return nil, err
	}

	defer func() {
		err = response.Body.Close()
	}()

	if err != nil {
		return nil, err
	}

	sourceImg, err := png.Decode(response.Body)
	if err != nil {
		return nil, err
	}

	return sourceImg, err
}

func (m mapImage) createLabelImg() (*image.RGBA, error) {
	flag.Parse()
	fontSize := 24.
	spacing := 1.5

	absPath, err := filepath.Abs("./fonts/luxisr.ttf")
	if err != nil {
		return nil, err
	}

	fontBytes, err := ioutil.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	f, err := freetype.ParseFont(fontBytes)
	if err != nil {
		return nil, err
	}

	fg, bg := image.Black, image.White
	rgba := image.NewRGBA(image.Rect(0, 0, 220, 50))
	draw.Draw(rgba, rgba.Bounds(), bg, image.ZP, draw.Src)
	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(f)
	c.SetFontSize(fontSize)
	c.SetClip(rgba.Bounds())
	c.SetDst(rgba)
	c.SetSrc(fg)
	c.SetHinting(font.HintingNone)

	pt := freetype.Pt(10, 10+int(c.PointToFixed(fontSize)>>6))

	_, err = c.DrawString(m.label, pt)
	if err != nil {
		return nil, err
	}

	pt.Y += c.PointToFixed(fontSize * spacing)

	return rgba, nil
}

func (m *mapImage) appendLabelImg(sourceImg image.Image, labelImg *image.RGBA) {
	img := image.NewRGBA(sourceImg.Bounds())

	draw.Draw(
		img,
		sourceImg.Bounds(),
		sourceImg,
		image.Point{X: 0, Y: 0},
		draw.Src,
	)

	draw.Draw(
		img,
		sourceImg.Bounds(),
		labelImg,
		image.Point{X: 0, Y: 0},
		draw.Src,
	)

	reImg := resize.Resize(700, 0, img, resize.Lanczos3)

	palettedImg := image.NewPaletted(reImg.Bounds(), palette.Plan9)
	draw.Draw(
		palettedImg,
		palettedImg.Rect,
		reImg,
		reImg.Bounds().Min,
		draw.Over,
	)

	(*m).image = palettedImg
}

func getBaseTime() time.Time {
	var initUTC int
	now := time.Now().UTC()

	switch hour := now.Hour(); {
	case hour < 7:
		now = now.AddDate(0, 0, -1)
		initUTC = 12
	case hour < 19:
		initUTC = 0
	default:
		initUTC = 12
	}

	return time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		initUTC,
		0,
		0,
		0,
		now.Location(),
	)
}

func writeImgUrl(baseTime time.Time, hourDelay int, zoneCode string, wind bool) string {
	baseUrl := "http://www.aemet.es/imagenes_d/eltiempo/prediccion/mod_maritima"

	mapType := "martot"

	if wind {
		mapType = "marvto"
	}

	return fmt.Sprintf(
		"%s/%s+%03d_%s_%s.png",
		baseUrl,
		baseTime.Format("2006010215"),
		hourDelay,
		zoneCode,
		mapType,
	)

}

func getImages(zoneCode string, forecastNbr int, wind bool) ([]*image.Paletted, error) {
	baseTime := getBaseTime()
	var images []mapImage
	var wg sync.WaitGroup

	for i := 0; i < forecastNbr; i++ {
		mapImg := newImg(i, i*3, baseTime, zoneCode, wind)
		wg.Add(1)
		go func() {
			defer wg.Done()
			sourceImg, err := mapImg.getImg()
			if err != nil {
				mapImg.err = err
				images = append(images, mapImg)
				return
			}
			labelImg, err := mapImg.createLabelImg()
			if err != nil {
				mapImg.err = err
				images = append(images, mapImg)
				return
			}
			mapImg.appendLabelImg(sourceImg, labelImg)
			images = append(images, mapImg)
		}()
	}

	wg.Wait()

	for _, mapImg := range images {
		if mapImg.err != nil {
			return nil, mapImg.err
		}
	}

	sort.Slice(images, func(i, j int) bool {
		return images[i].id < images[j].id
	})

	var result []*image.Paletted

	for _, mapImg := range images {
		result = append(result, mapImg.image)
	}

	return result, nil
}

func GetMaritimeForecastMapGIF(zoneCode string, forecastNbr, delay int, wind bool) (gif.GIF, error) {
	gifImage, err := getImages(zoneCode, forecastNbr, wind)
	if err != nil {
		return gif.GIF{}, err
	}

	var gifDelay []int

	for i := 0; i < forecastNbr; i++ {
		gifDelay = append(gifDelay, delay)
	}

	return gif.GIF{
		Image: gifImage,
		Delay: gifDelay,
	}, nil
}
