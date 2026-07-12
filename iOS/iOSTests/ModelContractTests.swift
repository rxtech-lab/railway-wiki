import XCTest
@testable import iOS

@MainActor
final class ModelContractTests: XCTestCase {
    func testResourceRegistryCoversAllUniqueManagementResources() {
        let resources = ResourceDefinition.all

        XCTAssertEqual(resources.count, 22)
        XCTAssertEqual(Set(resources.map(\.id)).count, 22)
        XCTAssertEqual(resources.first?.id, "companies")
        XCTAssertEqual(resources.filter { $0.specialization == .station }.map(\.id), ["stations"])
        XCTAssertEqual(resources.filter { $0.specialization == .route }.map(\.id), ["routes"])
        XCTAssertTrue(ResourceGroup.allCases.allSatisfy { group in resources.contains { $0.group == group } })
        XCTAssertEqual(Set(resources.filter(\.searchable).map(\.id)), [
            "companies", "stations", "routes", "operation-routes",
            "timetable-versions", "service-calendars", "trains", "media"
        ])
    }

    func testRelationshipRegistryResolvesBackendOwnedForeignKeys() {
        XCTAssertEqual(ResourceDefinition.relationResource(for: "stationId")?.id, "stations")
        XCTAssertEqual(ResourceDefinition.relationResource(for: "destinationStationId")?.id, "stations")
        XCTAssertEqual(ResourceDefinition.relationResource(for: "timetableVersionId")?.id, "timetable-versions")
        XCTAssertNil(ResourceDefinition.relationResource(for: "unknownId"))
    }

    func testMapBoundsRejectsInvalidAndOverlyBroadSearches() {
        XCTAssertTrue(MapBounds(south: 22.27, west: 114.13, north: 22.34, east: 114.22).isSearchable)
        XCTAssertFalse(MapBounds(south: 22, west: 114, north: 25, east: 115).isSearchable)
        XCTAssertFalse(MapBounds(south: 23, west: 114, north: 22, east: 115).isSearchable)
        XCTAssertFalse(MapBounds(south: -91, west: 114, north: 22, east: 115).isSearchable)
    }

    func testOverpassCandidatePrefillsOnlyReviewedStationFields() {
        let candidate = OverpassCandidate(
            elementType: "node",
            elementId: 123,
            name: "Central",
            nameEn: "Central Station",
            ref: "CEN",
            railway: "station",
            mode: "rail",
            operatorName: "Example Rail",
            network: "Metro",
            latitude: 22.2819,
            longitude: 114.1582,
            tags: ["railway": "station"],
            importedStationId: nil
        )

        XCTAssertEqual(candidate.id, "node:123")
        XCTAssertEqual(candidate.stationValues["name"], .string("Central"))
        XCTAssertEqual(candidate.stationValues["stationNumber"], .string("CEN"))
        XCTAssertEqual(candidate.stationValues["latitude"], .number(22.2819))
        XCTAssertNil(candidate.stationValues["osmTags"])
    }

    func testRecordHidesServerManagedFieldsFromEditing() {
        let record = TestFixtures.record(id: "station-1", name: "Central", additionalValues: [
            "createdAt": .string("2026-07-12T00:00:00Z"),
            "osmElementId": .number(123),
            "description": .string("Interchange")
        ])

        XCTAssertEqual(record.title, "Central")
        XCTAssertNil(record.editableValues["id"])
        XCTAssertNil(record.editableValues["createdAt"])
        XCTAssertNil(record.editableValues["osmElementId"])
        XCTAssertEqual(record.editableValues["description"], .string("Interchange"))
    }

    func testJSONValueCodableAndFormDataRoundTrip() throws {
        let value = JSONValue.object([
            "name": .string("Central"),
            "active": .bool(true),
            "coordinate": .array([.number(22.28), .number(114.15)]),
            "optional": .null
        ])

        let decoded = try JSONDecoder().decode(JSONValue.self, from: JSONEncoder().encode(value))
        XCTAssertEqual(decoded, value)
        XCTAssertEqual(JSONValue(formData: value.formData), value)
    }

    func testNavigationSelectionSurvivesSizeClassSynchronization() throws {
        let navigation = NavigationModel()
        let station = TestFixtures.record(id: "station-1", name: "Central")
        navigation.select(.resource("stations"))
        navigation.select(station)

        navigation.synchronizeForCompact()

        XCTAssertEqual(navigation.compactPath, [
            .destination(.resource("stations")),
            .record(resourceID: "stations", recordID: "station-1")
        ])
        navigation.synchronizeForRegular()
        XCTAssertEqual(navigation.selectedRecord?.stableID, "station-1")

        navigation.compactPath = [.destination(.resource("stations"))]
        navigation.synchronizeForRegular()
        XCTAssertNil(navigation.selectedRecord)

        navigation.compactPath.append(.record(resourceID: "stations", recordID: "station-1"))
        navigation.synchronizeForRegular()
        navigation.restoreCompactRecord(station, resourceID: "stations")
        XCTAssertEqual(navigation.selectedSidebar, .resource("stations"))
        XCTAssertEqual(navigation.selectedRecord?.stableID, "station-1")
    }

    func testDependenciesReuseRouteDraftStoreUntilDiscarded() throws {
        let dependencies = AppDependencies(configuration: try TestFixtures.configuration())
        let route = TestFixtures.record(id: "route-1", name: "Harbour Line")
        let first = dependencies.routeEditor(for: route)

        first.setRouteName("Unsaved Draft")

        XCTAssertTrue(first === dependencies.routeEditor(for: route))
        XCTAssertEqual(dependencies.routeEditor(for: route).configuration?.route.title, "Unsaved Draft")
        dependencies.discardRouteEditor(id: "route-1")
        XCTAssertFalse(first === dependencies.routeEditor(for: route))
    }
}
