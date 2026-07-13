import Foundation
import Observation

@MainActor
@Observable
final class MapStore {
    var savedStations: [ResourceRecord] = []
    var candidates: [OverpassCandidate] = []
    var selectedCandidate: OverpassCandidate?
    var reviewCandidate: OverpassCandidate?
    var bounds: MapBounds?
    var initialFocus: MapCoordinate?
    var isLoadingSaved = false
    var isSearching = false
    var errorMessage: String?
    private var viewportTask: Task<Void, Never>?

    var isViewportSearchable: Bool { bounds?.isSearchable == true }

    func updateBounds(_ bounds: MapBounds, api: APIClient) {
        self.bounds = bounds
        viewportTask?.cancel()
        viewportTask = Task {
            try? await Task.sleep(for: .milliseconds(350))
            guard !Task.isCancelled else { return }
            await loadSaved(api: api)
        }
    }

    func loadSaved(api: APIClient) async {
        guard let bounds, !isLoadingSaved,
              let stationResource = ResourceDefinition.all.first(where: { $0.id == "stations" })
        else { return }
        isLoadingSaved = true
        defer { isLoadingSaved = false }
        do {
            savedStations = try await api.list(stationResource, bounds: bounds).items
            if initialFocus == nil,
               let station = savedStations.first,
               let latitude = station.latitude,
               let longitude = station.longitude {
                initialFocus = MapCoordinate(latitude: latitude, longitude: longitude)
            }
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func searchViewport(api: APIClient) async {
        guard let bounds, bounds.isSearchable else {
            errorMessage = "Zoom in until the visible map area is at most 2° wide and high."
            return
        }
        await search(api: api) { try await api.overpassSearch(bounds: bounds) }
    }

    func search(latitude: Double, longitude: Double, api: APIClient) async {
        await search(api: api) { try await api.overpassSearch(latitude: latitude, longitude: longitude) }
    }

    @discardableResult
    func importCandidate(_ candidate: OverpassCandidate, api: APIClient) async -> Bool {
        let selectedCandidateID = selectedCandidate?.id
        do {
            let station = try await api.importCandidate(candidate)
            candidates = candidates.map { item in
                var item = item
                if item.id == candidate.id { item.importedStationId = station.stableID }
                return item
            }
            if selectedCandidateID == candidate.id {
                selectedCandidate = candidates.first { $0.id == candidate.id }
            }
            await loadSaved(api: api)
            return true
        } catch {
            errorMessage = error.localizedDescription
            return false
        }
    }

    private func search(api: APIClient, operation: () async throws -> OverpassCandidatePage) async {
        guard !isSearching else { return }
        isSearching = true
        defer { isSearching = false }
        do {
            candidates = try await operation().items
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
