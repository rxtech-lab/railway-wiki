package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/suite"

	"github.com/rxtech-lab/railway-wiki/internal/api"
	"github.com/rxtech-lab/railway-wiki/internal/media"
	"github.com/rxtech-lab/railway-wiki/internal/models"
	"github.com/rxtech-lab/railway-wiki/internal/overpass"
	"github.com/rxtech-lab/railway-wiki/internal/repo"
	"github.com/rxtech-lab/railway-wiki/internal/schema"
	"github.com/rxtech-lab/railway-wiki/internal/testutil"
)

type featureOverpass struct {
	search []overpass.Candidate
	fetch  *overpass.Candidate
}

func (f *featureOverpass) Search(context.Context, overpass.SearchRequest) ([]overpass.Candidate, error) {
	return f.search, nil
}

func (f *featureOverpass) Fetch(context.Context, string, int64) (*overpass.Candidate, error) {
	return f.fetch, nil
}

type ManagementFeatureSuite struct {
	testutil.DBTestSuite
	server   *Server
	overpass *featureOverpass
}

func (s *ManagementFeatureSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	registry, err := schema.NewRegistry()
	s.Require().NoError(err)
	s.overpass = &featureOverpass{}
	s.server = NewServer(s.DB, media.NoopPresigner{}, registry, s.overpass)
}

func (s *ManagementFeatureSuite) TestDashboardAndStationBounds() {
	s.Require().NoError(s.DB.Create(&models.Station{Name: "Inside", Latitude: ptrFloat(1), Longitude: ptrFloat(1)}).Error)
	s.Require().NoError(s.DB.Create(&models.Station{Name: "Outside", Latitude: ptrFloat(20), Longitude: ptrFloat(20)}).Error)

	response, err := s.server.AdminGetDashboard(context.Background(), api.AdminGetDashboardRequestObject{})
	s.Require().NoError(err)
	dashboard := response.(api.AdminGetDashboard200JSONResponse)
	s.Equal(int64(2), dashboard.StationCoverage.Total)
	s.Equal(int64(2), dashboard.StationCoverage.WithCoordinates)

	south, west, north, east := api.SouthParam(0), api.WestParam(0), api.NorthParam(2), api.EastParam(2)
	list, err := s.server.ListStations(context.Background(), api.ListStationsRequestObject{Params: api.ListStationsParams{
		South: &south, West: &west, North: &north, East: &east,
	}})
	s.Require().NoError(err)
	stations := list.(api.ListStations200JSONResponse)
	s.Len(stations.Items, 1)
	s.Equal("Inside", stations.Items[0].Name)

	_, err = s.server.ListStations(context.Background(), api.ListStationsRequestObject{Params: api.ListStationsParams{South: &south}})
	s.ErrorIs(err, repo.ErrBadRequest)
}

func (s *ManagementFeatureSuite) TestRouteConfigurationRejectsInvalidRouteDatesWithoutMutation() {
	route := models.Route{Name: "Original Route"}
	station := models.Station{Name: "Station"}
	s.Require().NoError(s.DB.Create(&route).Error)
	s.Require().NoError(s.DB.Create(&station).Error)

	validFrom := openapi_types.Date{Time: time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)}
	validTo := openapi_types.Date{Time: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)}
	body := api.RouteConfigurationRequest{
		Route:    api.UpdateRouteRequest{Name: "Should Not Persist", ValidFrom: &validFrom, ValidTo: &validTo},
		Stations: []api.RouteStationDraft{{StationId: uuid.MustParse(station.ID)}},
	}
	_, err := s.server.AdminUpdateRouteConfiguration(context.Background(), api.AdminUpdateRouteConfigurationRequestObject{
		Id: uuid.MustParse(route.ID), Body: &body,
	})
	s.ErrorIs(err, repo.ErrBadRequest)

	var persistedRoute models.Route
	s.Require().NoError(s.DB.First(&persistedRoute, "id = ?", route.ID).Error)
	s.Equal("Original Route", persistedRoute.Name)
	var stationRows int64
	s.Require().NoError(s.DB.Model(&models.RouteStation{}).Where("route_id = ?", route.ID).Count(&stationRows).Error)
	s.Zero(stationRows)
}

