package config

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetActiveWebhooks(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() = %v", err)
	}
	defer db.Close()

	webhooks := []byte(`{"active_flights":{"url":"https://discord.com/api/webhooks/1/t","frequency":"30m","filters":{}}}`)
	basic := []byte(`{"preferred_servers":["Expert"]}`)
	mock.ExpectQuery(regexp.QuoteMeta(getActiveWebhooksQuery)).
		WithArgs(TypeWebhooks, TypeBasic, "va-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "webhooks", "basic"}).
			AddRow("va-1", "TEST", webhooks, basic))

	got, err := NewRepository(db).GetActiveWebhooks(context.Background(), "va-1")
	if err != nil {
		t.Fatalf("GetActiveWebhooks() = %v", err)
	}
	if got.VAID != "va-1" || got.VACode != "TEST" || string(got.Webhooks) != string(webhooks) {
		t.Fatalf("row = %+v", got)
	}
	if len(got.PreferredServers) != 1 || got.PreferredServers[0] != "expert" {
		t.Fatalf("preferred servers = %#v", got.PreferredServers)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetActiveWebhooksNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() = %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(getActiveWebhooksQuery)).
		WithArgs(TypeWebhooks, TypeBasic, "va-missing").
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "webhooks", "basic"}))

	_, err = NewRepository(db).GetActiveWebhooks(context.Background(), "va-missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestListActiveBasicSkipsInvalidRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() = %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(listActiveBasicQuery)).
		WithArgs(TypeBasic).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "config"}).
			AddRow("va-1", "AAA", []byte(`{"game_callsign":{"prefix":"IE"}}`)).
			AddRow("va-2", "BBB", []byte(`{"game_callsign":{"prefix":1}}`)))

	got, err := NewRepository(db).ListActiveBasic(context.Background())
	if err != nil {
		t.Fatalf("ListActiveBasic() = %v", err)
	}
	if len(got) != 1 || got[0].VAID != "va-1" || got[0].Basic.GameCallsign.Prefix != "IE" {
		t.Fatalf("rows = %+v", got)
	}
}
