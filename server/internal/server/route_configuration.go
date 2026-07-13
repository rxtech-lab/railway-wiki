package server

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/rxtech-lab/railway-wiki/internal/api"
	"github.com/rxtech-lab/railway-wiki/internal/convert"
	"github.com/rxtech-lab/railway-wiki/internal/models"
	"github.com/rxtech-lab/railway-wiki/internal/repo"
)

func (s *Server) AdminGetRouteConfiguration(ctx context.Context, request api.AdminGetRouteConfigurationRequestObject) (api.AdminGetRouteConfigurationResponseObject, error) {
	configuration, err := s.routeConfiguration(ctx, request.Id.String())
	if err != nil {
		return nil, err
	}
	return api.AdminGetRouteConfiguration200JSONResponse(*configuration), nil
}

func (s *Server) AdminValidateRouteConfiguration(ctx context.Context, request api.AdminValidateRouteConfigurationRequestObject) (api.AdminValidateRouteConfigurationResponseObject, error) {
	if request.Body == nil {
		return api.AdminValidateRouteConfiguration200JSONResponse{Valid: false, Errors: []string{"request body is required"}}, nil
	}
	validationErrors, err := s.validateRouteConfiguration(ctx, request.Id.String(), request.Body)
	if err != nil {
		return nil, err
	}
	return api.AdminValidateRouteConfiguration200JSONResponse{
		Valid:  len(validationErrors) == 0,
		Errors: validationErrors,
	}, nil
}

func (s *Server) AdminUpdateRouteConfiguration(ctx context.Context, request api.AdminUpdateRouteConfigurationRequestObject) (api.AdminUpdateRouteConfigurationResponseObject, error) {
	if request.Body == nil {
		return nil, repo.ErrBadRequest
	}
	id := request.Id.String()
	validationErrors, err := s.validateRouteConfiguration(ctx, id, request.Body)
	if err != nil {
		return nil, err
	}
	if len(validationErrors) > 0 {
		return nil, fmt.Errorf("%w: %s", repo.ErrBadRequest, strings.Join(validationErrors, "; "))
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var route models.Route
		if err := tx.Where("id = ?", id).First(&route).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repo.ErrNotFound
			}
			return err
		}
		convert.RouteApplyUpdate(&request.Body.Route, &route)
		if err := tx.Save(&route).Error; err != nil {
			return err
		}

		kept := make([]string, 0, len(request.Body.Stations))
		for index, draft := range request.Body.Stations {
			row := models.RouteStation{
				RouteId:             id,
				StationId:           draft.StationId.String(),
				Sequence:            index + 1,
				DistanceFromStartKm: draft.DistanceFromStartKm,
			}
			if draft.ValidFrom != nil {
				value := draft.ValidFrom.Time
				row.ValidFrom = &value
			}
			if draft.ValidTo != nil {
				value := draft.ValidTo.Time
				row.ValidTo = &value
			}
			if draft.Id != nil {
				row.ID = draft.Id.String()
				result := tx.Model(&models.RouteStation{}).
					Where("id = ? AND route_id = ?", row.ID, id).
					Updates(map[string]any{
						"station_id": row.StationId, "sequence": row.Sequence,
						"distance_from_start_km": row.DistanceFromStartKm,
						"valid_from":             row.ValidFrom, "valid_to": row.ValidTo,
					})
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected != 1 {
					return repo.ErrBadRequest
				}
			} else if err := tx.Create(&row).Error; err != nil {
				return err
			}
			kept = append(kept, row.ID)
		}
		deleteQuery := tx.Where("route_id = ?", id)
		if len(kept) > 0 {
			deleteQuery = deleteQuery.Where("id NOT IN ?", kept)
		}
		return deleteQuery.Delete(&models.RouteStation{}).Error
	}); err != nil {
		return nil, err
	}
	configuration, err := s.routeConfiguration(ctx, id)
	if err != nil {
		return nil, err
	}
	return api.AdminUpdateRouteConfiguration200JSONResponse(*configuration), nil
}

func (s *Server) routeConfiguration(ctx context.Context, id string) (*api.RouteConfiguration, error) {
	var route models.Route
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&route).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}
	var rows []models.RouteStation
	if err := s.db.WithContext(ctx).Where("route_id = ?", id).Order("sequence ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	stations := make([]api.RouteStation, 0, len(rows))
	for index := range rows {
		stations = append(stations, convert.RouteStationToAPI(&rows[index]))
	}
	return &api.RouteConfiguration{Route: convert.RouteToAPI(&route), Stations: stations}, nil
}

func (s *Server) validateRouteConfiguration(ctx context.Context, routeID string, request *api.RouteConfigurationRequest) ([]string, error) {
	var routeCount int64
	if err := s.db.WithContext(ctx).Model(&models.Route{}).Where("id = ?", routeID).Count(&routeCount).Error; err != nil {
		return nil, err
	}
	if routeCount == 0 {
		return nil, repo.ErrNotFound
	}
	problems := make([]string, 0)
	if strings.TrimSpace(request.Route.Name) == "" {
		problems = append(problems, "route name is required")
	}
	if request.Route.ValidFrom != nil && request.Route.ValidTo != nil && request.Route.ValidTo.Time.Before(request.Route.ValidFrom.Time) {
		problems = append(problems, "route validity ends before it starts")
	}
	stationIDs := make([]string, 0, len(request.Stations))
	seenStations := make(map[string]struct{}, len(request.Stations))
	seenRows := make(map[string]struct{}, len(request.Stations))
	var previousDistance *float64
	for index, draft := range request.Stations {
		stationID := draft.StationId.String()
		stationIDs = append(stationIDs, stationID)
		if _, duplicate := seenStations[stationID]; duplicate {
			problems = append(problems, fmt.Sprintf("station %d duplicates a station already in the route", index+1))
		}
		seenStations[stationID] = struct{}{}
		if draft.Id != nil {
			rowID := draft.Id.String()
			if _, duplicate := seenRows[rowID]; duplicate {
				problems = append(problems, fmt.Sprintf("station row %d is duplicated", index+1))
			}
			seenRows[rowID] = struct{}{}
			var count int64
			if err := s.db.WithContext(ctx).Model(&models.RouteStation{}).
				Where("id = ? AND route_id = ?", rowID, routeID).Count(&count).Error; err != nil {
				return nil, err
			}
			if count == 0 {
				problems = append(problems, fmt.Sprintf("station row %d does not belong to this route", index+1))
			}
		}
		if draft.DistanceFromStartKm != nil {
			if *draft.DistanceFromStartKm < 0 {
				problems = append(problems, fmt.Sprintf("station %d has a negative distance", index+1))
			}
			if previousDistance != nil && *draft.DistanceFromStartKm < *previousDistance {
				problems = append(problems, fmt.Sprintf("station %d distance is before the previous station", index+1))
			}
			value := *draft.DistanceFromStartKm
			previousDistance = &value
		}
		if draft.ValidFrom != nil && draft.ValidTo != nil && draft.ValidTo.Time.Before(draft.ValidFrom.Time) {
			problems = append(problems, fmt.Sprintf("station %d validity ends before it starts", index+1))
		}
	}
	if len(stationIDs) > 0 {
		var count int64
		if err := s.db.WithContext(ctx).Model(&models.Station{}).Where("id IN ?", stationIDs).Count(&count).Error; err != nil {
			return nil, err
		}
		if count != int64(len(seenStations)) {
			problems = append(problems, "one or more stations do not exist")
		}
	}
	return problems, nil
}
