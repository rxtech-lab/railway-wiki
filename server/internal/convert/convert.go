package convert

import (
	"github.com/rxtech-lab/railway-wiki/internal/api"
	"github.com/rxtech-lab/railway-wiki/internal/models"
)

// Company ---------------------------------------------------------------------

func CompanyToAPI(m *models.Company) api.Company {
	return api.Company{
		Id:          idPtr(m.ID),
		Name:        m.Name,
		ShortName:   m.ShortName,
		CompanyType: m.CompanyType,
		Country:     m.Country,
		Website:     m.Website,
		ValidFrom:   datePtr(m.ValidFrom),
		ValidTo:     datePtr(m.ValidTo),
	}
}

func CompanyFromCreate(b *api.CreateCompanyRequest) models.Company {
	return models.Company{
		Name:        b.Name,
		ShortName:   b.ShortName,
		CompanyType: b.CompanyType,
		Country:     b.Country,
		Website:     b.Website,
		ValidFrom:   fromDatePtr(b.ValidFrom),
		ValidTo:     fromDatePtr(b.ValidTo),
	}
}

func CompanyApplyUpdate(b *api.UpdateCompanyRequest, m *models.Company) {
	m.Name = b.Name
	m.ShortName = b.ShortName
	m.CompanyType = b.CompanyType
	m.Country = b.Country
	m.Website = b.Website
	m.ValidFrom = fromDatePtr(b.ValidFrom)
	m.ValidTo = fromDatePtr(b.ValidTo)
}

// Station ---------------------------------------------------------------------

func StationToAPI(m *models.Station) api.Station {
	var osmElementType *api.StationOsmElementType
	if m.OsmElementType != nil {
		value := api.StationOsmElementType(*m.OsmElementType)
		osmElementType = &value
	}
	return api.Station{
		Id:             idPtr(m.ID),
		Name:           m.Name,
		NameEn:         m.NameEn,
		StationNumber:  m.StationNumber,
		Description:    m.Description,
		Latitude:       m.Latitude,
		Longitude:      m.Longitude,
		AreaGeo:        toGeo(m.AreaGeo),
		OpenedAt:       datePtr(m.OpenedAt),
		ClosedAt:       datePtr(m.ClosedAt),
		OsmElementType: osmElementType,
		OsmElementId:   m.OsmElementId,
		OsmTags:        toStringMap(m.OsmTags),
	}
}

func StationFromCreate(b *api.CreateStationRequest) models.Station {
	return models.Station{
		Name:          b.Name,
		NameEn:        b.NameEn,
		StationNumber: b.StationNumber,
		Description:   b.Description,
		Latitude:      b.Latitude,
		Longitude:     b.Longitude,
		AreaGeo:       fromGeo(b.AreaGeo),
		OpenedAt:      fromDatePtr(b.OpenedAt),
		ClosedAt:      fromDatePtr(b.ClosedAt),
	}
}

func StationApplyUpdate(b *api.UpdateStationRequest, m *models.Station) {
	m.Name = b.Name
	m.NameEn = b.NameEn
	m.StationNumber = b.StationNumber
	m.Description = b.Description
	m.Latitude = b.Latitude
	m.Longitude = b.Longitude
	m.AreaGeo = fromGeo(b.AreaGeo)
	m.OpenedAt = fromDatePtr(b.OpenedAt)
	m.ClosedAt = fromDatePtr(b.ClosedAt)
}

// StationCode -----------------------------------------------------------------

func StationCodeToAPI(m *models.StationCode) api.StationCode {
	return api.StationCode{
		Id:        idPtr(m.ID),
		StationId: toUUID(m.StationId),
		RouteId:   toUUIDPtr(m.RouteId),
		CompanyId: toUUIDPtr(m.CompanyId),
		Code:      m.Code,
		ValidFrom: datePtr(m.ValidFrom),
		ValidTo:   datePtr(m.ValidTo),
	}
}

func StationCodeFromCreate(b *api.CreateStationCodeRequest) models.StationCode {
	return models.StationCode{
		StationId: uuidStr(b.StationId),
		RouteId:   uuidStrPtr(b.RouteId),
		CompanyId: uuidStrPtr(b.CompanyId),
		Code:      b.Code,
		ValidFrom: fromDatePtr(b.ValidFrom),
		ValidTo:   fromDatePtr(b.ValidTo),
	}
}

