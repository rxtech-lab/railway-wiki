//
//  ContentView.swift
//  iOS
//
//  Created by Qiwei Li on 7/11/26.
//

import SwiftUI

struct ContentView: View {
    let bootstrap: AppBootstrap

    var body: some View {
        switch bootstrap {
        case .ready(let dependencies):
            AuthGateView()
                .environment(dependencies)
        case .failed(let message):
            ContentUnavailableView {
                Label("Configuration Required", systemImage: "gear.badge.xmark")
            } description: {
                Text(message)
            }
            .accessibilityIdentifier("configuration.error")
        }
    }
}
