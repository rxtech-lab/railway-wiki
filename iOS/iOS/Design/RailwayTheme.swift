import SwiftUI

enum RailwayTheme {
    static let accent = Color.indigo
    static let mapCandidate = Color.orange
    static let savedStation = Color.indigo
}

private struct GlassCardModifier: ViewModifier {
    func body(content: Content) -> some View {
        content
            .padding(16)
            .glassEffect(.regular, in: .rect(cornerRadius: 20))
    }
}

extension View {
    func railwayGlassCard() -> some View { modifier(GlassCardModifier()) }
}

struct LoadingOverlay: ViewModifier {
    let visible: Bool

    func body(content: Content) -> some View {
        content.overlay {
            if visible {
                ProgressView()
                    .padding(24)
                    .glassEffect(.regular, in: .rect(cornerRadius: 16))
            }
        }
    }
}
