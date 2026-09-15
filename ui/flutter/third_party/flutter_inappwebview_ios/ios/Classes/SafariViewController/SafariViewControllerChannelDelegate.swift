//
//  SafariViewControllerChannelDelegate.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 05/05/22.
//

import Foundation

public class SafariViewControllerChannelDelegate: ChannelDelegate {
    private weak var safariViewController: SafariViewController?
    
    public init(safariViewController: SafariViewController, channel: FlutterMethodChannel) {
        super.init(channel: channel)
        self.safariViewController = safariViewController
    }
    
    public override func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
        // let arguments = call.arguments as? NSDictionary
        switch call.method {
            case "close":
                if let safariViewController = safariViewController {
                    safariViewController.close(result: result)
                } else {
                    result(false)
                }
                break
            default:
                result(FlutterMethodNotImplemented)
                break
        }
    }
    
    public func onOpened() {
        let arguments: [String: Any?] = [:]
        channel?.invokeMethod("onOpened", arguments: arguments)
    }
    
    public func onCompletedInitialLoad(didLoadSuccessfully: Bool) {
        let arguments: [String: Any?] = [
            "didLoadSuccessfully": didLoadSuccessfully
        ]
        channel?.invokeMethod("onCompletedInitialLoad", arguments: arguments)
    }
    
    public func onInitialLoadDidRedirect(url: URL) {
        let arguments: [String: Any?] = [
            "url": url.absoluteString
        ]
        channel?.invokeMethod("onInitialLoadDidRedirect", arguments: arguments)
    }
    
    public func onWillOpenInBrowser() {
        let arguments: [String: Any?] = [:]
        channel?.invokeMethod("onWillOpenInBrowser", arguments: arguments)
    }
    
    public func onClosed() {
        let arguments: [String: Any?] = [:]
        channel?.invokeMethod("onClosed", arguments: arguments)
    }
    
    public func onItemActionPerform(id: Int64, url: URL, title: String?) {
        let arguments: [String: Any?] = [
            "id": id,
            "url": url.absoluteString,
            "title": title,
        ]
        channel?.invokeMethod("onItemActionPerform", arguments: arguments)
    }
    
    public override func dispose() {
        super.dispose()
        safariViewController = nil
    }
    
    deinit {
        dispose()
    }
}