func (s *ManagementFeatureSuite) TestOverpassImportIsIdempotentAndSearchMarksImported() {
	candidate := overpass.Candidate{
		ElementType: "node", ElementID: 42, Name: "Central", Latitude: 1, Longitude: 2,
		Ref: ptrString("CEN"), Tags: map[string]string{"railway": "station", "name": "Central"},
	}
	s.overpass.fetch = &candidate
	s.overpass.search = []overpass.Candidate{candidate}

	first, err := s.server.AdminImportOverpassStation(context.Background(), api.AdminImportOverpassStationRequestObject{
		ElementType: api.Node, ElementId: 42,
	})
	s.Require().NoError(err)
	created := first.(api.AdminImportOverpassStation201JSONResponse)
	s.Equal("Central", created.Name)
	s.NotNil(created.OsmElementId)

	second, err := s.server.AdminImportOverpassStation(context.Background(), api.AdminImportOverpassStationRequestObject{
		ElementType: api.Node, ElementId: 42,
	})
	s.Require().NoError(err)
	s.IsType(api.AdminImportOverpassStation200JSONResponse{}, second)

	lat, lon := 1.0, 2.0
	search, err := s.server.AdminSearchOverpassStations(context.Background(), api.AdminSearchOverpassStationsRequestObject{
		Params: api.AdminSearchOverpassStationsParams{Lat: &lat, Lon: &lon},
	})
	s.Require().NoError(err)
	items := search.(api.AdminSearchOverpassStations200JSONResponse).Items
	s.Require().Len(items, 1)
	s.NotNil(items[0].ImportedStationId)
}

func (s *ManagementFeatureSuite) TestRouteConfigurationValidatesAndSavesAtomically() {
	route := models.Route{Name: "Old Route"}
	stationA, stationB := models.Station{Name: "A"}, models.Station{Name: "B"}
	s.Require().NoError(s.DB.Create(&route).Error)
	s.Require().NoError(s.DB.Create(&stationA).Error)
	s.Require().NoError(s.DB.Create(&stationB).Error)
	routeID := uuid.MustParse(route.ID)
	stationAID := uuid.MustParse(stationA.ID)
	stationBID := uuid.MustParse(stationB.ID)
	distanceA, distanceB := 0.0, 5.5
	body := api.RouteConfigurationRequest{
		Route: api.UpdateRouteRequest{Name: "New Route"},
		Stations: []api.RouteStationDraft{
			{StationId: stationAID, DistanceFromStartKm: &distanceA},
			{StationId: stationBID, DistanceFromStartKm: &distanceB},
		},
	}
	response, err := s.server.AdminUpdateRouteConfiguration(context.Background(), api.AdminUpdateRouteConfigurationRequestObject{Id: routeID, Body: &body})
	s.Require().NoError(err)
	configuration := response.(api.AdminUpdateRouteConfiguration200JSONResponse)
	s.Equal("New Route", configuration.Route.Name)
	s.Require().Len(configuration.Stations, 2)
	s.Equal(1, configuration.Stations[0].Sequence)
	s.Equal(2, configuration.Stations[1].Sequence)

	body.Stations[1].StationId = stationAID
	validation, err := s.server.AdminValidateRouteConfiguration(context.Background(), api.AdminValidateRouteConfigurationRequestObject{Id: routeID, Body: &body})
	s.Require().NoError(err)
	result := validation.(api.AdminValidateRouteConfiguration200JSONResponse)
	s.False(result.Valid)
	s.NotEmpty(result.Errors)
}

func TestManagementFeatureSuite(t *testing.T) {
	suite.Run(t, new(ManagementFeatureSuite))
}

func ptrFloat(value float64) *float64 { return &value }
func ptrString(value string) *string  { return &value }
