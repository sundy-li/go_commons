package qrpie

import (
	"encoding/csv"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/vp8l"
	_ "golang.org/x/image/webp"

	"github.com/wswz/go_commons/log"
)

const (
	vecLen    = 5 //手动提取了5个特征
	Threshold = 0.6
)

type Qr struct {
	once  sync.Once
	model [][]string
}

//输入为决策树model的路径
func NewQr(modelPath string) *Qr {
	cs, err := os.Open(modelPath)
	if err != nil {
		panic(err.Error())
	}
	reader := csv.NewReader(cs)
	m, err := reader.ReadAll()
	if err != nil {
		panic(err.Error())
	}
	return &Qr{
		model: m,
	}
}

func loadImage(path string) (img image.Image, f string, err error) {
	file, err := os.Open(path)
	if err != nil {
		log.Error(err.Error())
		return
	}
	defer file.Close()
	img, f, err = image.Decode(file)
	return
}

func extractFeature(img image.Image) []float64 {
	width := img.Bounds().Size().X
	height := img.Bounds().Size().Y
	f1 := 0
	f2 := 0
	cc := true
	x := 0
	y := 0
	features := make([]float64, 0, vecLen)
	grayImg := image.NewGray(img.Bounds())
	th := grayMean(img)
	cluster := NewCluster()
	for h := 0; h < height; h++ {
		list := make([]int, 5, 5)
		p := 0
		for w := 0; w < width; w++ {
			c := color.GrayModel.Convert(img.At(w, h))
			if float64(c.(color.Gray).Y) < th {
				if cc || w == width-1 {
					p = (p + 1) % 5
					if isDemandBiLy(list, p) {
						if x == w || y == h {
							f2++
						}
						x = w
						y = h
						f1++
						cluster.add(newPoint(w, h))

					}
					list[p] = 0
				}
				cc = false
				list[p] = list[p] + 1
				grayImg.SetGray(w, h, color.Gray{Y: 0})

			} else {
				if !cc || w == width-1 {
					p = (p + 1) % 5
					if isDemandBiLy(list, p) {
						if x == w || y == h {
							f2++
						}
						x = w
						y = h
						f1++
						cluster.add(newPoint(w, h))
					}

					list[p] = 0
				}
				cc = true
				list[p] = list[p] + 1
				grayImg.SetGray(w, h, color.Gray{Y: 255})
			}
		}

	}

	features = append(features, float64(f1)/float64(height))
	features = append(features, float64(f2)/float64(height))
	features = append(features, getFeatureFromCluster(cluster.getMaxThreeHeapCenter()))
	features = append(features, float64(cluster.getLenth())/float64(height))
	features = append(features, cluster.getVariance())
	return features
}

//抽样获取灰度平均值，用10000个点来抽样
func grayMean(img image.Image) float64 {
	num := 10000
	width := img.Bounds().Size().X
	height := img.Bounds().Size().Y
	mean := 0.0
	for i := 0; i < num; i++ {
		c := color.GrayModel.Convert(img.At(rand.Intn(width), rand.Intn(height)))
		mean = float64(c.(color.Gray).Y) + mean
	}
	return mean / float64(num)
}

func getFeatureFromCluster(points []point) float64 {
	if len(points) < 3 {
		return 3
	}
	fn := func(p1 point, p2 point) float64 {
		return math.Abs(p1.x*p2.x+p1.y*p2.y) / distant(p1, point{0, 0}) / distant(p2, point{0, 0})
	}

	result := math.MaxFloat64
	for i, _ := range points {
		vec1 := pointMinus(points[i], points[(i+1)%3])
		vec2 := pointMinus(points[i], points[(i+2)%3])
		r := fn(vec1, vec2)
		r = r + math.Min(fn(vec1, point{0, 1}), fn(vec1, point{1, 0}))
		r = r + math.Min(fn(vec2, point{0, 1}), fn(vec2, point{1, 0}))
		if r < result {
			result = r
		}

	}

	return result
}

func isDemandBiLy(list []int, p int) bool {
	p = p % 5
	if isSim(list[(p+1)%5], list[p]) && isSim(list[(p+1)%5], list[(p+3)%5]) && isSim(list[(p+1)%5], list[(p+4)%5]) && isSim(list[(p+2)%5], list[(p+3)%5]*3) {
		if list[p] < 2 {
			return false
		}
		return true
	} else {
		return false
	}
}

