//
//  ChromeSafariBrowserManager.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 18/12/2019.
//

import Flutter
import UIKit
import WebKit
import Foundation
import AVFoundation
import SafariServices

public class ChromeSafariBrowserManager: ChannelDelegate {
    static let METHOD_CHANNEL_NAME = "com.pichillilorenzo/flutter_chromesafaribrowser"
    var plugin: SwiftFlutterPlugin?
    var browsers: [String: SafariViewController?] = [:]
    var prewarmingTokens: [String: Any?] = [:]
    
    init(plugin: SwiftFlutterPlugin) {
        super.init(channel: FlutterMethodChannel(name: ChromeSafariBrowserManager.METHOD_CHANNEL_NAME, binaryMessenger: plugin.registrar!.messenger()))
        self.plugin = plugin
    }
    
    public override func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
        let arguments = call.arguments as? NSDictionary

        switch call.method {
            case "open":
                let id = arguments!["id"] as! String
                let url = arguments!["url"] as! String
                let settings = arguments!["settings"] as! [String: Any?]
                let menuItemList = arguments!["menuItemList"] as! [[String: Any]]
                open(id: id, url: url, settings: settings,  menuItemList: menuItemList, result: result)
                break
            case "isAvailable":
                if #available(iOS 9.0, *) {
                    result(true)
                } else {
                    result(false)
                }
                break
            case "clearWebsiteData":
                if #available(iOS 16.0, *) {
                    SFSafariViewController.DataStore.default.clearWebsiteData {
                        result(true)
                    }
                } else {
                    result(false)
                }
            case "prewarmConnections":
                if #available(iOS 15.0, *) {
                    let stringURLs = arguments!["URLs"] as! [String]
                    var URLs: [URL] = []
                    for stringURL in stringURLs {
                        if let url = URL(string: stringURL) {
                            URLs.append(url)
                        }
                    }
                    let prewarmingToken = SFSafariViewController.prewarmConnections(to: URLs)
                    let prewarmingTokenId = NSUUID().uuidString
                    prewarmingTokens[prewarmingTokenId] = prewarmingToken
                    result([
                        "id": prewarmingTokenId
                    ])
                } else {
                    result(nil)
                }
            case "invalidatePrewarmingToken":
                if #available(iOS 15.0, *) {
                    let prewarmingToken = arguments!["prewarmingToken"] as! [String:Any?]
                    if let prewarmingTokenId = prewarmingToken["id"] as? String,
                       let prewarmingToken = prewarmingTokens[prewarmingTokenId] as? SFSafariViewController.PrewarmingToken? {
                        prewarmingToken?.invalidate()
                        prewarmingTokens[prewarmingTokenId] = nil
                    }
                    result(true)
                } else {
                    result(false)
                }
            default:
                result(FlutterMethodNotImplemented)
                break
        }
    }
    
    public func open(id: String, url: String, settings: [String: Any?], menuItemList: [[String: Any]], result: @escaping FlutterResult) {
        let absoluteUrl = URL(string: url)!.absoluteURL
        
        if #available(iOS 9.0, *), let plugin = plugin {
            
            if let flutterViewController = UIApplication.shared.visibleViewController {
                // flutterViewController could be casted to FlutterViewController if needed
                
                let safariSettings = SafariBrowserSettings()
                let _ = safariSettings.parse(settings: settings)
                
                let safari: SafariViewController
                
                if #available(iOS 11.0, *) {
                    let config = SFSafariViewController.Configuration()
                    safari = SafariViewController(plugin: plugin, id: id, url: absoluteUrl, configuration: config,
                                                  menuItemList: menuItemList, safariSettings: safariSettings)
                } else {
                    // Fallback on earlier versions
                    safari = SafariViewController(plugin: plugin, id: id, url: absoluteUrl, entersReaderIfAvailable: safariSettings.entersReaderIfAvailable,
                                                  menuItemList: menuItemList, safariSettings: safariSettings)
                }
                
                safari.prepareSafariBrowser()
                
                flutterViewController.present(safari, animated: true) {
                    result(true)
                }
                
                browsers[id] = safari
            }
            return
        }
        
        result(FlutterError.init(code: "ChromeSafariBrowserManager", message: "SafariViewController is not available!", details: nil))
    }
    
    public override func dispose() {
        super.dispose()
        let browserValues = browsers.values
        browserValues.forEach { (browser: SafariViewController?) in
            browser?.close(result: nil)
            browser?.dispose()
        }
        browsers.removeAll()
        if #available(iOS 15.0, *) {
            let prewarmingTokensValues = prewarmingTokens.values
            prewarmingTokensValues.forEach { (prewarmingToken: Any?) in
                if let prewarmingToken = prewarmingToken as? SFSafariViewController.PrewarmingToken? {
                    prewarmingToken?.invalidate()
                }
            }
            prewarmingTokens.removeAll()
        }
        plugin = nil
    }
    
    deinit {
        dispose()
    }
}
