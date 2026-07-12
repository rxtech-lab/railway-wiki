package server

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/rxtech-lab/railway-wiki/internal/api"
	"github.com/rxtech-lab/railway-wiki/internal/auth"
	"github.com/rxtech-lab/railway-wiki/internal/convert"
	"github.com/rxtech-lab/railway-wiki/internal/models"
	"github.com/rxtech-lab/railway-wiki/internal/overpass"
	"github.com/rxtech-lab/railway-wiki/internal/repo"
)

func (s *Server) AdminSearchOverpassStations(ctx context.Context, request api.AdminSearchOverpassStationsRequestObject) (api.AdminSearchOverpassStationsResponseObject, error) {
	search, err := overpassSearchRequest(request.Params)
	if err != nil {
		return nil, err
	}
	if principal, ok := auth.PrincipalFromContext(ctx); ok {
		search.Caller = principal.Subject
	}
	items, err := s.overpass.Search(ctx, search)
	if err != nil {
		return nil, err
	}
	existing, err := s.importedStationIDs(ctx, items)
	if err != nil {
		return nil, err
	}
	response := make([]api.OverpassStationCandidate, 0, len(items))
	for _, item := range items {
		candidate := overpassCandidateToAPI(item)
		candidate.ImportedStationId = existing[osmIdentity(item.ElementType, item.ElementID)]
		response = append(response, candidate)
	}
	return api.AdminSearchOverpassStations200JSONResponse{
		Items:       response,
		GeneratedAt: time.Now().UTC(),
	}, nil
}

func overpassSearchRequest(params api.AdminSearchOverpassStationsParams) (overpass.SearchRequest, error) {
	hasBounds := params.South != nil || params.West != nil || params.North != nil || params.East != nil
	hasPoint := params.Lat != nil || params.Lon != nil || params.RadiusMeters != nil
	if hasBounds == hasPoint {
		return overpass.SearchRequest{}, overpass.ErrBadRequest
	}
	if hasBounds {
		if params.South == nil || params.West == nil || params.North == nil || params.East == nil {
			return overpass.SearchRequest{}, overpass.ErrBadRequest
		}
		return overpass.SearchRequest{Bounds: &overpass.Bounds{
			South: float64(*params.South),
			West:  float64(*params.West),
			North: float64(*params.North),
			East:  float64(*params.East),
		}}, nil
	}
	if params.Lat == nil || params.Lon == nil {
		return overpass.SearchRequest{}, overpass.ErrBadRequest
	}
	radius := 2000
	if params.RadiusMeters != nil {
		radius = *params.RadiusMeters
	}
	return overpass.SearchRequest{Point: &overpass.PointSearch{
		Latitude:     *params.Lat,
		Longitude:    *params.Lon,
		RadiusMeters: radius,
	}}, nil
}

func (s *Server) AdminImportOverpassStation(ctx context.Context, request api.AdminImportOverpassStationRequestObject) (api.AdminImportOverpassStationResponseObject, error) {
	elementType := string(request.ElementType)
	if existing, err := s.stationByOSMIdentity(ctx, elementType, request.ElementId); err == nil {
		return api.AdminImportOverpassStation200JSONResponse(convert.StationToAPI(existing)), nil
	} else if !errors.Is(err, repo.ErrNotFound) {
		return nil, err
	}
	candidate, err := s.overpass.Fetch(ctx, elementType, request.ElementId)
	if err != nil {
		return nil, err
	}
	if candidate.ElementType != elementType || candidate.ElementID != request.ElementId {
		return nil, overpass.ErrBadRequest
	}
	encodedTags, err := json.Marshal(candidate.Tags)
	if err != nil {
		return nil, err
	}
	station := models.Station{
		Name:           candidate.Name,
		NameEn:         candidate.NameEn,
		StationNumber:  candidate.Ref,
		Description:    optionalTag(candidate.Tags, "description"),
		Latitude:       floatPointer(candidate.Latitude),
		Longitude:      floatPointer(candidate.Longitude),
		OsmElementType: stringPointer(candidate.ElementType),
		OsmElementId:   int64Pointer(candidate.ElementID),
		OsmTags:        datatypes.JSON(encodedTags),
	}
	if request.Body != nil && request.Body.Overrides != nil {
		convert.StationApplyUpdate(request.Body.Overrides, &station)
	}
	if station.Name == "" {
		return nil, repo.ErrBadRequest
	}
	if err := s.db.WithContext(ctx).Create(&station).Error; err != nil {
		// A concurrent import may have won the composite unique constraint.
		if existing, lookupErr := s.stationByOSMIdentity(ctx, elementType, request.ElementId); lookupErr == nil {
			return api.AdminImportOverpassStation200JSONResponse(convert.StationToAPI(existing)), nil
		}
		return nil, err
	}
	return api.AdminImportOverpassStation201JSONResponse(convert.StationToAPI(&station)), nil
}

func (s *Server) stationByOSMIdentity(ctx context.Context, elementType string, elementID int64) (*models.Station, error) {
	var station models.Station
	err := s.db.WithContext(ctx).
		Where("osm_element_type = ? AND osm_element_id = ?", elementType, elementID).
		First(&station).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repo.ErrNotFound
	}
	return &station, err
}

func (s *Server) importedStationIDs(ctx context.Context, candidates []overpass.Candidate) (map[string]*openapi_types.UUID, error) {
	result := make(map[string]*openapi_types.UUID)
	byType := map[string][]int64{"node": {}, "way": {}, "relation": {}}
	for _, item := range candidates {
		byType[item.ElementType] = append(byType[item.ElementType], item.ElementID)
	}
	for elementType, ids := range byType {
		if len(ids) == 0 {
			continue
		}
		var stations []models.Station
		if err := s.db.WithContext(ctx).
			Where("osm_element_type = ? AND osm_element_id IN ?", elementType, ids).
			Find(&stations).Error; err != nil {
			return nil, err
		}
		for _, station := range stations {
			if station.OsmElementType == nil || station.OsmElementId == nil {
				continue
			}
			id, err := uuid.Parse(station.ID)
			if err != nil {
				continue
			}
			result[osmIdentity(*station.OsmElementType, *station.OsmElementId)] = &id
		}
	}
	return result, nil
}

func overpassCandidateToAPI(item overpass.Candidate) api.OverpassStationCandidate {
	return api.OverpassStationCandidate{
		ElementType: api.OverpassStationCandidateElementType(item.ElementType),
		ElementId:   item.ElementID,
		Name:        item.Name,
		NameEn:      item.NameEn,
		Ref:         item.Ref,
		Railway:     item.Railway,
		Mode:        item.Mode,
		Operator:    item.Operator,
		Network:     item.Network,
		Latitude:    item.Latitude,
		Longitude:   item.Longitude,
		Tags:        item.Tags,
	}
}

func osmIdentity(elementType string, elementID int64) string {
	return elementType + ":" + strconv.FormatInt(elementID, 10)
}

func optionalTag(tags map[string]string, key string) *string {
	if value := tags[key]; value != "" {
		return &value
	}
	return nil
}

func floatPointer(value float64) *float64 { return &value }
func int64Pointer(value int64) *int64     { return &value }
func stringPointer(value string) *string  { return &value }
