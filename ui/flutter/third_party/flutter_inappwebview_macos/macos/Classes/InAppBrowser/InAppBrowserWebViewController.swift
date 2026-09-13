//
//  InAppBrowserWebViewController.swift
//  flutter_inappwebview
//
//  Created by Lorenzo on 17/09/18.
//

import FlutterMacOS
import AppKit
import WebKit
import Foundation

public class InAppBrowserWebViewController: NSViewController, InAppBrowserDelegate, Disposable {
    static var METHOD_CHANNEL_NAME_PREFIX = "com.pichillilorenzo/flutter_inappbrowser_";

    var progressBar: NSProgressIndicator!
    
    var window: InAppBrowserWindow?
    var id: String = ""
    var plugin: InAppWebViewFlutterPlugin?
    var windowId: Int64?
    var webView: InAppWebView?
    var channelDelegate: InAppBrowserChannelDelegate?
    var initialUrlRequest: URLRequest?
    var initialFile: String?
    var browserSettings: InAppBrowserSettings?
    var webViewSettings: InAppWebViewSettings?
    var initialData: String?
    var initialMimeType: String?
    var initialEncoding: String?
    var initialBaseUrl: String?
    var initialUserScripts: [[String: Any]] = []
    var isHidden = false

    public override func loadView() {
        guard let plugin = plugin, let registrar = plugin.registrar else {
            return
        }
        
        let channel = FlutterMethodChannel(name: InAppBrowserWebViewController.METHOD_CHANNEL_NAME_PREFIX + id, binaryMessenger: registrar.messenger)
        channelDelegate = InAppBrowserChannelDelegate(channel: channel)
        
        var userScripts: [UserScript] = []
        for initialUserScript in initialUserScripts {
            userScripts.append(UserScript.fromMap(map: initialUserScript, windowId: windowId)!)
        }
        
        let preWebviewConfiguration = InAppWebView.preWKWebViewConfiguration(settings: webViewSettings)
        if let wId = windowId, let webViewTransport = plugin.inAppWebViewManager?.windowWebViews[wId] {
            webView = webViewTransport.webView
            webView!.initialUserScripts = userScripts
        } else {
            webView = InAppWebView(id: nil,
                                   plugin: nil,
                                   frame: .zero,
                                   configuration: preWebviewConfiguration,
                                   userScripts: userScripts)
        }
        
        guard let webView = webView else {
            return
        }
        
        webView.inAppBrowserDelegate = self
        webView.id = id
        webView.plugin = plugin
        webView.channelDelegate = WebViewChannelDelegate(webView: webView, channel: channel)
        
        let findInteractionController = FindInteractionController(
            plugin: plugin,
            id: id, webView: webView, settings: nil)
        webView.findInteractionController = findInteractionController
        findInteractionController.prepare()
        
        prepareWebView()
        webView.windowCreated = true
        
        progressBar = NSProgressIndicator()
        progressBar.style = .bar
        progressBar.isIndeterminate = false
        progressBar.startAnimation(self)
        
        view = NSView(frame: NSApplication.shared.mainWindow?.frame ?? .zero)
        view.addSubview(webView)
        view.addSubview(progressBar, positioned: .above, relativeTo: webView)
    }
    
