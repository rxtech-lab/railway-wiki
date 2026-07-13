import Foundation

@MainActor
final class APIClient {
    private let configuration: AppConfiguration
    private let auth: AuthSession
    private let session: URLSession
    private let decoder: JSONDecoder

    init(configuration: AppConfiguration, auth: AuthSession, session: URLSession = .shared) {
        self.configuration = configuration
        self.auth = auth
        self.session = session
        decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
    }

    func dashboard() async throws -> DashboardSnapshot {
        try await request(base: configuration.railwayAPIBaseURL, path: "api/management/dashboard")
    }

    func list(
        _ resource: ResourceDefinition,
        query: String = "",
        cursor: String? = nil,
        bounds: MapBounds? = nil,
        filters: [String: String] = [:]
    ) async throws -> ResourcePage {
        var items = [URLQueryItem(name: "limit", value: "100")]
        let normalizedQuery = query.trimmingCharacters(in: .whitespacesAndNewlines)
        if resource.searchable, !normalizedQuery.isEmpty { items.append(.init(name: "q", value: normalizedQuery)) }
        if let cursor { items.append(.init(name: "cursor", value: cursor)) }
        if let bounds {
            items.append(contentsOf: [
                .init(name: "south", value: bounds.south.description),
                .init(name: "west", value: bounds.west.description),
                .init(name: "north", value: bounds.north.description),
                .init(name: "east", value: bounds.east.description)
            ])
        }
        items.append(contentsOf: filters.sorted { $0.key < $1.key }.map { .init(name: $0.key, value: $0.value) })
        return try await request(
            base: configuration.railwayAPIBaseURL,
            path: "api/management/\(resource.id)",
            queryItems: items
        )
    }

    func get(_ resource: ResourceDefinition, id: String) async throws -> ResourceRecord {
        try await request(base: configuration.railwayAPIBaseURL, path: "api/management/\(resource.id)/\(id)")
    }

    func schema(_ resource: ResourceDefinition, action: String) async throws -> [String: JSONValue] {
        try await request(
            base: configuration.railwayAPIBaseURL,
            path: "api/management/\(resource.id)/schema",
            queryItems: [.init(name: "action", value: action)]
        )
    }

    func create(_ resource: ResourceDefinition, values: [String: JSONValue]) async throws -> ResourceRecord {
        try await request(
            base: configuration.railwayAPIBaseURL,
            path: "api/management/\(resource.id)",
            method: "POST",
            body: JSONValue.object(values)
        )
    }

    func update(_ resource: ResourceDefinition, id: String, values: [String: JSONValue]) async throws -> ResourceRecord {
        try await request(
            base: configuration.railwayAPIBaseURL,
            path: "api/management/\(resource.id)/\(id)",
            method: "PUT",
            body: JSONValue.object(values)
        )
    }

    func delete(_ resource: ResourceDefinition, id: String) async throws {
        let _: EmptyResponse = try await request(
            base: configuration.railwayAPIBaseURL,
            path: "api/management/\(resource.id)/\(id)",
            method: "DELETE"
        )
    }

    func overpassSearch(bounds: MapBounds) async throws -> OverpassCandidatePage {
        try await request(
            base: configuration.overpassAPIBaseURL,
            path: "stations",
            queryItems: [
                .init(name: "south", value: bounds.south.description),
                .init(name: "west", value: bounds.west.description),
                .init(name: "north", value: bounds.north.description),
                .init(name: "east", value: bounds.east.description)
            ]
        )
    }

    func overpassSearch(latitude: Double, longitude: Double) async throws -> OverpassCandidatePage {
        try await request(
            base: configuration.overpassAPIBaseURL,
            path: "stations",
            queryItems: [
                .init(name: "lat", value: latitude.description),
                .init(name: "lon", value: longitude.description),
                .init(name: "radiusMeters", value: "2000")
            ]
        )
    }

    func importCandidate(_ candidate: OverpassCandidate, overrides: [String: JSONValue]? = nil) async throws -> ResourceRecord {
        let body = JSONValue.object(
            overrides.map { ["overrides": .object($0)] } ?? [:]
        )
        return try await request(
            base: configuration.overpassAPIBaseURL,
            path: "stations/\(candidate.elementType)/\(candidate.elementId)/import",
            method: "POST",
            body: body
        )
    }

