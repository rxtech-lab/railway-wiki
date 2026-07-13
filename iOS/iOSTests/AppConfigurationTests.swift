import XCTest
@testable import iOS

@MainActor
final class AppConfigurationTests: XCTestCase {
    func testValidConfigurationBuildsStyleURLAndScopes() throws {
        let configuration = try TestFixtures.configuration(environment: [:])

        XCTAssertEqual(configuration.mapStyleURL.absoluteString, "https://maps.example.test/maps/styles/basic/style.json")
        XCTAssertEqual(configuration.authScopes, ["openid", "profile", "email"])
        XCTAssertNil(configuration.testAuthToken)
    }

    func testEnvironmentOverridesServiceEndpointsAndTestToken() throws {
        let configuration = try TestFixtures.configuration(environment: [
            "E2E_API_BASE_URL": "http://127.0.0.1:18080",
            "E2E_OVERPASS_API_BASE_URL": "http://127.0.0.1:18080/overpass",
            "E2E_OSM_STACK_BASE_URL": "http://127.0.0.1:18081",
            "E2E_OSM_MAPS_API_KEY": "fixture-key",
            "E2E_AUTH_CLIENT_ID": "fixture-client",
            "E2E_AUTH_TOKEN": "fixture-token"
        ])

        XCTAssertEqual(configuration.railwayAPIBaseURL.absoluteString, "http://127.0.0.1:18080")
        XCTAssertEqual(configuration.overpassAPIBaseURL.absoluteString, "http://127.0.0.1:18080/overpass")
        XCTAssertEqual(configuration.osmStackBaseURL.absoluteString, "http://127.0.0.1:18081")
        XCTAssertEqual(configuration.osmMapsAPIKey, "fixture-key")
        XCTAssertEqual(configuration.authClientID, "fixture-client")
        XCTAssertEqual(configuration.testAuthToken, "fixture-token")
    }

    func testLaunchArgumentsTakePrecedenceOverEnvironment() throws {
        let configuration = try TestFixtures.configuration(
            environment: ["E2E_API_BASE_URL": "https://environment.example", "E2E_AUTH_TOKEN": "environment-token"],
            launchOverrides: ["E2E_API_BASE_URL": "https://arguments.example", "E2E_AUTH_TOKEN": "arguments-token"]
        )

        XCTAssertEqual(configuration.railwayAPIBaseURL.host, "arguments.example")
        XCTAssertEqual(configuration.testAuthToken, "arguments-token")
    }

    func testLaunchArgumentParserAcceptsOnlyE2EKeyValuePairs() {
        let overrides = AppConfiguration.e2eOverrides(from: [
            "Railway Wiki",
            "-AppleLanguages", "(en)",
            "-E2E_API_BASE_URL", "http://127.0.0.1:18080",
            "-E2E_AUTH_TOKEN", "fixture-token",
            "unpaired-value"
        ])

        XCTAssertEqual(overrides, [
            "E2E_API_BASE_URL": "http://127.0.0.1:18080",
            "E2E_AUTH_TOKEN": "fixture-token"
        ])
    }

    func testRejectsUnresolvedPlaceholders() {
        var values = TestFixtures.configurationValues
        values["OSMMapsAPIKey"] = "REQUIRED_DEBUG_OSM_MAPS_API_KEY"

        XCTAssertThrowsError(try AppConfiguration(values: values)) { error in
            XCTAssertEqual(error as? ConfigurationError, .missing("OSMMapsAPIKey"))
        }
    }

    func testRejectsNonWebAPIURLAndWrongCallbackScheme() {
        var values = TestFixtures.configurationValues
        values["RailwayAPIBaseURL"] = "railwaywiki://api"
        XCTAssertThrowsError(try AppConfiguration(values: values)) { error in
            XCTAssertEqual(error as? ConfigurationError, .invalidURL("RailwayAPIBaseURL"))
        }

        values = TestFixtures.configurationValues
        values["AuthRedirectURI"] = "otherapp://oauth-callback"
        XCTAssertThrowsError(try AppConfiguration(values: values)) { error in
            XCTAssertEqual(error as? ConfigurationError, .invalidURL("AuthRedirectURI"))
        }
    }
}
