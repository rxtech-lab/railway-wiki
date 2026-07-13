import XCTest

final class RailwayWikiUITests: XCTestCase {
    private var server: MockRailwayServer!

    override func setUpWithError() throws {
        continueAfterFailure = false
        server = try MockRailwayServer()
    }

    override func tearDownWithError() throws {
        server.stop()
        server = nil
    }

    @MainActor
    func testAuthenticatedDashboardAndStationNavigation() throws {
        let app = launchFixtureApp()

        openDashboard(in: app)
        XCTAssertTrue(element("dashboard", in: app).waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["Station Coordinate Coverage"].exists)

        openStations(in: app)
        XCTAssertTrue(element("resource.stations.list", in: app).waitForExistence(timeout: 5))
        let station = element("record.station-central", in: app)
        XCTAssertTrue(station.waitForExistence(timeout: 5))
        station.tap()
        XCTAssertTrue(element("resource.stations.detail", in: app).waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["station-central"].exists)
    }

    @MainActor
    func testCreateSheetCannotBeDismissedInteractivelyAndHasCloseAction() throws {
        let app = launchFixtureApp()
        openStations(in: app)

        let add = app.buttons["resource.stations.add"]
        XCTAssertTrue(add.waitForExistence(timeout: 5))
        add.tap()
        let close = app.buttons["form.close"]
        XCTAssertTrue(close.waitForExistence(timeout: 5))

        app.swipeDown(velocity: .fast)
        XCTAssertTrue(close.exists, "Schema-backed sheets must ignore interactive dismissal")
        close.tap()
        XCTAssertFalse(close.waitForExistence(timeout: 2))
    }

    @MainActor
    func testSelectionSurvivesOrientationChange() throws {
        XCUIDevice.shared.orientation = .landscapeLeft
        let app = launchFixtureApp()
        openStations(in: app)
        XCTAssertTrue(element("resource.stations.list", in: app).waitForExistence(timeout: 5))

        XCUIDevice.shared.orientation = .portrait

        XCTAssertTrue(app.navigationBars["Stations"].waitForExistence(timeout: 5))
        XCTAssertTrue(element("record.station-central", in: app).exists)
    }

    @MainActor
    func testMapUsesLeadingPanelAndPreservesResultsAcrossRotation() throws {
        XCUIDevice.shared.orientation = .portrait
        let app = launchFixtureApp()
        openMap(in: app)

        let map = element("map.page", in: app)
        let locationControl = element("map.currentLocation", in: app)
        let searchControl = element("map.searchViewport", in: app)
        let candidatePanel = element("map.candidateList", in: app)
        XCTAssertTrue(map.waitForExistence(timeout: 5))
        XCTAssertTrue(locationControl.waitForExistence(timeout: 5))
        XCTAssertTrue(searchControl.waitForExistence(timeout: 5))
        XCTAssertTrue(candidatePanel.waitForExistence(timeout: 5))

        expectation(for: NSPredicate(format: "enabled == true"), evaluatedWith: searchControl)
        waitForExpectations(timeout: 10)
        searchControl.tap()

        let candidate = element("map.candidate.node:123", in: app)
        XCTAssertTrue(candidate.waitForExistence(timeout: 5))
        candidate.tap()
        XCTAssertTrue(app.buttons["map.candidate.review"].exists)
        app.buttons["Close"].tap()

        XCUIDevice.shared.orientation = .landscapeLeft

        XCTAssertTrue(candidate.waitForExistence(timeout: 5))
        XCTAssertTrue(candidatePanel.waitForExistence(timeout: 5))
        XCTAssertLessThanOrEqual(candidatePanel.frame.width, 380)
        XCTAssertLessThan(candidatePanel.frame.maxX, locationControl.frame.minX)

        candidate.tap()
        let review = app.buttons["map.candidate.review"]
        XCTAssertTrue(review.waitForExistence(timeout: 3))
        review.tap()
        let formClose = app.buttons["form.close"]
        XCTAssertTrue(formClose.waitForExistence(timeout: 5))
        formClose.tap()

        candidate.tap()
        let add = app.buttons["map.candidate.add"]
        XCTAssertTrue(add.waitForExistence(timeout: 3))
        add.tap()
        XCTAssertTrue(app.staticTexts["Already in Railway Wiki"].waitForExistence(timeout: 5))
    }

    @MainActor
    private func launchFixtureApp() -> XCUIApplication {
        let app = XCUIApplication()
        app.launchArguments += [
            "-E2E_API_BASE_URL", server.baseURL,
            "-E2E_OVERPASS_API_BASE_URL", "\(server.baseURL)/api/management/overpass",
            "-E2E_OSM_STACK_BASE_URL", server.baseURL,
            "-E2E_OSM_MAPS_API_KEY", "ui-fixture-maps-key",
            "-E2E_AUTH_CLIENT_ID", "ui-fixture-client",
            "-E2E_AUTH_TOKEN", "ui-fixture-admin-token"
        ]
        app.launch()
        return app
    }

    @MainActor
    private func openDashboard(in app: XCUIApplication) {
        if element("dashboard", in: app).waitForExistence(timeout: 2) { return }
        let dashboard = element("sidebar.dashboard", in: app)
        XCTAssertTrue(dashboard.waitForExistence(timeout: 5))
        dashboard.tap()
    }

    @MainActor
    private func openStations(in app: XCUIApplication) {
        var stations = element("sidebar.stations", in: app)
        if !stations.isHittable, app.navigationBars.buttons.firstMatch.exists {
            app.navigationBars.buttons.firstMatch.tap()
            stations = element("sidebar.stations", in: app)
        }
        XCTAssertTrue(stations.waitForExistence(timeout: 5))
        stations.tap()
    }

    @MainActor
    private func openMap(in app: XCUIApplication) {
        let map = element("sidebar.map", in: app)
        XCTAssertTrue(map.waitForExistence(timeout: 5))
        map.tap()
    }

    private func element(_ identifier: String, in app: XCUIApplication) -> XCUIElement {
        app.descendants(matching: .any).matching(identifier: identifier).firstMatch
    }
}