    public override func viewDidLoad() {
        super.viewDidLoad()
        
        webView?.translatesAutoresizingMaskIntoConstraints = false
        progressBar.translatesAutoresizingMaskIntoConstraints = false
        
        webView?.topAnchor.constraint(equalTo: self.view.topAnchor, constant: 0.0).isActive = true
        webView?.bottomAnchor.constraint(equalTo: self.view.bottomAnchor, constant: 0.0).isActive = true
        webView?.leadingAnchor.constraint(equalTo: self.view.leadingAnchor, constant: 0.0).isActive = true
        webView?.trailingAnchor.constraint(equalTo: self.view.trailingAnchor, constant: 0.0).isActive = true
        
        progressBar.topAnchor.constraint(equalTo: self.view.topAnchor, constant: -6.0).isActive = true
        progressBar.leadingAnchor.constraint(equalTo: self.view.leadingAnchor, constant: 0.0).isActive = true
        progressBar.trailingAnchor.constraint(equalTo: self.view.trailingAnchor, constant: 0.0).isActive = true
        
        if let wId = windowId {
            channelDelegate?.onBrowserCreated()
            webView?.runWindowBeforeCreatedCallbacks()
        } else {
            if #available(macOS 10.13, *) {
                if let contentBlockers = webView?.settings?.contentBlockers, contentBlockers.count > 0 {
                    do {
                        let jsonData = try JSONSerialization.data(withJSONObject: contentBlockers, options: [])
                        let blockRules = String(data: jsonData, encoding: .utf8)
                        WKContentRuleListStore.default().compileContentRuleList(
                            forIdentifier: "ContentBlockingRules",
                            encodedContentRuleList: blockRules) { (contentRuleList, error) in

                                if let error = error {
                                    print(error.localizedDescription)
                                    return
                                }

                                let configuration = self.webView!.configuration
                                configuration.userContentController.add(contentRuleList!)

                                self.initLoad()
                        }
                        return
                    } catch {
                        print(error.localizedDescription)
                    }
                }
            }
            
            initLoad()
        }
    }
    
    public override func viewDidAppear() {
        super.viewDidAppear()
        window = view.window as? InAppBrowserWindow
    }
    
    public func initLoad() {
        if let initialFile = initialFile {
            do {
                try webView?.loadFile(assetFilePath: initialFile)
            }
            catch let error as NSError {
                dump(error)
            }
        }
        else if let initialData = initialData {
            let baseUrl = URL(string: initialBaseUrl ?? "about:blank")!
            var allowingReadAccessToURL: URL? = nil
            if let allowingReadAccessTo = webView?.settings?.allowingReadAccessTo, baseUrl.scheme == "file" {
                allowingReadAccessToURL = URL(string: allowingReadAccessTo)
                if allowingReadAccessToURL?.scheme != "file" {
                    allowingReadAccessToURL = nil
                }
            }
            webView?.loadData(data: initialData, mimeType: initialMimeType!, encoding: initialEncoding!, baseUrl: baseUrl, allowingReadAccessTo: allowingReadAccessToURL)
        }
        else if let initialUrlRequest = initialUrlRequest {
            var allowingReadAccessToURL: URL? = nil
            if let allowingReadAccessTo = webView?.settings?.allowingReadAccessTo, let url = initialUrlRequest.url, url.scheme == "file" {
                allowingReadAccessToURL = URL(string: allowingReadAccessTo)
                if allowingReadAccessToURL?.scheme != "file" {
                    allowingReadAccessToURL = nil
                }
            }
            webView?.loadUrl(urlRequest: initialUrlRequest, allowingReadAccessTo: allowingReadAccessToURL)
        }
        
        channelDelegate?.onBrowserCreated()
    }

    public func prepareWebView() {
        webView?.settings = webViewSettings
        webView?.prepare()

        if let browserSettings = browserSettings {
            if browserSettings.hideProgressBar {
                progressBar.isHidden = true
            }
        }
    }
    
    public func didChangeTitle(title: String?) {
        guard let title = title else {
            return
        }
        if let browserSettings = browserSettings,
           let toolbarTopFixedTitle = browserSettings.toolbarTopFixedTitle {
            window?.title = toolbarTopFixedTitle
        } else {
            window?.title = title
        }
        window?.update()
    }
    
    public func didStartNavigation(url: URL?) {
        window?.forwardButton?.isEnabled = webView?.canGoForward ?? false
        window?.backButton?.isEnabled = webView?.canGoBack ?? false
        progressBar.doubleValue = 0.0
        progressBar.isHidden = false
        guard let url = url else {
            return
        }
        window?.searchBar?.stringValue = url.absoluteString
    }
    
    public func didUpdateVisitedHistory(url: URL?) {
        window?.forwardButton?.isEnabled = webView?.canGoForward ?? false
        window?.backButton?.isEnabled = webView?.canGoBack ?? false
        guard let url = url else {
            return
        }
        window?.searchBar?.stringValue = url.absoluteString
    }
    
    public func didFinishNavigation(url: URL?) {
        window?.forwardButton?.isEnabled = webView?.canGoForward ?? false
        window?.backButton?.isEnabled = webView?.canGoBack ?? false
        progressBar.doubleValue = 0.0
        progressBar.isHidden = true
        guard let url = url else {
            return
        }
        window?.searchBar?.stringValue = url.absoluteString
    }
    
    public func didFailNavigation(url: URL?, error: Error) {
        window?.forwardButton?.isEnabled = webView?.canGoForward ?? false
        window?.backButton?.isEnabled = webView?.canGoBack ?? false
        progressBar.doubleValue = 0.0
        progressBar.isHidden = true
    }
    
    public func didChangeProgress(progress: Double) {
        progressBar.isHidden = false
        progressBar.doubleValue = progress * 100
        if progress == 100.0 {
            progressBar.isHidden = true
        }
    }
    
    @objc public func reload() {
        webView?.reload()
        didUpdateVisitedHistory(url: webView?.url)
    }
    
    @objc public func goBack() {
        if let webView = webView, webView.canGoBack {
            webView.goBack()
        }
    }
    
    @objc public func goForward() {
        if let webView = webView, webView.canGoForward {
            webView.goForward()
        }
    }
    
    @objc public func goBackOrForward(steps: Int) {
        webView?.goBackOrForward(steps: steps)
    }
    
    @objc public func onMenuItemClicked(sender: Any?) {
        var identifier: String?
        if let sender = sender as? NSMenuItem {
            identifier = sender.identifier?.rawValue
        }
        if identifier == nil, let sender = sender as? NSButton {
            identifier = sender.identifier?.rawValue
        }
        if let identifier = identifier, let window = window {
            let menuItem = window.menuItems.first { item in
                return item.id == Int64(identifier)
            }
            if let menuItem = menuItem {
                channelDelegate?.onMenuItemClicked(menuItem: menuItem)
            }
        }
    }

    public func setSettings(newSettings: InAppBrowserSettings, newSettingsMap: [String: Any]) {
        window?.setSettings(newSettings: newSettings, newSettingsMap: newSettingsMap)
        
        let newInAppWebViewSettings = InAppWebViewSettings()
        let _ = newInAppWebViewSettings.parse(settings: newSettingsMap)
        webView?.setSettings(newSettings: newInAppWebViewSettings, newSettingsMap: newSettingsMap)
        
        if newSettingsMap["hideProgressBar"] != nil, browserSettings?.hideProgressBar != newSettings.hideProgressBar {
            progressBar.isHidden = newSettings.hideProgressBar
        }
        
        browserSettings = newSettings
        webViewSettings = newInAppWebViewSettings
    }
    
    public func getSettings() -> [String: Any?]? {
        let webViewSettingsMap = webView?.getSettings()
        if (self.browserSettings == nil || webViewSettingsMap == nil) {
            return nil
        }
        var settingsMap = self.browserSettings!.getRealSettings(obj: self)
        settingsMap.merge(webViewSettingsMap!, uniquingKeysWith: { (current, _) in current })
        return settingsMap
    }
    
    public func hide() {
        isHidden = true
        window?.hide()
    }
    
    public func show() {
        isHidden = false
        window?.show()
    }
    
    public func close() {
        window?.close()
    }
    
    public func dispose() {
        channelDelegate?.onExit()
        channelDelegate?.dispose()
        channelDelegate = nil
        webView?.dispose()
        webView?.removeFromSuperview()
        webView = nil
        window = nil
        plugin = nil
    }
    
    deinit {
        debugPrint("InAppBrowserWebViewController - dealloc")
        dispose()
    }
}
