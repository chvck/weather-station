package weatherstn

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"
)

type mockHTTPClient struct {
	req *http.Request
	mock.Mock
}

func (mh *mockHTTPClient) Do(r *http.Request) (*http.Response, error) {
	mh.req = r
	args := mh.Called(r)
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestPublisher_Process(t *testing.T) {
	var dataset []WeatherDataRow
	err := loadJSONTestDataset("unpublished_observations", &dataset)
	if err != nil {
		t.Fatalf("unexpected error loading dataset: %v", err)
	}

	mockDS := &MockDataStore{}
	mockDS.On("ReadUnpublished").Return(dataset, nil)
	mockDS.On("UpdatePublished", dataset[0].Timestamp, dataset[len(dataset)-1].Timestamp).Return(nil)

	mockCli := &mockHTTPClient{}
	resp := &http.Response{
		StatusCode: 201,
	}
	mockCli.On("Do", mock.AnythingOfType("*http.Request")).Return(resp, nil)

	endpointCfg := EndpointConfig{
		Host: "anearbyserver:111",
		SendObservations: Endpoint{
			Method: "PUT",
			Path:   "observations",
		},
	}

	publisher := NewPublisher(mockDS, endpointCfg, mockCli)
	publisher.Process()

	if !mockDS.AssertExpectations(t) {
		t.FailNow()
	}

	if !mockCli.AssertExpectations(t) {
		t.FailNow()
	}

	if mockCli.req.Host != endpointCfg.Host {
		t.Fatalf("expected host to be %s but was %s", endpointCfg.Host, mockCli.req.Host)
	}

	if mockCli.req.Method != endpointCfg.SendObservations.Method {
		t.Fatalf("expected method to be %s but was %s", endpointCfg.SendObservations.Method, mockCli.req.Method)
	}

	body, err := ioutil.ReadAll(mockCli.req.Body)
	if err != nil {
		t.Fatalf("unexpected error reading body from request: %v", err)
	}

	var jsonBody []WeatherDataRow
	err = json.Unmarshal(body, &jsonBody)
	if err != nil {
		t.Fatalf("unexpected error unmarshalling body from request: %v", err)
	}

	if len(jsonBody) != len(dataset) {
		t.Fatalf("Expected body to be len %d but was %d", len(dataset), len(body))
	}

	for i, d := range dataset {
		if d != jsonBody[i] {
			t.Fatalf("Expected observation to be %#v but was %#v", d, jsonBody[i])
		}
	}
}
