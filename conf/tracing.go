package conf

import (
	"fmt"

	"github.com/opentracing/opentracing-go"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/opentracer"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type TracingConfig struct {
	Enabled     bool `default:"false"`
	Host        string
	Port        string
	ServiceName string `default:"gotrue" split_words:"true"`
	Tags        map[string]string
}

func (tc *TracingConfig) tracingAddr() string {
	return fmt.Sprintf("%s:%s", tc.Host, tc.Port)
}

func ConfigureTracing(tc *TracingConfig) {
	var t opentracing.Tracer = opentracing.NoopTracer{}
	if tc.Enabled {
		tracerOps := []tracer.StartOption{
			tracer.WithServiceName(tc.ServiceName),
			tracer.WithAgentAddr(tc.tracingAddr()),
		}

		for k, v := range tc.Tags {
			tracerOps = append(tracerOps, tracer.WithGlobalTag(k, v))
		}

		t = opentracer.New(tracerOps...)
	}
	opentracing.SetGlobalTracer(t)
}
