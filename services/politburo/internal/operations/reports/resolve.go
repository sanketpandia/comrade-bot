package reports

import (
	"context"
	"errors"
)

var (
	ErrReportNotFound        = errors.New("report not found")
	ErrUnsupportedReportKind = errors.New("unsupported report kind")
)

// ReportResolver performs operator resolution for open platform reports.
type ReportResolver interface {
	ResolveOccupiedIFC(ctx context.Context, reportID, operatorDiscordID string) error
	ResolveGuildMigration(ctx context.Context, reportID, operatorDiscordID string) error
}

// ResolveOpenReport loads the report and delegates resolution by kind.
func ResolveOpenReport(ctx context.Context, repo *Repository, resolver ReportResolver, reportID, operatorDiscordID string) error {
	report, err := repo.GetByID(ctx, reportID)
	if err != nil {
		return err
	}
	if report == nil {
		return ErrReportNotFound
	}
	switch report.Kind {
	case KindOccupiedIFC:
		return resolver.ResolveOccupiedIFC(ctx, reportID, operatorDiscordID)
	case KindMigrateDiscordServer:
		return resolver.ResolveGuildMigration(ctx, reportID, operatorDiscordID)
	default:
		return ErrUnsupportedReportKind
	}
}
