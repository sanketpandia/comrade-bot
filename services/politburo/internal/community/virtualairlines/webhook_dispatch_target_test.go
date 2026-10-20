package virtualairlines

import (
	"context"
	"errors"
	"regexp"
	"testing"

	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestLoadWebhookDispatchTarget(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() = %v", err)
	}
	defer db.Close()

	webhooks := []byte(`{"active_flights":{"url":"https://discord.com/api/webhooks/1/t","frequency":"30m","filters":{}}}`)
	basic := []byte(`{"preferred_servers":["Expert"]}`)

	mock.ExpectQuery(regexp.QuoteMeta(webhookDispatchTargetQuery)).
		WithArgs("va-1", vaconfig.TypeWebhooks, vaconfig.TypeBasic).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "webhooks", "basic"}).
			AddRow("va-1", "TEST", webhooks, basic))

	got, err := (&Service{db: db}).LoadWebhookDispatchTarget(context.Background(), "va-1", "active_flights")
	if err != nil {
		t.Fatalf("LoadWebhookDispatchTarget() = %v", err)
	}
	if got.VAID != "va-1" || got.VACode != "TEST" || got.WebhookType != "active_flights" {
		t.Fatalf("target = %+v", got)
	}
	if got.URL != "https://discord.com/api/webhooks/1/t" {
		t.Fatalf("url = %q", got.URL)
	}
	if len(got.PreferredServers) != 1 || got.PreferredServers[0] != "expert" {
		t.Fatalf("preferred servers = %#v", got.PreferredServers)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadWebhookDispatchTargetNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() = %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(webhookDispatchTargetQuery)).
		WithArgs("va-missing", vaconfig.TypeWebhooks, vaconfig.TypeBasic).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "webhooks", "basic"}))

	_, err = (&Service{db: db}).LoadWebhookDispatchTarget(context.Background(), "va-missing", "active_flights")
	if !errors.Is(err, ErrWebhookConfigNotFound) {
		t.Fatalf("err = %v, want ErrWebhookConfigNotFound", err)
	}
}

func TestLoadWebhookDispatchTargetUnknownType(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() = %v", err)
	}
	defer db.Close()

	webhooks := []byte(`{"active_flights":{"url":"https://discord.com/api/webhooks/1/t","frequency":"30m","filters":{}}}`)
	mock.ExpectQuery(regexp.QuoteMeta(webhookDispatchTargetQuery)).
		WithArgs("va-1", vaconfig.TypeWebhooks, vaconfig.TypeBasic).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "webhooks", "basic"}).
			AddRow("va-1", "TEST", webhooks, nil))

	_, err = (&Service{db: db}).LoadWebhookDispatchTarget(context.Background(), "va-1", "hourly")
	if !errors.Is(err, ErrWebhookTypeNotFound) {
		t.Fatalf("err = %v, want ErrWebhookTypeNotFound", err)
	}
}
