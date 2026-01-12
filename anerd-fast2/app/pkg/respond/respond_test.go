package respond_test

import (
	"errors"
	"gofast/pkg"
	"gofast/pkg/respond"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSON(t *testing.T) {
	t.Parallel()

	type responseData struct {
		Name string `json:"name"`
	}

	testCases := []struct {
		name           string
		data           any
		err            error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "successful response with data",
			data:           responseData{Name: "test"},
			err:            nil,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"name":"test"}`,
		},
		{
			name:           "successful response with nil data",
			data:           nil,
			err:            nil,
			expectedStatus: http.StatusNoContent,
			expectedBody:   "",
		},
		{
			name:           "successful response with empty struct",
			data:           struct{}{},
			err:            nil,
			expectedStatus: http.StatusNoContent,
			expectedBody:   "",
		},
		{
			name:           "unauthorized error",
			data:           nil,
			err:            pkg.UnauthorizedError{Err: errors.New("token expired")},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"unauthorized"}`,
		},
		{
			name:           "internal error",
			data:           nil,
			err:            pkg.InternalError{Err: errors.New("db error")},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal error"}`,
		},
		{
			name:           "bad request error",
			data:           nil,
			err:            pkg.BadRequestError{Err: errors.New("invalid input")},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"bad request"}`,
		},
		{
			name:           "not found error",
			data:           nil,
			err:            pkg.NotFoundError{Err: errors.New("user not found")},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"not found"}`,
		},
		{
			name: "validation error",
			data: nil,
			err: pkg.ValidationErrors{
				{Field: "name", Tag: "required", Message: "Name is required"},
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   `{"fields":[{"field":"name","tag":"required","message":"Name is required"}]}`,
		},
		{
			name:           "default error",
			data:           nil,
			err:            errors.New("default error"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"default error"}`,
		},
		{
			name:           "json marshal error",
			data:           make(chan int),
			err:            nil,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Error writing response\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			respond.JSON(w, r, tc.data, tc.err)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedBody != "" {
				if w.Header().Get("Content-Type") == "application/json" {
					assert.JSONEq(t, tc.expectedBody, w.Body.String())
				} else {
					assert.Equal(t, tc.expectedBody, w.Body.String())
				}
			} else {
				assert.Empty(t, w.Body.String())
			}
		})
	}
}

// This test case is to cover the scenario where writing to the response writer fails.
type failingResponseWriter struct {
	*httptest.ResponseRecorder
}

func (w *failingResponseWriter) Write(data []byte) (int, error) {
	return 0, errors.New("write error")
}

func TestJSON_WriteError(t *testing.T) {
	t.Parallel()
	w := &failingResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
	}
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	respond.JSON(w, r, map[string]string{"name": "test"}, nil)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestJSON_EncodeError(t *testing.T) {
	t.Parallel()

	// Let's test the validation error case with a failing writer to see what happens
	failingRecorder := &failingResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
	}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	respond.JSON(failingRecorder, r, nil, pkg.ValidationErrors{})
	assert.Equal(t, http.StatusUnprocessableEntity, failingRecorder.Code)

	// Let's test the unauthorized error case with a failing writer
	failingRecorder = &failingResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
	}
	respond.JSON(failingRecorder, r, nil, pkg.UnauthorizedError{})
	assert.Equal(t, http.StatusUnauthorized, failingRecorder.Code)

	// Let's test the internal error case with a failing writer
	failingRecorder = &failingResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
	}
	respond.JSON(failingRecorder, r, nil, pkg.InternalError{})
	assert.Equal(t, http.StatusInternalServerError, failingRecorder.Code)

	// Let's test the bad request error case with a failing writer
	failingRecorder = &failingResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
	}
	respond.JSON(failingRecorder, r, nil, pkg.BadRequestError{})
	assert.Equal(t, http.StatusBadRequest, failingRecorder.Code)

	// Let's test the not found error case with a failing writer
	failingRecorder = &failingResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
	}
	respond.JSON(failingRecorder, r, nil, pkg.NotFoundError{})
	assert.Equal(t, http.StatusNotFound, failingRecorder.Code)

	// Let's test the default error case with a failing writer
	failingRecorder = &failingResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
	}
	respond.JSON(failingRecorder, r, nil, errors.New("some error"))
	assert.Equal(t, http.StatusInternalServerError, failingRecorder.Code)
}
