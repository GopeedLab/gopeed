//
//  WebMessageChannel.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 10/03/21.
//

import Foundation

public class WebMessageChannel: FlutterMethodCallDelegate {
    static var METHOD_CHANNEL_NAME_PREFIX = "com.pichillilorenzo/flutter_inappwebview_web_message_channel_"
    var id: String
    var plugin: SwiftFlutterPlugin?
    var channelDelegate: WebMessageChannelChannelDelegate?
    weak var webView: InAppWebView?
    var ports: [WebMessagePort] = []
    
    public init(plugin: SwiftFlutterPlugin, id: String) {
        self.id = id
        self.plugin = plugin
        super.init()
        if let registrar = plugin.registrar {
            let channel = FlutterMethodChannel(name: WebMessageChannel.METHOD_CHANNEL_NAME_PREFIX + id,
                                               binaryMessenger: registrar.messenger())
            self.channelDelegate = WebMessageChannelChannelDelegate(webMessageChannel: self, channel: channel)
        }
        self.ports = [
            WebMessagePort(name: "port1", index: 0, webMessageChannelId: self.id, webMessageChannel: self),
            WebMessagePort(name: "port2", index: 1, webMessageChannelId: self.id, webMessageChannel: self)
        ]
    }
    
    public func initJsInstance(webView: InAppWebView, completionHandler: ((WebMessageChannel?) -> Void)? = nil) {
        self.webView = webView
        if let webView = self.webView {
            webView.evaluateJavascript(source: """
            (function() {
                \(WEB_MESSAGE_CHANNELS_VARIABLE_NAME)["\(id)"] = new MessageChannel();
            })();
            """) { (_) in
                completionHandler?(self)
            }
        } else {
            completionHandler?(nil)
        }
    }
    
    public func toMap() -> [String:Any?] {
        return [
            "id": id
        ]
    }
    
    public func dispose() {
        channelDelegate?.dispose()
        channelDelegate = nil
        for port in ports {
            port.dispose()
        }
        ports.removeAll()
        webView?.evaluateJavascript(source: """
        (function() {
            var webMessageChannel = \(WEB_MESSAGE_CHANNELS_VARIABLE_NAME)["\(id)"];
            if (webMessageChannel != null) {
                webMessageChannel.port1.close();
                webMessageChannel.port2.close();
                delete \(WEB_MESSAGE_CHANNELS_VARIABLE_NAME)["\(id)"];
            }
        })();
        """)
        webView = nil
        plugin = nil
    }
    
    deinit {
        debugPrint("WebMessageChannel - dealloc")
        dispose()
    }
}
