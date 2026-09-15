//
//  WebMessageChannelChannelDelegate.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 07/05/22.
//

import Foundation
import FlutterMacOS

public class WebMessageChannelChannelDelegate: ChannelDelegate {
    private weak var webMessageChannel: WebMessageChannel?
    
    public init(webMessageChannel: WebMessageChannel, channel: FlutterMethodChannel) {
        super.init(channel: channel)
        self.webMessageChannel = webMessageChannel
    }
    
    public override func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
        let arguments = call.arguments as? NSDictionary
        
        switch call.method {
        case "setWebMessageCallback":
            if let _ = webMessageChannel?.webView, let ports = webMessageChannel?.ports, ports.count > 0 {
                let index = arguments!["index"] as! Int
                let port = ports[index]
                do {
                    try port.setWebMessageCallback { (_) in
                        result(true)
                    }
                } catch let error as NSError {
                    result(FlutterError(code: "WebMessageChannel", message: error.domain, details: nil))
                }
                
            } else {
                result(true)
            }
            break
        case "postMessage":
            if let webView = webMessageChannel?.webView, let ports = webMessageChannel?.ports, ports.count > 0 {
                let index = arguments!["index"] as! Int
                let port = ports[index]
                var message = WebMessage.fromMap(map: arguments!["message"] as! [String: Any?])
                
                var ports: [WebMessagePort] = []
                if let notConnectedPorts = message.ports {
                    for notConnectedPort in notConnectedPorts {
                        if let webMessageChannel = webView.webMessageChannels[notConnectedPort.webMessageChannelId] {
                            ports.append(webMessageChannel.ports[Int(notConnectedPort.index)])
                        }
                    }
                }
                message.ports = ports
                
                do {
                    try port.postMessage(message: message) { (_) in
                        result(true)
                    }
                } catch let error as NSError {
                    result(FlutterError(code: "WebMessageChannel", message: error.domain, details: nil))
                }
            } else {
                result(true)
            }
            break
        case "close":
            if let _ = webMessageChannel?.webView, let ports = webMessageChannel?.ports, ports.count > 0 {
                let index = arguments!["index"] as! Int
                let port = ports[index]
                do {
                    try port.close { (_) in
                        result(true)
                    }
                } catch let error as NSError {
                    result(FlutterError(code: "WebMessageChannel", message: error.domain, details: nil))
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
    
    public func onMessage(index: Int64, message: WebMessage?) {
        let arguments: [String:Any?] = [
            "index": index,
            "message": message?.toMap()
        ]
        channel?.invokeMethod("onMessage", arguments: arguments)
    }
    
    public override func dispose() {
        super.dispose()
        webMessageChannel = nil
    }
    
    deinit {
        dispose()
    }
}
