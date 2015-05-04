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
  "crypto/rand"
  "image"
  _ "image/jpeg"
)

func main() {
  port := os.Getenv("PORT")
  if port == "" {
    port = "8080"
  }

  http.HandleFunc("/files/new", FileCreateHandler)
  http.HandleFunc("/instagram", InstagramHandler)
  http.Handle("/", http.FileServer(http.Dir("public")))
  log.Fatal(http.ListenAndServe(":"+port, nil))
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
  fmt.Fprint(w, "Photo-mosaic Generator")
}

func InstagramHandler(w http.ResponseWriter, r *http.Request) {
  err := setEnv("./.env")

  if err != nil {
    log.Fatal(err)
  }

  fmt.Fprint(w, "s")


  res, err := http.Get("https://api.instagram.com/v1/tags/nofilter/media/recent?client_id="+ os.Getenv("CLIENT_ID"))
  if err != nil {
    fmt.Fprint(w, "Failed to create request.")
  }
  json, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", json)
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

  fmt.Printf("%-14s %6s %6s %6s %6s\n", "bin", "red", "green", "blue", "alpha")
	for i, x := range histogram {
		fmt.Printf("0x%04x-0x%04x: %6d %6d %6d %6d\n", i<<12, (i+1)<<12-1, x[0], x[1], x[2], x[3])
	}

  fmt.Println(w, "File uploaded successfully")
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
