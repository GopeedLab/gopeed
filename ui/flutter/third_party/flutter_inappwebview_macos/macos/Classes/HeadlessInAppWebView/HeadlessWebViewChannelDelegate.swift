//
//  HeadlessWebViewChannelDelegate.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 05/05/22.
//

import Foundation
import FlutterMacOS

public class HeadlessWebViewChannelDelegate: ChannelDelegate {
    private weak var headlessWebView: HeadlessInAppWebView?
    
    public init(headlessWebView: HeadlessInAppWebView, channel: FlutterMethodChannel) {
        super.init(channel: channel)
        self.headlessWebView = headlessWebView
    }
    
    public override func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
        let arguments = call.arguments as? NSDictionary
        
        switch call.method {
        case "dispose":
            if let headlessWebView = headlessWebView {
                headlessWebView.dispose()
                result(true)
            } else {
                result(false)
            }
            break
        case "setSize":
            if let headlessWebView = headlessWebView {
                let sizeMap = arguments!["size"] as? [String: Any?]
                if let size = Size2D.fromMap(map: sizeMap) {
                    headlessWebView.setSize(size: size)
                }
                result(true)
            } else {
                result(false)
            }
            break
        case "getSize":
            result(headlessWebView?.getSize()?.toMap())
            break
        default:
            result(FlutterMethodNotImplemented)
            break
        }
    }
    
    public func onWebViewCreated() {
        let arguments: [String: Any?] = [:]
        channel?.invokeMethod("onWebViewCreated", arguments: arguments)
    }
    
    public override func dispose() {
        super.dispose()
        headlessWebView = nil
    }
    
    deinit {
        dispose()
    }
}
