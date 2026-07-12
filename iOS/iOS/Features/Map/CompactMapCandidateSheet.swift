import SwiftUI

enum MapCandidatePanelDetent {
    case collapsed
    case medium
    case large
}

private struct CandidatePanelHeights {
    let collapsed: CGFloat
    let medium: CGFloat
    let large: CGFloat
}

struct CompactMapCandidateSheet<Content: View>: View {
    @Binding var detent: MapCandidatePanelDetent
    @GestureState private var dragTranslation: CGFloat = 0
    let content: Content

    init(detent: Binding<MapCandidatePanelDetent>, @ViewBuilder content: () -> Content) {
        _detent = detent
        self.content = content()
    }

    var body: some View {
        GeometryReader { proxy in
            let heights = panelHeights(for: proxy.size.height)
            let draggedHeight = height(for: detent, in: heights) - dragTranslation

            VStack(spacing: 0) {
                Spacer(minLength: 0)
                    .allowsHitTesting(false)
                VStack(spacing: 0) {
                    Capsule()
                        .fill(.secondary)
                        .frame(width: 36, height: 5)
                        .frame(maxWidth: .infinity)
                        .frame(height: 24)
                        .contentShape(.rect)
                        .gesture(dragGesture(heights: heights))
                        .accessibilityLabel("Resize railway candidates")
                        .accessibilityIdentifier("map.candidates.resize")
                        .accessibilityAdjustableAction { direction in
                            adjustDetent(direction)
                        }
                    content
                }
                .frame(maxWidth: .infinity)
                .frame(
                    height: min(max(draggedHeight, heights.collapsed), heights.large),
                    alignment: .top
                )
                .background(
                    .thinMaterial,
                    in: UnevenRoundedRectangle(topLeadingRadius: 28, topTrailingRadius: 28)
                )
                .overlay {
                    UnevenRoundedRectangle(topLeadingRadius: 28, topTrailingRadius: 28)
                        .stroke(.separator.opacity(0.35), lineWidth: 0.5)
                }
            }
            .animation(.snappy, value: detent)
        }
        .ignoresSafeArea(edges: .bottom)
    }

    private func panelHeights(for availableHeight: CGFloat) -> CandidatePanelHeights {
        let large = max(140, availableHeight)
        return CandidatePanelHeights(
            collapsed: 140,
            medium: max(140, min(large * 0.52, 460)),
            large: large
        )
    }

    private func height(for detent: MapCandidatePanelDetent, in heights: CandidatePanelHeights) -> CGFloat {
        switch detent {
        case .collapsed: heights.collapsed
        case .medium: heights.medium
        case .large: heights.large
        }
    }

    private func dragGesture(heights: CandidatePanelHeights) -> some Gesture {
        DragGesture(minimumDistance: 4)
            .updating($dragTranslation) { value, state, _ in
                state = value.translation.height
            }
            .onEnded { value in
                let proposedHeight = height(for: detent, in: heights) - value.predictedEndTranslation.height
                detent = nearestDetent(to: proposedHeight, heights: heights)
            }
    }

    private func nearestDetent(to height: CGFloat, heights: CandidatePanelHeights) -> MapCandidatePanelDetent {
        let candidates: [(MapCandidatePanelDetent, CGFloat)] = [
            (.collapsed, heights.collapsed),
            (.medium, heights.medium),
            (.large, heights.large)
        ]
        return candidates.min { abs($0.1 - height) < abs($1.1 - height) }?.0 ?? .medium
    }

    private func adjustDetent(_ direction: AccessibilityAdjustmentDirection) {
        switch (detent, direction) {
        case (.collapsed, .increment), (.large, .decrement): detent = .medium
        case (.medium, .increment): detent = .large
        case (.medium, .decrement): detent = .collapsed
        default: break
        }
    }
}
