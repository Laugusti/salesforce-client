package restclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/Laugusti/go-sforce/credentials"
	"github.com/Laugusti/go-sforce/internal/testserver"
	"github.com/Laugusti/go-sforce/session"
	"github.com/stretchr/testify/assert"
)

const (
	accessToken = "MOCK_TOKEN"
	apiVersion  = "mock"
)

var (
	loginSuccessHandler = func(w http.ResponseWriter, r *http.Request) {
		serverURL := fmt.Sprintf("http://%s", r.Host)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(session.RequestToken{
			AccessToken: accessToken,
			InstanceURL: serverURL,
		})
	}
	unauthorizedHandler = testserver.StaticJSONHandler(&testing.T{}, APIError{
		Message:   "Session expired or invalid",
		ErrorCode: "INVALID_SESSION_ID",
	}, http.StatusUnauthorized)

	// api error
	genericErr = APIError{Message: "Generic API error", ErrorCode: "GENERIC_ERROR"}

	// request validators
	jsonContentTypeValidator = &testserver.HeaderValidator{Key: "Content-Type", Value: "application/json"}
	authTokenValidator       = &testserver.HeaderValidator{Key: "Authorization", Value: "Bearer " + accessToken}
	emptyQueryValidator      = &testserver.QueryValidator{Query: url.Values{}}
	emptyBodyValidator       = &testserver.JSONBodyValidator{Body: nil}
	getMethodValidator       = &testserver.MethodValidator{Method: http.MethodGet}
	postMethodValidator      = &testserver.MethodValidator{Method: http.MethodPost}
	patchMethodValidator     = &testserver.MethodValidator{Method: http.MethodPatch}
	deleteMethodValidator    = &testserver.MethodValidator{Method: http.MethodDelete}
)

func createClientAndServer(t *testing.T) (*Client, *testserver.Server) {
	// start server
	s := testserver.New(t)

	// create session and login
	s.HandlerFunc = loginSuccessHandler
	sess := session.Must(session.New(s.URL(), apiVersion, credentials.New("user", "pass", "cid", "csecret")))
	if err := sess.Login(); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, s.RequestCount, "expected single request (login)")
	s.RequestCount = 0 // reset counter

	// create client
	client := &Client{sess, s.Client()}

	return client, s
}

func TestCreateSObject(t *testing.T) {
	client, server := createClientAndServer(t)
	defer server.Stop()

	tests := []struct {
		objectType   string
		object       SObject
		statusCode   int
		requestCount int
		errSnippet   string
	}{
		{"", nil, 0, 0, "sobject name is required"},
		{"Object", nil, 0, 0, "sobject value is required"},
		{"Object", map[string]interface{}{}, 0, 0, "sobject value is required"},
		{"", map[string]interface{}{"Field1": "one", "Field2": 2}, 0, 0, "sobject name is required"},
		{"Object", map[string]interface{}{"Field1": "one", "Field2": 2}, 201, 1, ""},
		{"Object", map[string]interface{}{"Field1": "one", "Field2": 2}, 400, 1, "GENERIC_ERROR"},
	}

	for _, test := range tests {
		assertMsg := fmt.Sprintf("input: %v", test)
		path := fmt.Sprintf("/services/data/%s/sobjects/%s", apiVersion, test.objectType)
		validators := []testserver.RequestValidator{authTokenValidator, jsonContentTypeValidator,
			emptyQueryValidator, &testserver.JSONBodyValidator{Body: test.object},
			&testserver.PathValidator{Path: path}, postMethodValidator}
		requestFunc := func() (interface{}, error) {
			return client.CreateSObject(test.objectType, test.object)
		}
		successFunc := func(res interface{}) {
			if assert.NotNil(t, res, assertMsg) {
				assert.True(t, res.(*UpsertResult).Success, assertMsg)
			}
		}
		handler := &testserver.JSONResponseHandler{
			StatusCode: test.statusCode,
			Body:       UpsertResult{ID: "id", Success: true, Errors: []interface{}{}},
		}

		assertRequest(t, assertMsg, server, test.errSnippet, requestFunc, successFunc,
			test.requestCount, validators, handler)
	}
}

