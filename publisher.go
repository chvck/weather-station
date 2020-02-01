package weatherstn

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
)

// Endpoint represents a target endpoint.
type Endpoint struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// EndpointConfig represents the configuration of all endpoints that the publisher will send to.
type EndpointConfig struct {
	Host             string   `json:"host"`
	SendObservations Endpoint `json:"sendObservations"`
}

// Publisher is responsible for sending data upstream.
type Publisher struct {
	endpointConfig EndpointConfig
	datastore      DataStore
	cli            PublisherHTTPClient
	stopCh         chan struct{}
}

// PublisherHTTPClient is the http client that will be used by a Publisher.
type PublisherHTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// NewPublisher creates a new Publisher.
func NewPublisher(store DataStore, config EndpointConfig, cli PublisherHTTPClient) *Publisher {
	return &Publisher{
		datastore:      store,
		endpointConfig: config,
		cli:            cli,
		stopCh:         make(chan struct{}),
	}
}

// Run starts the publisher send loop.
func (p *Publisher) Run(interval time.Duration) {
	for {
		select {
		case <-p.stopCh:
			return
		case <-time.After(interval):
		}

		p.Process()
	}
}

// Process is called each Run iteration and is exposed for testing.
func (p *Publisher) Process() {
	unpublishedObs, err := p.datastore.ReadUnpublished()
	if err != nil {
		log.WithError(err).
			WithField("component", "Publisher").
			WithField("event", "Run").
			Error("failed to read unpublished observations from store")
		return
	}

	if len(unpublishedObs) == 0 {
		log.WithError(err).
			WithField("component", "Publisher").
			WithField("event", "Run").
			Info("no unpublished observations seen")
		return
	}

	body, err := json.Marshal(unpublishedObs)
	if err != nil {
		log.WithError(err).
			WithField("component", "Publisher").
			WithField("event", "Run").
			Error("failed to marshal unpublished observations")
		return
	}

	req, err := http.NewRequest(
		p.endpointConfig.SendObservations.Method,
		(&url.URL{
			Scheme: "https",
			Host:   p.endpointConfig.Host,
			Path:   p.endpointConfig.SendObservations.Path,
		}).String(),
		ioutil.NopCloser(bytes.NewReader(body)),
	)
	if err != nil {
		log.WithError(err).
			WithField("component", "Publisher").
			WithField("event", "Run").
			Error("failed to create request")
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.cli.Do(req)
	if err != nil {
		log.WithError(err).
			WithField("component", "Publisher").
			WithField("event", "Run").
			Error("failed to send http request")
		return
	}

	if resp.StatusCode != http.StatusCreated {
		log.WithField("component", "Publisher").
			WithField("event", "Run").
			WithField("statusCode", resp.StatusCode).
			Error("unexpected status code received")
		return
	}

	err = p.datastore.UpdatePublished(
		unpublishedObs[0].Timestamp,
		unpublishedObs[len(unpublishedObs)-1].Timestamp,
	)
	if err != nil {
		log.WithError(err).
			WithField("component", "Publisher").
			WithField("event", "Run").
			Error("failed to update published rows")
	}
}

// Stop causes the run loop to be halted, returning once the run loop has completed any work.
func (p *Publisher) Stop() {
	p.stopCh <- struct{}{}
	return
}
