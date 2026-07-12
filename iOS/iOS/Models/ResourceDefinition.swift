import Foundation

enum ResourceGroup: String, CaseIterable, Identifiable {
    case infrastructure = "Infrastructure"
    case operations = "Operations"
    case timetables = "Timetables"
    case media = "Media"
    var id: String { rawValue }
}

enum ResourceSpecialization: Hashable, Sendable {
    case generic
    case station
    case route
}

struct ResourceDefinition: Identifiable, Hashable, Sendable {
    let id: String
    let singular: String
    let plural: String
    let icon: String
    let group: ResourceGroup
    let searchable: Bool
    let specialization: ResourceSpecialization

    static let all: [ResourceDefinition] = [
        .init("companies", "Company", "Companies", "building.2", .infrastructure, true),
        .init("stations", "Station", "Stations", "tram.fill", .infrastructure, true, .station),
        .init("station-codes", "Station Code", "Station Codes", "number", .infrastructure),
        .init("station-transfers", "Station Transfer", "Station Transfers", "arrow.left.arrow.right", .infrastructure),
        .init("platforms", "Platform", "Platforms", "rectangle.split.3x1", .infrastructure),
        .init("routes", "Route", "Routes", "point.topleft.down.curvedto.point.bottomright.up", .infrastructure, true, .route),
        .init("route-companies", "Route Company", "Route Companies", "building.2.crop.circle", .infrastructure),
        .init("route-stations", "Route Station", "Route Stations", "list.number", .infrastructure),
        .init("track-segments", "Track Segment", "Track Segments", "road.lanes", .infrastructure),
        .init("platform-tracks", "Platform Track", "Platform Tracks", "link", .infrastructure),
        .init("operation-routes", "Operation Route", "Operation Routes", "arrow.triangle.branch", .operations, true),
        .init(
            "operation-route-companies", "Operation Route Company", "Operation Route Companies",
            "building.columns", .operations
        ),
        .init(
            "operation-route-sections", "Operation Route Section", "Operation Route Sections",
            "square.split.2x1", .operations
        ),
        .init("operation-route-stops", "Operation Route Stop", "Operation Route Stops", "mappin.and.ellipse", .operations),
        .init("timetable-versions", "Timetable Version", "Timetable Versions", "calendar.badge.clock", .timetables, true),
        .init("service-calendars", "Service Calendar", "Service Calendars", "calendar", .timetables, true),
        .init(
            "service-calendar-exceptions", "Calendar Exception", "Calendar Exceptions",
            "calendar.badge.exclamationmark", .timetables
        ),
        .init("trains", "Train", "Trains", "train.side.front.car", .operations, true),
        .init("train-runs", "Train Run", "Train Runs", "clock.arrow.trianglehead.counterclockwise.rotate.90", .timetables),
        .init("train-run-stops", "Train Run Stop", "Train Run Stops", "signpost.right", .timetables),
        .init("media", "Media", "Media", "photo.on.rectangle", .media, true),
        .init("media-attachments", "Media Attachment", "Media Attachments", "paperclip", .media)
    ]

    init(
        _ id: String,
        _ singular: String,
        _ plural: String,
        _ icon: String,
        _ group: ResourceGroup,
        _ searchable: Bool = false,
        _ specialization: ResourceSpecialization = .generic
    ) {
        self.id = id
        self.singular = singular
        self.plural = plural
        self.icon = icon
        self.group = group
        self.searchable = searchable
        self.specialization = specialization
    }

    static func relationResource(for field: String) -> ResourceDefinition? {
        let id: String? = switch field {
        case "companyId": "companies"
        case "stationId", "fromStationId", "toStationId", "originStationId", "destinationStationId": "stations"
        case "routeId": "routes"
        case "operationRouteId": "operation-routes"
        case "calendarId": "service-calendars"
        case "timetableVersionId": "timetable-versions"
        case "trainId": "trains"
        case "trainRunId": "train-runs"
        case "platformId": "platforms"
        case "trackSegmentId": "track-segments"
        case "mediaId": "media"
        default: nil
        }
        return all.first { $0.id == id }
    }
}
