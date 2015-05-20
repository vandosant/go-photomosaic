package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"os"
	"strings"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func random(size int) string {
	rb := make([]byte, size)
	_, err := rand.Read(rb)
	check(err)

	rs := base64.URLEncoding.EncodeToString(rb)

	return rs
}

func setEnv() (string, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"

		_, err := os.Stat("./.env")
		if err != nil {
			return port, err
		}

		f, err := os.Open("./.env")
		if err != nil {
			log.Fatal(err)
		}

		defer f.Close()

		b := make([]byte, 80)

		n, err := f.Read(b)
		if err != nil {
			return port, err
		}
		if n == 0 {
			return port, errors.New("No bytes read.")
		}

		x := strings.Split(string(b), "\n")

		for _, line := range x {
			kv := strings.Split(line, "=")
			if len(kv) == 2 {
				os.Setenv(kv[0], kv[1])
			}
		}
		return port, err
	}
	return port, nil
}