    func uploadMedia(
        fileURL: URL,
        contentType: String,
        mediaType: String?
    ) async throws -> URL {
        let values = try fileURL.resourceValues(forKeys: [.fileSizeKey])
        var body: [String: JSONValue] = [
            "fileName": .string(fileURL.lastPathComponent),
            "contentType": .string(contentType),
            "sizeBytes": .number(Double(values.fileSize ?? 0))
        ]
        if let mediaType { body["mediaType"] = .string(mediaType) }
        let target: MediaUploadTarget = try await request(
            base: configuration.railwayAPIBaseURL,
            path: "api/management/media/upload-url",
            method: "POST",
            body: .object(body)
        )
        var upload = URLRequest(url: target.uploadUrl)
        upload.httpMethod = target.method
        target.headers?.forEach { upload.setValue($0.value, forHTTPHeaderField: $0.key) }
        let (_, response) = try await session.upload(for: upload, fromFile: fileURL)
        guard let http = response as? HTTPURLResponse,
              (200 ..< 300).contains(http.statusCode)
        else { throw APIError.server(status: 502, message: "Object storage rejected the upload.") }
        return target.url
    }

    func routeConfiguration(id: String) async throws -> RouteConfiguration {
        try await request(base: configuration.railwayAPIBaseURL, path: "api/management/routes/\(id)/configuration")
    }

    func validateRoute(id: String, configuration: RouteConfiguration) async throws -> RouteValidationResponse {
        try await request(
            base: self.configuration.railwayAPIBaseURL,
            path: "api/management/routes/\(id)/configuration/validate",
            method: "POST",
            body: routeConfigurationBody(configuration)
        )
    }

    func saveRoute(id: String, configuration: RouteConfiguration) async throws -> RouteConfiguration {
        try await request(
            base: self.configuration.railwayAPIBaseURL,
            path: "api/management/routes/\(id)/configuration",
            method: "PUT",
            body: routeConfigurationBody(configuration)
        )
    }

    private func routeConfigurationBody(_ configuration: RouteConfiguration) -> JSONValue {
        let stations = configuration.stations.map { station -> JSONValue in
            var values: [String: JSONValue] = ["stationId": .string(station.stationId)]
            if let id = station.id { values["id"] = .string(id) }
            if let distance = station.distanceFromStartKm { values["distanceFromStartKm"] = .number(distance) }
            if let validFrom = station.validFrom { values["validFrom"] = .string(validFrom) }
            if let validTo = station.validTo { values["validTo"] = .string(validTo) }
            return .object(values)
        }
        return .object([
            "route": .object(configuration.route.editableValues),
            "stations": .array(stations)
        ])
    }

    private func request<Response: Decodable>(
        base: URL,
        path: String,
        queryItems: [URLQueryItem] = [],
        method: String = "GET",
        body: JSONValue? = nil,
        didRefresh: Bool = false
    ) async throws -> Response {
        guard var components = URLComponents(url: base.appending(path: path), resolvingAgainstBaseURL: false) else {
            throw APIError.invalidURL
        }
        components.queryItems = queryItems.isEmpty ? nil : queryItems
        guard let url = components.url else { throw APIError.invalidURL }
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        if let token = auth.accessToken { request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization") }
        if let body {
            request.httpBody = try JSONEncoder().encode(body)
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        }
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse else { throw APIError.invalidResponse }
        if http.statusCode == 401, !didRefresh, await auth.refreshedAccessToken() != nil {
            return try await self.request(
                base: base, path: path, queryItems: queryItems, method: method, body: body, didRefresh: true
            )
        }
        if http.statusCode == 403 {
            auth.markAdminAccessDenied()
            throw APIError.forbidden
        }
        guard (200 ..< 300).contains(http.statusCode) else {
            let server = try? decoder.decode(APIErrorBody.self, from: data)
            throw APIError.server(status: http.statusCode, message: server?.error ?? "Request failed")
        }
        let responseData = data.isEmpty && Response.self == EmptyResponse.self
            ? Data("{}".utf8)
            : data
        return try decoder.decode(Response.self, from: responseData)
    }
}

struct EmptyResponse: Codable {}

enum APIError: LocalizedError, Equatable {
    case invalidURL
    case invalidResponse
    case forbidden
    case server(status: Int, message: String)

    var errorDescription: String? {
        switch self {
        case .invalidURL: "The configured API URL is invalid."
        case .invalidResponse: "The server returned an invalid response."
        case .forbidden: "Your account does not have Railway Wiki administrator access."
        case .server(_, let message): message
        }
    }
}
