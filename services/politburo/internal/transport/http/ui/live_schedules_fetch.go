package ui

import (
	"context"
	"fmt"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

type liveSchedulesUpstreamData struct {
	Aircraft  []infiniteflight.LiveAircraft
	Schedules []infiniteflight.LiveSchedule
	Truncated bool
}

type liveSchedulesUpstreamClient interface {
	ListOrganizationAircraft(context.Context, string, string) ([]infiniteflight.LiveAircraft, error)
	ListOrganizationSchedules(context.Context, string, string, int, int, []string) (infiniteflight.LiveSchedulePage, error)
	ListAircraftSchedules(context.Context, string, string) ([]infiniteflight.LiveSchedule, error)
}

func fetchLiveSchedulesUpstream(
	ctx context.Context,
	live liveSchedulesUpstreamClient,
	accessToken, organizationID, filterAircraftID string,
	queryAircraftIDs []string,
) (liveSchedulesUpstreamData, error) {
	aircraft, err := live.ListOrganizationAircraft(ctx, accessToken, organizationID)
	if err != nil {
		return liveSchedulesUpstreamData{}, fmt.Errorf("list organization aircraft: %w", err)
	}

	if aircraftID, ok := singleAircraftScheduleFilter(filterAircraftID, queryAircraftIDs); ok {
		rows, err := live.ListAircraftSchedules(ctx, accessToken, aircraftID)
		if err != nil {
			return liveSchedulesUpstreamData{}, fmt.Errorf("list aircraft schedules: %w", err)
		}
		return liveSchedulesUpstreamData{
			Aircraft:  aircraft,
			Schedules: filterOpenLiveSchedules(rows),
		}, nil
	}

	schedules, truncated, err := fetchOrganizationSchedulesPages(ctx, live, accessToken, organizationID, queryAircraftIDs)
	if err != nil {
		return liveSchedulesUpstreamData{}, fmt.Errorf("list organization schedules: %w", err)
	}
	return liveSchedulesUpstreamData{
		Aircraft:  aircraft,
		Schedules: schedules,
		Truncated: truncated,
	}, nil
}

func singleAircraftScheduleFilter(filterAircraftID string, queryAircraftIDs []string) (aircraftID string, ok bool) {
	if filterAircraftID != "" {
		return filterAircraftID, true
	}
	if len(queryAircraftIDs) == 1 {
		return queryAircraftIDs[0], true
	}
	return "", false
}

func fetchOrganizationSchedulesPages(
	ctx context.Context,
	live liveSchedulesUpstreamClient,
	accessToken, organizationID string,
	aircraftIDs []string,
) (items []infiniteflight.LiveSchedule, truncated bool, err error) {
	for page := 1; page <= liveSchedulesMaxPages; page++ {
		result, err := live.ListOrganizationSchedules(ctx, accessToken, organizationID, page, liveSchedulesPageSize, aircraftIDs)
		if err != nil {
			return nil, false, err
		}
		items = append(items, result.Items...)
		if page >= result.TotalPages {
			return items, false, nil
		}
		if page == liveSchedulesMaxPages && result.TotalPages > liveSchedulesMaxPages {
			return items, true, nil
		}
	}
	return items, false, nil
}

// filterOpenLiveSchedules matches organization list semantics (exclude Cancelled and Arrived).
func filterOpenLiveSchedules(rows []infiniteflight.LiveSchedule) []infiniteflight.LiveSchedule {
	if len(rows) == 0 {
		return rows
	}
	out := make([]infiniteflight.LiveSchedule, 0, len(rows))
	for _, row := range rows {
		if row.Status == 9 || row.Status == 11 {
			continue
		}
		out = append(out, row)
	}
	return out
}
