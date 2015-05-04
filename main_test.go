package main

import (
  "testing"
  "net/http/httptest"
  "net/http"
  "strings"
  "os"
)

func TestIndexHandlerReturnsText (t *testing.T) {
  recorder := httptest.NewRecorder()

  req, err := http.NewRequest("GET", "http://example.com", nil)
  if err != nil {
    t.Errorf("Failed to create request.")
  }

  IndexHandler(recorder, req)

  expected := "Photo-mosaic Generator"

  result := recorder.Body.String()

  if strings.Contains(result, expected) != true {
    t.Errorf("json format incorrect. Actual: %s, Expected: %s", result, expected)
  }
}

func TestSettingEnvVariables (t *testing.T) {
  // var env []string
    err := setEnv("./.env")
    if err != nil {
      t.Errorf("Failed to set env variables. %s", err)
    }
    // env = os.Environ()

    // fmt.Println("List of Environtment variables : \n")

    // for index, value := range env {
      //  name := strings.Split(value, "=") // split by = sign
      //  fmt.Printf("[%d] %s : %v\n", index, name[0], name[1])
    // }

    v := os.Getenv("TEST")
    expected := "someuuid1234"
    if v != expected {
      t.Errorf("env value incorrect. Actual: %s, Expected: %s", v, expected)
    }

}
