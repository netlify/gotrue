package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gobuffalo/uuid"
	"github.com/netlify/gotrue/conf"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TracerTestSuite struct {
	suite.Suite
	API    *API
	Config *conf.Configuration

	instanceID uuid.UUID
}

func TestTracer(t *testing.T) {
	api, config, instanceID, err := setupAPIForTestForInstance()
	require.NoError(t, err)

	ts := &TracerTestSuite{
		API:        api,
		Config:     config,
		instanceID: instanceID,
	}
	defer api.db.Close()

	suite.Run(t, ts)
}

func (ts *TracerTestSuite) TestTracer_Spans() {
	spans := []string{}

	tt := &testTracer{spans: &spans, tags: &map[string]string{}}
	opentracing.SetGlobalTracer(tt)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://localhost/something1", nil)
	ts.API.handler.ServeHTTP(w, req)
	req = httptest.NewRequest(http.MethodPost, "http://localhost/something2", nil)
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusNotFound, w.Code)
	assert.Equal(ts.T(), spans, []string{"http.handler", "http.handler"})
}

func (ts *TracerTestSuite) TestTracer_Tags() {
	tags := map[string]string{}

	tt := &testTracer{tags: &tags, spans: &[]string{}}
	opentracing.SetGlobalTracer(tt)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://localhost/something", nil)
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusNotFound, w.Code)
	assert.Equal(ts.T(), tags["http.method"], "POST")
	assert.Equal(ts.T(), tags["http.status_code"], "404")
	assert.Equal(ts.T(), tags["http.url"], "/something")
	assert.Equal(ts.T(), tags["resource.name"], "POST /something")
	assert.NotEmpty(ts.T(), tags["http.request_id"])
}

type testTracer struct {
	noopTracer opentracing.NoopTracer
	spans      *[]string
	tags       *map[string]string
}

func (tt testTracer) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {
	*tt.spans = append(*(tt.spans), operationName)
	noopSpan := tt.noopTracer.StartSpan(operationName, opts...)
	return &testSpan{tags: tt.tags, noopSpan: noopSpan}
}
func (tt testTracer) Inject(sm opentracing.SpanContext, format interface{}, carrier interface{}) error {
	return nil
}
func (tt testTracer) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	return tt.noopTracer.Extract(format, carrier)
}

type testSpan struct {
	noopSpan opentracing.Span
	tags     *map[string]string
}

func (ts testSpan) Finish()                                          {}
func (ts testSpan) FinishWithOptions(opts opentracing.FinishOptions) {}
func (ts testSpan) Context() opentracing.SpanContext                 { return ts.noopSpan.Context() }
func (ts testSpan) SetOperationName(operationName string) opentracing.Span {
	return ts.noopSpan.SetOperationName(operationName)
}
func (ts testSpan) SetTag(key string, value interface{}) opentracing.Span {
	(*ts.tags)[key] = fmt.Sprintf("%v", value)
	return ts.noopSpan.SetTag(key, value)
}
func (ts testSpan) LogFields(fields ...log.Field)             {}
func (ts testSpan) LogKV(alternatingKeyValues ...interface{}) {}
func (ts testSpan) SetBaggageItem(restrictedKey, value string) opentracing.Span {
	return ts.noopSpan.SetBaggageItem(restrictedKey, value)
}
func (ts testSpan) BaggageItem(restrictedKey string) string {
	return ts.noopSpan.BaggageItem(restrictedKey)
}
func (ts testSpan) Tracer() opentracing.Tracer                            { return ts.noopSpan.Tracer() }
func (ts testSpan) LogEvent(event string)                                 {}
func (ts testSpan) LogEventWithPayload(event string, payload interface{}) {}
func (ts testSpan) Log(data opentracing.LogData)                          {}