func TestGetSObject(t *testing.T) {
	client, server := createClientAndServer(t)
	defer server.Stop()

	tests := []struct {
		objectType   string
		objectID     string
		statusCode   int
		requestCount int
		errSnippet   string
		wantedObject SObject
	}{
		{"", "", 0, 0, "sobject name is required", nil},
		{"", "A", 0, 0, "sobject name is required", nil},
		{"Object", "", 0, 0, "sobject id is required", nil},
		{"Object", "A", 200, 1, "", map[string]interface{}{"A": "one", "B": 2.0, "C": true}},
		{"Object", "A", 400, 1, "GENERIC_ERROR", nil},
	}

	for _, test := range tests {
		assertMsg := fmt.Sprintf("input: %v", test)
		path := fmt.Sprintf("/services/data/%s/sobjects/%s/%s", apiVersion, test.objectType,
			test.objectID)
		validators := []testserver.RequestValidator{authTokenValidator, jsonContentTypeValidator,
			emptyQueryValidator, emptyBodyValidator, &testserver.PathValidator{Path: path},
			getMethodValidator}

		requestFunc := func() (interface{}, error) {
			return client.GetSObject(test.objectType, test.objectID)
		}
		successFunc := func(res interface{}) {
			if assert.NotNil(t, res, assertMsg) {
				assert.Equal(t, test.wantedObject, res, assertMsg)
			}
		}
		handler := &testserver.JSONResponseHandler{
			StatusCode: test.statusCode,
			Body:       test.wantedObject,
		}

		assertRequest(t, assertMsg, server, test.errSnippet, requestFunc, successFunc,
			test.requestCount, validators, handler)
	}
}

func TestGetSObjectByExternalID(t *testing.T) {
	client, server := createClientAndServer(t)
	defer server.Stop()

	tests := []struct {
		objectType      string
		externalIDField string
		externalID      string
		wantedObject    SObject
		statusCode      int
		requestCount    int
		errSnippet      string
	}{
		{"", "", "", nil, 0, 0, "sobject name is required"},
		{"", "A", "", nil, 0, 0, "sobject name is required"},
		{"", "A", "a", nil, 0, 0, "sobject name is required"},
		{"Object", "", "", nil, 0, 0, "external id field is required"},
		{"Object", "", "a", nil, 0, 0, "external id field is required"},
		{"Object", "A", "", nil, 0, 0, "external id is required"},
		{"Object", "A", "a", map[string]interface{}{"A": "one", "B": 2.0, "C": true}, 200, 1, ""},
		{"Object", "A", "a", nil, 400, 1, "GENERIC_ERROR"},
	}

	for _, test := range tests {
		assertMsg := fmt.Sprintf("input: %v", test)
		path := fmt.Sprintf("/services/data/%s/sobjects/%s/%s/%s", apiVersion, test.objectType,
			test.externalIDField, test.externalID)
		validators := []testserver.RequestValidator{authTokenValidator, jsonContentTypeValidator,
			emptyQueryValidator, emptyBodyValidator, &testserver.PathValidator{Path: path},
			getMethodValidator}

		requestFunc := func() (interface{}, error) {
			return client.GetSObjectByExternalID(test.objectType,
				test.externalIDField, test.externalID)
		}
		successFunc := func(res interface{}) {
			if assert.NotNil(t, res, assertMsg) {
				assert.Equal(t, test.wantedObject, res, assertMsg)
			}
		}
		handler := &testserver.JSONResponseHandler{
			StatusCode: test.statusCode,
			Body:       test.wantedObject,
		}

		assertRequest(t, assertMsg, server, test.errSnippet, requestFunc, successFunc,
			test.requestCount, validators, handler)
	}
}

