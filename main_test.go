package main

import (
  "testing"
  "net/http/httptest"
  "net/http"
  "strings"
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
