import SwiftUI

struct StationPickerSheet: View {
    @Environment(AppDependencies.self) private var dependencies
    @Environment(\.dismiss) private var dismiss
    let onSelect: (ResourceRecord) -> Void
    @State private var stations: [ResourceRecord] = []
    @State private var query = ""
    @State private var isLoading = false

    var body: some View {
        NavigationStack {
            List(filteredStations) { station in
                Button {
                    onSelect(station)
                    dismiss()
                } label: {
                    Label(station.title, systemImage: "tram.fill")
                }
            }
            .scrollDismissesKeyboard(.interactively)
            .overlay { if isLoading { ProgressView() } }
            .searchable(text: $query, prompt: "Search stations")
            .navigationTitle("Add Station")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Close", systemImage: "xmark") { dismiss() }
                }
            }
        }
        .interactiveDismissDisabled(true)
        .presentationDragIndicator(.hidden)
        .task { await load() }
    }

    private var filteredStations: [ResourceRecord] {
        guard !query.isEmpty else { return stations }
        return stations.filter { $0.title.localizedStandardContains(query) }
    }

    private func load() async {
        guard let resource = ResourceDefinition.all.first(where: { $0.id == "stations" }) else { return }
        isLoading = true
        defer { isLoading = false }
        stations = (try? await dependencies.api.list(resource, query: query).items) ?? []
    }
}