func StationCodeApplyUpdate(b *api.UpdateStationCodeRequest, m *models.StationCode) {
	m.StationId = uuidStr(b.StationId)
	m.RouteId = uuidStrPtr(b.RouteId)
	m.CompanyId = uuidStrPtr(b.CompanyId)
	m.Code = b.Code
	m.ValidFrom = fromDatePtr(b.ValidFrom)
	m.ValidTo = fromDatePtr(b.ValidTo)
}

// StationTransfer -------------------------------------------------------------

func StationTransferToAPI(m *models.StationTransfer) api.StationTransfer {
	return api.StationTransfer{
		Id:             idPtr(m.ID),
		FromStationId:  toUUID(m.FromStationId),
		ToStationId:    toUUID(m.ToStationId),
		TransferType:   m.TransferType,
		WalkingMinutes: m.WalkingMinutes,
		Description:    m.Description,
	}
}

func StationTransferFromCreate(b *api.CreateStationTransferRequest) models.StationTransfer {
	return models.StationTransfer{
		FromStationId:  uuidStr(b.FromStationId),
		ToStationId:    uuidStr(b.ToStationId),
		TransferType:   b.TransferType,
		WalkingMinutes: b.WalkingMinutes,
		Description:    b.Description,
	}
}

func StationTransferApplyUpdate(b *api.UpdateStationTransferRequest, m *models.StationTransfer) {
	m.FromStationId = uuidStr(b.FromStationId)
	m.ToStationId = uuidStr(b.ToStationId)
	m.TransferType = b.TransferType
	m.WalkingMinutes = b.WalkingMinutes
	m.Description = b.Description
}

// Platform --------------------------------------------------------------------

func PlatformToAPI(m *models.Platform) api.Platform {
	return api.Platform{
		Id:             idPtr(m.ID),
		StationId:      toUUID(m.StationId),
		PlatformNumber: m.PlatformNumber,
		Name:           m.Name,
		PlatformType:   m.PlatformType,
		Latitude:       m.Latitude,
		Longitude:      m.Longitude,
		Geo:            toGeo(m.Geo),
		Description:    m.Description,
	}
}

func PlatformFromCreate(b *api.CreatePlatformRequest) models.Platform {
	return models.Platform{
		StationId:      uuidStr(b.StationId),
		PlatformNumber: b.PlatformNumber,
		Name:           b.Name,
		PlatformType:   b.PlatformType,
		Latitude:       b.Latitude,
		Longitude:      b.Longitude,
		Geo:            fromGeo(b.Geo),
		Description:    b.Description,
	}
}

func PlatformApplyUpdate(b *api.UpdatePlatformRequest, m *models.Platform) {
	m.StationId = uuidStr(b.StationId)
	m.PlatformNumber = b.PlatformNumber
	m.Name = b.Name
	m.PlatformType = b.PlatformType
	m.Latitude = b.Latitude
	m.Longitude = b.Longitude
	m.Geo = fromGeo(b.Geo)
	m.Description = b.Description
}

// Route -----------------------------------------------------------------------

func RouteToAPI(m *models.Route) api.Route {
	return api.Route{
		Id:              idPtr(m.ID),
		Name:            m.Name,
		NameEn:          m.NameEn,
		Description:     m.Description,
		RouteType:       m.RouteType,
		LocalType:       m.LocalType,
		OperationStatus: m.OperationStatus,
		ValidFrom:       datePtr(m.ValidFrom),
		ValidTo:         datePtr(m.ValidTo),
		Geo:             toGeo(m.Geo),
	}
}

func RouteFromCreate(b *api.CreateRouteRequest) models.Route {
	return models.Route{
		Name:            b.Name,
		NameEn:          b.NameEn,
		Description:     b.Description,
		RouteType:       b.RouteType,
		LocalType:       b.LocalType,
		OperationStatus: b.OperationStatus,
		ValidFrom:       fromDatePtr(b.ValidFrom),
		ValidTo:         fromDatePtr(b.ValidTo),
		Geo:             fromGeo(b.Geo),
	}
}

