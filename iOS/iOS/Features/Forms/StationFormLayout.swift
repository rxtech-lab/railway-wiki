import Foundation

struct FormLayoutStage: Identifiable, Hashable {
    let id: String
    let title: String
    let fields: [String]

    var includesEveryField: Bool { fields.isEmpty }

    static func stages(from schema: [String: JSONValue]) -> [FormLayoutStage] {
        guard case .array(let values) = schema["x-ui-layout"] else { return [] }
        return values.compactMap { value in
            guard case .object(let object) = value,
                  let id = object["id"]?.stringValue,
                  let title = object["title"]?.stringValue,
                  case .array(let fieldValues) = object["fields"]
            else { return nil }
            return FormLayoutStage(
                id: id,
                title: title,
                fields: fieldValues.compactMap(\.stringValue)
            )
        }
    }
}

enum FormSchemaFilter {
    static func schema(_ wire: [String: JSONValue], for stage: FormLayoutStage?) -> [String: JSONValue] {
        guard let stage, !stage.includesEveryField else { return wire }
        let fields = Set(stage.fields)
        var filtered = wire

        if case .object(let properties) = wire["properties"] {
            filtered["properties"] = .object(properties.filter { fields.contains($0.key) })
        }
        if case .array(let required) = wire["required"] {
            filtered["required"] = .array(required.filter { value in
                value.stringValue.map(fields.contains) ?? false
            })
        }
        if case .object(let uiSchema) = wire["x-ui-schema"] {
            var filteredUI = uiSchema.filter { fields.contains($0.key) }
            filteredUI["ui:order"] = .array(stage.fields.map(JSONValue.string))
            filtered["x-ui-schema"] = .object(filteredUI)
        }
        return filtered
    }
}
