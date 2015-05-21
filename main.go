package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	r.ParseMultipartForm(32 << 20)
	file, _, err := r.FormFile("file")
	if err != nil {
		fmt.Println(w, err)
		return
	}

	defer file.Close()

	id := random(32)

	out, err := os.OpenFile("./tmp/testfile"+id+".jpg", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println(w, "Unable to create file.")
		return
	}

	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		fmt.Println(w, err)
		return
	}

	histograms := make([]Histogram, 0)

	parent_histogram, err := generateHistogramFromFile("./tmp/testfile" + id + ".jpg")
	if err != nil {
		log.Fatal(err)
	}

	var data MediasResponse
	instagramUrl := "https://api.instagram.com/v1/tags/nofilter/media/recent?client_id=" + os.Getenv("CLIENT_ID")
	count := 300

	for len(histograms) == 0 {
		err = getInstagramData(instagramUrl, count, &data)
		if err != nil {
			log.Fatal(err)
		}

		for _, media := range data.Medias {
			fmt.Printf("Image: %v\n", media.Images.LowResolution.Url)

			out_of_bounds, histogram, err := compareMedia(media.Images.LowResolution.Url, parent_histogram)
			if err != nil {
				log.Fatal(err)
			}

			if out_of_bounds == false {
				histograms = append(histograms, histogram)
				postFile(media.Images.LowResolution.Url)
			}
		}
		instagramUrl = data.PaginationResponse.Pagination.NextUrl
	}
}

func getInstagramData(url string, count int, data *MediasResponse) (error){
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

	for i, x := range histogram {
		r, g, b := parent_histogram[i][0]-x[0], parent_histogram[i][1]-x[1], parent_histogram[i][2]-x[2]
		if r > 12000 || g > 12000 || b > 12000 || r < -12000 || g < -12000 || b < -12000 {
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
