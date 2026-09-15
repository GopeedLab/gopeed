//
//  InAppBrowserManager.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 18/12/2019.
//

import Flutter
import UIKit
import WebKit
import Foundation
import AVFoundation

public class InAppBrowserManager: ChannelDelegate {
    static let METHOD_CHANNEL_NAME = "com.pichillilorenzo/flutter_inappbrowser"
    static let WEBVIEW_STORYBOARD = "WebView"
    static let WEBVIEW_STORYBOARD_CONTROLLER_ID = "viewController"
    static let NAV_STORYBOARD_CONTROLLER_ID = "navController"
    var plugin: SwiftFlutterPlugin?
    
    private var previousStatusBarStyle = -1
    
    init(plugin: SwiftFlutterPlugin) {
        super.init(channel: FlutterMethodChannel(name: InAppBrowserManager.METHOD_CHANNEL_NAME, binaryMessenger: plugin.registrar!.messenger()))
        self.plugin = plugin
    }
    
    public override func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
        let arguments = call.arguments as? NSDictionary

        switch call.method {
            case "open":
                open(arguments: arguments!)
                result(true)
                break
            case "openWithSystemBrowser":
                let url = arguments!["url"] as! String
                openWithSystemBrowser(url: url, result: result)
                break
            default:
                result(FlutterMethodNotImplemented)
                break
        }
    }
    
    public func prepareInAppBrowserWebViewController(settings: [String: Any?]) -> InAppBrowserWebViewController {
        if previousStatusBarStyle == -1 {
            previousStatusBarStyle = UIApplication.shared.statusBarStyle.rawValue
        }
        
        let browserSettings = InAppBrowserSettings()
        let _ = browserSettings.parse(settings: settings)
        
        let webViewSettings = InAppWebViewSettings()
        let _ = webViewSettings.parse(settings: settings)
        
        let webViewController = InAppBrowserWebViewController()
        webViewController.plugin = plugin
        webViewController.browserSettings = browserSettings
        webViewController.isHidden = browserSettings.hidden
        webViewController.webViewSettings = webViewSettings
        webViewController.previousStatusBarStyle = previousStatusBarStyle
        return webViewController
    }
    
    public func open(arguments: NSDictionary) {
        let id = arguments["id"] as! String
        let urlRequest = arguments["urlRequest"] as? [String:Any?]
        let assetFilePath = arguments["assetFilePath"] as? String
        let data = arguments["data"] as? String
        let mimeType = arguments["mimeType"] as? String
        let encoding = arguments["encoding"] as? String
        let baseUrl = arguments["baseUrl"] as? String
        let settings = arguments["settings"] as! [String: Any?]
        let contextMenu = arguments["contextMenu"] as! [String: Any]
        let windowId = arguments["windowId"] as? Int64
        let initialUserScripts = arguments["initialUserScripts"] as? [[String: Any]]
        let pullToRefreshInitialSettings = arguments["pullToRefreshSettings"] as! [String: Any?]
        let menuItems = arguments["menuItems"] as! [[String: Any?]]
        
        let webViewController = prepareInAppBrowserWebViewController(settings: settings)
        
        webViewController.id = id
        webViewController.initialUrlRequest = urlRequest != nil ? URLRequest.init(fromPluginMap: urlRequest!) : nil
        webViewController.initialFile = assetFilePath
        webViewController.initialData = data
        webViewController.initialMimeType = mimeType
        webViewController.initialEncoding = encoding
        webViewController.initialBaseUrl = baseUrl
        webViewController.contextMenu = contextMenu
        webViewController.windowId = windowId
        webViewController.initialUserScripts = initialUserScripts ?? []
        webViewController.pullToRefreshInitialSettings = pullToRefreshInitialSettings
        for menuItem in menuItems {
            webViewController.menuItems.append(InAppBrowserMenuItem.fromMap(map: menuItem)!)
        }
        
        presentViewController(webViewController: webViewController)
    }
    
    public func presentViewController(webViewController: InAppBrowserWebViewController) {
        let storyboard = UIStoryboard(name: InAppBrowserManager.WEBVIEW_STORYBOARD, bundle: Bundle(for: InAppWebViewFlutterPlugin.self))
        let navController = storyboard.instantiateViewController(withIdentifier: InAppBrowserManager.NAV_STORYBOARD_CONTROLLER_ID) as! InAppBrowserNavigationController
        webViewController.edgesForExtendedLayout = []
        navController.pushViewController(webViewController, animated: false)
        webViewController.prepareNavigationControllerBeforeViewWillAppear()
        
        var animated = true
        if let browserSettings = webViewController.browserSettings, browserSettings.hidden {
            animated = false
        }
        
        guard let visibleViewController = UIApplication.shared.visibleViewController else {
            assertionFailure("Failure init the visibleViewController!")
            return
        }

        if let popover = navController.popoverPresentationController {
            let sourceView = visibleViewController.view ?? UIView()

            popover.sourceRect = CGRect(x: sourceView.bounds.midX, y: sourceView.bounds.midY, width: 0, height: 0)
            popover.permittedArrowDirections = []
            popover.sourceView = sourceView
        }

        visibleViewController.present(navController, animated: animated)
    }
    
    public func openWithSystemBrowser(url: String, result: @escaping FlutterResult) {
        let absoluteUrl = URL(string: url)!.absoluteURL
        if !UIApplication.shared.canOpenURL(absoluteUrl) {
            result(FlutterError(code: "InAppBrowserManager", message: url + " cannot be opened!", details: nil))
            return
        }
        else {
            if #available(iOS 10.0, *) {
                UIApplication.shared.open(absoluteUrl)
            } else {
                UIApplication.shared.openURL(absoluteUrl)
            }
        }
        result(true)
    }
    
    public override func dispose() {
        super.dispose()
        plugin = nil
    }
    
    deinit {
        dispose()
    }
}
