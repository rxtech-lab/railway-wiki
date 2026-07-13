import Foundation

struct AppConfiguration: Equatable, Sendable {
    let railwayAPIBaseURL: URL
    let overpassAPIBaseURL: URL
    let osmStackBaseURL: URL
    let osmMapStylePath: String
    let osmMapsAPIKey: String
    let authIssuerURL: URL
    let authClientID: String
    let authRedirectURI: String
    let authScopes: [String]
    let testAuthToken: String?

    var mapStyleURL: URL {
        osmStackBaseURL.appending(path: osmMapStylePath.trimmingCharacters(in: CharacterSet(charactersIn: "/")))
    }

    static func load(
        bundle: Bundle = .main,
        environment: [String: String] = ProcessInfo.processInfo.environment,
        launchArguments: [String] = ProcessInfo.processInfo.arguments
    ) throws -> AppConfiguration {
        var values: [String: String] = [:]
        let keys = [
            "RailwayAPIBaseURL", "OverpassAPIBaseURL", "OSMStackBaseURL",
            "OSMMapStylePath", "OSMMapsAPIKey", "AuthIssuerURL",
            "AuthClientID", "AuthRedirectURI", "AuthScopes"
        ]
        for key in keys {
            if let value = bundle.object(forInfoDictionaryKey: key) as? String {
                values[key] = value
            }
        }
        return try AppConfiguration(
            values: values,
            environment: environment,
            launchOverrides: e2eOverrides(from: launchArguments)
        )
    }

    init(
        values: [String: String],
        environment: [String: String] = [:],
        launchOverrides: [String: String] = [:]
    ) throws {
        func value(_ key: String, override: String? = nil) throws -> String {
            let raw = override.flatMap { launchOverrides[$0] ?? environment[$0] } ?? values[key] ?? ""
            let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !trimmed.isEmpty,
                  !trimmed.contains("$("),
                  !trimmed.hasPrefix("REQUIRED_"),
                  !trimmed.hasPrefix("REPLACE_")
            else {
                throw ConfigurationError.missing(key)
            }
            return trimmed
        }

        func webURL(_ key: String, override: String? = nil) throws -> URL {
            let raw = try value(key, override: override)
            guard let url = URL(string: raw), ["http", "https"].contains(url.scheme?.lowercased()) else {
                throw ConfigurationError.invalidURL(key)
            }
            return url
        }

        railwayAPIBaseURL = try webURL("RailwayAPIBaseURL", override: "E2E_API_BASE_URL")
        overpassAPIBaseURL = try webURL("OverpassAPIBaseURL", override: "E2E_OVERPASS_API_BASE_URL")
        osmStackBaseURL = try webURL("OSMStackBaseURL", override: "E2E_OSM_STACK_BASE_URL")
        osmMapStylePath = try value("OSMMapStylePath")
        osmMapsAPIKey = try value("OSMMapsAPIKey", override: "E2E_OSM_MAPS_API_KEY")
        authIssuerURL = try webURL("AuthIssuerURL")
        authClientID = try value("AuthClientID", override: "E2E_AUTH_CLIENT_ID")
        authRedirectURI = try value("AuthRedirectURI")
        guard let redirect = URL(string: authRedirectURI), redirect.scheme == "railwaywiki" else {
            throw ConfigurationError.invalidURL("AuthRedirectURI")
        }
        authScopes = try value("AuthScopes").split(separator: " ").map(String.init)
        testAuthToken = (launchOverrides["E2E_AUTH_TOKEN"] ?? environment["E2E_AUTH_TOKEN"])
            .flatMap { $0.isEmpty ? nil : $0 }
    }

    static func e2eOverrides(from arguments: [String]) -> [String: String] {
        var overrides: [String: String] = [:]
        var index = 0
        while index + 1 < arguments.count {
            let key = arguments[index]
            if key.hasPrefix("-E2E_") {
                overrides[String(key.dropFirst())] = arguments[index + 1]
                index += 2
            } else {
                index += 1
            }
        }
        return overrides
    }
}

enum ConfigurationError: LocalizedError, Equatable {
    case missing(String)
    case invalidURL(String)

    var errorDescription: String? {
        switch self {
        case .missing(let key):
            "Configure \(key) in Config/Secrets.Debug.xcconfig or Config/Secrets.Release.xcconfig."
        case .invalidURL(let key):
            "\(key) is not a valid configured URL."
        }
    }
}