func TestUpsertSObject(t *testing.T) {
	client, server := createClientAndServer(t)
	defer server.Stop()

	tests := []struct {
		objectType   string
		objectID     string
		object       SObject
		statusCode   int
		requestCount int
		errSnippet   string
	}{
		{"", "", nil, 0, 0, "sobject name is required"},
		{"", "A", nil, 0, 0, "sobject name is required"},
		{"", "A", map[string]interface{}{"A": "one", "B": 2}, 0, 0, "sobject name is required"},
		{"Object", "", nil, 0, 0, "sobject id is required"},
		{"Object", "", map[string]interface{}{"A": "one", "B": 2}, 0, 0, "sobject id is required"},
		{"Object", "A", nil, 0, 0, "sobject value is required"},
		{"Object", "A", map[string]interface{}{}, 0, 0, "sobject value is required"},
		{"Object", "A", map[string]interface{}{"A": "one", "B": 2}, 200, 1, ""},
		{"Object", "A", map[string]interface{}{"A": "one", "B": 2}, 400, 1, "GENERIC_ERROR"},
	}

	for _, test := range tests {
		assertMsg := fmt.Sprintf("input: %v", test)
		path := fmt.Sprintf("/services/data/%s/sobjects/%s/%s", apiVersion, test.objectType,
			test.objectID)
		validators := []testserver.RequestValidator{authTokenValidator, jsonContentTypeValidator,
			emptyQueryValidator, &testserver.JSONBodyValidator{Body: test.object},
			&testserver.PathValidator{Path: path}, patchMethodValidator}

		requestFunc := func() (interface{}, error) {
			return client.UpsertSObject(test.objectType, test.objectID, test.object)
		}
		successFunc := func(res interface{}) {
			if assert.NotNil(t, res, assertMsg) {
				assert.True(t, res.(*UpsertResult).Success, assertMsg)
				assert.Equal(t, test.objectID, res.(*UpsertResult).ID, assertMsg)
			}
		}
		handler := &testserver.JSONResponseHandler{
			StatusCode: test.statusCode,
			Body:       UpsertResult{ID: test.objectID, Success: true, Errors: []interface{}{}},
		}

		assertRequest(t, assertMsg, server, test.errSnippet, requestFunc, successFunc,
			test.requestCount, validators, handler)
	}
}

func TestUpsertSObjectByExternalID(t *testing.T) {
	client, server := createClientAndServer(t)
	defer server.Stop()

	tests := []struct {
		objectType      string
		externalIDField string
		externalID      string
		object          SObject
		statusCode      int
		requestCount    int
		errSnippet      string
	}{
		{"", "", "", nil, 0, 0, "sobject name is required"},
		{"", "A", "", nil, 0, 0, "sobject name is required"},
		{"", "A", "a", nil, 0, 0, "sobject name is required"},
		{"", "A", "a", map[string]interface{}{"A": "one", "B": 2}, 0, 0, "sobject name is required"},
		{"Object", "", "", nil, 0, 0, "external id field is required"},
		{"Object", "", "a", nil, 0, 0, "external id field is required"},
		{"Object", "", "a", map[string]interface{}{"A": "one", "B": 2}, 0, 0, "external id field is required"},
		{"Object", "A", "", nil, 0, 0, "external id is required"},
		{"Object", "A", "", map[string]interface{}{"A": "one", "B": 2}, 0, 0, "external id is required"},
		{"Object", "A", "a", nil, 0, 0, "sobject value is required"},
		{"Object", "A", "a", map[string]interface{}{}, 0, 0, "sobject value is required"},
		{"Object", "A", "a", map[string]interface{}{"A": "one", "B": 2}, 200, 1, ""},
		{"Object", "A", "a", map[string]interface{}{"A": "one", "B": 2}, 400, 1, "GENERIC_ERROR"},
	}

	for _, test := range tests {
		assertMsg := fmt.Sprintf("input: %v", test)
		path := fmt.Sprintf("/services/data/%s/sobjects/%s/%s/%s", apiVersion, test.objectType,
			test.externalIDField, test.externalID)
		validators := []testserver.RequestValidator{authTokenValidator, jsonContentTypeValidator,
			emptyQueryValidator, &testserver.JSONBodyValidator{Body: test.object},
			&testserver.PathValidator{Path: path}, patchMethodValidator}

		requestFunc := func() (interface{}, error) {
			return client.UpsertSObjectByExternalID(test.objectType, test.externalIDField,
				test.externalID, test.object)
		}
		successFunc := func(res interface{}) {
			if assert.NotNil(t, res, assertMsg) {
				assert.True(t, res.(*UpsertResult).Success, assertMsg)
			}
		}
		handler := &testserver.JSONResponseHandler{
			StatusCode: test.statusCode,
			Body:       UpsertResult{ID: "id", Success: true, Errors: []interface{}{}},
		}

		assertRequest(t, assertMsg, server, test.errSnippet, requestFunc, successFunc,
			test.requestCount, validators, handler)
	}
}

