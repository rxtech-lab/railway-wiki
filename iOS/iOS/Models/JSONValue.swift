import Foundation
import JSONSchemaForm

enum JSONValue: Codable, Hashable, Sendable {
    case object([String: JSONValue])
    case array([JSONValue])
    case string(String)
    case number(Double)
    case bool(Bool)
    case null

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if container.decodeNil() {
            self = .null
        } else if let value = try? container.decode(Bool.self) {
            self = .bool(value)
        } else if let value = try? container.decode(Double.self) {
            self = .number(value)
        } else if let value = try? container.decode(String.self) {
            self = .string(value)
        } else if let value = try? container.decode([JSONValue].self) {
            self = .array(value)
        } else if let value = try? container.decode([String: JSONValue].self) {
            self = .object(value)
        } else {
            throw DecodingError.dataCorruptedError(in: container, debugDescription: "Unsupported JSON value")
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch self {
        case .object(let value): try container.encode(value)
        case .array(let value): try container.encode(value)
        case .string(let value): try container.encode(value)
        case .number(let value): try container.encode(value)
        case .bool(let value): try container.encode(value)
        case .null: try container.encodeNil()
        }
    }

    nonisolated init(formData: FormData) {
        switch formData {
        case .object(let properties): self = .object(properties.mapValues(JSONValue.init))
        case .array(let items): self = .array(items.map(JSONValue.init))
        case .string(let value): self = .string(value)
        case .number(let value): self = .number(value)
        case .boolean(let value): self = .bool(value)
        case .null: self = .null
        }
    }

    nonisolated var formData: FormData {
        switch self {
        case .object(let properties): .object(properties: properties.mapValues(\.formData))
        case .array(let items): .array(items: items.map(\.formData))
        case .string(let value): .string(value)
        case .number(let value): .number(value)
        case .bool(let value): .boolean(value)
        case .null: .null
        }
    }

    var stringValue: String? {
        if case .string(let value) = self { return value }
        return nil
    }

    var numberValue: Double? {
        if case .number(let value) = self { return value }
        return nil
    }

    var displayValue: String {
        switch self {
        case .string(let value): value
        case .number(let value): value.formatted()
        case .bool(let value): value ? "Yes" : "No"
        case .array(let values): "\(values.count) values"
        case .object: "JSON object"
        case .null: "—"
        }
    }
}

struct ResourceRecord: Codable, Hashable, Identifiable, Sendable {
    var values: [String: JSONValue]

    init(values: [String: JSONValue]) { self.values = values }

    init(from decoder: Decoder) throws {
        values = try decoder.singleValueContainer().decode([String: JSONValue].self)
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        try container.encode(values)
    }

    var id: String { values["id"]?.stringValue ?? UUID().uuidString }
    var stableID: String? { values["id"]?.stringValue }

    var title: String {
        for key in ["name", "trainNumber", "code", "title", "stationNumber", "id"] {
            if let value = values[key]?.stringValue, !value.isEmpty { return value }
        }
        return "Untitled"
    }

    var subtitle: String? {
        for key in ["nameEn", "shortName", "description", "routeType", "mediaType"] {
            if let value = values[key]?.stringValue, !value.isEmpty { return value }
        }
        return nil
    }

    var latitude: Double? { values["latitude"]?.numberValue }
    var longitude: Double? { values["longitude"]?.numberValue }

    var editableValues: [String: JSONValue] {
        values.filter { key, _ in
            !["id", "createdAt", "updatedAt", "osmElementType", "osmElementId", "osmTags"].contains(key)
        }
    }
}
