import Foundation
import XCTest
@testable import iOS

@MainActor
final class MapNetworkConfiguratorTests: XCTestCase {
    override func tearDown() {
        MapAPIKeyURLProtocol.configure(host: nil, apiKey: "")
        super.tearDown()
    }

    func testMapProtocolInterceptsOnlyConfiguredHost() {
        MapAPIKeyURLProtocol.configure(host: "maps.example.test", apiKey: "maps-only-key")

        XCTAssertTrue(MapAPIKeyURLProtocol.canInit(with: request("https://maps.example.test/maps/styles/basic/style.json")))
        XCTAssertTrue(MapAPIKeyURLProtocol.canInit(with: request("https://MAPS.EXAMPLE.TEST/maps/tiles/1/2/3.pbf")))
        XCTAssertFalse(MapAPIKeyURLProtocol.canInit(with: request("https://api.example.test/api/management/stations")))
        XCTAssertFalse(MapAPIKeyURLProtocol.canInit(with: request("https://maps.example.test.evil.invalid/maps/tiles/1/2/3.pbf")))
    }

    func testMapProtocolDoesNotHandleARequestTwice() {
        MapAPIKeyURLProtocol.configure(host: "maps.example.test", apiKey: "maps-only-key")
        let mutable = NSMutableURLRequest(url: URL(string: "https://maps.example.test/maps/style.json")!)
        URLProtocol.setProperty(true, forKey: "RailwayWikiMapAPIKeyHandled", in: mutable)

        XCTAssertFalse(MapAPIKeyURLProtocol.canInit(with: mutable as URLRequest))
    }

    private func request(_ string: String) -> URLRequest {
        URLRequest(url: URL(string: string)!)
    }
}
