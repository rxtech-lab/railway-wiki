import Foundation
import Observation
import SwiftUI

@MainActor
@Observable
final class RouteEditorStore {
    let routeID: String
    var configuration: RouteConfiguration?
    var stationTitles: [String: String] = [:]
    var selectedStationID: String?
    var validationErrors: [String] = []
    var isLoading = false
    var isSaving = false
    var errorMessage: String?
    private(set) var hasLoaded = false
    private var undoStack: [RouteConfiguration] = []
    private var redoStack: [RouteConfiguration] = []

    init(routeID: String, initial: ResourceRecord) {
        self.routeID = routeID
        configuration = RouteConfiguration(route: initial, stations: [])
    }

    var canUndo: Bool { !undoStack.isEmpty }
    var canRedo: Bool { !redoStack.isEmpty }

    func load(api: APIClient) async {
        guard !hasLoaded, !isLoading else { return }
        isLoading = true
        defer { isLoading = false }
        do {
            configuration = try await api.routeConfiguration(id: routeID)
            selectedStationID = configuration?.stations.first?.stableID
            await loadStationTitles(api: api)
            hasLoaded = true
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func setRouteName(_ name: String) {
        mutate { $0.route.values["name"] = .string(name) }
    }

    func addStation(_ station: ResourceRecord) {
        guard let id = station.stableID,
              configuration?.stations.contains(where: { $0.stationId == id }) == false
        else { return }
        mutate { $0.stations.append(.init(id: nil, routeId: routeID, stationId: id, sequence: nil)) }
        stationTitles[id] = station.title
        selectedStationID = "new:\(id)"
    }

    func updateStation(_ draft: RouteStationDraft) {
        mutate { configuration in
            guard let index = configuration.stations.firstIndex(where: { $0.stableID == draft.stableID }) else { return }
            configuration.stations[index] = draft
        }
    }

    func removeSelectedStation() {
        guard let selectedStationID else { return }
        mutate { $0.stations.removeAll { $0.stableID == selectedStationID } }
        self.selectedStationID = configuration?.stations.first?.stableID
    }

    func move(from source: IndexSet, to destination: Int) {
        mutate { $0.stations.move(fromOffsets: source, toOffset: destination) }
    }

    func undo() {
        guard let previous = undoStack.popLast(), let current = configuration else { return }
        redoStack.append(current)
        configuration = previous
    }

    func redo() {
        guard let next = redoStack.popLast(), let current = configuration else { return }
        undoStack.append(current)
        configuration = next
    }

    func validate(api: APIClient) async {
        guard let configuration else { return }
        do {
            let result = try await api.validateRoute(id: routeID, configuration: configuration)
            validationErrors = result.errors
            if result.valid { validationErrors = ["Configuration is valid."] }
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func save(api: APIClient) async {
        guard let configuration else { return }
        isSaving = true
        defer { isSaving = false }
        do {
            let validation = try await api.validateRoute(id: routeID, configuration: configuration)
            guard validation.valid else {
                validationErrors = validation.errors
                return
            }
            self.configuration = try await api.saveRoute(id: routeID, configuration: configuration)
            undoStack.removeAll()
            redoStack.removeAll()
            validationErrors = ["Changes saved."]
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func mutate(_ change: (inout RouteConfiguration) -> Void) {
        guard var configuration else { return }
        undoStack.append(configuration)
        if undoStack.count > 50 { undoStack.removeFirst() }
        redoStack.removeAll()
        change(&configuration)
        self.configuration = configuration
    }

    private func loadStationTitles(api: APIClient) async {
        guard let stationResource = ResourceDefinition.all.first(where: { $0.id == "stations" }),
              let stationIDs = configuration?.stations.map(\.stationId)
        else { return }
        await withTaskGroup(of: (String, String?).self) { group in
            for id in Set(stationIDs) {
                group.addTask { (id, try? await api.get(stationResource, id: id).title) }
            }
            for await (id, title) in group where title != nil { stationTitles[id] = title }
        }
    }
}