func RouteApplyUpdate(b *api.UpdateRouteRequest, m *models.Route) {
	m.Name = b.Name
	m.NameEn = b.NameEn
	m.Description = b.Description
	m.RouteType = b.RouteType
	m.LocalType = b.LocalType
	m.OperationStatus = b.OperationStatus
	m.ValidFrom = fromDatePtr(b.ValidFrom)
	m.ValidTo = fromDatePtr(b.ValidTo)
	m.Geo = fromGeo(b.Geo)
}

// RouteCompany ----------------------------------------------------------------

func RouteCompanyToAPI(m *models.RouteCompany) api.RouteCompany {
	return api.RouteCompany{
		Id:        idPtr(m.ID),
		RouteId:   toUUID(m.RouteId),
		CompanyId: toUUID(m.CompanyId),
		Role:      m.Role,
		ValidFrom: datePtr(m.ValidFrom),
		ValidTo:   datePtr(m.ValidTo),
	}
}

func RouteCompanyFromCreate(b *api.CreateRouteCompanyRequest) models.RouteCompany {
	return models.RouteCompany{
		RouteId:   uuidStr(b.RouteId),
		CompanyId: uuidStr(b.CompanyId),
		Role:      b.Role,
		ValidFrom: fromDatePtr(b.ValidFrom),
		ValidTo:   fromDatePtr(b.ValidTo),
	}
}

func RouteCompanyApplyUpdate(b *api.UpdateRouteCompanyRequest, m *models.RouteCompany) {
	m.RouteId = uuidStr(b.RouteId)
	m.CompanyId = uuidStr(b.CompanyId)
	m.Role = b.Role
	m.ValidFrom = fromDatePtr(b.ValidFrom)
	m.ValidTo = fromDatePtr(b.ValidTo)
}

// RouteStation ----------------------------------------------------------------

func RouteStationToAPI(m *models.RouteStation) api.RouteStation {
	return api.RouteStation{
		Id:                  idPtr(m.ID),
		RouteId:             toUUID(m.RouteId),
		StationId:           toUUID(m.StationId),
		Sequence:            m.Sequence,
		DistanceFromStartKm: m.DistanceFromStartKm,
		ValidFrom:           datePtr(m.ValidFrom),
		ValidTo:             datePtr(m.ValidTo),
	}
}

func RouteStationFromCreate(b *api.CreateRouteStationRequest) models.RouteStation {
	return models.RouteStation{
		RouteId:             uuidStr(b.RouteId),
		StationId:           uuidStr(b.StationId),
		Sequence:            b.Sequence,
		DistanceFromStartKm: b.DistanceFromStartKm,
		ValidFrom:           fromDatePtr(b.ValidFrom),
		ValidTo:             fromDatePtr(b.ValidTo),
	}
}

func RouteStationApplyUpdate(b *api.UpdateRouteStationRequest, m *models.RouteStation) {
	m.RouteId = uuidStr(b.RouteId)
	m.StationId = uuidStr(b.StationId)
	m.Sequence = b.Sequence
	m.DistanceFromStartKm = b.DistanceFromStartKm
	m.ValidFrom = fromDatePtr(b.ValidFrom)
	m.ValidTo = fromDatePtr(b.ValidTo)
}

// TrackSegment ----------------------------------------------------------------

func TrackSegmentToAPI(m *models.TrackSegment) api.TrackSegment {
	return api.TrackSegment{
		Id:              idPtr(m.ID),
		RouteId:         toUUID(m.RouteId),
		FromStationId:   toUUIDPtr(m.FromStationId),
		ToStationId:     toUUIDPtr(m.ToStationId),
		TrackNumber:     m.TrackNumber,
		TrackCount:      m.TrackCount,
		DistanceKm:      m.DistanceKm,
		Gauge:           m.Gauge,
		Electrification: m.Electrification,
		MaxSpeedKmh:     m.MaxSpeedKmh,
		Geo:             toGeo(m.Geo),
		ValidFrom:       datePtr(m.ValidFrom),
		ValidTo:         datePtr(m.ValidTo),
	}
}

