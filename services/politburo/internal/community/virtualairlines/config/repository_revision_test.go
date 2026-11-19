package config

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUpsertIfRevisionInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	cfg := json.RawMessage(`{"version":1,"revision":1}`)
	mock.ExpectQuery("INSERT INTO public.va_configs").
		WithArgs("va-1", TypeDataConnections, []byte(cfg), sqlmock.AnyArg(), "actor-1").
		WillReturnRows(sqlmock.NewRows([]string{"va_id", "config_type", "config", "updated_at", "updated_by"}).
			AddRow("va-1", TypeDataConnections, []byte(cfg), time.Now().UTC(), "actor-1"))

	block, err := NewRepository(db).UpsertIfRevision(context.Background(), "va-1", TypeDataConnections, cfg, 0, "actor-1")
	if err != nil {
		t.Fatalf("UpsertIfRevision() = %v", err)
	}
	if block.VAID != "va-1" {
		t.Fatalf("block = %+v", block)
	}
}

func TestUpsertIfRevisionConflictOnInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	cfg := json.RawMessage(`{"version":1,"revision":1}`)
	mock.ExpectQuery("INSERT INTO public.va_configs").
		WillReturnRows(sqlmock.NewRows([]string{"va_id", "config_type", "config", "updated_at", "updated_by"}))

	_, err = NewRepository(db).UpsertIfRevision(context.Background(), "va-1", TypeDataConnections, cfg, 0, "actor-1")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("err = %v", err)
	}
}