func TestDeleteSObject(t *testing.T) {
	client, server := createClientAndServer(t)
	defer server.Stop()

	tests := []struct {
		objectType   string
		objectID     string
		statusCode   int
		requestCount int
		errSnippet   string
	}{
		{"", "", 0, 0, "sobject name is required"},
		{"", "A", 0, 0, "sobject name is required"},
		{"Object", "", 0, 0, "sobject id is required"},
		{"Object", "A", 204, 1, ""},
		{"Object", "A", 400, 1, "GENERIC_ERROR"},
	}

	for _, test := range tests {
		assertMsg := fmt.Sprintf("input: %v", test)
		path := fmt.Sprintf("/services/data/%s/sobjects/%s/%s", apiVersion, test.objectType,
			test.objectID)
		validators := []testserver.RequestValidator{authTokenValidator, jsonContentTypeValidator,
			emptyQueryValidator, emptyBodyValidator, &testserver.PathValidator{Path: path},
			deleteMethodValidator}

		requestFunc := func() (interface{}, error) {
			return nil, client.DeleteSObject(test.objectType, test.objectID)
		}
		successFunc := func(res interface{}) {
			assert.Nil(t, res, assertMsg)
		}
		handler := &testserver.JSONResponseHandler{
			StatusCode: test.statusCode,
			Body:       nil,
		}

		assertRequest(t, assertMsg, server, test.errSnippet, requestFunc, successFunc,
			test.requestCount, validators, handler)
	}
}

func TestUnauthorizedClient(t *testing.T) {
	client, server := createClientAndServer(t)
	defer server.Stop()
	// server handler return 401

	server.HandlerFunc = unauthorizedHandler
	_, err := client.CreateSObject("Object", map[string]interface{}{"A": "B"})
	assert.NotNil(t, err, "expected client error")
	assert.Contains(t, err.Error(), "INVALID_SESSION_ID", "expected invalid session response")
	assert.Equal(t, 2, server.RequestCount, "expected 2 request (create and login)")

	server.RequestCount = 0 // reset counter
	// 1st request fails, 2nd returns login, other return upsert result
	server.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		switch server.RequestCount {
		case 0:
			t.Error("request count can't be 0")
		case 1:
			unauthorizedHandler(w, r)
		case 2:
			loginSuccessHandler(w, r)
		default:
			testserver.StaticJSONHandler(t, UpsertResult{"id", true, nil}, http.StatusCreated)(w, r)
		}
	}
	_, err = client.CreateSObject("Object", map[string]interface{}{"A": "B"})
	assert.Nil(t, err, "client request should've succeeded")
	// 3 requests (create POST and login POST and retry create POST)
	assert.Equal(t, 3, server.RequestCount, "expected 3 requests (create, login, retry)")
}

func assertRequest(t *testing.T, assertMsg string, server *testserver.Server, wantErr string,
	invokeFunc func() (interface{}, error), successFunc func(interface{}),
	expectedRequestCount int, validators []testserver.RequestValidator,
	respHandler *testserver.JSONResponseHandler) {
	shouldErr := wantErr != ""
	// set server response
	if shouldErr {
		respHandler.Body = genericErr
	}
	server.HandlerFunc = testserver.ValidateAndSetResponseHandler(t, assertMsg, respHandler, validators...)

	// invoke request
	server.RequestCount = 0 // reset counter
	res, err := invokeFunc()

	// assertions
	assert.Equal(t, expectedRequestCount, server.RequestCount, assertMsg)
	if shouldErr {
		if assert.Error(t, err, assertMsg) {
			assert.Contains(t, err.Error(), wantErr, assertMsg)
		}
		assert.Nil(t, res, assertMsg)
	} else {
		assert.Nil(t, err, assertMsg)
		if successFunc != nil {
			successFunc(res)
		}
	}
}