func TrackSegmentFromCreate(b *api.CreateTrackSegmentRequest) models.TrackSegment {
	return models.TrackSegment{
		RouteId:         uuidStr(b.RouteId),
		FromStationId:   uuidStrPtr(b.FromStationId),
		ToStationId:     uuidStrPtr(b.ToStationId),
		TrackNumber:     b.TrackNumber,
		TrackCount:      b.TrackCount,
		DistanceKm:      b.DistanceKm,
		Gauge:           b.Gauge,
		Electrification: b.Electrification,
		MaxSpeedKmh:     b.MaxSpeedKmh,
		Geo:             fromGeo(b.Geo),
		ValidFrom:       fromDatePtr(b.ValidFrom),
		ValidTo:         fromDatePtr(b.ValidTo),
	}
}

func TrackSegmentApplyUpdate(b *api.UpdateTrackSegmentRequest, m *models.TrackSegment) {
	m.RouteId = uuidStr(b.RouteId)
	m.FromStationId = uuidStrPtr(b.FromStationId)
	m.ToStationId = uuidStrPtr(b.ToStationId)
	m.TrackNumber = b.TrackNumber
	m.TrackCount = b.TrackCount
	m.DistanceKm = b.DistanceKm
	m.Gauge = b.Gauge
	m.Electrification = b.Electrification
	m.MaxSpeedKmh = b.MaxSpeedKmh
	m.Geo = fromGeo(b.Geo)
	m.ValidFrom = fromDatePtr(b.ValidFrom)
	m.ValidTo = fromDatePtr(b.ValidTo)
}

// PlatformTrack ---------------------------------------------------------------

func PlatformTrackToAPI(m *models.PlatformTrack) api.PlatformTrack {
	return api.PlatformTrack{
		Id:             idPtr(m.ID),
		PlatformId:     toUUID(m.PlatformId),
		TrackSegmentId: toUUID(m.TrackSegmentId),
		Direction:      m.Direction,
		ValidFrom:      datePtr(m.ValidFrom),
		ValidTo:        datePtr(m.ValidTo),
	}
}

func PlatformTrackFromCreate(b *api.CreatePlatformTrackRequest) models.PlatformTrack {
	return models.PlatformTrack{
		PlatformId:     uuidStr(b.PlatformId),
		TrackSegmentId: uuidStr(b.TrackSegmentId),
		Direction:      b.Direction,
		ValidFrom:      fromDatePtr(b.ValidFrom),
		ValidTo:        fromDatePtr(b.ValidTo),
	}
}

func PlatformTrackApplyUpdate(b *api.UpdatePlatformTrackRequest, m *models.PlatformTrack) {
	m.PlatformId = uuidStr(b.PlatformId)
	m.TrackSegmentId = uuidStr(b.TrackSegmentId)
	m.Direction = b.Direction
	m.ValidFrom = fromDatePtr(b.ValidFrom)
	m.ValidTo = fromDatePtr(b.ValidTo)
}

// OperationRoute --------------------------------------------------------------

func OperationRouteToAPI(m *models.OperationRoute) api.OperationRoute {
	return api.OperationRoute{
		Id:                   idPtr(m.ID),
		Name:                 m.Name,
		NameEn:               m.NameEn,
		Description:          m.Description,
		ServiceType:          m.ServiceType,
		LocalType:            m.LocalType,
		OriginStationId:      toUUIDPtr(m.OriginStationId),
		DestinationStationId: toUUIDPtr(m.DestinationStationId),
		ValidFrom:            datePtr(m.ValidFrom),
		ValidTo:              datePtr(m.ValidTo),
	}
}

func OperationRouteFromCreate(b *api.CreateOperationRouteRequest) models.OperationRoute {
	return models.OperationRoute{
		Name:                 b.Name,
		NameEn:               b.NameEn,
		Description:          b.Description,
		ServiceType:          b.ServiceType,
		LocalType:            b.LocalType,
		OriginStationId:      uuidStrPtr(b.OriginStationId),
		DestinationStationId: uuidStrPtr(b.DestinationStationId),
		ValidFrom:            fromDatePtr(b.ValidFrom),
		ValidTo:              fromDatePtr(b.ValidTo),
	}
}

func OperationRouteApplyUpdate(b *api.UpdateOperationRouteRequest, m *models.OperationRoute) {
	m.Name = b.Name
	m.NameEn = b.NameEn
	m.Description = b.Description
	m.ServiceType = b.ServiceType
	m.LocalType = b.LocalType
	m.OriginStationId = uuidStrPtr(b.OriginStationId)
	m.DestinationStationId = uuidStrPtr(b.DestinationStationId)
	m.ValidFrom = fromDatePtr(b.ValidFrom)
	m.ValidTo = fromDatePtr(b.ValidTo)
}

