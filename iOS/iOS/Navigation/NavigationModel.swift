import Foundation
import Observation

enum SidebarDestination: Hashable, Sendable {
    case dashboard
    case map
    case resource(String)
}

enum CompactRoute: Hashable, Sendable {
    case destination(SidebarDestination)
    case record(resourceID: String, recordID: String)
}

@MainActor
@Observable
final class NavigationModel {
    var selectedSidebar: SidebarDestination = .dashboard
    var selectedRecord: ResourceRecord?
    var compactPath: [CompactRoute] = []

    func select(_ destination: SidebarDestination) {
        selectedSidebar = destination
        selectedRecord = nil
    }

    func select(_ record: ResourceRecord) {
        selectedRecord = record
    }

    func synchronizeForCompact() {
        var path: [CompactRoute] = [.destination(selectedSidebar)]
        if let recordID = selectedRecord?.stableID,
           case .resource(let resourceID) = selectedSidebar {
            path.append(.record(resourceID: resourceID, recordID: recordID))
        }
        compactPath = path
    }

    func synchronizeForRegular() {
        var compactRecord: (resourceID: String, recordID: String)?
        for route in compactPath {
            switch route {
            case .destination(let destination):
                selectedSidebar = destination
            case .record(let resourceID, let recordID):
                compactRecord = (resourceID, recordID)
            }
        }
        guard let compactRecord else {
            selectedRecord = nil
            return
        }
        selectedSidebar = .resource(compactRecord.resourceID)
        if selectedRecord?.stableID != compactRecord.recordID {
            selectedRecord = nil
        }
    }

    func restoreCompactRecord(_ record: ResourceRecord, resourceID: String) {
        selectedSidebar = .resource(resourceID)
        selectedRecord = record
    }
}
