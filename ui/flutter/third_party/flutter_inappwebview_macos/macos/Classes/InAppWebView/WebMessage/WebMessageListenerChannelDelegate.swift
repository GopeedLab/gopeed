//
//  WebMessageListenerChannelDelegate.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 07/05/22.
//

import Foundation
import FlutterMacOS

public class WebMessageListenerChannelDelegate: ChannelDelegate {
    private weak var webMessageListener: WebMessageListener?
    
    public init(webMessageListener: WebMessageListener, channel: FlutterMethodChannel) {
        super.init(channel: channel)
        self.webMessageListener = webMessageListener
    }
    
    public override func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
        let arguments = call.arguments as? NSDictionary
        
        switch call.method {
        case "postMessage":
            if let webView = webMessageListener?.webView, let jsObjectName = webMessageListener?.jsObjectName {
                let jsObjectNameEscaped = jsObjectName.replacingOccurrences(of: "\'", with: "\\'")
                let message = WebMessage.fromMap(map: arguments!["message"] as! [String: Any?])
                
                let source = """
                (function() {
                    var webMessageListener = window['\(jsObjectNameEscaped)'];
                    if (webMessageListener != null) {
                        var event = {data: \(message.jsData)};
                        if (webMessageListener.onmessage != null) {
                            webMessageListener.onmessage(event);
                        }
                        for (var listener of webMessageListener.listeners) {
                            listener(event);
                        }
                    }
                })();
                """
                webView.evaluateJavascript(source: source) { (_) in
                    result(true)
                }
            } else {
               result(true)
            }
            break
        default:
            result(FlutterMethodNotImplemented)
            break
        }
    }
    
    public func onPostMessage(message: WebMessage?, sourceOrigin: URL?, isMainFrame: Bool) {
        let arguments: [String:Any?] = [
            "message": message?.toMap(),
            "sourceOrigin": sourceOrigin?.absoluteString,
            "isMainFrame": isMainFrame
        ]
        channel?.invokeMethod("onPostMessage", arguments: arguments)
    }
    
    public override func dispose() {
        super.dispose()
        webMessageListener = nil
    }
    
    deinit {
        dispose()
    }
}

