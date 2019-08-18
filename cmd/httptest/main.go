package main

import (
	"context"
	"net/http"
	"os"

	"github.com/datawire/teleproxy/pkg/dlog"
)

func main() {
	log := dlog.GetLogger(context.Background())

	body := os.Getenv("HTTPTEST_BODY")
	if body == "" {
		body = "HTTPTEST"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(body))
		if err != nil {
			log.Print(err)
		}
	})

	log.Error(http.ListenAndServe(":8080", nil))
	os.Exit(1)
}
