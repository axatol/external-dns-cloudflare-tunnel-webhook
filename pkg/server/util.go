package server

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

func write(w http.ResponseWriter, status int, body []byte) {
	w.WriteHeader(status)

	raw := []byte(http.StatusText(status))
	if body != nil {
		raw = body
	}

	if _, err := w.Write(raw); err != nil {
		log.Error().Err(fmt.Errorf("failed to write response: %s", err)).Send()
	}
}