// OperationRouteCompany -------------------------------------------------------

func OperationRouteCompanyToAPI(m *models.OperationRouteCompany) api.OperationRouteCompany {
	return api.OperationRouteCompany{
		Id:               idPtr(m.ID),
		OperationRouteId: toUUID(m.OperationRouteId),
		CompanyId:        toUUID(m.CompanyId),
		Role:             m.Role,
		ValidFrom:        datePtr(m.ValidFrom),
		ValidTo:          datePtr(m.ValidTo),
	}
}

func OperationRouteCompanyFromCreate(b *api.CreateOperationRouteCompanyRequest) models.OperationRouteCompany {
	return models.OperationRouteCompany{
		OperationRouteId: uuidStr(b.OperationRouteId),
		CompanyId:        uuidStr(b.CompanyId),
		Role:             b.Role,
		ValidFrom:        fromDatePtr(b.ValidFrom),
		ValidTo:          fromDatePtr(b.ValidTo),
	}
}

func OperationRouteCompanyApplyUpdate(b *api.UpdateOperationRouteCompanyRequest, m *models.OperationRouteCompany) {
	m.OperationRouteId = uuidStr(b.OperationRouteId)
	m.CompanyId = uuidStr(b.CompanyId)
	m.Role = b.Role
	m.ValidFrom = fromDatePtr(b.ValidFrom)
	m.ValidTo = fromDatePtr(b.ValidTo)
}

// OperationRouteSection -------------------------------------------------------

func OperationRouteSectionToAPI(m *models.OperationRouteSection) api.OperationRouteSection {
	return api.OperationRouteSection{
		Id:               idPtr(m.ID),
		OperationRouteId: toUUID(m.OperationRouteId),
		RouteId:          toUUID(m.RouteId),
		Sequence:         m.Sequence,
		FromStationId:    toUUIDPtr(m.FromStationId),
		ToStationId:      toUUIDPtr(m.ToStationId),
		ValidFrom:        datePtr(m.ValidFrom),
		ValidTo:          datePtr(m.ValidTo),
	}
}

func OperationRouteSectionFromCreate(b *api.CreateOperationRouteSectionRequest) models.OperationRouteSection {
	return models.OperationRouteSection{
		OperationRouteId: uuidStr(b.OperationRouteId),
		RouteId:          uuidStr(b.RouteId),
		Sequence:         b.Sequence,
		FromStationId:    uuidStrPtr(b.FromStationId),
		ToStationId:      uuidStrPtr(b.ToStationId),
		ValidFrom:        fromDatePtr(b.ValidFrom),
		ValidTo:          fromDatePtr(b.ValidTo),
	}
}

func OperationRouteSectionApplyUpdate(b *api.UpdateOperationRouteSectionRequest, m *models.OperationRouteSection) {
	m.OperationRouteId = uuidStr(b.OperationRouteId)
	m.RouteId = uuidStr(b.RouteId)
	m.Sequence = b.Sequence
	m.FromStationId = uuidStrPtr(b.FromStationId)
	m.ToStationId = uuidStrPtr(b.ToStationId)
	m.ValidFrom = fromDatePtr(b.ValidFrom)
	m.ValidTo = fromDatePtr(b.ValidTo)
}

// OperationRouteStop ----------------------------------------------------------

func OperationRouteStopToAPI(m *models.OperationRouteStop) api.OperationRouteStop {
	return api.OperationRouteStop{
		Id:                 idPtr(m.ID),
		OperationRouteId:   toUUID(m.OperationRouteId),
		StationId:          toUUID(m.StationId),
		Sequence:           m.Sequence,
		DefaultStopType:    m.DefaultStopType,
		PassengerBoarding:  m.PassengerBoarding,
		PassengerAlighting: m.PassengerAlighting,
		ValidFrom:          datePtr(m.ValidFrom),
		ValidTo:            datePtr(m.ValidTo),
	}
}

