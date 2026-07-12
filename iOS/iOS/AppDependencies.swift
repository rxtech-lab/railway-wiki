import Foundation
import Observation

@MainActor
@Observable
final class AppDependencies {
    let configuration: AppConfiguration
    let auth: AuthSession
    let api: APIClient
    let navigation = NavigationModel()
    let mapStore = MapStore()
    @ObservationIgnored private var routeEditors: [String: RouteEditorStore] = [:]

    init(configuration: AppConfiguration, session: URLSession = .shared) {
        self.configuration = configuration
        auth = AuthSession(configuration: configuration)
        api = APIClient(configuration: configuration, auth: auth, session: session)
    }

    func routeEditor(for record: ResourceRecord) -> RouteEditorStore {
        let routeID = record.stableID ?? "unsaved-route"
        if let existing = routeEditors[routeID] { return existing }
        let store = RouteEditorStore(routeID: routeID, initial: record)
        routeEditors[routeID] = store
        return store
    }

    func discardRouteEditor(id: String) {
        routeEditors[id] = nil
    }
}
