package conf

import (
	"os"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	defer os.Clearenv()
	os.Exit(m.Run())
}

func TestGlobal(t *testing.T) {
	os.Setenv("GOTRUE_DB_DRIVER", "mysql")
	os.Setenv("GOTRUE_DB_DATABASE_URL", "fake")
	os.Setenv("GOTRUE_OPERATOR_TOKEN", "token")
	os.Setenv("GOTRUE_API_REQUEST_ID_HEADER", "X-Request-ID")
	gc, err := LoadGlobal("")
	require.NoError(t, err)
	require.NotNil(t, gc)
	assert.Equal(t, "X-Request-ID", gc.API.RequestIDHeader)
}

func TestTracing(t *testing.T) {
	os.Setenv("GOTRUE_DB_DRIVER", "mysql")
	os.Setenv("GOTRUE_DB_DATABASE_URL", "fake")
	os.Setenv("GOTRUE_OPERATOR_TOKEN", "token")
	os.Setenv("GOTRUE_TRACING_SERVICE_NAME", "identity")
	os.Setenv("GOTRUE_TRACING_PORT", "8126")
	os.Setenv("GOTRUE_TRACING_HOST", "127.0.0.1")
	os.Setenv("GOTRUE_TRACING_TAGS", "tag1:value1,tag2:value2")

	gc, _ := LoadGlobal("")
	tc := opentracing.GlobalTracer()

	assert.Equal(t, opentracing.NoopTracer{}, tc)
	assert.Equal(t, false, gc.Tracing.Enabled)
	assert.Equal(t, "identity", gc.Tracing.ServiceName)
	assert.Equal(t, "8126", gc.Tracing.Port)
	assert.Equal(t, "127.0.0.1", gc.Tracing.Host)
	assert.Equal(t, map[string]string{"tag1": "value1", "tag2": "value2"}, gc.Tracing.Tags)
}
