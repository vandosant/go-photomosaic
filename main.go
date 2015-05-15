package main

import (
  "net/http"
  "log"
  "fmt"
  "os"
  "io"
  "strings"
  "errors"
  "io/ioutil"
  "encoding/base64"
  "encoding/json"
  "crypto/rand"
  "image"
  "path/filepath"
  "time"
  "strconv"
  _ "image/jpeg"
)

func main() {
  port := os.Getenv("PORT")
  if port == "" {
    port = "8080"
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

  out, err := os.OpenFile("./tmp/testfile"+id, os.O_WRONLY|os.O_CREATE, 0666)
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

  reader, err := os.Open("./tmp/testfile"+id)
  if err != nil {
    log.Fatal(err)
  }

  defer reader.Close()

  m, _, err := image.Decode(reader)
	if err != nil {
		log.Fatal(err)
	}
	bounds := m.Bounds()

  var histogram [16][4]int
  histograms := make([][16][4]int, 0)

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
  histograms = append(histograms, histogram)

  fmt.Printf("%-14s %6s %6s %6s %6s\n", "bin", "red", "green", "blue", "alpha")
	for i, x := range histogram {
		fmt.Printf("0x%04x-0x%04x: %6d %6d %6d %6d\n", i<<12, (i+1)<<12-1, x[0], x[1], x[2], x[3])
	}

  fmt.Println(w, "File uploaded successfully")

  if _, err := os.Stat("./.env"); err == nil {
    err := setEnv("./.env")
    if err != nil {
      log.Fatal(err)
    }
  }

  res, err := http.Get("https://api.instagram.com/v1/tags/nofilter/media/recent?client_id="+ os.Getenv("CLIENT_ID"))
  if err != nil {
    fmt.Fprint(w, "Failed to create request.")
  }

  response, err := ioutil.ReadAll(res.Body)
  res.Body.Close()
  if err != nil {
    log.Fatal(err)
  }

  var data MediasResponse

  err = json.Unmarshal(response, &data)
  if err != nil {
    log.Fatal(err)
  }
  fmt.Printf("Results: %v\n", data)
  fmt.Printf("Medias: %v\n", data.Medias[0].Images.LowResolution.Url)

  file_path, err := postFile(data.Medias[0].Images.LowResolution.Url)
  if err != nil {
    log.Fatal(err)
  }

  fmt.Fprint(w, file_path)
}

func postFile(targetUrl string) (string, error) {
  type_ext := filepath.Ext(targetUrl)

  dir, err := dir_full_path()
  if err != nil {
    return "", err
  }

  file_name := random(20) + type_ext
  file_path := dir + file_name

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

func dir_full_path() (string, error) {
    path, err := filepath.Abs("tmp")

    if err != nil {
        return "", err
    }

    t := time.Now()

    s := path +
        string(os.PathSeparator) +
        strconv.Itoa(t.Day()) +
        "_" +
        strconv.Itoa(int(t.Month())) +
        "_" +
        strconv.Itoa(t.Year()) +
        string(os.PathSeparator)

    return s, nil
}

type MediasResponse struct {
    MetaResponse
    Medias []Media `json:"data"`
}

type MetaResponse struct {
    Meta *Meta
}

type Meta struct {
    Code         int
    ErrorType    string `json:"error_type"`
    ErrorMessage string `json:"error_message"`
}

type Media struct {
    Images       *Images
}

type Images struct {
    LowResolution      *Image `json:"low_resolution"`
    Thumbnail          *Image
    StandardResolution *Image `json:"standard_resolution"`
}

type Image struct {
    Url    string
    Width  int64
    Height int64
}

// helpers
func check(e error) {
    if e != nil {
        panic(e)
    }
}

func random(size int) string {
  rb := make([]byte,size)
  _, err := rand.Read(rb)
  check(err)

  rs := base64.URLEncoding.EncodeToString(rb)

  return rs
}

func setEnv(p string) (error) {
  f, err := os.Open(p)
  check(err)

  defer f.Close()

  b := make([]byte, 80)

  n, err := f.Read(b)
  if err != nil {
    return err
  }
  if n == 0 {
    return errors.New("No bytes read.")
  }

  x := strings.Split(string(b), "\n")

  for _, line := range x {
    kv := strings.Split(line, "=")
    if len(kv) == 2 {
      os.Setenv(kv[0], kv[1])
    }
      }
  return err
}
