package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
)

type Histogram [16][3]int

type writer struct {
	writer io.Writer
}

type ImagePage struct {
	Urls      []string
	ImageSize int
	RowCount  int
}

type ImageUrls struct {
	sync.Mutex
	Urls []ImageUrl
}

func (a ImageUrls) Len() int           { return len(a.Urls) }
func (a ImageUrls) Swap(i, j int)      { a.Urls[i], a.Urls[j] = a.Urls[j], a.Urls[i] }
func (a ImageUrls) Less(i, j int) bool { return a.Urls[i].Index < a.Urls[j].Index }

type ImageUrl struct {
	Index int
	Url   string
}

func main() {
	port, err := setEnv()
	if err != nil {
		log.Fatal("Failed to set env variables.")
	}

	http.HandleFunc("/files/new", FileCreateHandler)
	http.Handle("/", http.FileServer(http.Dir("public")))
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Photo-mosaic Generator")
}

func FileCreateHandler(w http.ResponseWriter, r *http.Request) {
	var wg sync.WaitGroup
	imageUrls := ImageUrls{}

	r.ParseMultipartForm(32 << 20)
	file, _, err := r.FormFile("file")
	if err != nil {
		fmt.Fprint(w, "Failed to read FormFile.")
	}
	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Fprint(w, "Failed to read file contents.")
	}
	r.Body.Close()

	reader := bytes.NewReader(fileContents)

	m, _, err := image.Decode(reader)
	if err != nil {
		fmt.Println(w, "Unable to decode file.")
	}

	parentBounds := m.Bounds()

	startX := parentBounds.Min.X
	startY := parentBounds.Min.Y
	size := 30
	for parentBounds.Max.X%size != 0 {
		size = size + 1
	}

	maxX := parentBounds.Max.X
	across := int(parentBounds.Max.X / size)
	tall := int(parentBounds.Max.Y / size)

	fmt.Println(across * tall)
	for i := 0; i < across*tall; i++ {
		wg.Add(1)
		var data MediasResponse
		instagramUrl := "https://api.instagram.com/v1/tags/nofilter/media/recent?client_id=" + os.Getenv("CLIENT_ID")

		err = getInstagramData(instagramUrl, &data)
		if err != nil {
			panic(err)
		}

		go func(d MediasResponse, i int) {
			nextUrl := d.PaginationResponse.Pagination.NextUrl

			parentSubImage := m.(interface {
				SubImage(r image.Rectangle) image.Image
			}).SubImage(image.Rect(startX, startY, startX+size, startY+size))
			parentBounds := parentSubImage.Bounds()

			subImageHistogram, err := (generateHistogramFromImage(parentSubImage))
			if err != nil {
				panic(err)
			}

			imageUrl := ""
			outOfBounds := true
			for imageUrl == "" && outOfBounds {
				for _, media := range d.Medias {
					url := media.Images.Thumbnail.Url

					outOfBounds, _, err := compareMedia(url, subImageHistogram, parentBounds)
					if err != nil {
						panic(err)
					}

					if outOfBounds == false {
						imageUrl = url
						break
					}
				}
				if imageUrl == "" {
					err = getInstagramData(nextUrl, &d)
					if err != nil {
						panic(err)
					}
				}
			}
			imageUrls.Lock()
			imageUrls.Urls = append(imageUrls.Urls, ImageUrl{Index: i, Url: imageUrl})
			fmt.Println(len(imageUrls.Urls))
			imageUrls.Unlock()
			defer wg.Done()
		}(data, i)
		startX = startX + size
		if startX > maxX {
			startX = 0
			startY = startY + size
		}
	}

	wg.Wait()

	sort.Sort(ImageUrls(imageUrls))
	var urls []string
	for _, v := range imageUrls.Urls {
		urls = append(urls, v.Url)
	}

	ip := ImagePage{
		Urls:      urls,
		ImageSize: size,
		RowCount:  across,
	}

	w.WriteHeader(http.StatusCreated)
	t, _ := template.ParseFiles("./public/image.html")
	err = t.Execute(w, ip)
	if err != nil {
		fmt.Println(w, err)
	}
}

func getInstagramData(url string, data *MediasResponse) error {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url+"&count=100", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Connection", "close")

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	response, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	res.Close = true
	res.Header.Set("Connection", "close")

	err = json.Unmarshal(response, &data)
	if err != nil {
		return err
	}

	return nil
}

func compareMedia(url string, parentHistogram Histogram, parentBounds image.Rectangle) (bool, Histogram, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return true, Histogram{}, err
	}
	req.Header.Set("Connection", "close")

	res, err := client.Do(req)
	if err != nil {
		return true, Histogram{}, err
	}
	defer res.Body.Close()

	fileContent, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return true, Histogram{}, err
	}

	res.Close = true
	res.Header.Set("Connection", "close")

	histogram, compareBounds, err := generateHistogramFromContents(fileContent)
	if err != nil {
		return true, histogram, err
	}

	tolerance := 650

	parentResolution := (parentBounds.Max.X * parentBounds.Max.Y) >> 12
	compareImageRes := (compareBounds.Max.X * compareBounds.Max.Y) >> 12
	if parentResolution == 0 {
		parentResolution = 1
	}
	if compareImageRes == 0 {
		compareImageRes = 1
	}
	for i, x := range histogram {
		r, g, b := (parentHistogram[i][0]/parentResolution)-(x[0]/compareImageRes),
			(parentHistogram[i][1]/parentResolution)-(x[1]/compareImageRes),
			(parentHistogram[i][2]/parentResolution)-(x[2]/compareImageRes)
		if r > tolerance || g > tolerance || b > tolerance || r < -tolerance || g < -tolerance || b < -tolerance {
			return true, histogram, nil
			break
		}
	}

	return false, histogram, nil
}

func generateHistogramFromContents(fileContent []byte) (Histogram, image.Rectangle, error) {
	histogram := Histogram{}

	reader := bytes.NewReader(fileContent)

	m, _, err := image.Decode(reader)
	if err != nil {
		return histogram, image.Rectangle{}, err
	}
	bounds := m.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := m.At(x, y).RGBA()
			histogram[r>>12][0]++
			histogram[g>>12][1]++
			histogram[b>>12][2]++
		}
	}
	return histogram, bounds, nil
}

func generateHistogramFromImage(img image.Image) (Histogram, error) {
	histogram := Histogram{}

	bounds := img.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			histogram[r>>12][0]++
			histogram[g>>12][1]++
			histogram[b>>12][2]++
		}
	}
	return histogram, nil
}
