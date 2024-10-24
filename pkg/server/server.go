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
	log := log.With().Str("action", "handleNegotiation").Logger()

	return func(w http.ResponseWriter, r *http.Request) {
		raw, err := json.Marshal(p.GetDomainFilter())
		if err != nil {
			err = fmt.Errorf("failed to marshal response: %w", err)
			log.Error().Err(err).Send()
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		log.Debug().RawJSON("domain_filter", raw).Send()
		w.Header().Set(contentTypeHeader, externalDNSMediaType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(raw)
	}
}

func handleGetRecords(p provider.Provider) http.HandlerFunc {
	log := log.With().Str("action", "handleGetRecords").Logger()

	return func(w http.ResponseWriter, r *http.Request) {
		records, err := p.Records(r.Context())
		if err != nil {
			err = fmt.Errorf("failed to get records: %w", err)
			log.Error().Err(err).Send()
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		raw, err := json.Marshal(records)
		if err != nil {
			err = fmt.Errorf("failed to marshal records to json: %w", err)
			log.Error().Err(err).Send()
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		log.Debug().RawJSON("records", raw).Send()
		w.Header().Set(contentTypeHeader, externalDNSMediaType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(raw)
	}
}

func handleApplyChanges(p provider.Provider) http.HandlerFunc {
	log := log.With().Str("action", "handleApplyChanges").Logger()

	return func(w http.ResponseWriter, r *http.Request) {
		var changes plan.Changes
		if err := json.NewDecoder(r.Body).Decode(&changes); err != nil {
			err = fmt.Errorf("failed to decode changes: %w", err)
			log.Error().Err(err).Send()
			w.WriteHeader(http.StatusBadRequest)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(http.StatusText(http.StatusBadRequest)))
			return
		}

		if err := p.ApplyChanges(r.Context(), &changes); err != nil {
			err = fmt.Errorf("failed to apply changes: %w", err)
			log.Error().Err(err).Any("changes", changes).Send()
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		log.Debug().Any("changes", changes).Send()
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleAdjustEndpoints(p provider.Provider) http.HandlerFunc {
	log := log.With().Str("action", "handleAdjustEndpoints").Logger()

	return func(w http.ResponseWriter, r *http.Request) {
		var endpoints []*endpoint.Endpoint
		if err := json.NewDecoder(r.Body).Decode(&endpoints); err != nil {
			err = fmt.Errorf("failed to decode endpoints: %w", err)
			log.Error().Err(err).Send()
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(http.StatusText(http.StatusBadRequest)))
			return
		}

		endpoints, err := p.AdjustEndpoints(endpoints)
		if err != nil {
			err = fmt.Errorf("failed to adjust endpoints: %w", err)
			log.Error().Err(err).Any("endpoints", endpoints).Send()
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		raw, err := json.Marshal(endpoints)
		if err != nil {
			err = fmt.Errorf("failed to marshal endpoints to json: %w", err)
			log.Error().Err(err).Any("endpoints", endpoints).Send()
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		log.Debug().RawJSON("endpoints", raw).Send()
		w.Header().Set(contentTypeHeader, externalDNSMediaType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(raw)
	}
}