func OperationRouteStopFromCreate(b *api.CreateOperationRouteStopRequest) models.OperationRouteStop {
	return models.OperationRouteStop{
		OperationRouteId:   uuidStr(b.OperationRouteId),
		StationId:          uuidStr(b.StationId),
		Sequence:           b.Sequence,
		DefaultStopType:    b.DefaultStopType,
		PassengerBoarding:  b.PassengerBoarding,
		PassengerAlighting: b.PassengerAlighting,
		ValidFrom:          fromDatePtr(b.ValidFrom),
		ValidTo:            fromDatePtr(b.ValidTo),
	}
}

func OperationRouteStopApplyUpdate(b *api.UpdateOperationRouteStopRequest, m *models.OperationRouteStop) {
	m.OperationRouteId = uuidStr(b.OperationRouteId)
	m.StationId = uuidStr(b.StationId)
	m.Sequence = b.Sequence
	m.DefaultStopType = b.DefaultStopType
	m.PassengerBoarding = b.PassengerBoarding
	m.PassengerAlighting = b.PassengerAlighting
	m.ValidFrom = fromDatePtr(b.ValidFrom)
	m.ValidTo = fromDatePtr(b.ValidTo)
}

// TimetableVersion ------------------------------------------------------------

func TimetableVersionToAPI(m *models.TimetableVersion) api.TimetableVersion {
	return api.TimetableVersion{
		Id:          idPtr(m.ID),
		Name:        m.Name,
		Description: m.Description,
		CompanyId:   toUUIDPtr(m.CompanyId),
		ValidFrom:   datePtr(m.ValidFrom),
		ValidTo:     datePtr(m.ValidTo),
		SourceUrl:   m.SourceUrl,
		ImportedAt:  m.ImportedAt,
	}
}

func TimetableVersionFromCreate(b *api.CreateTimetableVersionRequest) models.TimetableVersion {
	return models.TimetableVersion{
		Name:        b.Name,
		Description: b.Description,
		CompanyId:   uuidStrPtr(b.CompanyId),
		ValidFrom:   fromDatePtr(b.ValidFrom),
		ValidTo:     fromDatePtr(b.ValidTo),
		SourceUrl:   b.SourceUrl,
		ImportedAt:  b.ImportedAt,
	}
}

func TimetableVersionApplyUpdate(b *api.UpdateTimetableVersionRequest, m *models.TimetableVersion) {
	m.Name = b.Name
	m.Description = b.Description
	m.CompanyId = uuidStrPtr(b.CompanyId)
	m.ValidFrom = fromDatePtr(b.ValidFrom)
	m.ValidTo = fromDatePtr(b.ValidTo)
	m.SourceUrl = b.SourceUrl
	m.ImportedAt = b.ImportedAt
}

// ServiceCalendar -------------------------------------------------------------

func ServiceCalendarToAPI(m *models.ServiceCalendar) api.ServiceCalendar {
	return api.ServiceCalendar{
		Id:        idPtr(m.ID),
		Name:      m.Name,
		StartDate: datePtr(m.StartDate),
		EndDate:   datePtr(m.EndDate),
		Monday:    m.Monday,
		Tuesday:   m.Tuesday,
		Wednesday: m.Wednesday,
		Thursday:  m.Thursday,
		Friday:    m.Friday,
		Saturday:  m.Saturday,
		Sunday:    m.Sunday,
	}
}

func ServiceCalendarFromCreate(b *api.CreateServiceCalendarRequest) models.ServiceCalendar {
	return models.ServiceCalendar{
		Name:      b.Name,
		StartDate: fromDatePtr(b.StartDate),
		EndDate:   fromDatePtr(b.EndDate),
		Monday:    b.Monday,
		Tuesday:   b.Tuesday,
		Wednesday: b.Wednesday,
		Thursday:  b.Thursday,
		Friday:    b.Friday,
		Saturday:  b.Saturday,
		Sunday:    b.Sunday,
	}
}

func ServiceCalendarApplyUpdate(b *api.UpdateServiceCalendarRequest, m *models.ServiceCalendar) {
	m.Name = b.Name
	m.StartDate = fromDatePtr(b.StartDate)
	m.EndDate = fromDatePtr(b.EndDate)
	m.Monday = b.Monday
	m.Tuesday = b.Tuesday
	m.Wednesday = b.Wednesday
	m.Thursday = b.Thursday
	m.Friday = b.Friday
	m.Saturday = b.Saturday
	m.Sunday = b.Sunday
}

// ServiceCalendarException ----------------------------------------------------

