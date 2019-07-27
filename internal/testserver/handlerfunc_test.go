package testserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticJSONHandler(t *testing.T) {
	// start server
	s := New()
	s.Start()
	defer s.Stop()

	// set server response to static json
	want := map[string]interface{}{
		"field1": "one",
		"field2": 2.0,
	}
	s.HandlerFunc = StaticJSONHandler(want, http.StatusCreated)

	// get response using http client
	resp, err := s.Client().Get(s.URL())
	assert.Nil(t, err)
	defer func() { _ = resp.Body.Close() }()

	// assert response body matches expected
	assert.Equal(t, resp.StatusCode, http.StatusCreated)
	got := make(map[string]interface{})
	err = json.NewDecoder(resp.Body).Decode(&got)
	assert.Nil(t, err)
	assert.Equal(t, want, got)
}

func TestValidateJSONBodyHandler(t *testing.T) {
	// start server
	s := New()
	s.Start()
	defer s.Stop()

	// set server handler
	reqBody := map[string]interface{}{
		"field1": "one",
		"field2": 2.0,
	}
	respBody := map[string]interface{}{
		"field2": "one",
		"field1": 2.0,
	}
	s.HandlerFunc = ValidateJSONBodyHandler(t, reqBody, respBody, http.StatusAccepted, "wrong body")

	// get response using http client
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(reqBody)
	assert.Nil(t, err)
	resp, err := s.Client().Post(s.URL(), "application/json", &buf)
	assert.Nil(t, err)
	defer func() { _ = resp.Body.Close() }()

	// assert response body matches expected
	assert.Equal(t, resp.StatusCode, http.StatusAccepted)
	got := make(map[string]interface{})
	err = json.NewDecoder(resp.Body).Decode(&got)
	assert.Nil(t, err)
	assert.Equal(t, respBody, got)
}

func TestJSONObjectToMap(t *testing.T) {
	tests := []struct {
		object interface{}
		want   map[string]interface{}
	}{
		{struct{ A string }{"value"}, map[string]interface{}{"A": "value"}},
		{struct{ B int }{3}, map[string]interface{}{"B": 3.0}},
		{struct{ C bool }{true}, map[string]interface{}{"C": true}},
	}

	for _, test := range tests {
		got, err := jsonObjectToMap(test.object)
		assert.Nil(t, err)
		assert.Equal(t, test.want, got)
	}
}
