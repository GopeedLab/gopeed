//
//  FindInteractionController.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 07/10/22.
//

import Foundation
import FlutterMacOS

public class FindInteractionController: NSObject, Disposable {
    
    static var METHOD_CHANNEL_NAME_PREFIX = "com.pichillilorenzo/flutter_inappwebview_find_interaction_";
    var plugin: InAppWebViewFlutterPlugin?
    var webView: InAppWebView?
    var channelDelegate: FindInteractionChannelDelegate?
    var settings: FindInteractionSettings?
    var shouldCallOnRefresh = false
    var searchText: String?
    var activeFindSession: FindSession?
    
    public init(plugin: InAppWebViewFlutterPlugin, id: Any, webView: InAppWebView, settings: FindInteractionSettings?) {
        super.init()
        self.plugin = plugin
        self.webView = webView
        self.settings = settings
        if let registrar = plugin.registrar {
            let channel = FlutterMethodChannel(name: FindInteractionController.METHOD_CHANNEL_NAME_PREFIX + String(describing: id),
                                               binaryMessenger: registrar.messenger)
            self.channelDelegate = FindInteractionChannelDelegate(findInteractionController: self, channel: channel)
        }
    }
    
    public func prepare() {
//        if let settings = settings {
//
//        }
    }
    
    public func findAll(find: String?, completionHandler: ((Any?, Error?) -> Void)?) {
        guard let webView else {
            if let completionHandler = completionHandler {
                completionHandler(nil, nil)
            }
            return
        }
        
        var find = find
        if find == nil {
            find = searchText
        } else {
            // updated searchText
            searchText = find
        }
        
        guard let find else {
            if let completionHandler = completionHandler {
                completionHandler(nil, nil)
            }
            return
        }
        
        if find != "" {
            let startSearch = "window.\(JAVASCRIPT_BRIDGE_NAME)._findAllAsync('\(find)');"
            webView.evaluateJavaScript(startSearch, completionHandler: completionHandler)
        }
    }

    public func findNext(forward: Bool, completionHandler: ((Any?, Error?) -> Void)?) {
        guard let webView else {
            if let completionHandler = completionHandler {
                completionHandler(nil, nil)
            }
            return
        }
        webView.evaluateJavaScript("window.\(JAVASCRIPT_BRIDGE_NAME)._findNext(\(forward ? "true" : "false"));", completionHandler: completionHandler)
    }

    public func clearMatches(completionHandler: ((Any?, Error?) -> Void)?) {
        guard let webView else {
            if let completionHandler = completionHandler {
                completionHandler(nil, nil)
            }
            return
        }
        webView.evaluateJavaScript("window.\(JAVASCRIPT_BRIDGE_NAME)._clearMatches();", completionHandler: completionHandler)
    }
    
    public func dispose() {
        channelDelegate?.dispose()
        channelDelegate = nil
        webView = nil
        activeFindSession = nil
        plugin = nil
    }
    
    deinit {
        debugPrint("FindInteractionControl - dealloc")
        dispose()
    }
}
