//
//  FindInteractionController.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 07/10/22.
//

import Foundation
import Flutter

public class FindInteractionController: NSObject, Disposable {
    static let METHOD_CHANNEL_NAME_PREFIX = "com.pichillilorenzo/flutter_inappwebview_find_interaction_"

    var webView: InAppWebView?
    var channelDelegate: FindInteractionChannelDelegate?

    private var plugin: SwiftFlutterPlugin?
    private var settings: FindInteractionSettings?
    
    private var _searchText: String? = nil
    var searchText: String? {
        get {
            if #available(iOS 16.0, *), let interaction = webView?.findInteraction {
                return  interaction.searchText
            }
            return _searchText
        }
        set {
            if #available(iOS 16.0, *), let interaction = webView?.findInteraction {
                interaction.searchText = newValue
            }
            self._searchText = newValue
        }
    }
    
    private var _activeFindSession: FindSession? = nil
    var activeFindSession: FindSession? {
        get {
            if #available(iOS 16.0, *), let interaction = webView?.findInteraction {
                if let activeFindSession = interaction.activeFindSession {
                    return FindSession.fromUIFindSession(uiFindSession: activeFindSession)
                }
                return nil
            }
            return _activeFindSession
        }
        set {
            self._activeFindSession = newValue
        }
    }
    
    public init(plugin: SwiftFlutterPlugin, id: Any, webView: InAppWebView, settings: FindInteractionSettings?) {
        super.init()
        self.plugin = plugin
        self.webView = webView
        self.settings = settings
        if let registrar = plugin.registrar {
            let channel = FlutterMethodChannel(name: FindInteractionController.METHOD_CHANNEL_NAME_PREFIX + String(describing: id),
                                               binaryMessenger: registrar.messenger())
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
        
        if #available(iOS 16.0, *), webView.isFindInteractionEnabled {
            if let interaction = webView.findInteraction {
                interaction.searchText = find
                interaction.presentFindNavigator(showingReplace: false)
            }
            if let completionHandler = completionHandler {
                completionHandler(nil, nil)
            }
        } else if find != "" {
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
        if #available(iOS 16.0, *), webView.isFindInteractionEnabled {
            if let interaction = webView.findInteraction {
                if forward {
                    interaction.findNext()
                } else {
                    interaction.findPrevious()
                }
            }
            if let completionHandler = completionHandler {
                completionHandler(nil, nil)
            }
        } else {
            webView.evaluateJavaScript("window.\(JAVASCRIPT_BRIDGE_NAME)._findNext(\(forward ? "true" : "false"));", completionHandler: completionHandler)
        }
    }

    public func clearMatches(completionHandler: ((Any?, Error?) -> Void)?) {
        guard let webView else {
            if let completionHandler = completionHandler {
                completionHandler(nil, nil)
            }
            return
        }
        if #available(iOS 16.0, *), webView.isFindInteractionEnabled {
            if let interaction = webView.findInteraction {
                interaction.searchText = nil
                interaction.dismissFindNavigator()
            }
            if let completionHandler = completionHandler {
                completionHandler(nil, nil)
            }
        } else {
            webView.evaluateJavaScript("window.\(JAVASCRIPT_BRIDGE_NAME)._clearMatches();", completionHandler: completionHandler)
        }
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
