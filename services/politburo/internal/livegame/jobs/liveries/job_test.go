package liveries

import (
	"context"
	"strings"
	"testing"

	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

type liveriesClientStub struct {
	liveries []infiniteflight.Livery
	err      error
}

func (s liveriesClientStub) GetAircraftLiveries(context.Context) ([]infiniteflight.Livery, error) {
	return s.liveries, s.err
}

func TestJobName(t *testing.T) {
	job := New(liveriesClientStub{}, gameliveries.NewRepository(nil), gameliveries.NewLookup(gameliveries.NewRepository(nil)))
	if job.Name() != "infinite-flight-liveries" {
		t.Fatalf("Name() = %q", job.Name())
	}
}

func TestJobRunWrapsClientError(t *testing.T) {
	job := New(liveriesClientStub{err: context.DeadlineExceeded}, gameliveries.NewRepository(nil), gameliveries.NewLookup(gameliveries.NewRepository(nil)))
	err := job.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "refresh liveries") {
		t.Fatalf("Run() error = %v", err)
	}
}
