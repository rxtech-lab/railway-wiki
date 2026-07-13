package server

import (
	"context"
	"time"

	"github.com/rxtech-lab/railway-wiki/internal/api"
)

var dashboardTables = []struct {
	resource string
	table    string
}{
	{"companies", "companies"},
	{"stations", "stations"},
	{"station-codes", "station_codes"},
	{"station-transfers", "station_transfers"},
	{"platforms", "platforms"},
	{"routes", "routes"},
	{"route-companies", "route_companies"},
	{"route-stations", "route_stations"},
	{"track-segments", "track_segments"},
	{"platform-tracks", "platform_tracks"},
	{"operation-routes", "operation_routes"},
	{"operation-route-companies", "operation_route_companies"},
	{"operation-route-sections", "operation_route_sections"},
	{"operation-route-stops", "operation_route_stops"},
	{"timetable-versions", "timetable_versions"},
	{"service-calendars", "service_calendars"},
	{"service-calendar-exceptions", "service_calendar_exceptions"},
	{"trains", "trains"},
	{"train-runs", "train_runs"},
	{"train-run-stops", "train_run_stops"},
	{"media", "media"},
	{"media-attachments", "media_attachments"},
}

func (s *Server) AdminGetDashboard(ctx context.Context, _ api.AdminGetDashboardRequestObject) (api.AdminGetDashboardResponseObject, error) {
	metrics := make([]api.DashboardResourceMetric, 0, len(dashboardTables))
	var stationTotal, stationWithCoordinates int64
	for _, item := range dashboardTables {
		var count int64
		if err := s.db.WithContext(ctx).Table(item.table).Count(&count).Error; err != nil {
			return nil, err
		}
		metrics = append(metrics, api.DashboardResourceMetric{Resource: item.resource, Count: count})
		if item.table == "stations" {
			stationTotal = count
		}
	}
	if err := s.db.WithContext(ctx).Table("stations").
		Where("latitude IS NOT NULL AND longitude IS NOT NULL").
		Count(&stationWithCoordinates).Error; err != nil {
		return nil, err
	}
	return api.AdminGetDashboard200JSONResponse{
		Resources: metrics,
		StationCoverage: api.DashboardStationCoverage{
			Total:           stationTotal,
			WithCoordinates: stationWithCoordinates,
		},
		GeneratedAt: time.Now().UTC(),
	}, nil
}
