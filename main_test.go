package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestIndexHandlerReturnsText(t *testing.T) {
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

func TestSettingEnvVariables(t *testing.T) {
	err := setEnv("./specs/examples/.envtest")
	if err != nil {
		t.Errorf("Failed to set env variables. %s", err)
	}

	v := os.Getenv("TEST")
	expected := "someuuid1234"
	if v != expected {
		t.Errorf("env value incorrect. Actual: %s, Expected: %s", v, expected)
	}

}
