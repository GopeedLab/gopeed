//
//  WebAuthenticationSessionChannelDelegate.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 08/05/22.
//

import Foundation
import FlutterMacOS

public class WebAuthenticationSessionChannelDelegate: ChannelDelegate {
    private weak var webAuthenticationSession: WebAuthenticationSession?
    
    public init(webAuthenticationSession: WebAuthenticationSession, channel: FlutterMethodChannel) {
        super.init(channel: channel)
        self.webAuthenticationSession = webAuthenticationSession
    }
    
    public override func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
        // let arguments = call.arguments as? NSDictionary
        switch call.method {
            case "canStart":
                if let webAuthenticationSession = webAuthenticationSession {
                    result(webAuthenticationSession.canStart())
                } else {
                    result(false)
                }
                break
            case "start":
                if let webAuthenticationSession = webAuthenticationSession {
                    result(webAuthenticationSession.start())
                } else {
                    result(false)
                }
                break
            case "cancel":
                if let webAuthenticationSession = webAuthenticationSession {
                    webAuthenticationSession.cancel()
                    result(true)
                } else {
                    result(false)
                }
                break
            case "dispose":
                if let webAuthenticationSession = webAuthenticationSession {
                    webAuthenticationSession.dispose()
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
    
    public func onComplete(url: URL?, errorCode: Int?) {
        let arguments: [String: Any?] = [
            "url": url?.absoluteString,
            "errorCode": errorCode
        ]
        DispatchQueue.main.async { [weak self] in
            self?.channel?.invokeMethod("onComplete", arguments: arguments)
        }
    }
    
    public override func dispose() {
        super.dispose()
        webAuthenticationSession = nil
    }
    
    deinit {
        dispose()
    }
}
