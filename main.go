package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	_ "image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type Histogram [16][4]int

type writer struct {
	writer io.Writer
}

type ImagePage struct {
	Urls      []string
	ImageSize int
	RowCount  int
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
	imageUrls := make([]string, 0)

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

	var data MediasResponse
	instagramUrl := "https://api.instagram.com/v1/tags/nofilter/media/recent?client_id=" + os.Getenv("CLIENT_ID")
	count := 100

	err = getInstagramData(instagramUrl, count, &data)
	if err != nil {
		log.Fatal(err)
	}

	startX := parentBounds.Min.X
	startY := parentBounds.Min.Y
	size := 20
	maxX := parentBounds.Max.X
	across := int(parentBounds.Max.X / size)
	tall := int(parentBounds.Max.Y / size)

	fmt.Println(w, across)
	fmt.Println(w, across*tall)
	for len(imageUrls) < across*tall {
		parentSubImage := m.(interface {
			SubImage(r image.Rectangle) image.Image
		}).SubImage(image.Rect(startX, startY, startX+size, startY+size))

		subImageHistogram, err := (generateHistogramFromImage(parentSubImage))
		if err != nil {
			log.Fatal(err)
		}

		// match this sub image
		imageUrl := ""
		for imageUrl == "" {
			for _, media := range data.Medias {
				url := media.Images.Thumbnail.Url

				out_of_bounds, _, err := compareMedia(url, subImageHistogram)
				if err != nil {
					log.Fatal(err)
				}

				if out_of_bounds == false {
					imageUrl = url
					break
				}
			}
			err = getInstagramData(data.PaginationResponse.Pagination.NextUrl, count, &data)
			if err != nil {
				log.Fatal(err)
			}
		}
		imageUrls = append(imageUrls, imageUrl)

		startX = startX + size
		if startX > maxX {
			startX = 0
			startY = startY + size
		}
		fmt.Println(len(imageUrls))
	}

	ip := ImagePage{
		Urls:      imageUrls,
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

func getInstagramData(url string, count int, data *MediasResponse) error {
	res, err := http.Get(url + "&count=" + string(count))
	if err != nil {
		return err
	}

	response, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return err
	}

	err = json.Unmarshal(response, &data)
	if err != nil {
		return err
	}

	return nil
}

func compareMedia(url string, parent_histogram Histogram) (bool, Histogram, error) {
	res, err := http.Get(url)
	if err != nil {
		return true, Histogram{}, err
	}
	defer res.Body.Close()

	file_content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return true, Histogram{}, err
	}

	histogram, err := generateHistogramFromContents(file_content)
	if err != nil {
		return true, histogram, err
	}

	tolerance := 3000

	for i, x := range histogram {
		r, g, b := parent_histogram[i][0]-x[0], parent_histogram[i][1]-x[1], parent_histogram[i][2]-x[2]
		if r > tolerance || g > tolerance || b > tolerance || r < -tolerance || g < -tolerance || b < -tolerance {
			return true, histogram, nil
			break
		}
	}

	return false, histogram, nil
}

func generateHistogramFromFile(file_path string) (Histogram, error) {
	histogram := Histogram{}

	reader, err := os.Open(file_path)
	if err != nil {
		return histogram, err
	}

	defer reader.Close()

	m, _, err := image.Decode(reader)
	if err != nil {
		return histogram, err
	}
	bounds := m.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := m.At(x, y).RGBA()
			// A color's RGBA method returns values in the range [0, 65535].
			// Shifting by 12 reduces this to the range [0, 15].
			histogram[r>>12][0]++
			histogram[g>>12][1]++
			histogram[b>>12][2]++
			histogram[a>>12][3]++
		}
	}

	return histogram, nil
}

func generateHistogramFromContents(file_content []byte) (Histogram, error) {
	histogram := Histogram{}

	reader := bytes.NewReader(file_content)

	m, _, err := image.Decode(reader)
	if err != nil {
		return histogram, err
	}
	bounds := m.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := m.At(x, y).RGBA()
			// A color's RGBA method returns values in the range [0, 65535].
			// Shifting by 12 reduces this to the range [0, 15].
			histogram[r>>12][0]++
			histogram[g>>12][1]++
			histogram[b>>12][2]++
			histogram[a>>12][3]++
		}
	}

	return histogram, nil
}

func generateHistogramFromImage(img image.Image) (Histogram, error) {
	histogram := Histogram{}

	bounds := img.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			// A color's RGBA method returns values in the range [0, 65535].
			// Shifting by 12 reduces this to the range [0, 15].
			histogram[r>>12][0]++
			histogram[g>>12][1]++
			histogram[b>>12][2]++
			histogram[a>>12][3]++
		}
	}

	return histogram, nil
}

func postFile(targetUrl string) (string, error) {
	type_ext := filepath.Ext(targetUrl)

	dir := "./tmp"
	file_name := random(20) + type_ext
	file_path := dir + "/" + file_name

	os.Mkdir(dir, 0666)

	file, err := os.Create(file_path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	res, err := http.Get(targetUrl)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	file_content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	i, err := file.Write(file_content)
	if err != nil {
		return "", err
	}
	fmt.Print(string(i))

	return file_path, nil
}
