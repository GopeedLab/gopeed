//
//  InAppBrowserChannelDelegate.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 05/05/22.
//

import Foundation
import FlutterMacOS

public class InAppBrowserChannelDelegate: ChannelDelegate {
    public override init(channel: FlutterMethodChannel) {
        super.init(channel: channel)
    }
    
    public func onBrowserCreated() {
        let arguments: [String: Any?] = [:]
        channel?.invokeMethod("onBrowserCreated", arguments: arguments)
    }
    
    public func onMenuItemClicked(menuItem: InAppBrowserMenuItem) {
        let arguments: [String: Any?] = [
            "id": menuItem.id
        ]
        channel?.invokeMethod("onMenuItemClicked", arguments: arguments)
    }
    
    public func onMainWindowWillClose() {
        let arguments: [String: Any?] = [:]
        channel?.invokeMethod("onMainWindowWillClose", arguments: arguments)
    }
    
    public func onExit() {
        let arguments: [String: Any?] = [:]
        channel?.invokeMethod("onExit", arguments: arguments)
    }
    
    deinit {
        dispose()
    }
}
