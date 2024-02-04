package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"
)

const (
	contentTypeHeader    = "Content-Type"
	externalDNSMediaType = "application/external.dns.webhook+json;version=1"
)

func NewServer(port int64, p provider.Provider, readTimeout, writeTimeout time.Duration) *http.Server {
	mux := chi.NewMux()
	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.Heartbeat("/healthz"))
	mux.Get("/", handleNegotiation(p))
	mux.Get("/records", handleGetRecords(p))
	mux.Post("/records", handleApplyChanges(p))
	mux.Post("/adjustendpoints", handleAdjustEndpoints(p))

	return &http.Server{
		Handler:      mux,
		Addr:         fmt.Sprintf(":%d", port),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}
}

func handleNegotiation(p provider.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, err := json.Marshal(p.GetDomainFilter())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		log.Debug().Str("action", "handleNegotiation").RawJSON("domain_filter", raw).Send()
		w.Header().Set(contentTypeHeader, externalDNSMediaType)
		w.WriteHeader(http.StatusOK)
		w.Write(raw)
	}
}

func handleGetRecords(p provider.Provider) http.HandlerFunc {
	log := log.With().Str("action", "handleGetRecords").Logger()

	return func(w http.ResponseWriter, r *http.Request) {
		records, err := p.Records(r.Context())
		if err != nil {
			log.Error().Err(err).Msg("failed to get records")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		raw, err := json.Marshal(records)
		if err != nil {
			log.Error().Err(err).Msg("failed to marshal records to json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		log.Debug().RawJSON("records", raw).Send()
		w.Header().Set(contentTypeHeader, externalDNSMediaType)
		w.WriteHeader(http.StatusOK)
		w.Write(raw)
	}
}

func handleApplyChanges(p provider.Provider) http.HandlerFunc {
	log := log.With().Str("action", "handleApplyChanges").Logger()

	return func(w http.ResponseWriter, r *http.Request) {
		var changes plan.Changes
		if err := json.NewDecoder(r.Body).Decode(&changes); err != nil {
			log.Error().Err(err).Msg("failed to decode changes")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(http.StatusText(http.StatusBadRequest)))
			return
		}

		if err := p.ApplyChanges(r.Context(), &changes); err != nil {
			log.Error().Err(err).Any("changes", changes).Msg("failed to apply changes")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		log.Debug().Any("changes", changes).Send()
		w.WriteHeader(http.StatusNoContent)
		w.Write([]byte(http.StatusText(http.StatusNoContent)))
	}
}

func handleAdjustEndpoints(p provider.Provider) http.HandlerFunc {
	log := log.With().Str("action", "handleAdjustEndpoints").Logger()

	return func(w http.ResponseWriter, r *http.Request) {
		var endpoints []*endpoint.Endpoint
		if err := json.NewDecoder(r.Body).Decode(&endpoints); err != nil {
			log.Error().Err(err).Msg("failed to decode endpoints")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(http.StatusText(http.StatusBadRequest)))
			return
		}

		endpoints, err := p.AdjustEndpoints(endpoints)
		if err != nil {
			log.Error().Err(err).Any("endpoints", endpoints).Msg("failed to adjust endpoints")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		raw, err := json.Marshal(endpoints)
		if err != nil {
			log.Error().Err(err).Any("endpoints", endpoints).Msg("failed to marshal endpoints to json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		log.Debug().Any("endpoints", endpoints).Send()
		w.Header().Set(contentTypeHeader, externalDNSMediaType)
		w.WriteHeader(http.StatusOK)
		w.Write(raw)
	}
}