func isSim(x int, y int) bool {
	if y == 0 {
		return false
	}
	if math.Abs((float64(x)-float64(y))/math.Max(float64(x), float64(y))) <= 0.4 {
		return true
	} else {
		return false
	}
}

//此方法是用来产生训练数据的
//qrpath:放二维码图片的文件夹地址
//other: 放非二维码图片的文件夹地址
//name : 产生的训练数据文件的文件名
func GenerateTrainData(qrPath string, other string, name string) (err error) {
	dirs := []string{qrPath, other}
	file, _ := os.Create(name)
	writer := csv.NewWriter(file)
	header := make([]string, 0, vecLen)
	header = append(header, "/")
	for i := 0; i < vecLen; i++ {
		f := "f" + strconv.Itoa(i+1)
		header = append(header, f)
	}
	header = append(header, "y")
	writer.Write(header)
	fail := 0

	for i, dir := range dirs {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, file := range files {
			if file.IsDir() {
				continue
			} else {
				img, _, err := loadImage(dir + "/" + file.Name())
				if err != nil {
					log.Debugf("load img fail error msg is %s,fileName is %s", err.Error(), file.Name())
					fail++
					continue
				}
				features := extractFeature(img)
				log.Debug(file.Name())
				record := make([]string, 0, vecLen+1)
				record = append(record, file.Name())
				for _, s := range features {
					record = append(record, strconv.FormatFloat(s, 'f', -1, 64))
				}
				if i == 0 {
					record = append(record, strconv.Itoa(1))
				} else {
					record = append(record, strconv.Itoa(0))
				}
				writer.Write(record)
			}
		}
	}

	writer.Flush()
	file.Close()
	return
}

func (q *Qr) predict(features []float64) bool {
	fm := make(map[string]float64)
	for i := 0; i < vecLen; i++ {
		key := "f" + strconv.Itoa(i+1)
		fm[key] = features[i]
	}
	i := 0
	var gain float64
	nextNode := "0-0"
	for _, record := range q.model {
		if i == 0 {
			i = 1
			continue
		}
		if nextNode != record[2] && string(record[2][2]) != "0" {
			continue
		}
		if record[3] != "Leaf" {
			split, _ := strconv.ParseFloat(record[4], 64)
			if fm[record[3]] < split {
				nextNode = record[5]
			} else {
				nextNode = record[6]
			}

		} else {
			g, _ := strconv.ParseFloat(record[8], 64)
			gain += g

		}
	}
	if math.Exp(gain)/(1+math.Exp(gain)) > Threshold {
		return true
	}
	return false
}

func (q *Qr) IsQr(img image.Image) (bool, error) {
	features := extractFeature(img)
	return q.predict(features), nil
}

func downLoadImg(url string) (image.Image, string, error) {
	req, _ := http.NewRequest("GET", url, nil)
	client := http.Client{
		Timeout: time.Duration(5) * time.Second,
	}
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	response, e := client.Do(req)
	if e != nil {
		return nil, ",", e
	}
	defer response.Body.Close()
	img, f, err := image.Decode(response.Body)
	return img, f, err
}

func (q *Qr) IsQrUrl(url string) (bool, error) {
	img, _, err := downLoadImg(url)
	if err == nil {
		return q.IsQr(img)
	} else {
		return false, err
	}
}

func (q *Qr) IsQrPath(path string) (bool, error) {
	img, _, err := loadImage(path)
	if err == nil {
		return q.IsQr(img)
	} else {
		return false, err
	}
}

type point struct {
	x float64
	y float64
}

func newPoint(x, y int) point {
	return point{float64(x), float64(y)}
}

func pointAdd(p1 point, p2 point) point {
	return point{p1.x + p2.x, p1.y + p2.y}
}

func pointMinus(p1 point, p2 point) point {
	return point{p1.x - p2.x, p1.y - p2.y}
}

func distant(p1 point, p2 point) float64 {
	d := math.Sqrt(math.Pow(p1.x-p2.x, 2) + math.Pow(p1.y-p2.y, 2))
	return d
}