func ServiceCalendarExceptionToAPI(m *models.ServiceCalendarException) api.ServiceCalendarException {
	return api.ServiceCalendarException{
		Id:            idPtr(m.ID),
		CalendarId:    toUUID(m.CalendarId),
		ServiceDate:   date(m.ServiceDate),
		ExceptionType: m.ExceptionType,
		Description:   m.Description,
	}
}

func ServiceCalendarExceptionFromCreate(b *api.CreateServiceCalendarExceptionRequest) models.ServiceCalendarException {
	return models.ServiceCalendarException{
		CalendarId:    uuidStr(b.CalendarId),
		ServiceDate:   fromDate(b.ServiceDate),
		ExceptionType: b.ExceptionType,
		Description:   b.Description,
	}
}

func ServiceCalendarExceptionApplyUpdate(b *api.UpdateServiceCalendarExceptionRequest, m *models.ServiceCalendarException) {
	m.CalendarId = uuidStr(b.CalendarId)
	m.ServiceDate = fromDate(b.ServiceDate)
	m.ExceptionType = b.ExceptionType
	m.Description = b.Description
}

// Train -----------------------------------------------------------------------

func TrainToAPI(m *models.Train) api.Train {
	return api.Train{
		Id:          idPtr(m.ID),
		TrainNumber: m.TrainNumber,
		TrainName:   m.TrainName,
		TrainType:   m.TrainType,
		Description: m.Description,
		ImageUrl:    m.ImageUrl,
		ValidFrom:   datePtr(m.ValidFrom),
		ValidTo:     datePtr(m.ValidTo),
	}
}

func TrainFromCreate(b *api.CreateTrainRequest) models.Train {
	return models.Train{
		TrainNumber: b.TrainNumber,
		TrainName:   b.TrainName,
		TrainType:   b.TrainType,
		Description: b.Description,
		ImageUrl:    b.ImageUrl,
		ValidFrom:   fromDatePtr(b.ValidFrom),
		ValidTo:     fromDatePtr(b.ValidTo),
	}
}

func TrainApplyUpdate(b *api.UpdateTrainRequest, m *models.Train) {
	m.TrainNumber = b.TrainNumber
	m.TrainName = b.TrainName
	m.TrainType = b.TrainType
	m.Description = b.Description
	m.ImageUrl = b.ImageUrl
	m.ValidFrom = fromDatePtr(b.ValidFrom)
	m.ValidTo = fromDatePtr(b.ValidTo)
}

// TrainRun --------------------------------------------------------------------

func TrainRunToAPI(m *models.TrainRun) api.TrainRun {
	return api.TrainRun{
		Id:                   idPtr(m.ID),
		TimetableVersionId:   toUUID(m.TimetableVersionId),
		OperationRouteId:     toUUIDPtr(m.OperationRouteId),
		CalendarId:           toUUIDPtr(m.CalendarId),
		TrainId:              toUUIDPtr(m.TrainId),
		TrainNumber:          m.TrainNumber,
		Direction:            m.Direction,
		OriginStationId:      toUUIDPtr(m.OriginStationId),
		DestinationStationId: toUUIDPtr(m.DestinationStationId),
		ServiceType:          m.ServiceType,
		Note:                 m.Note,
	}
}

func TrainRunFromCreate(b *api.CreateTrainRunRequest) models.TrainRun {
	return models.TrainRun{
		TimetableVersionId:   uuidStr(b.TimetableVersionId),
		OperationRouteId:     uuidStrPtr(b.OperationRouteId),
		CalendarId:           uuidStrPtr(b.CalendarId),
		TrainId:              uuidStrPtr(b.TrainId),
		TrainNumber:          b.TrainNumber,
		Direction:            b.Direction,
		OriginStationId:      uuidStrPtr(b.OriginStationId),
		DestinationStationId: uuidStrPtr(b.DestinationStationId),
		ServiceType:          b.ServiceType,
		Note:                 b.Note,
	}
}

