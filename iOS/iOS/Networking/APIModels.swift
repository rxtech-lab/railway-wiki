import Foundation

struct Pagination: Codable, Sendable { let next: String? }
struct ResourcePage: Codable, Sendable { let items: [ResourceRecord]; let pagination: Pagination }

struct DashboardResourceMetric: Codable, Identifiable, Sendable {
    let resource: String
    let count: Int
    var id: String { resource }
}

struct DashboardStationCoverage: Codable, Sendable { let total: Int; let withCoordinates: Int }

struct DashboardSnapshot: Codable, Sendable {
    let resources: [DashboardResourceMetric]
    let stationCoverage: DashboardStationCoverage
    let generatedAt: Date
}

struct MapBounds: Equatable, Sendable {
    let south: Double
    let west: Double
    let north: Double
    let east: Double

    var isSearchable: Bool {
        south >= -90 && north <= 90 && west >= -180 && east <= 180 &&
            south < north && west < east && north - south <= 2 && east - west <= 2
    }
}

struct OverpassCandidatePage: Codable, Sendable { let items: [OverpassCandidate]; let generatedAt: Date }

struct OverpassCandidate: Codable, Hashable, Identifiable, Sendable {
    let elementType: String
    let elementId: Int64
    let name: String
    let nameEn: String?
    let ref: String?
    let railway: String?
    let mode: String?
    let operatorName: String?
    let network: String?
    let latitude: Double
    let longitude: Double
    let tags: [String: String]
    var importedStationId: String?

    var id: String { "\(elementType):\(elementId)" }

    enum CodingKeys: String, CodingKey {
        case elementType, elementId, name, nameEn, ref, railway, mode, network
        case operatorName = "operator"
        case latitude, longitude, tags, importedStationId
    }

    var stationValues: [String: JSONValue] {
        var values: [String: JSONValue] = [
            "name": .string(name),
            "latitude": .number(latitude),
            "longitude": .number(longitude)
        ]
        if let nameEn { values["nameEn"] = .string(nameEn) }
        if let ref { values["stationNumber"] = .string(ref) }
        return values
    }
}

struct RouteConfiguration: Codable, Hashable, Sendable {
    var route: ResourceRecord
    var stations: [RouteStationDraft]
}

struct RouteStationDraft: Codable, Hashable, Identifiable, Sendable {
    var id: String?
    var routeId: String?
    var stationId: String
    var sequence: Int?
    var distanceFromStartKm: Double?
    var validFrom: String?
    var validTo: String?

    var stableID: String { id ?? "new:\(stationId)" }
}

struct RouteValidationResponse: Codable, Sendable {
    let valid: Bool
    let errors: [String]
}

struct APIErrorBody: Codable { let error: String; let code: String }

struct MediaUploadTarget: Codable, Sendable {
    let uploadUrl: URL
    let method: String
    let headers: [String: String]?
    let url: URL
    let key: String?
    let expiresAt: Date
}
