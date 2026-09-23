package platform

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorEnvelope(t *testing.T) {
	w := httptest.NewRecorder()
	Wrap(func(http.ResponseWriter, *http.Request) (any, error) {
		return nil, Fail("INSUFFICIENT_STOCK", "Unavailable")
	})(w, httptest.NewRequest("POST", "/inventory", nil))
	require.Equal(t, 409, w.Code)
	var v map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &v))
	require.Equal(t, "INSUFFICIENT_STOCK", v["code"])
	require.NotEmpty(t, v["traceId"])
	require.NotEmpty(t, w.Header().Get("X-Correlation-ID"))
}
func TestBodyRejectsExtraJSON(t *testing.T) {
	for _, body := range []string{`{"n":1} {"n":2}`, `{"unknown":1}`} {
		_, e := Body[struct {
			N int `json:"n"`
		}](httptest.NewRecorder(), httptest.NewRequest("POST", "/", strings.NewReader(body)))
		require.Error(t, e)
	}
}
