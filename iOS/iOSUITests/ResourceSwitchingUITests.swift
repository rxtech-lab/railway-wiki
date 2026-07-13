import Darwin
import XCTest

/// Drives the **real** railway-wiki server running in `E2E_MODE` (local SQLite,
/// management auth disabled, seeded sample data — see the server's `make run-e2e`
/// and `scripts/e2e-ios.sh`).
///
/// Guards the split-view bug where switching sidebar resources reused a stale
/// `@State` store in `ResourceListView`, so navigating Companies → Stations →
/// Station Codes → Platforms rendered the Companies list every time (every row
/// showing the seeded company "MTR"). The fix adds `.id(resource.id)` to the list
/// column in `AppShellView`.
///
/// The assertions key off `resource.<id>.list`, the list container's accessibility
/// id, which is derived from the list store's OWN resource. When the list is
/// switched but the store is stale (the bug), the id keeps reading the previous
/// resource, so the expected `resource.<new>.list` never appears. This is a
/// data-independent signal — it does not rely on server-generated record ids or on
/// the combined row accessibility labels.
///
/// The bug only reproduces in the regular size class, so the harness runs this on
/// an iPad destination.
final class ResourceSwitchingUITests: XCTestCase {
    /// Base URL of the e2e server the app should hit. Defaults to the port the
    /// harness starts the server on; overridable for other setups.
    private var baseURL: String {
        ProcessInfo.processInfo.environment["E2E_IOS_API_BASE_URL"] ?? "http://localhost:8080"
    }

    override func setUpWithError() throws {
        continueAfterFailure = false
        // This test drives the real server (unlike the mock-based UI tests), so it
        // only runs when that server is up. Skip gracefully otherwise so a plain
        // `xcodebuild test` / Cmd-U stays green — run it via scripts/e2e-ios.sh,
        // which boots the server in E2E_MODE first.
        try XCTSkipUnless(
            isServerReachable(),
            "railway-wiki server not reachable at \(baseURL). Run this via scripts/e2e-ios.sh (starts the server in E2E_MODE)."
        )
    }

    /// Raw TCP connect to the e2e server's port. Uses a BSD socket (not URLSession)
    /// so App Transport Security does not apply to the reachability probe. Every
    /// address getaddrinfo returns is tried, because "localhost" typically resolves
    /// to IPv6 (::1) first while the server answers on IPv4 loopback.
    private func isServerReachable() -> Bool {
        guard let url = URL(string: baseURL), let host = url.host else { return false }
        let port = url.port ?? 80
        var hints = addrinfo(
            ai_flags: 0, ai_family: AF_UNSPEC, ai_socktype: SOCK_STREAM,
            ai_protocol: 0, ai_addrlen: 0, ai_canonname: nil, ai_addr: nil, ai_next: nil
        )
        var result: UnsafeMutablePointer<addrinfo>?
        guard getaddrinfo(host, String(port), &hints, &result) == 0 else { return false }
        defer { freeaddrinfo(result) }
        var candidate = result
        while let info = candidate {
            let fd = socket(info.pointee.ai_family, info.pointee.ai_socktype, info.pointee.ai_protocol)
            if fd >= 0 {
                let connected = connect(fd, info.pointee.ai_addr, info.pointee.ai_addrlen) == 0
                close(fd)
                if connected { return true }
            }
            candidate = info.pointee.ai_next
        }
        return false
    }

    override func tearDown() {
        // Don't leak orientation into other tests in the same run.
        XCUIDevice.shared.orientation = .portrait
        super.tearDown()
    }

    @MainActor
    func testSwitchingResourcesDoesNotShowStaleCompanyRows() throws {
        // The stale-store bug lives in the regular size class split view. Pin
        // landscape so all three columns (sidebar / content / detail) are visible
        // and the sidebar is directly tappable. In portrait the iPad split view can
        // collapse the sidebar behind a toolbar toggle, which makes navigation
        // non-deterministic — the simulator (and CI) default to portrait.
        XCUIDevice.shared.orientation = .landscapeLeft

        let app = launchAgainstE2EServer()

        // Absorb app launch + admin-access verification (a dashboard round-trip)
        // before driving the sidebar. Generous timeout to tolerate a cold sim.
        XCTAssertTrue(
            element("sidebar.companies", in: app).waitForExistence(timeout: 60),
            "App shell sidebar should appear after admin-access verification"
        )

        // Companies first — this is the resource whose data leaked into the others.
        openResource("companies", in: app)
        assertResourceLoaded("companies", in: app)

        // Each subsequent switch must rebuild the list from its OWN store: the new
        // container id must appear and the stale Companies container must be gone.
        for resource in ["stations", "station-codes", "platforms"] {
            openResource(resource, in: app)
            assertResourceLoaded(resource, in: app)
            XCTAssertFalse(
                element("resource.companies.list", in: app).exists,
                "A stale Companies store must not back the \(resource) list"
            )
        }
    }

    // MARK: - Helpers

    @MainActor
    private func assertResourceLoaded(_ id: String, in app: XCUIApplication) {
        XCTAssertTrue(
            element("resource.\(id).list", in: app).waitForExistence(timeout: 20),
            "The \(id) list must be backed by the \(id) store"
        )
    }

    @MainActor
    private func launchAgainstE2EServer() -> XCUIApplication {
        let app = XCUIApplication()
        // Point the app at the local no-auth e2e server and supply a test auth
        // token so `AuthSession.isTestAuthenticated` skips the OAuth sign-in UI.
        app.launchArguments += [
            "-E2E_API_BASE_URL", baseURL,
            "-E2E_OVERPASS_API_BASE_URL", "\(baseURL)/api/management/overpass",
            "-E2E_OSM_STACK_BASE_URL", baseURL,
            "-E2E_OSM_MAPS_API_KEY", "ui-e2e-maps-key",
            "-E2E_AUTH_CLIENT_ID", "ui-e2e-client",
            "-E2E_AUTH_TOKEN", "ui-e2e-admin-token"
        ]
        app.launch()
        return app
    }

    @MainActor
    private func openResource(_ id: String, in app: XCUIApplication) {
        var item = element("sidebar.\(id)", in: app)
        // In compact width the sidebar is a pushed screen; pop back to reach it.
        if !item.waitForExistence(timeout: 2), app.navigationBars.buttons.firstMatch.exists {
            app.navigationBars.buttons.firstMatch.tap()
            item = element("sidebar.\(id)", in: app)
        }
        XCTAssertTrue(item.waitForExistence(timeout: 20), "sidebar.\(id) should exist")
        item.tap()
    }

    private func element(_ identifier: String, in app: XCUIApplication) -> XCUIElement {
        app.descendants(matching: .any).matching(identifier: identifier).firstMatch
    }
}
