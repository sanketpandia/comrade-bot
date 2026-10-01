package registration

import (
	"context"
	"errors"
	"testing"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

type ifStub struct {
	userID  string
	flights []infiniteflight.LogbookFlight
}

func (s ifStub) ResolveUserIDByIFCUsername(context.Context, string) (string, error) {
	return s.userID, nil
}

func (s ifStub) RecentLogbookFlights(context.Context, string, int) ([]infiniteflight.LogbookFlight, error) {
	return s.flights, nil
}

func TestRegisterRequiresIFClient(t *testing.T) {
	svc := NewService(nil, nil)
	_, err := svc.Register(context.Background(), RegisterInput{
		DiscordID: "1", IFCUsername: "Pilot", RouteProof: "EGLL-KSEA",
	})
	if !errors.Is(err, ErrIFClientUnavailable) {
		t.Fatalf("err = %v", err)
	}
}
