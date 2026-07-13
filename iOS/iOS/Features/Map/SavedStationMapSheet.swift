import SwiftUI

struct SavedStationMapSheet: View {
    @Environment(\.dismiss) private var dismiss
    let station: ResourceRecord

    var body: some View {
        NavigationStack {
            Group {
                if let resource = ResourceDefinition.all.first(where: { $0.id == "stations" }) {
                    StationDetailView(
                        resource: resource,
                        record: station,
                        onDeleted: { dismiss() }
                    )
                } else {
                    ContentUnavailableView("Station Unavailable", systemImage: "tram.fill")
                }
            }
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Close") { dismiss() }
                }
            }
        }
        .presentationDetents([.large])
        .presentationDragIndicator(.visible)
        .accessibilityIdentifier("map.station.sheet")
    }
}
