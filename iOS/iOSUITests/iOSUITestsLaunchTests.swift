import XCTest

final class RailwayWikiUITestsLaunchTests: XCTestCase {
    private var server: MockRailwayServer!

    override class var runsForEachTargetApplicationUIConfiguration: Bool { true }

    override func setUpWithError() throws {
        continueAfterFailure = false
        server = try MockRailwayServer()
    }

    override func tearDownWithError() throws {
        server.stop()
        server = nil
    }

    @MainActor
    func testLaunch() throws {
        let app = XCUIApplication()
        app.launchArguments += [
            "-E2E_API_BASE_URL", server.baseURL,
            "-E2E_OVERPASS_API_BASE_URL", "\(server.baseURL)/api/management/overpass",
            "-E2E_OSM_STACK_BASE_URL", server.baseURL,
            "-E2E_OSM_MAPS_API_KEY", "launch-fixture-key",
            "-E2E_AUTH_CLIENT_ID", "launch-fixture-client",
            "-E2E_AUTH_TOKEN", "launch-fixture-token"
        ]
        app.launch()

        XCTAssertTrue(app.descendants(matching: .any)["sidebar.dashboard"].waitForExistence(timeout: 5))
        let attachment = XCTAttachment(screenshot: app.screenshot())
        attachment.name = "Authenticated Launch"
        attachment.lifetime = .keepAlways
        add(attachment)
    }
}
