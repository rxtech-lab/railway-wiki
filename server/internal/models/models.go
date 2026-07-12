// Package models defines the GORM database models for railway-wiki.
//
// All models embed Base, which supplies a string UUID primary key generated in
// Go (libSQL/SQLite has no gen_random_uuid()) plus created/updated timestamps.
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Base is embedded by every model. It provides the primary key + timestamps and
// satisfies the repo.Model interface (GetID) used by the generic repository.
type Base struct {
	ID        string    `gorm:"primaryKey;type:text" json:"id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// GetID returns the primary key. Value receiver so it is promoted to both the
// value and pointer method sets of the embedding model.
func (b Base) GetID() string { return b.ID }

// BeforeCreate assigns a UUID when one was not supplied.
func (b *Base) BeforeCreate(*gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
	return nil
}

// Company - a railway operator / owner.
type Company struct {
	Base
	Name        string `gorm:"not null"`
	ShortName   *string
	CompanyType *string
	Country     *string
	Website     *string
	ValidFrom   *time.Time
	ValidTo     *time.Time
}

func (Company) TableName() string { return "companies" }

// Station - a physical station.
type Station struct {
	Base
	Name           string `gorm:"not null"`
	NameEn         *string
	StationNumber  *string
	Description    *string
	Latitude       *float64
	Longitude      *float64
	AreaGeo        datatypes.JSON
	OpenedAt       *time.Time
	ClosedAt       *time.Time
	OsmElementType *string        `gorm:"index:idx_stations_osm_identity,unique"`
	OsmElementId   *int64         `gorm:"index:idx_stations_osm_identity,unique"`
	OsmTags        datatypes.JSON `gorm:"type:json"`
}

func (Station) TableName() string { return "stations" }

// StationCode - a code assigned to a station (optionally per route/company).
type StationCode struct {
	Base
	StationId string `gorm:"not null;index"`
	RouteId   *string
	CompanyId *string
	Code      string `gorm:"not null"`
	ValidFrom *time.Time
	ValidTo   *time.Time
}

func (StationCode) TableName() string { return "station_codes" }

// StationTransfer - a walkable transfer between two stations.
type StationTransfer struct {
	Base
	FromStationId  string `gorm:"not null;index"`
	ToStationId    string `gorm:"not null;index"`
	TransferType   *string
	WalkingMinutes *int
	Description    *string
}

func (StationTransfer) TableName() string { return "station_transfers" }

// Platform - a platform within a station.
type Platform struct {
	Base
	StationId      string `gorm:"not null;index"`
	PlatformNumber *string
	Name           *string
	PlatformType   *string
	Latitude       *float64
	Longitude      *float64
	Geo            datatypes.JSON
	Description    *string
}

func (Platform) TableName() string { return "platforms" }

// Route - a physical/infrastructure line.
type Route struct {
	Base
	Name            string `gorm:"not null"`
	NameEn          *string
	Description     *string
	RouteType       *string
	LocalType       *string
	OperationStatus *string
	ValidFrom       *time.Time
	ValidTo         *time.Time
	Geo             datatypes.JSON
}

func (Route) TableName() string { return "routes" }

// RouteCompany - a company associated with a route.
type RouteCompany struct {
	Base
	RouteId   string `gorm:"not null;index"`
	CompanyId string `gorm:"not null;index"`
	Role      *string
	ValidFrom *time.Time
	ValidTo   *time.Time
}

func (RouteCompany) TableName() string { return "route_companies" }

// RouteStation - a station on a route, in sequence.
type RouteStation struct {
	Base
	RouteId             string `gorm:"not null;index"`
	StationId           string `gorm:"not null;index"`
	Sequence            int    `gorm:"not null"`
	DistanceFromStartKm *float64
	ValidFrom           *time.Time
	ValidTo             *time.Time
}

func (RouteStation) TableName() string { return "route_stations" }

// TrackSegment - a segment of track on a route.
type TrackSegment struct {
	Base
	RouteId         string `gorm:"not null;index"`
	FromStationId   *string
	ToStationId     *string
	TrackNumber     *string
	TrackCount      *int
	DistanceKm      *float64
	Gauge           *string
	Electrification *string
	MaxSpeedKmh     *int
	Geo             datatypes.JSON
	ValidFrom       *time.Time
	ValidTo         *time.Time
}

func (TrackSegment) TableName() string { return "track_segments" }

// PlatformTrack - links a platform to a track segment.
type PlatformTrack struct {
	Base
	PlatformId     string `gorm:"not null;index"`
	TrackSegmentId string `gorm:"not null;index"`
	Direction      *string
	ValidFrom      *time.Time
	ValidTo        *time.Time
}

func (PlatformTrack) TableName() string { return "platform_tracks" }

// OperationRoute - a service/operation line (e.g. a named limited express).
type OperationRoute struct {
	Base
	Name                 string `gorm:"not null"`
	NameEn               *string
	Description          *string
	ServiceType          *string
	LocalType            *string
	OriginStationId      *string
	DestinationStationId *string
	ValidFrom            *time.Time
	ValidTo              *time.Time
}

func (OperationRoute) TableName() string { return "operation_routes" }

// OperationRouteCompany - a company associated with an operation route.
type OperationRouteCompany struct {
	Base
	OperationRouteId string `gorm:"not null;index"`
	CompanyId        string `gorm:"not null;index"`
	Role             *string
	ValidFrom        *time.Time
	ValidTo          *time.Time
}

func (OperationRouteCompany) TableName() string { return "operation_route_companies" }

// OperationRouteSection - a routed section of an operation route.
type OperationRouteSection struct {
	Base
	OperationRouteId string `gorm:"not null;index"`
	RouteId          string `gorm:"not null;index"`
	Sequence         int    `gorm:"not null"`
	FromStationId    *string
	ToStationId      *string
	ValidFrom        *time.Time
	ValidTo          *time.Time
}

func (OperationRouteSection) TableName() string { return "operation_route_sections" }

// OperationRouteStop - a scheduled stop template on an operation route.
type OperationRouteStop struct {
	Base
	OperationRouteId   string `gorm:"not null;index"`
	StationId          string `gorm:"not null;index"`
	Sequence           int    `gorm:"not null"`
	DefaultStopType    *string
	PassengerBoarding  *bool
	PassengerAlighting *bool
	ValidFrom          *time.Time
	ValidTo            *time.Time
}

func (OperationRouteStop) TableName() string { return "operation_route_stops" }

// TimetableVersion - a versioned timetable dataset.
type TimetableVersion struct {
	Base
	Name        string `gorm:"not null"`
	Description *string
	CompanyId   *string
	ValidFrom   *time.Time
	ValidTo     *time.Time
	SourceUrl   *string
	ImportedAt  *time.Time
}

func (TimetableVersion) TableName() string { return "timetable_versions" }

// ServiceCalendar - a weekly service pattern with a validity window.
type ServiceCalendar struct {
	Base
	Name      string `gorm:"not null"`
	StartDate *time.Time
	EndDate   *time.Time
	Monday    *bool
	Tuesday   *bool
	Wednesday *bool
	Thursday  *bool
	Friday    *bool
	Saturday  *bool
	Sunday    *bool
}

func (ServiceCalendar) TableName() string { return "service_calendars" }

// ServiceCalendarException - a per-date exception to a service calendar.
type ServiceCalendarException struct {
	Base
	CalendarId    string    `gorm:"not null;index"`
	ServiceDate   time.Time `gorm:"not null"`
	ExceptionType *string
	Description   *string
}

func (ServiceCalendarException) TableName() string { return "service_calendar_exceptions" }

// Train - a train (rolling stock / named service).
type Train struct {
	Base
	TrainNumber string `gorm:"not null"`
	TrainName   *string
	TrainType   *string
	Description *string
	ImageUrl    *string
	ValidFrom   *time.Time
	ValidTo     *time.Time
}

func (Train) TableName() string { return "trains" }

// TrainRun - a concrete scheduled run of a train.
type TrainRun struct {
	Base
	TimetableVersionId   string `gorm:"not null;index"`
	OperationRouteId     *string
	CalendarId           *string
	TrainId              *string
	TrainNumber          *string
	Direction            *string
	OriginStationId      *string
	DestinationStationId *string
	ServiceType          *string
	Note                 *string
}

func (TrainRun) TableName() string { return "train_runs" }

// TrainRunStop - a stop on a train run.
type TrainRunStop struct {
	Base
	TrainRunId         string `gorm:"not null;index"`
	StationId          string `gorm:"not null;index"`
	PlatformId         *string
	Sequence           int `gorm:"not null"`
	ArrivalTime        *string
	DepartureTime      *string
	StopType           *string
	PassengerBoarding  *bool
	PassengerAlighting *bool
	Note               *string
}

func (TrainRunStop) TableName() string { return "train_run_stops" }

// Media - an image/video/document asset.
type Media struct {
	Base
	MediaType string `gorm:"not null"`
	Url       string `gorm:"not null"`
	Title     *string
	Source    *string
	License   *string
}

func (Media) TableName() string { return "media" }

// MediaAttachment - attaches a media asset to some entity.
type MediaAttachment struct {
	Base
	MediaId    string `gorm:"not null;index"`
	EntityType string `gorm:"not null;index"`
	EntityId   string `gorm:"not null;index"`
	SortOrder  *int
}

func (MediaAttachment) TableName() string { return "media_attachments" }

// AllModels lists every model for AutoMigrate and test setup.
var AllModels = []any{
	&Company{},
	&Station{},
	&StationCode{},
	&StationTransfer{},
	&Platform{},
	&Route{},
	&RouteCompany{},
	&RouteStation{},
	&TrackSegment{},
	&PlatformTrack{},
	&OperationRoute{},
	&OperationRouteCompany{},
	&OperationRouteSection{},
	&OperationRouteStop{},
	&TimetableVersion{},
	&ServiceCalendar{},
	&ServiceCalendarException{},
	&Train{},
	&TrainRun{},
	&TrainRunStop{},
	&Media{},
	&MediaAttachment{},
}
