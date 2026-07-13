@preconcurrency import CoreLocation
import Combine
import Foundation

@MainActor
final class StationLocationProvider: NSObject, ObservableObject, CLLocationManagerDelegate {
    @Published private(set) var coordinate: MapCoordinate?
    @Published private(set) var errorMessage: String?

    private let manager = CLLocationManager()

    override init() {
        super.init()
        manager.delegate = self
        manager.desiredAccuracy = kCLLocationAccuracyBest
    }

    func requestCurrentLocation() {
        errorMessage = nil
        switch manager.authorizationStatus {
        case .notDetermined:
            manager.requestWhenInUseAuthorization()
        case .authorizedAlways, .authorizedWhenInUse:
            manager.requestLocation()
        case .denied, .restricted:
            errorMessage = "Location access is disabled. You can still choose a coordinate on the map."
        @unknown default:
            errorMessage = "Location is not available right now."
        }
    }

    func clearError() {
        errorMessage = nil
    }

    func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        if manager.authorizationStatus == .authorizedAlways || manager.authorizationStatus == .authorizedWhenInUse {
            manager.requestLocation()
        }
    }

    func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        guard let location = locations.last else { return }
        coordinate = MapCoordinate(
            latitude: location.coordinate.latitude,
            longitude: location.coordinate.longitude
        )
    }

    func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {
        errorMessage = error.localizedDescription
    }
}
