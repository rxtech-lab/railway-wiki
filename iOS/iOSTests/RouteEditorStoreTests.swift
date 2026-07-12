import XCTest
@testable import iOS

@MainActor
final class RouteEditorStoreTests: XCTestCase {
    func testAddMoveUpdateUndoAndRedo() throws {
        let store = RouteEditorStore(
            routeID: "route-1",
            initial: TestFixtures.record(id: "route-1", name: "Harbour Line")
        )
        let central = TestFixtures.record(id: "station-central", name: "Central")
        let admiralty = TestFixtures.record(id: "station-admiralty", name: "Admiralty")

        store.addStation(central)
        store.addStation(admiralty)
        XCTAssertEqual(store.configuration?.stations.map(\.stationId), ["station-central", "station-admiralty"])

        store.move(from: IndexSet(integer: 1), to: 0)
        XCTAssertEqual(store.configuration?.stations.map(\.stationId), ["station-admiralty", "station-central"])

        var selected = try XCTUnwrap(store.configuration?.stations.first)
        selected.distanceFromStartKm = 3.2
        store.updateStation(selected)
        XCTAssertEqual(store.configuration?.stations.first?.distanceFromStartKm, 3.2)

        store.undo()
        XCTAssertNil(store.configuration?.stations.first?.distanceFromStartKm)
        store.redo()
        XCTAssertEqual(store.configuration?.stations.first?.distanceFromStartKm, 3.2)
    }

    func testDuplicateStationIsIgnoredAndNewStationCanBeRemoved() {
        let store = RouteEditorStore(
            routeID: "route-1",
            initial: TestFixtures.record(id: "route-1", name: "Harbour Line")
        )
        let station = TestFixtures.record(id: "station-central", name: "Central")

        store.addStation(station)
        store.addStation(station)
        XCTAssertEqual(store.configuration?.stations.count, 1)

        store.removeSelectedStation()
        XCTAssertTrue(store.configuration?.stations.isEmpty == true)
        XCTAssertNil(store.selectedStationID)
        XCTAssertTrue(store.canUndo)
    }

    func testRouteNameParticipatesInUndoHistory() {
        let store = RouteEditorStore(
            routeID: "route-1",
            initial: TestFixtures.record(id: "route-1", name: "Old Name")
        )

        store.setRouteName("New Name")
        XCTAssertEqual(store.configuration?.route.values["name"], .string("New Name"))
        store.undo()
        XCTAssertEqual(store.configuration?.route.values["name"], .string("Old Name"))
    }
}
