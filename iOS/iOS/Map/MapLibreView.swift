import CoreLocation
import MapLibre
import SwiftUI
import UIKit

struct MapCoordinate: Equatable, Sendable {
    let latitude: Double
    let longitude: Double
}

struct MapLibreView: UIViewRepresentable {
    let configuration: AppConfiguration
    let savedStations: [ResourceRecord]
    let candidates: [OverpassCandidate]
    var focus: MapCoordinate?
    var focusRequestID = 0
    var initialBounds: MapBounds?
    var cameraPadding: UIEdgeInsets = .zero
    var showsUserLocation = false
    let onBoundsChanged: (MapBounds) -> Void
    var onCameraInteraction: (() -> Void)?
    let onMapTap: ((Double, Double) -> Void)?
    let onCandidateTap: (OverpassCandidate, CGPoint) -> Void
    let onSavedStationTap: (String) -> Void

    func makeCoordinator() -> Coordinator { Coordinator(parent: self) }

    func makeUIView(context: Context) -> MLNMapView {
        MapNetworkConfigurator.configure(configuration: configuration)
        let mapView = MLNMapView(frame: .zero, styleURL: configuration.mapStyleURL)
        mapView.delegate = context.coordinator
        mapView.automaticallyAdjustsContentInset = false
        mapView.contentInset = cameraPadding
        mapView.showsUserLocation = showsUserLocation
        mapView.logoView.isHidden = false
        mapView.attributionButton.isHidden = false
        if let initialBounds {
            mapView.setVisibleCoordinateBounds(
                .init(
                    sw: .init(latitude: initialBounds.south, longitude: initialBounds.west),
                    ne: .init(latitude: initialBounds.north, longitude: initialBounds.east)
                ),
                animated: false
            )
        } else if let focus {
            mapView.setCenter(.init(latitude: focus.latitude, longitude: focus.longitude), zoomLevel: 13, animated: false)
        } else {
            mapView.setCenter(.init(latitude: 20, longitude: 0), zoomLevel: 1.5, animated: false)
        }
        let tap = UITapGestureRecognizer(target: context.coordinator, action: #selector(Coordinator.handleMapTap(_:)))
        tap.cancelsTouchesInView = false
        tap.delegate = context.coordinator
        mapView.addGestureRecognizer(tap)
        context.coordinator.mapView = mapView
        return mapView
    }

    func updateUIView(_ mapView: MLNMapView, context: Context) {
        context.coordinator.parent = self
        if mapView.contentInset != cameraPadding {
            mapView.contentInset = cameraPadding
        }
        context.coordinator.updateFocus(focus, requestID: focusRequestID, on: mapView)
        context.coordinator.updateSavedStations(savedStations, on: mapView)
        context.coordinator.updateCandidates(candidates, on: mapView)
    }

    final class Coordinator: NSObject, MLNMapViewDelegate, UIGestureRecognizerDelegate {
        var parent: MapLibreView
        weak var mapView: MLNMapView?
        private var styleLoaded = false
        private var candidateAnnotations: [CandidateAnnotation] = []
        private var currentFocus: MapCoordinate?
        private var currentFocusRequestID = 0
        private var hasFittedSavedStations = false

        init(parent: MapLibreView) { self.parent = parent }

        func mapView(_ mapView: MLNMapView, didFinishLoading style: MLNStyle) {
            styleLoaded = true
            updateSavedStations(parent.savedStations, on: mapView)
        }

        func mapView(_ mapView: MLNMapView, regionDidChangeAnimated animated: Bool) {
            let bounds = mapView.visibleCoordinateBounds
            parent.onBoundsChanged(.init(
                south: bounds.sw.latitude,
                west: bounds.sw.longitude,
                north: bounds.ne.latitude,
                east: bounds.ne.longitude
            ))
        }

        func mapView(_ mapView: MLNMapView, regionWillChangeAnimated animated: Bool) {
            let isUserInteraction = mapView.gestureRecognizers?.contains {
                $0.state == .began || $0.state == .changed
            } == true
            if isUserInteraction {
                parent.onCameraInteraction?()
            }
        }

        func mapView(_ mapView: MLNMapView, didSelect annotation: MLNAnnotation) {
            guard let annotation = annotation as? CandidateAnnotation,
                  let candidate = parent.candidates.first(where: { $0.id == annotation.candidateID })
            else { return }
            let anchor = mapView.convert(annotation.coordinate, toPointTo: mapView)
            parent.onCandidateTap(candidate, anchor)
        }

        func mapView(_ mapView: MLNMapView, viewFor annotation: MLNAnnotation) -> MLNAnnotationView? {
            guard let annotation = annotation as? CandidateAnnotation else { return nil }
            let identifier = "overpass-candidate"
            let view = mapView.dequeueReusableAnnotationView(withIdentifier: identifier) as? CandidateAnnotationView
                ?? CandidateAnnotationView(reuseIdentifier: identifier)
            view.candidateID = annotation.candidateID
            view.isUserInteractionEnabled = true
            if !view.hasCandidateTapRecognizer {
                let tap = UITapGestureRecognizer(
                    target: self,
                    action: #selector(handleCandidateAnnotationTap(_:))
                )
                tap.cancelsTouchesInView = false
                tap.delegate = self
                view.addGestureRecognizer(tap)
                view.hasCandidateTapRecognizer = true
            }
            view.isAccessibilityElement = true
            view.accessibilityLabel = annotation.title
            view.accessibilityIdentifier = "map.annotation.\(annotation.candidateID)"
            return view
        }

        @objc func handleCandidateAnnotationTap(_ recognizer: UITapGestureRecognizer) {
            guard recognizer.state == .ended,
                  let view = recognizer.view as? CandidateAnnotationView,
                  let mapView,
                  let annotation = candidateAnnotations.first(where: { $0.candidateID == view.candidateID }),
                  let candidate = parent.candidates.first(where: { $0.id == view.candidateID })
            else { return }
            let anchor = mapView.convert(annotation.coordinate, toPointTo: mapView)
            parent.onCandidateTap(candidate, anchor)
        }

        func gestureRecognizer(
            _ gestureRecognizer: UIGestureRecognizer,
            shouldRecognizeSimultaneouslyWith otherGestureRecognizer: UIGestureRecognizer
        ) -> Bool {
            true
        }

        @objc func handleMapTap(_ recognizer: UITapGestureRecognizer) {
            guard recognizer.state == .ended, let mapView else { return }
            let point = recognizer.location(in: mapView)
            let candidateHitArea = CGRect(x: point.x - 22, y: point.y - 22, width: 44, height: 44)
            if let annotation = mapView.visibleAnnotations(in: candidateHitArea)?
                .compactMap({ $0 as? CandidateAnnotation })
                .min(by: {
                    mapView.convert($0.coordinate, toPointTo: mapView).distance(to: point) <
                        mapView.convert($1.coordinate, toPointTo: mapView).distance(to: point)
                }),
               let candidate = parent.candidates.first(where: { $0.id == annotation.candidateID }) {
                let anchor = mapView.convert(annotation.coordinate, toPointTo: mapView)
                parent.onCandidateTap(candidate, anchor)
                return
            }
            let features = mapView.visibleFeatures(
                at: point,
                styleLayerIdentifiers: ["saved-station-points", "saved-station-clusters"]
            )
            if let feature = features.first as? MLNPointFeature {
                if let stationID = feature.attributes["stationID"] as? String, !stationID.isEmpty {
                    parent.onSavedStationTap(stationID)
                    return
                }
                if feature.attributes["cluster"] as? Bool == true {
                    mapView.setCenter(feature.coordinate, zoomLevel: mapView.zoomLevel + 2, animated: true)
                    return
                }
            }
            if let onMapTap = parent.onMapTap {
                let coordinate = mapView.convert(point, toCoordinateFrom: mapView)
                onMapTap(coordinate.latitude, coordinate.longitude)
            }
        }

        func updateCandidates(_ candidates: [OverpassCandidate], on mapView: MLNMapView) {
            if !candidateAnnotations.isEmpty { mapView.removeAnnotations(candidateAnnotations) }
            candidateAnnotations = candidates.map { candidate in
                let annotation = CandidateAnnotation(candidateID: candidate.id)
                annotation.coordinate = .init(latitude: candidate.latitude, longitude: candidate.longitude)
                annotation.title = candidate.name
                return annotation
            }
            mapView.addAnnotations(candidateAnnotations)
        }

        func updateFocus(_ focus: MapCoordinate?, requestID: Int, on mapView: MLNMapView) {
            guard let focus,
                  focus != currentFocus || requestID != currentFocusRequestID
            else { return }
            currentFocus = focus
            currentFocusRequestID = requestID
            mapView.setCenter(.init(latitude: focus.latitude, longitude: focus.longitude), zoomLevel: 13, animated: false)
        }

        func updateSavedStations(_ records: [ResourceRecord], on mapView: MLNMapView) {
            guard styleLoaded, let style = mapView.style else { return }
            for identifier in ["saved-station-points", "saved-station-clusters"] {
                if let layer = style.layer(withIdentifier: identifier) { style.removeLayer(layer) }
            }
            if let source = style.source(withIdentifier: "saved-stations") { style.removeSource(source) }

            let features: [MLNPointFeature] = records.compactMap { record in
                guard let latitude = record.latitude, let longitude = record.longitude else { return nil }
                let feature = MLNPointFeature()
                feature.coordinate = .init(latitude: latitude, longitude: longitude)
                feature.attributes = ["stationID": record.stableID ?? "", "name": record.title]
                return feature
            }
            let source = MLNShapeSource(
                identifier: "saved-stations",
                features: features,
                options: [.clustered: true, .clusterRadius: 50]
            )
            style.addSource(source)

            let points = MLNCircleStyleLayer(identifier: "saved-station-points", source: source)
            points.predicate = NSPredicate(format: "cluster != YES")
            points.circleColor = NSExpression(forConstantValue: UIColor.systemIndigo)
            points.circleRadius = NSExpression(forConstantValue: 6)
            points.circleStrokeColor = NSExpression(forConstantValue: UIColor.white)
            points.circleStrokeWidth = NSExpression(forConstantValue: 2)
            style.addLayer(points)

            let clusters = MLNCircleStyleLayer(identifier: "saved-station-clusters", source: source)
            clusters.predicate = NSPredicate(format: "cluster == YES")
            clusters.circleColor = NSExpression(forConstantValue: UIColor.systemIndigo.withAlphaComponent(0.82))
            clusters.circleRadius = NSExpression(forConstantValue: 16)
            clusters.circleStrokeColor = NSExpression(forConstantValue: UIColor.white)
            clusters.circleStrokeWidth = NSExpression(forConstantValue: 2)
            style.addLayer(clusters)
            fitSavedStationsIfNeeded(records, on: mapView)
        }

        private func fitSavedStationsIfNeeded(_ records: [ResourceRecord], on mapView: MLNMapView) {
            guard !hasFittedSavedStations, currentFocus == nil else { return }
            let coordinates = records.compactMap { record -> CLLocationCoordinate2D? in
                guard let latitude = record.latitude, let longitude = record.longitude else { return nil }
                return .init(latitude: latitude, longitude: longitude)
            }
            guard let first = coordinates.first else { return }
            hasFittedSavedStations = true
            if coordinates.count == 1 {
                mapView.setCenter(first, zoomLevel: 13, animated: false)
                return
            }
            let bounds = coordinates.dropFirst().reduce(
                MLNCoordinateBounds(sw: first, ne: first)
            ) { bounds, coordinate in
                MLNCoordinateBounds(
                    sw: .init(
                        latitude: min(bounds.sw.latitude, coordinate.latitude),
                        longitude: min(bounds.sw.longitude, coordinate.longitude)
                    ),
                    ne: .init(
                        latitude: max(bounds.ne.latitude, coordinate.latitude),
                        longitude: max(bounds.ne.longitude, coordinate.longitude)
                    )
                )
            }
            mapView.setVisibleCoordinateBounds(
                bounds,
                edgePadding: UIEdgeInsets(top: 80, left: 60, bottom: 80, right: 60),
                animated: false,
                completionHandler: nil
            )
        }
    }
}

private extension CGPoint {
    func distance(to other: CGPoint) -> CGFloat {
        hypot(x - other.x, y - other.y)
    }
}

private final class CandidateAnnotation: MLNPointAnnotation {
    let candidateID: String
    init(candidateID: String) {
        self.candidateID = candidateID
        super.init()
    }
    required init?(coder: NSCoder) { nil }
}

private final class CandidateAnnotationView: MLNAnnotationView {
    var candidateID = ""
    var hasCandidateTapRecognizer = false

    override init(reuseIdentifier: String?) {
        super.init(reuseIdentifier: reuseIdentifier)
        bounds = CGRect(x: 0, y: 0, width: 44, height: 44)

        let marker = UIView(frame: CGRect(x: 13, y: 13, width: 18, height: 18))
        marker.isUserInteractionEnabled = false
        marker.layer.cornerRadius = 9
        marker.layer.borderWidth = 3
        marker.layer.borderColor = UIColor.white.cgColor
        marker.backgroundColor = .systemOrange
        addSubview(marker)
    }

    required init?(coder: NSCoder) { nil }
}
