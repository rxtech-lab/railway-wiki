import JSONSchema
import JSONSchemaForm
import SwiftUI

struct RouteStationInspector: View {
    @Environment(AppDependencies.self) private var dependencies
    let routeID: String
    let draft: RouteStationDraft
    let onChange: (RouteStationDraft) -> Void
    @State private var schema: JSONSchema?
    @State private var schemaJSON: String?
    @State private var uiSchema: [String: Any]?
    @State private var formData: FormData

    init(routeID: String, draft: RouteStationDraft, onChange: @escaping (RouteStationDraft) -> Void) {
        self.routeID = routeID
        self.draft = draft
        self.onChange = onChange
        var values: [String: FormData] = [
            "routeId": .string(routeID),
            "stationId": .string(draft.stationId),
            "sequence": .number(Double(draft.sequence ?? 1))
        ]
        if let distance = draft.distanceFromStartKm { values["distanceFromStartKm"] = .number(distance) }
        if let validFrom = draft.validFrom { values["validFrom"] = .string(validFrom) }
        if let validTo = draft.validTo { values["validTo"] = .string(validTo) }
        _formData = State(initialValue: .object(properties: values))
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                Text("Station Inspector").font(.headline)
                if let schema {
                    JSONSchemaForm(
                        schema: schema,
                        uiSchema: uiSchema,
                        formData: $formData,
                        schemaJSON: schemaJSON,
                        liveValidate: true,
                        showErrorList: true,
                        showSubmitButton: false,
                        widgets: FormWidgetRegistry.widgets(api: dependencies.api),
                        idPrefix: "route_station"
                    )
                } else {
                    ProgressView()
                }
            }
            .padding()
        }
        .scrollDismissesKeyboard(.interactively)
        .task { await loadSchema() }
        .onChange(of: formData) { _, value in updateDraft(value) }
    }

    private func loadSchema() async {
        guard let resource = ResourceDefinition.all.first(where: { $0.id == "route-stations" }) else { return }
        do {
            let wire = try await dependencies.api.schema(resource, action: "update")
            let data = try JSONEncoder().encode(JSONValue.object(wire))
            guard let json = String(data: data, encoding: .utf8),
                  let root = try JSONSerialization.jsonObject(with: data) as? [String: Any]
            else { return }
            schemaJSON = json
            uiSchema = root["x-ui-schema"] as? [String: Any]
            schema = try JSONSchema(jsonString: json)
        } catch {}
    }

    private func updateDraft(_ data: FormData) {
        guard let values = data.object else { return }
        var changed = draft
        changed.stationId = values["stationId"]?.string ?? draft.stationId
        changed.distanceFromStartKm = values["distanceFromStartKm"]?.number
        changed.validFrom = values["validFrom"]?.string
        changed.validTo = values["validTo"]?.string
        onChange(changed)
    }
}
