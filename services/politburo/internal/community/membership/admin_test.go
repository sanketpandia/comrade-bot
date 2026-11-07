package membership

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUpdateRoleTransfersAdministrator(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := NewRepository(db)
	svc := NewService(nil, repo, nil, nil)
	joined := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	mock.ExpectQuery(`SELECT vur.id`).
		WithArgs("target-user", "va-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "va_id", "role", "is_active", "callsign", "airtable_pilot_id", "joined_at", "name", "code",
		}).AddRow("m-1", "target-user", "va-1", RoleProletariat, true, nil, nil, joined, "VA", "VA"))
	mock.ExpectQuery(`SELECT user_id FROM public.va_user_roles`).
		WithArgs("va-1").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("admin-user"))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE public.va_user_roles`).
		WithArgs("admin-user", "va-1", RoleBourgeoisie).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE public.va_user_roles`).
		WithArgs("target-user", "va-1", RoleAdministrator).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = svc.UpdateRole(context.Background(), UpdateRoleInput{
		ActorUserID:  "admin-user",
		TargetUserID: "target-user",
		VAID:         "va-1",
		NewRole:      RoleAdministrator,
	})
	if err != nil {
		t.Fatalf("UpdateRole() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestRemoveMemberDeniesStaffRemovingBourgeoisie(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := NewRepository(db)
	svc := NewService(nil, repo, nil, nil)
	joined := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	mock.ExpectQuery(`SELECT vur.id`).
		WithArgs("staff-user", "va-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "va_id", "role", "is_active", "callsign", "airtable_pilot_id", "joined_at", "name", "code",
		}).AddRow("m-1", "staff-user", "va-1", RoleBourgeoisie, true, nil, nil, joined, "VA", "VA"))

	err = svc.RemoveMember(context.Background(), RemoveMemberInput{
		ActorUserID:  "actor",
		ActorRole:    RoleBourgeoisie,
		TargetUserID: "staff-user",
		VAID:         "va-1",
	})
	if err != ErrRemovalDenied {
		t.Fatalf("RemoveMember() error = %v, want ErrRemovalDenied", err)
	}
}
