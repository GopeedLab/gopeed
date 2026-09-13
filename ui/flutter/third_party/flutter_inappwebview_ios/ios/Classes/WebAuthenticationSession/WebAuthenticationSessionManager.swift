//
//  WebAuthenticationSessionManager.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 08/05/22.
//

import Flutter
import UIKit
import WebKit
import Foundation
import AVFoundation
import SafariServices

public class WebAuthenticationSessionManager: ChannelDelegate {
    static let METHOD_CHANNEL_NAME = "com.pichillilorenzo/flutter_webauthenticationsession"
    var plugin: SwiftFlutterPlugin?
    var sessions: [String: WebAuthenticationSession?] = [:]
    
    init(plugin: SwiftFlutterPlugin) {
        super.init(channel: FlutterMethodChannel(name: WebAuthenticationSessionManager.METHOD_CHANNEL_NAME, binaryMessenger: plugin.registrar!.messenger()))
        self.plugin = plugin
    }
    
    public override func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
        let arguments = call.arguments as? NSDictionary

        switch call.method {
            case "create":
                let id = arguments!["id"] as! String
                let url = arguments!["url"] as! String
                let callbackURLScheme = arguments!["callbackURLScheme"] as? String
                let initialSettings = arguments!["initialSettings"] as! [String: Any?]
                create(id: id, url: url, callbackURLScheme: callbackURLScheme, settings: initialSettings, result: result)
                break
            case "isAvailable":
                if #available(iOS 11.0, *) {
                    result(true)
                } else {
                    result(false)
                }
                break
            default:
                result(FlutterMethodNotImplemented)
                break
        }
    }
    
    public func create(id: String, url: String, callbackURLScheme: String?, settings: [String: Any?], result: @escaping FlutterResult) {
        if #available(iOS 11.0, *), let plugin = plugin {
            let sessionUrl = URL(string: url) ?? URL(string: "about:blank")!
            let initialSettings = WebAuthenticationSessionSettings()
            let _ = initialSettings.parse(settings: settings)
            let session = WebAuthenticationSession(plugin: plugin, id: id, url: sessionUrl, callbackURLScheme: callbackURLScheme, settings: initialSettings)
            session.prepare()
            sessions[id] = session
            result(true)
            return
        }
        
        result(FlutterError.init(code: "WebAuthenticationSessionManager", message: "WebAuthenticationSession is not available!", details: nil))
    }
    
    public override func dispose() {
        super.dispose()
        let sessionValues = sessions.values
        sessionValues.forEach { (session: WebAuthenticationSession?) in
            session?.cancel()
            session?.dispose()
        }
        sessions.removeAll()
        plugin = nil
    }
    
    deinit {
        dispose()
    }
}
