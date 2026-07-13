import Foundation
import RxAuthSwift
import RxAuthSwiftUI
import SwiftUI

struct AuthGateView: View {
    @Environment(AppDependencies.self) private var dependencies

    var body: some View {
        if dependencies.auth.isTestAuthenticated {
            AdminAuthorizationView()
        } else {
            switch dependencies.auth.manager.authState {
            case .unknown:
                ProgressView("Restoring session…")
                    .task { await dependencies.auth.restore() }
            case .unauthenticated:
                RxSignInView(
                    manager: dependencies.auth.manager,
                    style: .web
                )
                .accessibilityIdentifier("auth.signIn")
            case .authenticated:
                AdminAuthorizationView()
            }
        }
    }
}

private struct AdminAuthorizationView: View {
    @Environment(AppDependencies.self) private var dependencies
    @State private var phase: Phase = .checking

    var body: some View {
        Group {
            if dependencies.auth.hasLostAdminAccess {
                accessDenied
            } else {
                switch phase {
                case .checking:
                    ProgressView("Verifying administrator access…")
                        .task { await verify() }
                case .allowed:
                    AppShellView()
                case .denied:
                    accessDenied
                case .failed(let message):
                    ContentUnavailableView {
                        Label("Could Not Verify Access", systemImage: "wifi.exclamationmark")
                    } description: {
                        Text(message)
                    } actions: {
                        Button("Retry") { Task { await verify() } }
                        Button("Sign Out") { Task { await dependencies.auth.signOut() } }
                    }
                }
            }
        }
    }

    private var accessDenied: some View {
        ContentUnavailableView {
            Label("Administrator Access Required", systemImage: "person.badge.shield.checkmark")
        } description: {
            Text("This account is signed in but does not have the admin role.")
        } actions: {
            Button("Sign Out") { Task { await dependencies.auth.signOut() } }
        }
        .accessibilityIdentifier("auth.accessDenied")
    }

    private func verify() async {
        phase = .checking
        do {
            _ = try await dependencies.api.dashboard()
            dependencies.auth.restoreAdminAccess()
            phase = .allowed
        } catch APIError.forbidden {
            phase = .denied
        } catch {
            phase = .failed(error.localizedDescription)
        }
    }

    private enum Phase {
        case checking
        case allowed
        case denied
        case failed(String)
    }
}