func TrainRunApplyUpdate(b *api.UpdateTrainRunRequest, m *models.TrainRun) {
	m.TimetableVersionId = uuidStr(b.TimetableVersionId)
	m.OperationRouteId = uuidStrPtr(b.OperationRouteId)
	m.CalendarId = uuidStrPtr(b.CalendarId)
	m.TrainId = uuidStrPtr(b.TrainId)
	m.TrainNumber = b.TrainNumber
	m.Direction = b.Direction
	m.OriginStationId = uuidStrPtr(b.OriginStationId)
	m.DestinationStationId = uuidStrPtr(b.DestinationStationId)
	m.ServiceType = b.ServiceType
	m.Note = b.Note
}

// TrainRunStop ----------------------------------------------------------------

func TrainRunStopToAPI(m *models.TrainRunStop) api.TrainRunStop {
	return api.TrainRunStop{
		Id:                 idPtr(m.ID),
		TrainRunId:         toUUID(m.TrainRunId),
		StationId:          toUUID(m.StationId),
		PlatformId:         toUUIDPtr(m.PlatformId),
		Sequence:           m.Sequence,
		ArrivalTime:        m.ArrivalTime,
		DepartureTime:      m.DepartureTime,
		StopType:           m.StopType,
		PassengerBoarding:  m.PassengerBoarding,
		PassengerAlighting: m.PassengerAlighting,
		Note:               m.Note,
	}
}

func TrainRunStopFromCreate(b *api.CreateTrainRunStopRequest) models.TrainRunStop {
	return models.TrainRunStop{
		TrainRunId:         uuidStr(b.TrainRunId),
		StationId:          uuidStr(b.StationId),
		PlatformId:         uuidStrPtr(b.PlatformId),
		Sequence:           b.Sequence,
		ArrivalTime:        b.ArrivalTime,
		DepartureTime:      b.DepartureTime,
		StopType:           b.StopType,
		PassengerBoarding:  b.PassengerBoarding,
		PassengerAlighting: b.PassengerAlighting,
		Note:               b.Note,
	}
}

func TrainRunStopApplyUpdate(b *api.UpdateTrainRunStopRequest, m *models.TrainRunStop) {
	m.TrainRunId = uuidStr(b.TrainRunId)
	m.StationId = uuidStr(b.StationId)
	m.PlatformId = uuidStrPtr(b.PlatformId)
	m.Sequence = b.Sequence
	m.ArrivalTime = b.ArrivalTime
	m.DepartureTime = b.DepartureTime
	m.StopType = b.StopType
	m.PassengerBoarding = b.PassengerBoarding
	m.PassengerAlighting = b.PassengerAlighting
	m.Note = b.Note
}

// Media -----------------------------------------------------------------------

func MediaToAPI(m *models.Media) api.Media {
	return api.Media{
		Id:        idPtr(m.ID),
		MediaType: m.MediaType,
		Url:       m.Url,
		Title:     m.Title,
		Source:    m.Source,
		License:   m.License,
		CreatedAt: timePtr(m.CreatedAt),
	}
}

func MediaFromCreate(b *api.CreateMediaRequest) models.Media {
	return models.Media{
		MediaType: b.MediaType,
		Url:       b.Url,
		Title:     b.Title,
		Source:    b.Source,
		License:   b.License,
	}
}

func MediaApplyUpdate(b *api.UpdateMediaRequest, m *models.Media) {
	m.MediaType = b.MediaType
	m.Url = b.Url
	m.Title = b.Title
	m.Source = b.Source
	m.License = b.License
}

// MediaAttachment -------------------------------------------------------------

func MediaAttachmentToAPI(m *models.MediaAttachment) api.MediaAttachment {
	return api.MediaAttachment{
		Id:         idPtr(m.ID),
		MediaId:    toUUID(m.MediaId),
		EntityType: m.EntityType,
		EntityId:   toUUID(m.EntityId),
		SortOrder:  m.SortOrder,
	}
}

func MediaAttachmentFromCreate(b *api.CreateMediaAttachmentRequest) models.MediaAttachment {
	return models.MediaAttachment{
		MediaId:    uuidStr(b.MediaId),
		EntityType: b.EntityType,
		EntityId:   uuidStr(b.EntityId),
		SortOrder:  b.SortOrder,
	}
}

func MediaAttachmentApplyUpdate(b *api.UpdateMediaAttachmentRequest, m *models.MediaAttachment) {
	m.MediaId = uuidStr(b.MediaId)
	m.EntityType = b.EntityType
	m.EntityId = uuidStr(b.EntityId)
	m.SortOrder = b.SortOrder
}
