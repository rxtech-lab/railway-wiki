import JSONSchemaForm
import SwiftUI

struct StationLocationTools: View {
    let configuration: AppConfiguration
    @Binding var formData: FormData
    @StateObject private var locationProvider = StationLocationProvider()

    private var coordinate: MapCoordinate? {
        guard case .object(let properties) = formData,
              let latitude = properties["latitude"]?.number,
              let longitude = properties["longitude"]?.number
        else { return nil }
        return MapCoordinate(latitude: latitude, longitude: longitude)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                VStack(alignment: .leading, spacing: 3) {
                    Text("Choose Coordinates").font(.headline)
                    Text("Tap the map or use your current location.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button("Current Location", systemImage: "location") {
                    locationProvider.requestCurrentLocation()
                }
                .buttonStyle(.bordered)
                .accessibilityIdentifier("form.station.currentLocation")
            }

            MapLibreView(
                configuration: configuration,
                savedStations: previewStation.map { [$0] } ?? [],
                candidates: [],
                focus: coordinate,
                initialBounds: nil,
                onBoundsChanged: { _ in },
                onMapTap: updateCoordinate,
                onCandidateTap: { _, _ in },
                onSavedStationTap: { _ in }
            )
            .frame(minHeight: 260)
            .clipShape(.rect(cornerRadius: 18))
            .accessibilityIdentifier("form.station.coordinateMap")

            if let coordinate {
                Text(coordinateText(coordinate))
                    .font(.caption.monospacedDigit())
                    .foregroundStyle(.secondary)
            }
            if let errorMessage = locationProvider.errorMessage {
                Label(errorMessage, systemImage: "location.slash")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .railwayGlassCard()
        .onChange(of: locationProvider.coordinate) { _, coordinate in
            guard let coordinate else { return }
            updateCoordinate(coordinate.latitude, coordinate.longitude)
        }
    }

    private var previewStation: ResourceRecord? {
        guard let coordinate else { return nil }
        return ResourceRecord(values: [
            "name": .string("Selected coordinate"),
            "latitude": .number(coordinate.latitude),
            "longitude": .number(coordinate.longitude)
        ])
    }

    private func updateCoordinate(_ latitude: Double, _ longitude: Double) {
        guard case .object(var properties) = formData else { return }
        properties["latitude"] = .number(latitude)
        properties["longitude"] = .number(longitude)
        formData = .object(properties: properties)
    }

    private func coordinateText(_ coordinate: MapCoordinate) -> String {
        let precision = FloatingPointFormatStyle<Double>.number.precision(.fractionLength(6))
        return "\(coordinate.latitude.formatted(precision)), \(coordinate.longitude.formatted(precision))"
    }
}
