package main

import (
  "net/http"
  "log"
  "fmt"
  "os"
  "io"
  "io/ioutil"
  "encoding/json"
  "image"
  "path/filepath"
  "bytes"
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

  histograms := make([][16][4]int, 0)

  parent_histogram, err := generateHistogramFromFile("./tmp/testfile"+id+".jpg")
  if err != nil {
    log.Fatal(err)
  }
  fmt.Printf("%-14s %6s %6s %6s %6s\n", "bin", "red", "green", "blue", "alpha")
  for i, x := range parent_histogram {
    fmt.Printf("hist: r - %d, g - %d, b - %d\n", x[0], x[1], x[2])
    fmt.Printf("0x%04x-0x%04x: %6d %6d %6d %6d\n", i<<12, (i+1)<<12-1, x[0], x[1], x[2], x[3])
  }

  err = setEnv()
  if err != nil {
    fmt.Fprint(w, "Failed to set env variables.")
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

  for _, media := range data.Medias {
    fmt.Printf("Image: %v\n", media.Images.LowResolution.Url)

    res, err := http.Get(media.Images.LowResolution.Url)
    if err != nil {
      log.Fatal(err)
    }
    defer res.Body.Close()

    file_content, err := ioutil.ReadAll(res.Body)
    if err != nil {
      log.Fatal(err)
    }

    histogram, err := generateHistogramFromContents(file_content)
    if err != nil {
      log.Fatal(err)
    }

    out_of_bounds := false
    for i, x := range histogram {
      r, g, b := parent_histogram[i][0] - x[0], parent_histogram[i][1] - x[1], parent_histogram[i][2] - x[2]
      if r > 10000 || g > 10000 || b > 10000 || r < -10000 || g < -10000 || b < -10000 {
        out_of_bounds = true
        fmt.Println("breaking...")
        break
      }
    }

    if out_of_bounds == false {
      histograms = append(histograms, histogram)
      fmt.Printf("WINNER: %-14s %6s %6s %6s %6s\n", "bin", "red", "green", "blue", "alpha")
      for i, x := range parent_histogram {
        fmt.Printf("hist: r - %d, g - %d, b - %d\n", x[0], x[1], x[2])
        fmt.Printf("0x%04x-0x%04x: %6d %6d %6d %6d\n", i<<12, (i+1)<<12-1, x[0], x[1], x[2], x[3])
      }

      postFile(media.Images.LowResolution.Url)
    }
  }
}

func generateHistogramFromFile(file_path string) ([16][4]int, error) {
  var histogram [16][4]int

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

func generateHistogramFromContents(file_content []byte) ([16][4]int, error) {
  var histogram [16][4]int

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
