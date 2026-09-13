//
//  InAppWebView.swift
//  flutter_inappwebview
//
//  Created by Lorenzo on 21/10/18.
//

import Flutter
import Foundation
import WebKit

public class InAppWebView: WKWebView, UIScrollViewDelegate, WKUIDelegate,
                            WKNavigationDelegate, WKScriptMessageHandler, UIGestureRecognizerDelegate,
                            WKDownloadDelegate,
                            PullToRefreshDelegate,
                            Disposable {
    static let METHOD_CHANNEL_NAME_PREFIX = "com.pichillilorenzo/flutter_inappwebview_"

    var id: Any? // viewId
    var plugin: SwiftFlutterPlugin?
    var windowId: Int64?
    var windowCreated = false
    var windowBeforeCreatedCallbacks: [() -> ()] = []
    var inAppBrowserDelegate: InAppBrowserDelegate?
    var channelDelegate: WebViewChannelDelegate?
    var settings: InAppWebViewSettings?
    var pullToRefreshControl: PullToRefreshControl?
    var findInteractionController: FindInteractionController?
    var webMessageChannels: [String: WebMessageChannel] = [:]
    var webMessageListeners: [WebMessageListener] = []
    var currentOriginalUrl: URL?
    var inFullscreen = false
    var preventGestureDelay = false
    
    private static var sslCertificatesMap: [String: SslCertificate] = [:] // [URL host name : SslCertificate]
    private static var credentialsProposed: [URLCredential] = []
    
    var lastScrollX: CGFloat = 0
    var lastScrollY: CGFloat = 0
    
    // Used to manage pauseTimers() and resumeTimers()
    var isPausedTimers = false
    var isPausedTimersCompletionHandler: (() -> Void)?

    var contextMenu: [String: Any]?
    var initialUserScripts: [UserScript] = []
    
    // https://github.com/mozilla-mobile/firefox-ios/blob/50531a7e9e4d459fb11d4fcb7d4322e08103501f/Client/Frontend/Browser/ContextMenuHelper.swift
    fileprivate var nativeHighlightLongPressRecognizer: UILongPressGestureRecognizer?
    fileprivate var nativeLoupeGesture: UILongPressGestureRecognizer?
    var longPressRecognizer: UILongPressGestureRecognizer!
    var recognizerForDisablingContextMenuOnLinks: UILongPressGestureRecognizer!
    var lastLongPressTouchPoint: CGPoint?
    
    var panGestureRecognizer: UIPanGestureRecognizer!
    
    var lastTouchPoint: CGPoint?
    var lastTouchPointTimestamp = Int64(Date().timeIntervalSince1970 * 1000)
    
    var contextMenuIsShowing = false
    // flag used for the workaround to trigger onCreateContextMenu event as the same on Android
    var onCreateContextMenuEventTriggeredWhenMenuDisabled = false
    
    var customIMPs: [IMP] = []
    
    var callAsyncJavaScriptBelowIOS14Results: [String:((Any?) -> Void)] = [:]
    
    var oldZoomScale = Float(1.0)
    
    fileprivate var interceptOnlyAsyncAjaxRequestsPluginScript: PluginScript?
    
    init(id: Any?, plugin: SwiftFlutterPlugin?, frame: CGRect, configuration: WKWebViewConfiguration,
         contextMenu: [String: Any]?, userScripts: [UserScript] = []) {
        super.init(frame: frame, configuration: configuration)
        self.id = id
        self.plugin = plugin
        if let id = id, let registrar = plugin?.registrar {
            let channel = FlutterMethodChannel(name: InAppWebView.METHOD_CHANNEL_NAME_PREFIX + String(describing: id),
                                               binaryMessenger: registrar.messenger())
            self.channelDelegate = WebViewChannelDelegate(webView: self, channel: channel)
        }
        self.contextMenu = contextMenu
        self.initialUserScripts = userScripts
        uiDelegate = self
        navigationDelegate = self
        scrollView.delegate = self
        longPressRecognizer = UILongPressGestureRecognizer()
        longPressRecognizer.delegate = self
        longPressRecognizer.addTarget(self, action: #selector(longPressGestureDetected))
        recognizerForDisablingContextMenuOnLinks = UILongPressGestureRecognizer()
        recognizerForDisablingContextMenuOnLinks.delegate = self
        recognizerForDisablingContextMenuOnLinks.addTarget(self, action: #selector(longPressGestureDetected))
        recognizerForDisablingContextMenuOnLinks?.minimumPressDuration = 0.45
        panGestureRecognizer = UIPanGestureRecognizer()
        panGestureRecognizer.delegate = self
        panGestureRecognizer.addTarget(self, action: #selector(endDraggingDetected))
    }
    
    override public var frame: CGRect {
        get {
            return super.frame
        }
        set {
            super.frame = newValue
            
            self.scrollView.contentInset = .zero
            if #available(iOS 11, *) {
                // Above iOS 11, adjust contentInset to compensate the adjustedContentInset so the sum will
                // always be 0.
                if (scrollView.adjustedContentInset != UIEdgeInsets.zero) {
                    let insetToAdjust = self.scrollView.adjustedContentInset
                    scrollView.contentInset = UIEdgeInsets(top: -insetToAdjust.top, left: -insetToAdjust.left,
                                                                bottom: -insetToAdjust.bottom, right: -insetToAdjust.right)
                }
            }
        }
    }
    
    required public init(coder aDecoder: NSCoder) {
        super.init(coder: aDecoder)!
    }

    public func gestureRecognizer(_ gestureRecognizer: UIGestureRecognizer, shouldRecognizeSimultaneouslyWith otherGestureRecognizer: UIGestureRecognizer) -> Bool {
        return true
    }
    
    // BVC KVO events for all changes on the webview will call this.
    // It is called frequently during a page load (particularly on progress changes and URL changes).
    // As of iOS 12, WKContentView gesture setup is async, but it has been called by the time
    // the webview is ready to load an URL. After this has happened, we can override the gesture.
    func replaceGestureHandlerIfNeeded() {
        DispatchQueue.main.async {
            if self.gestureRecognizerWithDescriptionFragment("InAppWebView") == nil {
                self.replaceWebViewLongPress()
            }
        }
    }
    
    private func replaceWebViewLongPress() {
        // WebKit installs gesture handlers async. If `replaceWebViewLongPress` is called after a wkwebview in most cases a small delay is sufficient
        // See also https://bugs.webkit.org/show_bug.cgi?id=193366
        nativeHighlightLongPressRecognizer = gestureRecognizerWithDescriptionFragment("action=_highlightLongPressRecognized:")
        nativeLoupeGesture = gestureRecognizerWithDescriptionFragment("action=loupeGesture:")

        if let nativeLongPressRecognizer = gestureRecognizerWithDescriptionFragment("action=_longPressRecognized:") {
            nativeLongPressRecognizer.removeTarget(nil, action: nil)
            nativeLongPressRecognizer.addTarget(self, action: #selector(self.longPressGestureDetected))
        }
    }
    
    private func gestureRecognizerWithDescriptionFragment(_ descriptionFragment: String) -> UILongPressGestureRecognizer? {
        let result = self.scrollView.subviews.compactMap({ $0.gestureRecognizers }).joined().first(where: {
            return (($0 as? UILongPressGestureRecognizer) != nil) && $0.description.contains(descriptionFragment)
        })
        return result as? UILongPressGestureRecognizer
    }
    
    @objc func longPressGestureDetected(_ sender: UIGestureRecognizer) {
        if sender.state == .cancelled {
            return
        }

        guard sender.state == .began else {
            return
        }
        
        if sender == recognizerForDisablingContextMenuOnLinks,
           let settings = settings, !settings.disableLongPressContextMenuOnLinks {
            return
        }
        
        if sender == longPressRecognizer {
            // To prevent the tapped link from proceeding with navigation, "cancel" the native WKWebView
            // `_highlightLongPressRecognizer`. This preserves the original behavior as seen here:
            // https://github.com/WebKit/webkit/blob/d591647baf54b4b300ca5501c21a68455429e182/Source/WebKit/UIProcess/ios/WKContentViewInteraction.mm#L1600-L1614
            if let nativeHighlightLongPressRecognizer = nativeHighlightLongPressRecognizer,
                nativeHighlightLongPressRecognizer.isEnabled {
                nativeHighlightLongPressRecognizer.isEnabled = false
                nativeHighlightLongPressRecognizer.isEnabled = true
            }
        }

        //Finding actual touch location in webView
        var touchLocation = sender.location(in: self)
        touchLocation.x -= scrollView.contentInset.left
        touchLocation.y -= scrollView.contentInset.top
        touchLocation.x /= scrollView.zoomScale
        touchLocation.y /= scrollView.zoomScale

        lastLongPressTouchPoint = touchLocation

        evaluateJavaScript("window.\(JAVASCRIPT_BRIDGE_NAME)._findElementsAtPoint(\(touchLocation.x),\(touchLocation.y))", completionHandler: {(value, error) in
            if error != nil {
                print("Long press gesture recognizer error: \(error?.localizedDescription ?? "")")
            } else if let value = value as? [String: Any?] {
                let hitTestResult = HitTestResult.fromMap(map: value)!
                self.nativeLoupeGesture = self.gestureRecognizerWithDescriptionFragment("action=loupeGesture:")
                
                if sender == self.recognizerForDisablingContextMenuOnLinks,
                   hitTestResult.type.rawValue > HitTestResultType.unknownType.rawValue,
                   hitTestResult.type.rawValue < HitTestResultType.editTextType.rawValue {
                    self.nativeLoupeGesture?.isEnabled = false
                    self.nativeLoupeGesture?.isEnabled = true
                } else {
                    self.channelDelegate?.onLongPressHitTestResult(hitTestResult: hitTestResult)
                }
            } else if sender == self.longPressRecognizer {
                self.channelDelegate?.onLongPressHitTestResult(hitTestResult: HitTestResult(type: .unknownType, extra: nil))
            }
        })
    }
    
    public override func hitTest(_ point: CGPoint, with event: UIEvent?) -> UIView? {
        lastTouchPoint = point
        lastTouchPointTimestamp = Int64(Date().timeIntervalSince1970 * 1000)
        SharedLastTouchPointTimestamp[self] = lastTouchPointTimestamp
        
        // re-build context menu items for the current webview
        UIMenuController.shared.menuItems = []
        if let menu = self.contextMenu {
            if let menuItems = menu["menuItems"] as? [[String : Any]] {
                for menuItem in menuItems {
                    let id = menuItem["id"]!
                    let title = menuItem["title"] as! String
                    let targetMethodName = "onContextMenuActionItemClicked-" + String(self.hash) + "-" +
                                            (id is Int64 ? String(id as! Int64) : id as! String)
                    if !self.responds(to: Selector(targetMethodName)) {
                        let customAction: () -> Void = {
                            self.channelDelegate?.onContextMenuActionItemClicked(id: id, title: title)
                            if #available(iOS 16.0, *) {
                                if #unavailable(iOS 16.4) {
                                    self.onHideContextMenu()
                                }
                            }
                        }
                        let castedCustomAction: AnyObject = unsafeBitCast(customAction as @convention(block) () -> Void, to: AnyObject.self)
                        let swizzledImplementation = imp_implementationWithBlock(castedCustomAction)
                        class_addMethod(InAppWebView.self, Selector(targetMethodName), swizzledImplementation, nil)
                        self.customIMPs.append(swizzledImplementation)
                    }
                    let item = UIMenuItem(title: title, action: Selector(targetMethodName))
                    UIMenuController.shared.menuItems!.append(item)
                }
            }
        }
        
        // https://github.com/pichillilorenzo/flutter_inappwebview/pull/1665
        if preventGestureDelay, let gestures = superview?.superview?.gestureRecognizers {
            for gesture in gestures {
                if NSStringFromClass(type(of: gesture)) == "DelayingGestureRecognizer" {
                    gesture.isEnabled = false
                }
            }
        }
        
        return super.hitTest(point, with: event)
    }
    
    @available(iOS 13.0, *)
    public override func buildMenu(with builder: UIMenuBuilder) {
        if #available(iOS 16.0, *) {
            if let menu = contextMenu {
                let contextMenuSettings = ContextMenuSettings()
                if let contextMenuSettingsMap = menu["settings"] as? [String: Any?] {
                    let _ = contextMenuSettings.parse(settings: contextMenuSettingsMap)
                    if contextMenuSettings.hideDefaultSystemContextMenuItems {
                        builder.remove(menu: .lookup)
                    }
                }
            }
            
            if #unavailable(iOS 16.4), settings?.disableContextMenu == false {
                contextMenuIsShowing = false
                DispatchQueue.main.asyncAfter(deadline: .now() + 0.25) {
                    self.onCreateContextMenu()
                }
            }
        }
        super.buildMenu(with: builder)
    }
    
    @available(iOS 16.4, *)
    public func webView(_ webView: WKWebView, willPresentEditMenuWithAnimator animator: UIEditMenuInteractionAnimating) {
        onCreateContextMenu()
    }
    
    @available(iOS 16.4, *)
    public func webView(_ webView: WKWebView, willDismissEditMenuWithAnimator animator: UIEditMenuInteractionAnimating) {
        onHideContextMenu()
    }
    
    public override func canPerformAction(_ action: Selector, withSender sender: Any?) -> Bool {
        var needCheck = sender is UIMenuController
        if #available(iOS 13.0, *) {
            needCheck = sender is UIMenuElement || sender is UIMenuController
        }
        
        if needCheck {
            if settings?.disableContextMenu == true {
                if !onCreateContextMenuEventTriggeredWhenMenuDisabled {
                    // workaround to trigger onCreateContextMenu event as the same on Android
                    onCreateContextMenu()
                    onCreateContextMenuEventTriggeredWhenMenuDisabled = true
                    DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
                        self.onCreateContextMenuEventTriggeredWhenMenuDisabled = false
                    }
                }
                return false
            }
            
            if let menu = contextMenu {
                let contextMenuSettings = ContextMenuSettings()
                if let contextMenuSettingsMap = menu["settings"] as? [String: Any?] {
                    let _ = contextMenuSettings.parse(settings: contextMenuSettingsMap)
                    if !action.description.starts(with: "onContextMenuActionItemClicked-") && contextMenuSettings.hideDefaultSystemContextMenuItems {
                        return false
                    }
                }
            }
            
            if contextMenuIsShowing, !action.description.starts(with: "onContextMenuActionItemClicked-") {
                let id = action.description.compactMap({ $0.asciiValue?.description }).joined()
                channelDelegate?.onContextMenuActionItemClicked(id: id, title: action.description)
                if #available(iOS 16.0, *) {
                    if #unavailable(iOS 16.4) {
                        onHideContextMenu()
                    }
                }
            }
        }
        return super.canPerformAction(action, withSender: sender)
    }
    
    // For some reasons, using the scrollViewDidEndDragging event, in some rare cases, could block
    // the scroll gesture
    @objc func endDraggingDetected() {
        // detect end dragging
        if panGestureRecognizer.state == .ended {
            // fix for pull-to-refresh jittering when the touch drag event is held
            if let pullToRefreshControl = pullToRefreshControl,
               pullToRefreshControl.shouldCallOnRefresh {
                pullToRefreshControl.onRefresh()
            }
        }
    }

    public func prepare() {
        scrollView.addGestureRecognizer(self.longPressRecognizer)
        scrollView.addGestureRecognizer(self.recognizerForDisablingContextMenuOnLinks)
        scrollView.addGestureRecognizer(self.panGestureRecognizer)
        scrollView.addObserver(self, forKeyPath: #keyPath(UIScrollView.contentOffset), options: [.new, .old], context: nil)
        scrollView.addObserver(self, forKeyPath: #keyPath(UIScrollView.zoomScale), options: [.new, .old], context: nil)
        scrollView.addObserver(self, forKeyPath: #keyPath(UIScrollView.contentSize), options: [.new, .old], context: nil)
        
        addObserver(self,
                    forKeyPath: #keyPath(WKWebView.estimatedProgress),
                    options: .new,
                    context: nil)
        
        addObserver(self,
                    forKeyPath: #keyPath(WKWebView.url),
                    options: [.new, .old],
                    context: nil)
        
        addObserver(self,
            forKeyPath: #keyPath(WKWebView.title),
            options: [.new, .old],
            context: nil)
        
        if #available(iOS 15.0, *) {
            addObserver(self,
                forKeyPath: #keyPath(WKWebView.cameraCaptureState),
                options: [.new, .old],
                context: nil)
            
            addObserver(self,
                forKeyPath: #keyPath(WKWebView.microphoneCaptureState),
                options: [.new, .old],
                context: nil)
        }
        
        if #unavailable(iOS 16.0) {
            NotificationCenter.default.addObserver(
                self,
                selector: #selector(onCreateContextMenu),
                name: UIMenuController.willShowMenuNotification,
                object: nil)
            
            NotificationCenter.default.addObserver(
                self,
                selector: #selector(onHideContextMenu),
                name: UIMenuController.didHideMenuNotification,
                object: nil)
        }
        
        // TODO: Still not working on iOS 16.0!
//        if #available(iOS 16.0, *) {
//            addObserver(self,
//                        forKeyPath: #keyPath(WKWebView.fullscreenState),
//                        options: .new,
//                context: nil)
//        } else {
            // listen for videos playing in fullscreen
            NotificationCenter.default.addObserver(self,
                                                   selector: #selector(onEnterFullscreen(_:)),
                                                   name: UIWindow.didBecomeVisibleNotification,
                                                   object: window)

            // listen for videos stopping to play in fullscreen
            NotificationCenter.default.addObserver(self,
                                                   selector: #selector(onExitFullscreen(_:)),
                                                   name: UIWindow.didBecomeHiddenNotification,
                                                   object: window)
//        }
        
        if let settings = settings {
            if settings.transparentBackground {
                isOpaque = false
                backgroundColor = UIColor.clear
                scrollView.backgroundColor = UIColor.clear
            }
            
            // prevent webView from bouncing
            if settings.disallowOverScroll {
                if responds(to: #selector(getter: scrollView)) {
                    scrollView.bounces = false
                }
                else {
                    for subview: UIView in subviews {
                        if subview is UIScrollView {
                            (subview as! UIScrollView).bounces = false
                        }
                    }
                }
            }
            
            if #available(iOS 11.0, *) {
                accessibilityIgnoresInvertColors = settings.accessibilityIgnoresInvertColors
                scrollView.contentInsetAdjustmentBehavior =
                    UIScrollView.ContentInsetAdjustmentBehavior.init(rawValue: settings.contentInsetAdjustmentBehavior)!
            }
            
            allowsBackForwardNavigationGestures = settings.allowsBackForwardNavigationGestures
            if #available(iOS 9.0, *) {
                allowsLinkPreview = settings.allowsLinkPreview
                if !settings.userAgent.isEmpty {
                    customUserAgent = settings.userAgent
                }
            }
            
            if #available(iOS 13.0, *) {
                scrollView.automaticallyAdjustsScrollIndicatorInsets = settings.automaticallyAdjustsScrollIndicatorInsets
            }
            
            scrollView.showsVerticalScrollIndicator = !settings.disableVerticalScroll
            scrollView.showsHorizontalScrollIndicator = !settings.disableHorizontalScroll
            scrollView.showsVerticalScrollIndicator = settings.verticalScrollBarEnabled
            scrollView.showsHorizontalScrollIndicator = settings.horizontalScrollBarEnabled
            scrollView.isScrollEnabled = !(settings.disableVerticalScroll && settings.disableHorizontalScroll)
            scrollView.isDirectionalLockEnabled = settings.isDirectionalLockEnabled

            scrollView.decelerationRate = Util.getDecelerationRate(type: settings.decelerationRate)
            scrollView.alwaysBounceVertical = settings.alwaysBounceVertical
            scrollView.alwaysBounceHorizontal = settings.alwaysBounceHorizontal
            scrollView.scrollsToTop = settings.scrollsToTop
            scrollView.isPagingEnabled = settings.isPagingEnabled
            scrollView.maximumZoomScale = CGFloat(settings.maximumZoomScale)
            scrollView.minimumZoomScale = CGFloat(settings.minimumZoomScale)
            
            if #available(iOS 14.0, *) {
                mediaType = settings.mediaType
                pageZoom = CGFloat(settings.pageZoom)
            }
            
            if #available(iOS 15.0, *) {
                if let underPageBackgroundColor = settings.underPageBackgroundColor, !underPageBackgroundColor.isEmpty {
                    self.underPageBackgroundColor = UIColor(hexString: underPageBackgroundColor)
                }
            }
            
            if #available(iOS 15.5, *) {
                if let minViewportInset = settings.minimumViewportInset, let maxViewportInset = settings.maximumViewportInset {
                    setMinimumViewportInset(minViewportInset, maximumViewportInset: maxViewportInset)
                }
            }
            
            if #available(iOS 16.0, *) {
                isFindInteractionEnabled = settings.isFindInteractionEnabled
            }
            
            if #available(iOS 16.4, *) {
                isInspectable = settings.isInspectable
            }
            
            if settings.clearCache {
                clearCache()
            }
        }
        
        prepareAndAddUserScripts()
        
        if windowId != nil {
            // The new created window webview has the same WKWebViewConfiguration variable reference.
            // So, we cannot set another WKWebViewConfiguration for it unfortunately!
            // This is a limitation of the official WebKit API.
            return
        }
        
        configuration.preferences = WKPreferences()
        if let settings = settings {
            if #available(iOS 9.0, *) {
                configuration.allowsAirPlayForMediaPlayback = settings.allowsAirPlayForMediaPlayback
                configuration.allowsPictureInPictureMediaPlayback = settings.allowsPictureInPictureMediaPlayback
            }
            
            configuration.preferences.javaScriptCanOpenWindowsAutomatically = settings.javaScriptCanOpenWindowsAutomatically
            configuration.preferences.minimumFontSize = CGFloat(settings.minimumFontSize)
            
            if #available(iOS 13.0, *) {
                configuration.preferences.isFraudulentWebsiteWarningEnabled = settings.isFraudulentWebsiteWarningEnabled
                configuration.defaultWebpagePreferences.preferredContentMode = WKWebpagePreferences.ContentMode(rawValue: settings.preferredContentMode)!
            }
            
            configuration.preferences.javaScriptEnabled = settings.javaScriptEnabled
            if #available(iOS 14.0, *) {
                configuration.defaultWebpagePreferences.allowsContentJavaScript = settings.javaScriptEnabled
            }
            
            if #available(iOS 15.0, *) {
                configuration.preferences.isTextInteractionEnabled = settings.isTextInteractionEnabled
            }
            if #available(iOS 15.4, *) {
                configuration.preferences.isSiteSpecificQuirksModeEnabled = settings.isSiteSpecificQuirksModeEnabled
                configuration.preferences.isElementFullscreenEnabled = settings.isElementFullscreenEnabled
            }
            if #available(iOS 16.4, *) {
                configuration.preferences.shouldPrintBackgrounds = settings.shouldPrintBackgrounds
            }
        }
    }
    
    public func prepareAndAddUserScripts() -> Void {
        if windowId != nil {
            // The new created window webview has the same WKWebViewConfiguration variable reference.
            // So, we cannot set another WKWebViewConfiguration for it unfortunately!
            // This is a limitation of the official WebKit API.
            return
        }
        configuration.userContentController = WKUserContentController()
        configuration.userContentController.initialize()
        
        if let applePayAPIEnabled = settings?.applePayAPIEnabled, applePayAPIEnabled {
            return
        }
        
        configuration.userContentController.addPluginScript(PROMISE_POLYFILL_JS_PLUGIN_SCRIPT)
        configuration.userContentController.addPluginScript(JAVASCRIPT_BRIDGE_JS_PLUGIN_SCRIPT)
        configuration.userContentController.addPluginScript(CONSOLE_LOG_JS_PLUGIN_SCRIPT)
        configuration.userContentController.addPluginScript(PRINT_JS_PLUGIN_SCRIPT)
        configuration.userContentController.addPluginScript(ON_WINDOW_BLUR_EVENT_JS_PLUGIN_SCRIPT)
        configuration.userContentController.addPluginScript(ON_WINDOW_FOCUS_EVENT_JS_PLUGIN_SCRIPT)
        configuration.userContentController.addPluginScript(FIND_ELEMENTS_AT_POINT_JS_PLUGIN_SCRIPT)
        configuration.userContentController.addPluginScript(LAST_TOUCHED_ANCHOR_OR_IMAGE_JS_PLUGIN_SCRIPT)
        configuration.userContentController.addPluginScript(FIND_TEXT_HIGHLIGHT_JS_PLUGIN_SCRIPT)
        configuration.userContentController.addPluginScript(ORIGINAL_VIEWPORT_METATAG_CONTENT_JS_PLUGIN_SCRIPT)
        if let settings = settings {
            interceptOnlyAsyncAjaxRequestsPluginScript = createInterceptOnlyAsyncAjaxRequestsPluginScript(onlyAsync: settings.interceptOnlyAsyncAjaxRequests)
            if settings.useShouldInterceptAjaxRequest {
                if let interceptOnlyAsyncAjaxRequestsPluginScript = interceptOnlyAsyncAjaxRequestsPluginScript {
                    configuration.userContentController.addPluginScript(interceptOnlyAsyncAjaxRequestsPluginScript)
                }
                configuration.userContentController.addPluginScript(INTERCEPT_AJAX_REQUEST_JS_PLUGIN_SCRIPT)
            }
            if settings.useShouldInterceptFetchRequest {
                configuration.userContentController.addPluginScript(INTERCEPT_FETCH_REQUEST_JS_PLUGIN_SCRIPT)
            }
            if settings.useOnLoadResource {
                configuration.userContentController.addPluginScript(ON_LOAD_RESOURCE_JS_PLUGIN_SCRIPT)
            }
            if !settings.supportZoom {
                configuration.userContentController.addPluginScript(NOT_SUPPORT_ZOOM_JS_PLUGIN_SCRIPT)
            } else if settings.enableViewportScale {
                configuration.userContentController.addPluginScript(ENABLE_VIEWPORT_SCALE_JS_PLUGIN_SCRIPT)
            }
        }
        configuration.userContentController.removeScriptMessageHandler(forName: "onCallAsyncJavaScriptResultBelowIOS14Received")
        configuration.userContentController.add(self, name: "onCallAsyncJavaScriptResultBelowIOS14Received")
        configuration.userContentController.removeScriptMessageHandler(forName: "onWebMessagePortMessageReceived")
        configuration.userContentController.add(self, name: "onWebMessagePortMessageReceived")
        configuration.userContentController.removeScriptMessageHandler(forName: "onWebMessageListenerPostMessageReceived")
        configuration.userContentController.add(self, name: "onWebMessageListenerPostMessageReceived")
        configuration.userContentController.addUserOnlyScripts(initialUserScripts)
        configuration.userContentController.sync(scriptMessageHandler: self)
    }
    
    public static func preWKWebViewConfiguration(settings: InAppWebViewSettings?) -> WKWebViewConfiguration {
        let configuration = WKWebViewConfiguration()
        
        configuration.processPool = WKProcessPoolManager.sharedProcessPool
        
        if let settings = settings {
            configuration.allowsInlineMediaPlayback = settings.allowsInlineMediaPlayback
            configuration.suppressesIncrementalRendering = settings.suppressesIncrementalRendering
            configuration.selectionGranularity = WKSelectionGranularity.init(rawValue: settings.selectionGranularity)!
            
            if settings.allowUniversalAccessFromFileURLs {
                configuration.setValue(settings.allowUniversalAccessFromFileURLs, forKey: "allowUniversalAccessFromFileURLs")
            }
            
            if settings.allowFileAccessFromFileURLs {
                configuration.preferences.setValue(settings.allowFileAccessFromFileURLs, forKey: "allowFileAccessFromFileURLs")
            }
            
            if #available(iOS 9.0, *) {
                if settings.incognito {
                    configuration.websiteDataStore = WKWebsiteDataStore.nonPersistent()
                } else if settings.cacheEnabled {
                    configuration.websiteDataStore = GopeedProfiles.store(settings.gopeedProfileId) ?? WKWebsiteDataStore.default()
                }
                if !settings.applicationNameForUserAgent.isEmpty {
                    if let applicationNameForUserAgent = configuration.applicationNameForUserAgent {
                        configuration.applicationNameForUserAgent = applicationNameForUserAgent + " " + settings.applicationNameForUserAgent
                    }
                }
            }
            
            if #available(iOS 10.0, *) {
                configuration.ignoresViewportScaleLimits = settings.ignoresViewportScaleLimits
                
                var dataDetectorTypes = WKDataDetectorTypes.init(rawValue: 0)
                for type in settings.dataDetectorTypes {
                    let dataDetectorType = Util.getDataDetectorType(type: type)
                    dataDetectorTypes = WKDataDetectorTypes(rawValue: dataDetectorTypes.rawValue | dataDetectorType.rawValue)
                }
                configuration.dataDetectorTypes = dataDetectorTypes
                
                configuration.mediaTypesRequiringUserActionForPlayback = settings.mediaPlaybackRequiresUserGesture ? .all : []
            } else {
                // Fallback on earlier versions
                configuration.mediaPlaybackRequiresUserAction = settings.mediaPlaybackRequiresUserGesture
            }
            
            if #available(iOS 11.0, *) {
                for scheme in settings.resourceCustomSchemes {
                    configuration.setURLSchemeHandler(CustomSchemeHandler(), forURLScheme: scheme)
                }
                if settings.sharedCookiesEnabled {
                    // More info to sending cookies with WKWebView
                    // https://stackoverflow.com/questions/26573137/can-i-set-the-cookies-to-be-used-by-a-wkwebview/26577303#26577303
                    // Set Cookies in iOS 11 and above, initialize websiteDataStore before setting cookies
                    // See also https://forums.developer.apple.com/thread/97194
                    // check if websiteDataStore has not been initialized before
                    if(!settings.incognito && !settings.cacheEnabled) {
                        configuration.websiteDataStore = WKWebsiteDataStore.nonPersistent()
                    }
                    for cookie in HTTPCookieStorage.shared.cookies ?? [] {
                        configuration.websiteDataStore.httpCookieStore.setCookie(cookie, completionHandler: nil)
                    }
                }
            }
            
            if #available(iOS 14.0, *) {
                configuration.limitsNavigationsToAppBoundDomains = settings.limitsNavigationsToAppBoundDomains
            }
            
            if #available(iOS 15.0, *) {
                configuration.upgradeKnownHostsToHTTPS = settings.upgradeKnownHostsToHTTPS
            }
        }
        
        return configuration
    }
    
    @objc func onCreateContextMenu() {
        let mapSorted = SharedLastTouchPointTimestamp.sorted { $0.value > $1.value }
        if (mapSorted.first?.key != self) {
            return
        }
        
        contextMenuIsShowing = true
        
        let hitTestResult = HitTestResult(type: .unknownType, extra: nil)
        
        if let lastLongPressTouhLocation = lastLongPressTouchPoint {
            if configuration.preferences.javaScriptEnabled {
                self.evaluateJavaScript("window.\(JAVASCRIPT_BRIDGE_NAME)._findElementsAtPoint(\(lastLongPressTouhLocation.x),\(lastLongPressTouhLocation.y))", completionHandler: {(value, error) in
                    if error != nil {
                        print("Long press gesture recognizer error: \(error?.localizedDescription ?? "")")
                    } else if let value = value as? [String: Any?] {
                        self.channelDelegate?.onCreateContextMenu(hitTestResult: HitTestResult.fromMap(map: value) ?? hitTestResult)
                    } else {
                        self.channelDelegate?.onCreateContextMenu(hitTestResult: hitTestResult)
                    }
                })
            } else {
                channelDelegate?.onCreateContextMenu(hitTestResult: hitTestResult)
            }
        } else {
            channelDelegate?.onCreateContextMenu(hitTestResult: hitTestResult)
        }
    }
    
    @objc func onHideContextMenu() {
        if contextMenuIsShowing == false {
            return
        }
        contextMenuIsShowing = false
        channelDelegate?.onHideContextMenu()
    }
    
    override public func observeValue(forKeyPath keyPath: String?, of object: Any?,
                               change: [NSKeyValueChangeKey : Any]?, context: UnsafeMutableRawPointer?) {
        if keyPath == #keyPath(WKWebView.estimatedProgress) {
            initializeWindowIdJS()
            let progress = Int(estimatedProgress * 100)
            channelDelegate?.onProgressChanged(progress: progress)
            inAppBrowserDelegate?.didChangeProgress(progress: estimatedProgress)
        } else if keyPath == #keyPath(WKWebView.url) && change?[.newKey] is URL {
            initializeWindowIdJS()
            let newUrl = change?[NSKeyValueChangeKey.newKey] as? URL
            channelDelegate?.onUpdateVisitedHistory(url: newUrl?.absoluteString, isReload: nil)
            inAppBrowserDelegate?.didUpdateVisitedHistory(url: newUrl)
        } else if keyPath == #keyPath(WKWebView.title) && change?[.newKey] is String {
            let newTitle = change?[.newKey] as? String
            channelDelegate?.onTitleChanged(title: newTitle)
            inAppBrowserDelegate?.didChangeTitle(title: newTitle)
        } else if keyPath == #keyPath(UIScrollView.contentOffset) {
            let newContentOffset = change?[.newKey] as? CGPoint
            let oldContentOffset = change?[.oldKey] as? CGPoint
            let startedByUser = scrollView.isDragging || scrollView.isDecelerating
            if newContentOffset != oldContentOffset {
                DispatchQueue.main.async {
                    self.onScrollChanged(startedByUser: startedByUser, oldContentOffset: oldContentOffset)
                }
            }
        } else if keyPath == #keyPath(UIScrollView.contentSize) {
            if let newContentSize = change?[.newKey] as? CGSize,
               let oldContentSize = change?[.oldKey] as? CGSize,
               newContentSize != oldContentSize {
                DispatchQueue.main.async {
                    self.onContentSizeChanged(oldContentSize: oldContentSize)
                }
            }
        }
        else if #available(iOS 15.0, *) {
            if keyPath == #keyPath(WKWebView.cameraCaptureState) || keyPath == #keyPath(WKWebView.microphoneCaptureState) {
                var oldState: WKMediaCaptureState? = nil
                if let oldValue = change?[.oldKey] as? Int {
                    oldState = WKMediaCaptureState.init(rawValue: oldValue)
                }
                var newState: WKMediaCaptureState? = nil
                if let newValue = change?[.newKey] as? Int {
                    newState = WKMediaCaptureState.init(rawValue: newValue)
                }
                if oldState != newState {
                    if keyPath == #keyPath(WKWebView.cameraCaptureState) {
                        channelDelegate?.onCameraCaptureStateChanged(oldState: oldState, newState: newState)
                    } else {
                        channelDelegate?.onMicrophoneCaptureStateChanged(oldState: oldState, newState: newState)
                    }
                }
            }
        } else if #available(iOS 16.0, *) {
            // TODO: Still not working on iOS 16.0!
//            if keyPath == #keyPath(WKWebView.fullscreenState) {
//                if fullscreenState == .enteringFullscreen {
//                    channelDelegate?.onEnterFullscreen()
//                } else if fullscreenState == .exitingFullscreen {
//                    channelDelegate?.onExitFullscreen()
//                }
//            }
        }
        replaceGestureHandlerIfNeeded()
    }
    
    public func initializeWindowIdJS() {
        if let windowId = windowId {
            if #available(iOS 14.0, *) {
                let contentWorlds = configuration.userContentController.getContentWorlds(with: windowId)
                for contentWorld in contentWorlds {
                    let source = WINDOW_ID_INITIALIZE_JS_SOURCE.replacingOccurrences(of: PluginScriptsUtil.VAR_PLACEHOLDER_VALUE, with: String(windowId))
                    evaluateJavascript(source: source, contentWorld: contentWorld)
                }
            } else {
                let source = WINDOW_ID_INITIALIZE_JS_SOURCE.replacingOccurrences(of: PluginScriptsUtil.VAR_PLACEHOLDER_VALUE, with: String(windowId))
                evaluateJavascript(source: source)
            }
        }
    }
    
    public func goBackOrForward(steps: Int) {
        if canGoBackOrForward(steps: steps) {
            if (steps > 0) {
                let index = steps - 1
                go(to: self.backForwardList.forwardList[index])
            }
            else if (steps < 0){
                let backListLength = self.backForwardList.backList.count
                let index = backListLength + steps
                go(to: self.backForwardList.backList[index])
            }
        }
    }
    
    public func canGoBackOrForward(steps: Int) -> Bool {
        let currentIndex = self.backForwardList.backList.count
        return (steps >= 0)
            ? steps <= self.backForwardList.forwardList.count
            : currentIndex + steps >= 0
    }
    
    @available(iOS 11.0, *)
    public func takeScreenshot (with: [String: Any?]?, completionHandler: @escaping (_ screenshot: Data?) -> Void) {
        var snapshotConfiguration: WKSnapshotConfiguration? = nil
        if let with = with {
            snapshotConfiguration = WKSnapshotConfiguration()
            if let rect = with["rect"] as? [String: Double] {
                snapshotConfiguration!.rect = CGRect.fromMap(map: rect)
            }
            if let snapshotWidth = with["snapshotWidth"] as? Double {
                snapshotConfiguration!.snapshotWidth = NSNumber(value: snapshotWidth)
            }
            if #available(iOS 13.0, *), let afterScreenUpdates = with["afterScreenUpdates"] as? Bool {
                snapshotConfiguration!.afterScreenUpdates = afterScreenUpdates
            }
        }
        takeSnapshot(with: snapshotConfiguration, completionHandler: {(image, error) -> Void in
            var imageData: Data? = nil
            if let screenshot = image {
                if let with = with {
                    switch with["compressFormat"] as! String {
                    case "JPEG":
                        let quality = Float(with["quality"] as! Int) / 100
                        imageData = screenshot.jpegData(compressionQuality: CGFloat(quality))
                        break
                    case "PNG":
                        imageData = screenshot.pngData()
                        break
                    default:
                        imageData = screenshot.pngData()
                    }
                }
                else {
                    imageData = screenshot.pngData()
                }
            }
            completionHandler(imageData)
        })
    }
    
    @available(iOS 14.0, *)
    public func createPdf (configuration: [String: Any?]?, completionHandler: @escaping (_ pdf: Data?) -> Void) {
        let pdfConfiguration: WKPDFConfiguration = .init()
        if let configuration = configuration {
            if let rect = configuration["rect"] as? [String: Double] {
                pdfConfiguration.rect = CGRect.fromMap(map: rect)
            }
        }
        createPDF(configuration: pdfConfiguration) { (result) in
            switch (result) {
            case .success(let data):
                completionHandler(data)
                return
            case .failure(let error):
                print(error.localizedDescription)
                completionHandler(nil)
                return
            }
        }
    }
    
    @available(iOS 14.0, *)
    public func createWebArchiveData (dataCompletionHandler: @escaping (_ webArchiveData: Data?) -> Void) {
        createWebArchiveData(completionHandler: { (result) in
            switch (result) {
            case .success(let data):
                dataCompletionHandler(data)
                return
            case .failure(let error):
                print(error.localizedDescription)
                dataCompletionHandler(nil)
                return
            }
        })
    }
    
    @available(iOS 14.0, *)
    public func saveWebArchive (filePath: String, autoname: Bool, completionHandler: @escaping (_ path: String?) -> Void) {
        createWebArchiveData(dataCompletionHandler: { (webArchiveData) in
            if let webArchiveData = webArchiveData {
                var localUrl = URL(fileURLWithPath: filePath)
                if autoname {
                    if let url = self.url {
                        // tries to mimic Android saveWebArchive method
                        let invalidCharacters = CharacterSet(charactersIn: "\\/:*?\"<>|")
                                    .union(.newlines)
                                    .union(.illegalCharacters)
                                    .union(.controlCharacters)
                                
                        let currentPageUrlFileName = url.path
                            .components(separatedBy: invalidCharacters)
                            .joined(separator: "")
                        
                        let fullPath = filePath + "/" + currentPageUrlFileName + ".webarchive"
                        localUrl = URL(fileURLWithPath: fullPath)
                    } else {
                        completionHandler(nil)
                        return
                    }
                }
                do {
                    try webArchiveData.write(to: localUrl)
                    completionHandler(localUrl.path)
                } catch {
                    // Catch any errors
                    print(error.localizedDescription)
                    completionHandler(nil)
                }
            } else {
                completionHandler(nil)
            }
        })
    }
    
    public func loadUrl(urlRequest: URLRequest, allowingReadAccessTo: URL?) {
        let url = urlRequest.url!
        
        if #available(iOS 9.0, *), let allowingReadAccessTo = allowingReadAccessTo, url.scheme == "file", allowingReadAccessTo.scheme == "file" {
            loadFileURL(url, allowingReadAccessTo: allowingReadAccessTo)
        } else {
            load(urlRequest)
        }
    }
    
    public func postUrl(url: URL, postData: Data) {
        var request = URLRequest(url: url)
        
        request.addValue("application/x-www-form-urlencoded", forHTTPHeaderField: "Content-Type")
        request.httpMethod = "POST"
        request.httpBody = postData
        load(request)
    }
    
    public func loadData(data: String, mimeType: String, encoding: String, baseUrl: URL, allowingReadAccessTo: URL?) {
        if #available(iOS 9.0, *), let allowingReadAccessTo = allowingReadAccessTo, baseUrl.scheme == "file", allowingReadAccessTo.scheme == "file" {
            loadFileURL(baseUrl, allowingReadAccessTo: allowingReadAccessTo)
        }
        
        if #available(iOS 9.0, *) {
            load(data.data(using: .utf8)!, mimeType: mimeType, characterEncodingName: encoding, baseURL: baseUrl)
        } else {
            loadHTMLString(data, baseURL: baseUrl)
        }
    }
    
    public func loadFile(assetFilePath: String) throws {
        if let plugin = plugin {
            let assetURL = try Util.getUrlAsset(plugin: plugin, assetFilePath: assetFilePath)
            let urlRequest = URLRequest(url: assetURL)
            loadUrl(urlRequest: urlRequest, allowingReadAccessTo: nil)
        }
    }
    
    func setSettings(newSettings: InAppWebViewSettings, newSettingsMap: [String: Any]) {
        
        // MUST be the first! In this way, all the settings that uses evaluateJavaScript can be applied/blocked!
        if #available(iOS 13.0, *) {
            if newSettingsMap["applePayAPIEnabled"] != nil && settings?.applePayAPIEnabled != newSettings.applePayAPIEnabled {
                if let settings = settings {
                    settings.applePayAPIEnabled = newSettings.applePayAPIEnabled
                }
                if !newSettings.applePayAPIEnabled {
                    // re-add WKUserScripts for the next page load
                    prepareAndAddUserScripts()
                } else {
                    configuration.userContentController.removeAllUserScripts()
                }
            }
        }
        
        if newSettingsMap["transparentBackground"] != nil && settings?.transparentBackground != newSettings.transparentBackground {
            if newSettings.transparentBackground {
                isOpaque = false
                backgroundColor = UIColor.clear
                scrollView.backgroundColor = UIColor.clear
            } else {
                isOpaque = true
                backgroundColor = nil
                scrollView.backgroundColor = UIColor(red: 1, green: 1, blue: 1, alpha: 1)
            }
        }
        
        if newSettingsMap["disallowOverScroll"] != nil && settings?.disallowOverScroll != newSettings.disallowOverScroll {
            if responds(to: #selector(getter: scrollView)) {
                scrollView.bounces = !newSettings.disallowOverScroll
            }
            else {
                for subview: UIView in subviews {
                    if subview is UIScrollView {
                        (subview as! UIScrollView).bounces = !newSettings.disallowOverScroll
                    }
                }
            }
        }
        
        if #available(iOS 9.0, *) {
            if (newSettingsMap["incognito"] != nil && settings?.incognito != newSettings.incognito && newSettings.incognito) {
                configuration.websiteDataStore = WKWebsiteDataStore.nonPersistent()
            } else if (newSettingsMap["cacheEnabled"] != nil && settings?.cacheEnabled != newSettings.cacheEnabled && newSettings.cacheEnabled) {
                configuration.websiteDataStore = GopeedProfiles.store(settings?.gopeedProfileId ?? "") ?? WKWebsiteDataStore.default()
            }
        }
        
        if #available(iOS 11.0, *) {
            if (newSettingsMap["sharedCookiesEnabled"] != nil && settings?.sharedCookiesEnabled != newSettings.sharedCookiesEnabled && newSettings.sharedCookiesEnabled) {
                if(!newSettings.incognito && !newSettings.cacheEnabled) {
                    configuration.websiteDataStore = WKWebsiteDataStore.nonPersistent()
                }
                for cookie in HTTPCookieStorage.shared.cookies ?? [] {
                    configuration.websiteDataStore.httpCookieStore.setCookie(cookie, completionHandler: nil)
                }
            }
            if newSettingsMap["accessibilityIgnoresInvertColors"] != nil && settings?.accessibilityIgnoresInvertColors != newSettings.accessibilityIgnoresInvertColors {
                accessibilityIgnoresInvertColors = newSettings.accessibilityIgnoresInvertColors
            }
            if newSettingsMap["contentInsetAdjustmentBehavior"] != nil && settings?.contentInsetAdjustmentBehavior != newSettings.contentInsetAdjustmentBehavior {
                scrollView.contentInsetAdjustmentBehavior =
                    UIScrollView.ContentInsetAdjustmentBehavior.init(rawValue: newSettings.contentInsetAdjustmentBehavior)!
            }
        }
        
        if newSettingsMap["enableViewportScale"] != nil && settings?.enableViewportScale != newSettings.enableViewportScale {
            if !newSettings.enableViewportScale {
                if configuration.userContentController.userScripts.contains(ENABLE_VIEWPORT_SCALE_JS_PLUGIN_SCRIPT) {
                    configuration.userContentController.removePluginScript(ENABLE_VIEWPORT_SCALE_JS_PLUGIN_SCRIPT)
                    evaluateJavaScript(NOT_ENABLE_VIEWPORT_SCALE_JS_SOURCE)
                }
            } else {
                evaluateJavaScript(ENABLE_VIEWPORT_SCALE_JS_SOURCE)
                configuration.userContentController.addUserScript(ENABLE_VIEWPORT_SCALE_JS_PLUGIN_SCRIPT)
            }
        }
        
        if newSettingsMap["supportZoom"] != nil && settings?.supportZoom != newSettings.supportZoom {
            if newSettings.supportZoom {
                if configuration.userContentController.userScripts.contains(NOT_SUPPORT_ZOOM_JS_PLUGIN_SCRIPT) {
                    configuration.userContentController.removePluginScript(NOT_SUPPORT_ZOOM_JS_PLUGIN_SCRIPT)
                    evaluateJavaScript(SUPPORT_ZOOM_JS_SOURCE)
                }
            } else {
                evaluateJavaScript(NOT_SUPPORT_ZOOM_JS_SOURCE)
                configuration.userContentController.addUserScript(NOT_SUPPORT_ZOOM_JS_PLUGIN_SCRIPT)
            }
        }
        
        if newSettingsMap["useOnLoadResource"] != nil && settings?.useOnLoadResource != newSettings.useOnLoadResource {
            if let applePayAPIEnabled = settings?.applePayAPIEnabled, !applePayAPIEnabled {
                enablePluginScriptAtRuntime(flagVariable: FLAG_VARIABLE_FOR_ON_LOAD_RESOURCE_JS_SOURCE,
                                            enable: newSettings.useOnLoadResource,
                                            pluginScript: ON_LOAD_RESOURCE_JS_PLUGIN_SCRIPT)
            } else {
                newSettings.useOnLoadResource = false
            }
        }
        
        if newSettingsMap["useShouldInterceptAjaxRequest"] != nil && settings?.useShouldInterceptAjaxRequest != newSettings.useShouldInterceptAjaxRequest {
            if let applePayAPIEnabled = settings?.applePayAPIEnabled, !applePayAPIEnabled {
                enablePluginScriptAtRuntime(flagVariable: FLAG_VARIABLE_FOR_SHOULD_INTERCEPT_AJAX_REQUEST_JS_SOURCE,
                                            enable: newSettings.useShouldInterceptAjaxRequest,
                                            pluginScript: INTERCEPT_AJAX_REQUEST_JS_PLUGIN_SCRIPT)
            } else {
                newSettings.useShouldInterceptAjaxRequest = false
            }
        }
        
        if newSettingsMap["interceptOnlyAsyncAjaxRequests"] != nil && settings?.interceptOnlyAsyncAjaxRequests != newSettings.interceptOnlyAsyncAjaxRequests {
            if let applePayAPIEnabled = settings?.applePayAPIEnabled, !applePayAPIEnabled,
               let interceptOnlyAsyncAjaxRequestsPluginScript = interceptOnlyAsyncAjaxRequestsPluginScript {
                enablePluginScriptAtRuntime(flagVariable: FLAG_VARIABLE_FOR_INTERCEPT_ONLY_ASYNC_AJAX_REQUESTS_JS_SOURCE,
                                            enable: newSettings.interceptOnlyAsyncAjaxRequests,
                                            pluginScript: interceptOnlyAsyncAjaxRequestsPluginScript)
            }
        }
        
        if newSettingsMap["useShouldInterceptFetchRequest"] != nil && settings?.useShouldInterceptFetchRequest != newSettings.useShouldInterceptFetchRequest {
            if let applePayAPIEnabled = settings?.applePayAPIEnabled, !applePayAPIEnabled {
                enablePluginScriptAtRuntime(flagVariable: FLAG_VARIABLE_FOR_SHOULD_INTERCEPT_FETCH_REQUEST_JS_SOURCE,
                                            enable: newSettings.useShouldInterceptFetchRequest,
                                            pluginScript: INTERCEPT_FETCH_REQUEST_JS_PLUGIN_SCRIPT)
            } else {
                newSettings.useShouldInterceptFetchRequest = false
            }
        }
        
        if newSettingsMap["mediaPlaybackRequiresUserGesture"] != nil && settings?.mediaPlaybackRequiresUserGesture != newSettings.mediaPlaybackRequiresUserGesture {
            if #available(iOS 10.0, *) {
                configuration.mediaTypesRequiringUserActionForPlayback = (newSettings.mediaPlaybackRequiresUserGesture) ? .all : []
            } else {
                // Fallback on earlier versions
                configuration.mediaPlaybackRequiresUserAction = newSettings.mediaPlaybackRequiresUserGesture
            }
        }
        
        if newSettingsMap["allowsInlineMediaPlayback"] != nil && settings?.allowsInlineMediaPlayback != newSettings.allowsInlineMediaPlayback {
            configuration.allowsInlineMediaPlayback = newSettings.allowsInlineMediaPlayback
        }
        
        if newSettingsMap["suppressesIncrementalRendering"] != nil && settings?.suppressesIncrementalRendering != newSettings.suppressesIncrementalRendering {
            configuration.suppressesIncrementalRendering = newSettings.suppressesIncrementalRendering
        }
        
        if newSettingsMap["allowsBackForwardNavigationGestures"] != nil && settings?.allowsBackForwardNavigationGestures != newSettings.allowsBackForwardNavigationGestures {
            allowsBackForwardNavigationGestures = newSettings.allowsBackForwardNavigationGestures
        }
        
        if newSettingsMap["javaScriptCanOpenWindowsAutomatically"] != nil && settings?.javaScriptCanOpenWindowsAutomatically != newSettings.javaScriptCanOpenWindowsAutomatically {
            configuration.preferences.javaScriptCanOpenWindowsAutomatically = newSettings.javaScriptCanOpenWindowsAutomatically
        }
        
        if newSettingsMap["minimumFontSize"] != nil && settings?.minimumFontSize != newSettings.minimumFontSize {
            configuration.preferences.minimumFontSize = CGFloat(newSettings.minimumFontSize)
        }
        
        if newSettingsMap["selectionGranularity"] != nil && settings?.selectionGranularity != newSettings.selectionGranularity {
            configuration.selectionGranularity = WKSelectionGranularity.init(rawValue: newSettings.selectionGranularity)!
        }
        
        if #available(iOS 10.0, *) {
            if newSettingsMap["ignoresViewportScaleLimits"] != nil && settings?.ignoresViewportScaleLimits != newSettings.ignoresViewportScaleLimits {
                configuration.ignoresViewportScaleLimits = newSettings.ignoresViewportScaleLimits
            }
            
            if newSettingsMap["dataDetectorTypes"] != nil && settings?.dataDetectorTypes != newSettings.dataDetectorTypes {
                var dataDetectorTypes = WKDataDetectorTypes.init(rawValue: 0)
                for type in newSettings.dataDetectorTypes {
                    let dataDetectorType = Util.getDataDetectorType(type: type)
                    dataDetectorTypes = WKDataDetectorTypes(rawValue: dataDetectorTypes.rawValue | dataDetectorType.rawValue)
                }
                configuration.dataDetectorTypes = dataDetectorTypes
            }
        }
        
        if #available(iOS 13.0, *) {
            if newSettingsMap["isFraudulentWebsiteWarningEnabled"] != nil && settings?.isFraudulentWebsiteWarningEnabled != newSettings.isFraudulentWebsiteWarningEnabled {
                configuration.preferences.isFraudulentWebsiteWarningEnabled = newSettings.isFraudulentWebsiteWarningEnabled
            }
            if newSettingsMap["preferredContentMode"] != nil && settings?.preferredContentMode != newSettings.preferredContentMode {
                configuration.defaultWebpagePreferences.preferredContentMode = WKWebpagePreferences.ContentMode(rawValue: newSettings.preferredContentMode)!
            }
            if newSettingsMap["automaticallyAdjustsScrollIndicatorInsets"] != nil && settings?.automaticallyAdjustsScrollIndicatorInsets != newSettings.automaticallyAdjustsScrollIndicatorInsets {
                scrollView.automaticallyAdjustsScrollIndicatorInsets = newSettings.automaticallyAdjustsScrollIndicatorInsets
            }
        }
        
        if newSettingsMap["disableVerticalScroll"] != nil && settings?.disableVerticalScroll != newSettings.disableVerticalScroll {
            scrollView.showsVerticalScrollIndicator = !newSettings.disableVerticalScroll
        }
        if newSettingsMap["disableHorizontalScroll"] != nil && settings?.disableHorizontalScroll != newSettings.disableHorizontalScroll {
            scrollView.showsHorizontalScrollIndicator = !newSettings.disableHorizontalScroll
        }
        
        if newSettingsMap["verticalScrollBarEnabled"] != nil && settings?.verticalScrollBarEnabled != newSettings.verticalScrollBarEnabled {
            scrollView.showsVerticalScrollIndicator = newSettings.verticalScrollBarEnabled
        }
        if newSettingsMap["horizontalScrollBarEnabled"] != nil && settings?.horizontalScrollBarEnabled != newSettings.horizontalScrollBarEnabled {
            scrollView.showsHorizontalScrollIndicator = newSettings.horizontalScrollBarEnabled
        }
        
        if newSettingsMap["isDirectionalLockEnabled"] != nil && settings?.isDirectionalLockEnabled != newSettings.isDirectionalLockEnabled {
            scrollView.isDirectionalLockEnabled = newSettings.isDirectionalLockEnabled
        }
        
        if newSettingsMap["decelerationRate"] != nil && settings?.decelerationRate != newSettings.decelerationRate {
            scrollView.decelerationRate = Util.getDecelerationRate(type: newSettings.decelerationRate)
        }
        if newSettingsMap["alwaysBounceVertical"] != nil && settings?.alwaysBounceVertical != newSettings.alwaysBounceVertical {
            scrollView.alwaysBounceVertical = newSettings.alwaysBounceVertical
        }
        if newSettingsMap["alwaysBounceHorizontal"] != nil && settings?.alwaysBounceHorizontal != newSettings.alwaysBounceHorizontal {
            scrollView.alwaysBounceHorizontal = newSettings.alwaysBounceHorizontal
        }
        if newSettingsMap["scrollsToTop"] != nil && settings?.scrollsToTop != newSettings.scrollsToTop {
            scrollView.scrollsToTop = newSettings.scrollsToTop
        }
        if newSettingsMap["isPagingEnabled"] != nil && settings?.isPagingEnabled != newSettings.isPagingEnabled {
            scrollView.scrollsToTop = newSettings.isPagingEnabled
        }
        if newSettingsMap["maximumZoomScale"] != nil && settings?.maximumZoomScale != newSettings.maximumZoomScale {
            scrollView.maximumZoomScale = CGFloat(newSettings.maximumZoomScale)
        }
        if newSettingsMap["minimumZoomScale"] != nil && settings?.minimumZoomScale != newSettings.minimumZoomScale {
            scrollView.minimumZoomScale = CGFloat(newSettings.minimumZoomScale)
        }
        
        if #available(iOS 9.0, *) {
            if newSettingsMap["allowsLinkPreview"] != nil && settings?.allowsLinkPreview != newSettings.allowsLinkPreview {
                allowsLinkPreview = newSettings.allowsLinkPreview
            }
            if newSettingsMap["allowsAirPlayForMediaPlayback"] != nil && settings?.allowsAirPlayForMediaPlayback != newSettings.allowsAirPlayForMediaPlayback {
                configuration.allowsAirPlayForMediaPlayback = newSettings.allowsAirPlayForMediaPlayback
            }
            if newSettingsMap["allowsPictureInPictureMediaPlayback"] != nil && settings?.allowsPictureInPictureMediaPlayback != newSettings.allowsPictureInPictureMediaPlayback {
                configuration.allowsPictureInPictureMediaPlayback = newSettings.allowsPictureInPictureMediaPlayback
            }
            if newSettingsMap["applicationNameForUserAgent"] != nil && settings?.applicationNameForUserAgent != newSettings.applicationNameForUserAgent && newSettings.applicationNameForUserAgent != "" {
                configuration.applicationNameForUserAgent = newSettings.applicationNameForUserAgent
            }
            if newSettingsMap["userAgent"] != nil && settings?.userAgent != newSettings.userAgent && newSettings.userAgent != "" {
                customUserAgent = newSettings.userAgent
            }
        }
        
        if newSettingsMap["allowUniversalAccessFromFileURLs"] != nil && settings?.allowUniversalAccessFromFileURLs != newSettings.allowUniversalAccessFromFileURLs {
            configuration.setValue(newSettings.allowUniversalAccessFromFileURLs, forKey: "allowUniversalAccessFromFileURLs")
        }
        
        if newSettingsMap["allowFileAccessFromFileURLs"] != nil && settings?.allowFileAccessFromFileURLs != newSettings.allowFileAccessFromFileURLs {
            configuration.preferences.setValue(newSettings.allowFileAccessFromFileURLs, forKey: "allowFileAccessFromFileURLs")
        }
        
        if newSettingsMap["clearCache"] != nil && newSettings.clearCache {
            clearCache()
        }
        
        if newSettingsMap["javaScriptEnabled"] != nil && settings?.javaScriptEnabled != newSettings.javaScriptEnabled {
            configuration.preferences.javaScriptEnabled = newSettings.javaScriptEnabled
        }
        
        if #available(iOS 14.0, *) {
            if settings?.mediaType != newSettings.mediaType {
                mediaType = newSettings.mediaType
            }
            
            if newSettingsMap["pageZoom"] != nil && settings?.pageZoom != newSettings.pageZoom {
                pageZoom = CGFloat(newSettings.pageZoom)
            }
            
            if newSettingsMap["limitsNavigationsToAppBoundDomains"] != nil && settings?.limitsNavigationsToAppBoundDomains != newSettings.limitsNavigationsToAppBoundDomains {
                configuration.limitsNavigationsToAppBoundDomains = newSettings.limitsNavigationsToAppBoundDomains
            }
            
            if newSettingsMap["javaScriptEnabled"] != nil && settings?.javaScriptEnabled != newSettings.javaScriptEnabled {
                configuration.defaultWebpagePreferences.allowsContentJavaScript = newSettings.javaScriptEnabled
            }
        }
        
        if #available(iOS 11.0, *), newSettingsMap["contentBlockers"] != nil {
            configuration.userContentController.removeAllContentRuleLists()
            let contentBlockers = newSettings.contentBlockers
            if contentBlockers.count > 0 {
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
                            self.configuration.userContentController.add(contentRuleList!)
                    }
                } catch {
                    print(error.localizedDescription)
                }
            }
        }
        
        if #available(iOS 15.0, *) {
            if newSettingsMap["upgradeKnownHostsToHTTPS"] != nil && settings?.upgradeKnownHostsToHTTPS != newSettings.upgradeKnownHostsToHTTPS {
                configuration.upgradeKnownHostsToHTTPS = newSettings.upgradeKnownHostsToHTTPS
            }
            if newSettingsMap["isTextInteractionEnabled"] != nil && settings?.isTextInteractionEnabled != newSettings.isTextInteractionEnabled {
                configuration.preferences.isTextInteractionEnabled = newSettings.isTextInteractionEnabled
            }
            if newSettingsMap["underPageBackgroundColor"] != nil, settings?.underPageBackgroundColor != newSettings.underPageBackgroundColor,
               let underPageBackgroundColor = newSettings.underPageBackgroundColor, !underPageBackgroundColor.isEmpty {
                self.underPageBackgroundColor = UIColor(hexString: underPageBackgroundColor)
            }
        }
        if #available(iOS 15.4, *) {
            if newSettingsMap["isSiteSpecificQuirksModeEnabled"] != nil, settings?.isSiteSpecificQuirksModeEnabled != newSettings.isSiteSpecificQuirksModeEnabled {
                configuration.preferences.isSiteSpecificQuirksModeEnabled = newSettings.isSiteSpecificQuirksModeEnabled
            }
        }
        if #available(iOS 15.5, *) {
            if ((newSettingsMap["minimumViewportInset"] != nil && settings?.minimumViewportInset != newSettings.minimumViewportInset) ||
               (newSettingsMap["maximumViewportInset"] != nil && settings?.maximumViewportInset != newSettings.maximumViewportInset)),
               let minViewportInset = newSettings.minimumViewportInset, let maxViewportInset = newSettings.maximumViewportInset {
                setMinimumViewportInset(minViewportInset, maximumViewportInset: maxViewportInset)
            }
        }
        if #available(iOS 16.0, *) {
            if newSettingsMap["isFindInteractionEnabled"] != nil, settings?.isFindInteractionEnabled != newSettings.isFindInteractionEnabled {
                isFindInteractionEnabled = newSettings.isFindInteractionEnabled
            }
        }
        if #available(iOS 16.4, *) {
            if newSettingsMap["isInspectable"] != nil, settings?.isInspectable != newSettings.isInspectable {
                isInspectable = newSettings.isInspectable
            }
            if newSettingsMap["shouldPrintBackgrounds"] != nil, settings?.shouldPrintBackgrounds != newSettings.shouldPrintBackgrounds {
                configuration.preferences.shouldPrintBackgrounds = newSettings.shouldPrintBackgrounds
            }
        }
        
        scrollView.isScrollEnabled = !(newSettings.disableVerticalScroll && newSettings.disableHorizontalScroll)
        
        self.settings = newSettings
    }
    
    func getSettings() -> [String: Any?]? {
        if (self.settings == nil) {
            return nil
        }
        return self.settings!.getRealSettings(obj: self)
    }
    
    public func enablePluginScriptAtRuntime(flagVariable: String, enable: Bool, pluginScript: PluginScript) {
        evaluateJavascript(source: flagVariable) { (alreadyLoaded) in
            if let alreadyLoaded = alreadyLoaded as? Bool, alreadyLoaded {
                let enableSource = "\(flagVariable) = \(enable);"
                if #available(iOS 14.0, *), pluginScript.requiredInAllContentWorlds {
                    for contentWorld in self.configuration.userContentController.contentWorlds {
                        self.evaluateJavaScript(enableSource, frame: nil, contentWorld: contentWorld, completionHandler: nil)
                    }
                } else {
                    self.evaluateJavaScript(enableSource, completionHandler: nil)
                }
                if !enable {
                    self.configuration.userContentController.removePluginScripts(with: pluginScript.groupName!)
                }
            }
            else if enable {
                if #available(iOS 14.0, *), pluginScript.requiredInAllContentWorlds {
                    for contentWorld in self.configuration.userContentController.contentWorlds {
                        self.evaluateJavaScript(pluginScript.source, frame: nil, contentWorld: contentWorld, completionHandler: nil)
                        self.configuration.userContentController.addPluginScript(pluginScript)
                    }
                } else {
                    self.evaluateJavaScript(pluginScript.source, completionHandler: nil)
                    self.configuration.userContentController.addPluginScript(pluginScript)
                }
                self.configuration.userContentController.sync(scriptMessageHandler: self)
            }
        }
    }
    
    @available(*, deprecated, message: "Use InAppWebViewManager.clearAllCache instead.")
    public func clearCache() {
        if #available(iOS 9.0, *) {
            let date = NSDate(timeIntervalSince1970: 0)
            WKWebsiteDataStore.default().removeData(ofTypes: WKWebsiteDataStore.allWebsiteDataTypes(), modifiedSince: date as Date, completionHandler:{ })
        } else {
            var libraryPath = NSSearchPathForDirectoriesInDomains(FileManager.SearchPathDirectory.libraryDirectory, FileManager.SearchPathDomainMask.userDomainMask, false).first!
            libraryPath += "/Cookies"
            
            do {
                try FileManager.default.removeItem(atPath: libraryPath)
            } catch {
                print("can't clear cache")
            }
            URLCache.shared.removeAllCachedResponses()
        }
    }
    
    public func injectDeferredObject(source: String, withWrapper jsWrapper: String?, completionHandler: ((Any?) -> Void)? = nil) {
        var jsToInject = source
        if let wrapper = jsWrapper {
            let jsonData: Data? = try? JSONSerialization.data(withJSONObject: [source], options: [])
            let sourceArrayString = String(data: jsonData!, encoding: .utf8)
            let sourceString: String? = (sourceArrayString! as NSString).substring(with: NSRange(location: 1, length: (sourceArrayString?.count ?? 0) - 2))
            jsToInject = String(format: wrapper, sourceString!)
        }
        
        evaluateJavaScript(jsToInject) { (value, error) in
            guard let completionHandler = completionHandler else {
                return
            }
            
            if let error = error {
                let userInfo = (error as NSError).userInfo
                let errorMessage = userInfo["WKJavaScriptExceptionMessage"] ??
                                   userInfo["NSLocalizedDescription"] as? String ??
                                   error.localizedDescription
                self.channelDelegate?.onConsoleMessage(message: String(describing: errorMessage), messageLevel: 3)
            }
            
            if value == nil {
                completionHandler(nil)
                return
            }
            
            completionHandler(value)
        }
    }
    
    @available(iOS 14.0, *)
    public func injectDeferredObject(source: String, contentWorld: WKContentWorld, withWrapper jsWrapper: String?, completionHandler: ((Any?) -> Void)? = nil) {
        var jsToInject = source
        if let wrapper = jsWrapper {
            let jsonData: Data? = try? JSONSerialization.data(withJSONObject: [source], options: [])
            let sourceArrayString = String(data: jsonData!, encoding: .utf8)
            let sourceString: String? = (sourceArrayString! as NSString).substring(with: NSRange(location: 1, length: (sourceArrayString?.count ?? 0) - 2))
            jsToInject = String(format: wrapper, sourceString!)
        }
        
        jsToInject = configuration.userContentController.generateCodeForScriptEvaluation(scriptMessageHandler: self, source: jsToInject, contentWorld: contentWorld)
        
        evaluateJavaScript(jsToInject, frame: nil, contentWorld: contentWorld) { (evalResult) in
            guard let completionHandler = completionHandler else {
                return
            }
            
            switch (evalResult) {
            case .success(let value):
                completionHandler(value)
                return
            case .failure(let error):
                let userInfo = (error as NSError).userInfo
                let errorMessage = userInfo["WKJavaScriptExceptionMessage"] ??
                                   userInfo["NSLocalizedDescription"] as? String ??
                                   error.localizedDescription
                self.channelDelegate?.onConsoleMessage(message: String(describing: errorMessage), messageLevel: 3)
                break
            }
            
            completionHandler(nil)
        }
    }
    
#if compiler(>=6.0)
    public override func evaluateJavaScript(_ javaScriptString: String, completionHandler: (@MainActor @Sendable (Any?, (any Error)?) -> Void)? = nil) {
        if let applePayAPIEnabled = settings?.applePayAPIEnabled, applePayAPIEnabled {
            if let completionHandler = completionHandler {
                completionHandler(nil, nil)
            }
            return
        }
        super.evaluateJavaScript(javaScriptString, completionHandler: completionHandler)
    }
#else
    public override func evaluateJavaScript(_ javaScriptString: String, completionHandler: ((Any?, Error?) -> Void)? = nil) {
        if let applePayAPIEnabled = settings?.applePayAPIEnabled, applePayAPIEnabled {
            if let completionHandler = completionHandler {
                completionHandler(nil, nil)
            }
            return
        }
        super.evaluateJavaScript(javaScriptString, completionHandler: completionHandler)
    }
#endif
    
    @available(iOS 14.0, *)
    public func evaluateJavaScript(_ javaScript: String, frame: WKFrameInfo? = nil, contentWorld: WKContentWorld, completionHandler: ((Result<Any, Error>) -> Void)? = nil) {
        if let applePayAPIEnabled = settings?.applePayAPIEnabled, applePayAPIEnabled {
            return
        }
        super.evaluateJavaScript(javaScript, in: frame, in: contentWorld, completionHandler: completionHandler)
    }
    
    public func evaluateJavascript(source: String, completionHandler: ((Any?) -> Void)? = nil) {
        injectDeferredObject(source: source, withWrapper: nil, completionHandler: completionHandler)
    }
    
    @available(iOS 14.0, *)
    public func evaluateJavascript(source: String, contentWorld: WKContentWorld, completionHandler: ((Any?) -> Void)? = nil) {
        injectDeferredObject(source: source, contentWorld: contentWorld, withWrapper: nil, completionHandler: completionHandler)
    }
    
    @available(iOS 14.0, *)
    public func callAsyncJavaScript(_ functionBody: String, arguments: [String : Any] = [:], frame: WKFrameInfo? = nil, contentWorld: WKContentWorld, completionHandler: ((Result<Any, Error>) -> Void)? = nil) {
        if let applePayAPIEnabled = settings?.applePayAPIEnabled, applePayAPIEnabled {
            return
        }
        super.callAsyncJavaScript(functionBody, arguments: arguments, in: frame, in: contentWorld, completionHandler: completionHandler)
    }
    
    @available(iOS 14.0, *)
    public func callAsyncJavaScript(functionBody: String, arguments: [String:Any], contentWorld: WKContentWorld, completionHandler: ((Any?) -> Void)? = nil) {
        let jsToInject = configuration.userContentController.generateCodeForScriptEvaluation(scriptMessageHandler: self, source: functionBody, contentWorld: contentWorld)
        
        callAsyncJavaScript(jsToInject, arguments: arguments, frame: nil, contentWorld: contentWorld) { (evalResult) in
            guard let completionHandler = completionHandler else {
                return
            }
            
            var body: [String: Any?] = [
                "value": nil,
                "error": nil
            ]
            
            switch (evalResult) {
            case .success(let value):
                body["value"] = value
                break
            case .failure(let error):
                let userInfo = (error as NSError).userInfo
                body["error"] = userInfo["WKJavaScriptExceptionMessage"] ??
                                userInfo["NSLocalizedDescription"] as? String ??
                                error.localizedDescription
                self.channelDelegate?.onConsoleMessage(message: String(describing: body["error"]), messageLevel: 3)
                break
            }
            
            completionHandler(body)
        }
    }
    
    @available(iOS 10.3, *)
    public func callAsyncJavaScript(functionBody: String, arguments: [String:Any], completionHandler: ((Any?) -> Void)? = nil) {
        if let applePayAPIEnabled = settings?.applePayAPIEnabled, applePayAPIEnabled {
            completionHandler?(nil)
        }
        
        var jsToInject = functionBody
        
        let resultUuid = NSUUID().uuidString
        if let completionHandler = completionHandler {
            callAsyncJavaScriptBelowIOS14Results[resultUuid] = completionHandler
        }
        
        var functionArgumentNamesList: [String] = []
        var functionArgumentValuesList: [String] = []
        let keys = arguments.keys
        keys.forEach { (key) in
            functionArgumentNamesList.append(key)
            functionArgumentValuesList.append("obj.\(key)")
        }
        
        let functionArgumentNames = functionArgumentNamesList.joined(separator: ", ")
        let functionArgumentValues = functionArgumentValuesList.joined(separator: ", ")
        
        jsToInject = CALL_ASYNC_JAVASCRIPT_BELOW_IOS_14_WRAPPER_JS
            .replacingOccurrences(of: PluginScriptsUtil.VAR_FUNCTION_ARGUMENT_NAMES, with: functionArgumentNames)
            .replacingOccurrences(of: PluginScriptsUtil.VAR_FUNCTION_ARGUMENT_VALUES, with: functionArgumentValues)
            .replacingOccurrences(of: PluginScriptsUtil.VAR_FUNCTION_ARGUMENTS_OBJ, with: Util.JSONStringify(value: arguments))
            .replacingOccurrences(of: PluginScriptsUtil.VAR_FUNCTION_BODY, with: jsToInject)
            .replacingOccurrences(of: PluginScriptsUtil.VAR_RESULT_UUID, with: resultUuid)
        
        evaluateJavaScript(jsToInject) { (value, error) in
            if let error = error {
                let userInfo = (error as NSError).userInfo
                let errorMessage = userInfo["WKJavaScriptExceptionMessage"] ??
                                   userInfo["NSLocalizedDescription"] as? String ??
                                   error.localizedDescription
                self.channelDelegate?.onConsoleMessage(message: String(describing: errorMessage), messageLevel: 3)
                completionHandler?(nil)
                self.callAsyncJavaScriptBelowIOS14Results.removeValue(forKey: resultUuid)
            }
        }
    }
    
    public func injectJavascriptFileFromUrl(urlFile: String, scriptHtmlTagAttributes: [String:Any?]?) {
        var scriptAttributes = ""
        if let scriptHtmlTagAttributes = scriptHtmlTagAttributes {
            if let typeAttr = scriptHtmlTagAttributes["type"] as? String {
                scriptAttributes += " script.type = '\(typeAttr.replacingOccurrences(of: "\'", with: "\\'"))'; "
            }
            if let idAttr = scriptHtmlTagAttributes["id"] as? String {
                let scriptIdEscaped = idAttr.replacingOccurrences(of: "\'", with: "\\'")
                scriptAttributes += " script.id = '\(scriptIdEscaped)'; "
                scriptAttributes += """
                script.onload = function() {
                    if (window.\(JAVASCRIPT_BRIDGE_NAME) != null) {
                        window.\(JAVASCRIPT_BRIDGE_NAME).callHandler('onInjectedScriptLoaded', '\(scriptIdEscaped)');
                    }
                };
                """
                scriptAttributes += """
                script.onerror = function() {
                    if (window.\(JAVASCRIPT_BRIDGE_NAME) != null) {
                        window.\(JAVASCRIPT_BRIDGE_NAME).callHandler('onInjectedScriptError', '\(scriptIdEscaped)');
                    }
                };
                """
            }
            if let asyncAttr = scriptHtmlTagAttributes["async"] as? Bool, asyncAttr {
                scriptAttributes += " script.async = true; "
            }
            if let deferAttr = scriptHtmlTagAttributes["defer"] as? Bool, deferAttr {
                scriptAttributes += " script.defer = true; "
            }
            if let crossOriginAttr = scriptHtmlTagAttributes["crossOrigin"] as? String {
                scriptAttributes += " script.crossOrigin = '\(crossOriginAttr.replacingOccurrences(of: "\'", with: "\\'"))'; "
            }
            if let integrityAttr = scriptHtmlTagAttributes["integrity"] as? String {
                scriptAttributes += " script.integrity = '\(integrityAttr.replacingOccurrences(of: "\'", with: "\\'"))'; "
            }
            if let noModuleAttr = scriptHtmlTagAttributes["noModule"] as? Bool, noModuleAttr {
                scriptAttributes += " script.noModule = true; "
            }
            if let nonceAttr = scriptHtmlTagAttributes["nonce"] as? String {
                scriptAttributes += " script.nonce = '\(nonceAttr.replacingOccurrences(of: "\'", with: "\\'"))'; "
            }
            if let referrerPolicyAttr = scriptHtmlTagAttributes["referrerPolicy"] as? String {
                scriptAttributes += " script.referrerPolicy = '\(referrerPolicyAttr.replacingOccurrences(of: "\'", with: "\\'"))'; "
            }
        }
        let jsWrapper = "(function(d) { var script = d.createElement('script'); \(scriptAttributes) script.src = %@; d.body.appendChild(script); })(document);"
        injectDeferredObject(source: urlFile, withWrapper: jsWrapper, completionHandler: nil)
    }
    
    public func injectCSSCode(source: String) {
        let jsWrapper = "(function(d) { var style = d.createElement('style'); style.innerHTML = %@; d.head.appendChild(style); })(document);"
        injectDeferredObject(source: source, withWrapper: jsWrapper, completionHandler: nil)
    }
    
    public func injectCSSFileFromUrl(urlFile: String, cssLinkHtmlTagAttributes: [String:Any?]?) {
        var cssLinkAttributes = ""
        var alternateStylesheet = ""
        if let cssLinkHtmlTagAttributes = cssLinkHtmlTagAttributes {
            if let idAttr = cssLinkHtmlTagAttributes["id"] as? String {
                cssLinkAttributes += " link.id = '\(idAttr.replacingOccurrences(of: "\'", with: "\\'"))'; "
            }
            if let mediaAttr = cssLinkHtmlTagAttributes["media"] as? String {
                cssLinkAttributes += " link.media = '\(mediaAttr.replacingOccurrences(of: "\'", with: "\\'"))'; "
            }
            if let crossOriginAttr = cssLinkHtmlTagAttributes["crossOrigin"] as? String {
                cssLinkAttributes += " link.crossOrigin = '\(crossOriginAttr.replacingOccurrences(of: "\'", with: "\\'"))'; "
            }
            if let integrityAttr = cssLinkHtmlTagAttributes["integrity"] as? String {
                cssLinkAttributes += " link.integrity = '\(integrityAttr.replacingOccurrences(of: "\'", with: "\\'"))'; "
            }
            if let referrerPolicyAttr = cssLinkHtmlTagAttributes["referrerPolicy"] as? String {
                cssLinkAttributes += " link.referrerPolicy = '\(referrerPolicyAttr.replacingOccurrences(of: "\'", with: "\\'"))'; "
            }
            if let disabledAttr = cssLinkHtmlTagAttributes["disabled"] as? Bool, disabledAttr {
                cssLinkAttributes += " link.disabled = true; "
            }
            if let alternateAttr = cssLinkHtmlTagAttributes["alternate"] as? Bool, alternateAttr {
                alternateStylesheet = "alternate "
            }
            if let titleAttr = cssLinkHtmlTagAttributes["title"] as? String {
                cssLinkAttributes += " link.title = '\(titleAttr.replacingOccurrences(of: "\'", with: "\\'"))'; "
            }
        }
        let jsWrapper = "(function(d) { var link = d.createElement('link'); link.rel='\(alternateStylesheet)stylesheet', link.type='text/css'; \(cssLinkAttributes) link.href = %@; d.head.appendChild(link); })(document);"
        injectDeferredObject(source: urlFile, withWrapper: jsWrapper, completionHandler: nil)
    }
    
    public func getCopyBackForwardList() -> [String: Any] {
        let currentList = backForwardList
        let currentIndex = currentList.backList.count
        var completeList = currentList.backList
        if currentList.currentItem != nil {
            completeList.append(currentList.currentItem!)
        }
        completeList.append(contentsOf: currentList.forwardList)
        
        var history: [[String: String]] = []
        
        for historyItem in completeList {
            var historyItemMap: [String: String] = [:]
            historyItemMap["originalUrl"] = historyItem.initialURL.absoluteString
            historyItemMap["title"] = historyItem.title
            historyItemMap["url"] = historyItem.url.absoluteString
            history.append(historyItemMap)
        }
        
        var result: [String: Any] = [:]
        result["list"] = history
        result["currentIndex"] = currentIndex
        
        return result;
    }

    @available(iOS 15.0, *)
    public func webView(_ webView: WKWebView,
                        requestMediaCapturePermissionFor origin: WKSecurityOrigin,
                        initiatedByFrame frame: WKFrameInfo,
                        type: WKMediaCaptureType,
                        decisionHandler: @escaping (WKPermissionDecision) -> Void) {
        let origin = "\(origin.protocol)://\(origin.host)\(origin.port != 0 ? ":" + String(origin.port) : "")"
        let permissionRequest = PermissionRequest(origin: origin, resources: [type.rawValue], frame: frame)
        
        var decisionHandlerCalled = false
        let callback = WebViewChannelDelegate.PermissionRequestCallback()
        callback.nonNullSuccess = { (response: PermissionResponse) in
            if let action = response.action {
                decisionHandlerCalled = true
                switch action {
                    case 1:
                        decisionHandler(.grant)
                        break
                    case 2:
                        decisionHandler(.prompt)
                        break
                    default:
                        decisionHandler(.deny)
                }
                return false
            }
            return true
        }
        callback.defaultBehaviour = { (response: PermissionResponse?) in
            if !decisionHandlerCalled {
                decisionHandlerCalled = true
                decisionHandler(.deny)
            }
        }
        callback.error = { [weak callback] (code: String, message: String?, details: Any?) in
            print(code + ", " + (message ?? ""))
            callback?.defaultBehaviour(nil)
        }
        
        if let channelDelegate = channelDelegate {
            channelDelegate.onPermissionRequest(request: permissionRequest, callback: callback)
        } else {
            callback.defaultBehaviour(nil)
        }
    }
    
    @available(iOS 15.0, *)
    public func webView(_ webView: WKWebView,
                        requestDeviceOrientationAndMotionPermissionFor origin: WKSecurityOrigin,
                        initiatedByFrame frame: WKFrameInfo,
                        decisionHandler: @escaping (WKPermissionDecision) -> Void) {
        let origin = "\(origin.protocol)://\(origin.host)\(origin.port != 0 ? ":" + String(origin.port) : "")"
        let permissionRequest = PermissionRequest(origin: origin, resources: ["deviceOrientationAndMotion"], frame: frame)
        
        var decisionHandlerCalled = false
        let callback = WebViewChannelDelegate.PermissionRequestCallback()
        callback.nonNullSuccess = { (response: PermissionResponse) in
            if let action = response.action {
                decisionHandlerCalled = true
                switch action {
                    case 1:
                        decisionHandler(.grant)
                        break
                    case 2:
                        decisionHandler(.prompt)
                        break
                    default:
                        decisionHandler(.deny)
                }
                return false
            }
            return true
        }
        callback.defaultBehaviour = { (response: PermissionResponse?) in
            if !decisionHandlerCalled {
                decisionHandlerCalled = true
                decisionHandler(.deny)
            }
        }
        callback.error = { [weak callback] (code: String, message: String?, details: Any?) in
            print(code + ", " + (message ?? ""))
            callback?.defaultBehaviour(nil)
        }
        
        if let channelDelegate = channelDelegate {
            channelDelegate.onPermissionRequest(request: permissionRequest, callback: callback)
        } else {
            callback.defaultBehaviour(nil)
        }
    }
    
    @available(iOS 13.0, *)
    public func webView(_ webView: WKWebView,
                 decidePolicyFor navigationAction: WKNavigationAction,
                 preferences: WKWebpagePreferences,
                 decisionHandler: @escaping (WKNavigationActionPolicy, WKWebpagePreferences) -> Void) {
        self.webView(webView, decidePolicyFor: navigationAction, decisionHandler: {(navigationActionPolicy) -> Void in
            decisionHandler(navigationActionPolicy, preferences)
        })
    }
    
    @available(iOS 14.5, *)
    public func download(_ download: WKDownload, decideDestinationUsing response: URLResponse, suggestedFilename: String, completionHandler: @escaping (URL?) -> Void) {
        if let url = response.url, let useOnDownloadStart = settings?.useOnDownloadStart, useOnDownloadStart {
            let downloadStartRequest = DownloadStartRequest(url: url.absoluteString,
                                                            userAgent: nil,
                                                            contentDisposition: nil,
                                                            mimeType: response.mimeType,
                                                            contentLength: response.expectedContentLength,
                                                            suggestedFilename: suggestedFilename,
                                                            textEncodingName: response.textEncodingName)
            channelDelegate?.onDownloadStartRequest(request: downloadStartRequest)
        }
        download.delegate = nil
        // cancel the download
        completionHandler(nil)
    }
    
    @available(iOS 14.5, *)
    public func webView(_ webView: WKWebView, navigationResponse: WKNavigationResponse, didBecome download: WKDownload) {
        let response = navigationResponse.response
        if let url = response.url, let useOnDownloadStart = settings?.useOnDownloadStart, useOnDownloadStart {
            let downloadStartRequest = DownloadStartRequest(url: url.absoluteString,
                                                            userAgent: nil,
                                                            contentDisposition: nil,
                                                            mimeType: response.mimeType,
                                                            contentLength: response.expectedContentLength,
                                                            suggestedFilename: response.suggestedFilename,
                                                            textEncodingName: response.textEncodingName)
            channelDelegate?.onDownloadStartRequest(request: downloadStartRequest)
        }
        download.delegate = nil
    }
    
    public func webView(_ webView: WKWebView,
                 decidePolicyFor navigationAction: WKNavigationAction,
                 decisionHandler: @escaping (WKNavigationActionPolicy) -> Void) {
        var decisionHandlerCalled = false
        let callback = WebViewChannelDelegate.ShouldOverrideUrlLoadingCallback()
        callback.nonNullSuccess = { (response: WKNavigationActionPolicy) in
            decisionHandlerCalled = true
            decisionHandler(response)
            return false
        }
        callback.defaultBehaviour = { (response: WKNavigationActionPolicy?) in
            if !decisionHandlerCalled {
                decisionHandlerCalled = true
                decisionHandler(.allow)
            }
        }
        callback.error = { [weak callback] (code: String, message: String?, details: Any?) in
            print(code + ", " + (message ?? ""))
            callback?.defaultBehaviour(nil)
        }

        let runCallback = {
            if let useShouldOverrideUrlLoading = self.settings?.useShouldOverrideUrlLoading, useShouldOverrideUrlLoading, let channelDelegate = self.channelDelegate {
                channelDelegate.shouldOverrideUrlLoading(navigationAction: navigationAction, callback: callback)
            } else {
                callback.defaultBehaviour(nil)
            }
        }
        
        if windowId != nil, !windowCreated {
            windowBeforeCreatedCallbacks.append(runCallback)
        } else {
            runCallback()
        }
    }
    
    public func webView(_ webView: WKWebView,
                 decidePolicyFor navigationResponse: WKNavigationResponse,
                 decisionHandler: @escaping (WKNavigationResponsePolicy) -> Void) {
        if let response = navigationResponse.response as? HTTPURLResponse, response.statusCode >= 400 {
            let request = WebResourceRequest.init(fromWKNavigationResponse: navigationResponse)
            let errorResponse = WebResourceResponse.init(fromWKNavigationResponse: navigationResponse)
            channelDelegate?.onReceivedHttpError(request: request, errorResponse: errorResponse)
        }
        
        let useOnNavigationResponse = settings?.useOnNavigationResponse
        
        if useOnNavigationResponse != nil, useOnNavigationResponse! {
            var decisionHandlerCalled = false
            let callback = WebViewChannelDelegate.NavigationResponseCallback()
            callback.nonNullSuccess = { (response: WKNavigationResponsePolicy) in
                decisionHandlerCalled = true
                decisionHandler(response)
                return false
            }
            callback.defaultBehaviour = { (response: WKNavigationResponsePolicy?) in
                if !decisionHandlerCalled {
                    decisionHandlerCalled = true
                    decisionHandler(.allow)
                }
            }
            callback.error = { [weak callback] (code: String, message: String?, details: Any?) in
                print(code + ", " + (message ?? ""))
                callback?.defaultBehaviour(nil)
            }
            
            if let channelDelegate = channelDelegate {
                channelDelegate.onNavigationResponse(navigationResponse: navigationResponse, callback: callback)
            } else {
                callback.defaultBehaviour(nil)
            }
        }
        
        if let useOnDownloadStart = settings?.useOnDownloadStart, useOnDownloadStart {
            if #available(iOS 14.5, *), !navigationResponse.canShowMIMEType, useOnNavigationResponse == nil || !useOnNavigationResponse! {
                decisionHandler(.download)
                return
            } else {
                let mimeType = navigationResponse.response.mimeType
                if let url = navigationResponse.response.url, navigationResponse.isForMainFrame {
                    if url.scheme != "file", mimeType != nil, !mimeType!.starts(with: "text/") {
                        let downloadStartRequest = DownloadStartRequest(url: url.absoluteString,
                                                                        userAgent: nil,
                                                                        contentDisposition: nil,
                                                                        mimeType: mimeType,
                                                                        contentLength: navigationResponse.response.expectedContentLength,
                                                                        suggestedFilename: navigationResponse.response.suggestedFilename,
                                                                        textEncodingName: navigationResponse.response.textEncodingName)
                        channelDelegate?.onDownloadStartRequest(request: downloadStartRequest)
                        if useOnNavigationResponse == nil || !useOnNavigationResponse! {
                            decisionHandler(.cancel)
                        }
                        return
                    }
                }
            }
        }
        
        if useOnNavigationResponse == nil || !useOnNavigationResponse! {
            decisionHandler(.allow)
        }
    }
    
    public func webView(_ webView: WKWebView, didStartProvisionalNavigation navigation: WKNavigation!) {
        currentOriginalUrl = url
        lastTouchPoint = nil
        
        disposeWebMessageChannels()
        initializeWindowIdJS()
        
        if #available(iOS 14.0, *) {
            configuration.userContentController.resetContentWorlds(windowId: windowId)
        }
        
        channelDelegate?.onLoadStart(url: url?.absoluteString)
        
        inAppBrowserDelegate?.didStartNavigation(url: url)
    }
    
    public func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        initializeWindowIdJS()
        
        InAppWebView.credentialsProposed = []
        evaluateJavaScript(PLATFORM_READY_JS_SOURCE, completionHandler: nil)
        
        // sometimes scrollView.contentSize doesn't fit all the frame.size available
        // so, we call setNeedsLayout to redraw the layout
        let webViewFrameSize = frame.size
        let scrollViewSize = scrollView.contentSize
        if (scrollViewSize.width < webViewFrameSize.width || scrollViewSize.height < webViewFrameSize.height) {
            setNeedsLayout()
        }

        channelDelegate?.onLoadStop(url: url?.absoluteString)
        
        inAppBrowserDelegate?.didFinishNavigation(url: url)
    }
    
    public func webView(_ view: WKWebView,
                 didFailProvisionalNavigation navigation: WKNavigation!,
                 withError error: Error) {
        webView(view, didFail: navigation, withError: error)
    }
    
    public func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
        InAppWebView.credentialsProposed = []
        
        var urlError: URL = url ?? URL(string: "about:blank")!
        var errorCode = -1
        var errorDescription = "domain=\(error._domain), code=\(error._code), \(error.localizedDescription)"
        
        if let info = error as? URLError {
            if let failingURL = info.failingURL {
                urlError = failingURL
            }
            errorCode = info.code.rawValue
            errorDescription = info.localizedDescription
        }
        else if let info = error._userInfo as? [String: Any] {
            if let failingUrl = info[NSURLErrorFailingURLErrorKey] as? URL {
                urlError = failingUrl
            }
            if let failingUrlString = info[NSURLErrorFailingURLStringErrorKey] as? String,
               let failingUrl = URL(string: failingUrlString) {
                urlError = failingUrl
            }
        }
        
        let webResourceRequest = WebResourceRequest(url: urlError, headers: nil)
        let webResourceError = WebResourceError(type: errorCode, errorDescription: errorDescription)
        
        channelDelegate?.onReceivedError(request: webResourceRequest, error: webResourceError)
        
        inAppBrowserDelegate?.didFailNavigation(url: url, error: error)
    }
    
    public func webView(_ webView: WKWebView, didReceive challenge: URLAuthenticationChallenge, completionHandler: @escaping (URLSession.AuthChallengeDisposition, URLCredential?) -> Void) {
        var completionHandlerCalled = false
        if challenge.protectionSpace.authenticationMethod == NSURLAuthenticationMethodHTTPBasic ||
            challenge.protectionSpace.authenticationMethod == NSURLAuthenticationMethodDefault ||
            challenge.protectionSpace.authenticationMethod == NSURLAuthenticationMethodHTTPDigest ||
            challenge.protectionSpace.authenticationMethod == NSURLAuthenticationMethodNegotiate ||
            challenge.protectionSpace.authenticationMethod == NSURLAuthenticationMethodNTLM {
            let host = challenge.protectionSpace.host
            let prot = challenge.protectionSpace.protocol
            let realm = challenge.protectionSpace.realm
            let port = challenge.protectionSpace.port
            
            let callback = WebViewChannelDelegate.ReceivedHttpAuthRequestCallback()
            callback.nonNullSuccess = { (response: HttpAuthResponse) in
                if let action = response.action {
                    completionHandlerCalled = true
                    switch action {
                        case 0:
                            InAppWebView.credentialsProposed = []
                            // used .performDefaultHandling to maintain consistency with Android
                            // because .cancelAuthenticationChallenge will call webView(_:didFail:withError:)
                            completionHandler(.performDefaultHandling, nil)
                            //completionHandler(.cancelAuthenticationChallenge, nil)
                            break
                        case 1:
                            let username = response.username
                            let password = response.password
                            let permanentPersistence = response.permanentPersistence
                            let persistence = (permanentPersistence) ? URLCredential.Persistence.permanent : URLCredential.Persistence.forSession
                            let credential = URLCredential(user: username, password: password, persistence: persistence)
                            completionHandler(.useCredential, credential)
                            break
                        case 2:
                            if InAppWebView.credentialsProposed.count == 0 {
                                for (protectionSpace, credentials) in CredentialDatabase.credentialStore.allCredentials {
                                    if protectionSpace.host == host && protectionSpace.realm == realm &&
                                    protectionSpace.protocol == prot && protectionSpace.port == port {
                                        for credential in credentials {
                                            InAppWebView.credentialsProposed.append(credential.value)
                                        }
                                        break
                                    }
                                }
                            }
                            if InAppWebView.credentialsProposed.count == 0, let credential = challenge.proposedCredential {
                                InAppWebView.credentialsProposed.append(credential)
                            }
                            
                            if let credential = InAppWebView.credentialsProposed.popLast() {
                                completionHandler(.useCredential, credential)
                            }
                            else {
                                completionHandler(.performDefaultHandling, nil)
                            }
                            break
                        default:
                            InAppWebView.credentialsProposed = []
                            completionHandler(.performDefaultHandling, nil)
                    }
                    return false
                }
                return true
            }
            callback.defaultBehaviour = { (response: HttpAuthResponse?) in
                if !completionHandlerCalled {
                    completionHandlerCalled = true
                    completionHandler(.performDefaultHandling, nil)
                }
            }
            callback.error = { [weak callback] (code: String, message: String?, details: Any?) in
                print(code + ", " + (message ?? ""))
                callback?.defaultBehaviour(nil)
            }
            
            let runCallback = {
                if let channelDelegate = self.channelDelegate {
                    channelDelegate.onReceivedHttpAuthRequest(challenge: HttpAuthenticationChallenge(fromChallenge: challenge), callback: callback)
                } else {
                    callback.defaultBehaviour(nil)
                }
            }
            
            if windowId != nil, !windowCreated {
                windowBeforeCreatedCallbacks.append(runCallback)
            } else {
                runCallback()
            }
        }
        else if challenge.protectionSpace.authenticationMethod == NSURLAuthenticationMethodServerTrust {
            guard let serverTrust = challenge.protectionSpace.serverTrust else {
                completionHandler(.performDefaultHandling, nil)
                return
            }
            
            if let scheme = challenge.protectionSpace.protocol, scheme == "https" {
                // workaround for ProtectionSpace SSL Certificate
                // https://github.com/pichillilorenzo/flutter_inappwebview/issues/1678
                DispatchQueue.global(qos: .background).async {
                    if let sslCertificate = challenge.protectionSpace.sslCertificate {
                        DispatchQueue.main.async {
                            InAppWebView.sslCertificatesMap[challenge.protectionSpace.host] = sslCertificate
                        }
                    }
                }
            }
            
            let callback = WebViewChannelDelegate.ReceivedServerTrustAuthRequestCallback()
            callback.nonNullSuccess = { (response: ServerTrustAuthResponse) in
                if let action = response.action {
                    completionHandlerCalled = true
                    switch action {
                        case 0:
                            InAppWebView.credentialsProposed = []
                            completionHandler(.cancelAuthenticationChallenge, nil)
                            break
                        case 1:
                            // workaround for https://github.com/pichillilorenzo/flutter_inappwebview/issues/1924
                            DispatchQueue.global(qos: .background).async {
                                let exceptions = SecTrustCopyExceptions(serverTrust)
                                SecTrustSetExceptions(serverTrust, exceptions)
                                let credential = URLCredential(trust: serverTrust)
                                completionHandler(.useCredential, credential)
                            }
                            break
                        default:
                            InAppWebView.credentialsProposed = []
                            completionHandler(.performDefaultHandling, nil)
                    }
                    return false
                }
                return true
            }
            callback.defaultBehaviour = { (response: ServerTrustAuthResponse?) in
                if !completionHandlerCalled {
                    completionHandlerCalled = true
                    completionHandler(.performDefaultHandling, nil)
                }
            }
            callback.error = { [weak callback] (code: String, message: String?, details: Any?) in
                print(code + ", " + (message ?? ""))
                callback?.defaultBehaviour(nil)
            }
            
            let runCallback = {
                if let channelDelegate = self.channelDelegate {
                    channelDelegate.onReceivedServerTrustAuthRequest(challenge: ServerTrustChallenge(fromChallenge: challenge), callback: callback)
                } else {
                    callback.defaultBehaviour(nil)
                }
            }
            
            if windowId != nil, !windowCreated {
                windowBeforeCreatedCallbacks.append(runCallback)
            } else {
                runCallback()
            }
        }
        else if challenge.protectionSpace.authenticationMethod == NSURLAuthenticationMethodClientCertificate {
            let callback = WebViewChannelDelegate.ReceivedClientCertRequestCallback()
            callback.nonNullSuccess = { (response: ClientCertResponse) in
                if let action = response.action {
                    completionHandlerCalled = true
                    switch action {
                        case 0:
                            completionHandler(.cancelAuthenticationChallenge, nil)
                            break
                        case 1:
                            let certificatePath = response.certificatePath
                            let certificatePassword = response.certificatePassword ?? "";
                            
                            var path: String = certificatePath
                            do {
                                if let plugin = self.plugin {
                                    path = try Util.getAbsPathAsset(plugin: plugin, assetFilePath: certificatePath)
                                }
                            } catch {}
                            
                            if let PKCS12Data = NSData(contentsOfFile: path),
                               let identityAndTrust: IdentityAndTrust = self.extractIdentity(PKCS12Data: PKCS12Data, password: certificatePassword) {
                                let urlCredential: URLCredential = URLCredential(
                                    identity: identityAndTrust.identityRef,
                                    certificates: identityAndTrust.certArray as? [AnyObject],
                                    persistence: URLCredential.Persistence.forSession);
                                completionHandler(.useCredential, urlCredential)
                            } else {
                                completionHandler(.performDefaultHandling, nil)
                            }
                            
                            break
                        case 2:
                            completionHandler(.cancelAuthenticationChallenge, nil)
                            break
                        default:
                            completionHandler(.performDefaultHandling, nil)
                    }
                    return false
                }
                return true
            }
            callback.defaultBehaviour = { (response: ClientCertResponse?) in
                if !completionHandlerCalled {
                    completionHandlerCalled = true
                    completionHandler(.performDefaultHandling, nil)
                }
            }
            callback.error = { [weak callback] (code: String, message: String?, details: Any?) in
                print(code + ", " + (message ?? ""))
                callback?.defaultBehaviour(nil)
            }
            
            let runCallback = {
                if let channelDelegate = self.channelDelegate {
                    channelDelegate.onReceivedClientCertRequest(challenge: ClientCertChallenge(fromChallenge: challenge), callback: callback)
                } else {
                    callback.defaultBehaviour(nil)
                }
            }
            
            if windowId != nil, !windowCreated {
                windowBeforeCreatedCallbacks.append(runCallback)
            } else {
                runCallback()
            }
        }
        else {
            completionHandler(.performDefaultHandling, nil)
        }
    }
    
    struct IdentityAndTrust {
        var identityRef: SecIdentity
        var trust: SecTrust
        var certArray: AnyObject
    }

    func extractIdentity(PKCS12Data: NSData, password: String) -> IdentityAndTrust? {
        var identityAndTrust: IdentityAndTrust?
        var securityError: OSStatus = errSecSuccess

        var importResult: CFArray?
        securityError = SecPKCS12Import(
            PKCS12Data as NSData,
            [kSecImportExportPassphrase as String: password] as NSDictionary,
            &importResult
        )

        if securityError == errSecSuccess {
            let certItems: CFArray = importResult! as CFArray
            let certItemsArray: Array = certItems as Array
            let dict: AnyObject? = certItemsArray.first
            if let certEntry: Dictionary = dict as? Dictionary<String, AnyObject> {
                // grab the identity
                let identityPointer: AnyObject? = certEntry["identity"]
                let secIdentityRef:SecIdentity = (identityPointer as! SecIdentity?)!
                // grab the trust
                let trustPointer: AnyObject? = certEntry["trust"]
                let trustRef:SecTrust = trustPointer as! SecTrust
                // grab the cert
                let chainPointer: AnyObject? = certEntry["chain"]
                identityAndTrust = IdentityAndTrust(identityRef: secIdentityRef, trust: trustRef, certArray:  chainPointer!)
            }
        } else {
            print("Security Error: " + securityError.description)
            if #available(iOS 11.3, *) {
                print(SecCopyErrorMessageString(securityError, nil) ?? "")
            }
        }
        return identityAndTrust
    }
    
    func createAlertDialog(message: String?, responseMessage: String?, confirmButtonTitle: String?, completionHandler: @escaping () -> Void) {
        let title = responseMessage != nil && !responseMessage!.isEmpty ? responseMessage : message
        let okButton = confirmButtonTitle != nil && !confirmButtonTitle!.isEmpty ? confirmButtonTitle : NSLocalizedString("Ok", comment: "")
        let alertController = UIAlertController(title: title, message: nil,
                                                preferredStyle: UIAlertController.Style.alert);
        
        alertController.addAction(UIAlertAction(title: okButton, style: UIAlertAction.Style.default) {
            _ in completionHandler()}
        )
        
        guard let presentingViewController = inAppBrowserDelegate != nil ? inAppBrowserDelegate as? InAppBrowserWebViewController : window?.rootViewController else {
            completionHandler()
            return
        }
        presentingViewController.present(alertController, animated: true, completion: {})
    }
    
    public func webView(_ webView: WKWebView, runJavaScriptAlertPanelWithMessage message: String,
                 initiatedByFrame frame: WKFrameInfo, completionHandler: @escaping () -> Void) {
        
        if (isPausedTimers) {
            isPausedTimersCompletionHandler = completionHandler
            return
        }
        
        var completionHandlerCalled = false
        
        let callback = WebViewChannelDelegate.JsAlertCallback()
        callback.nonNullSuccess = { (response: JsAlertResponse) in
            if response.handledByClient {
                completionHandlerCalled = true
                let action = response.action ?? 1
                switch action {
                    case 0:
                        completionHandler()
                        break
                    default:
                        completionHandler()
                }
                return false
            }
            return true
        }
        callback.defaultBehaviour = { [weak self] (response: JsAlertResponse?) in
            if !completionHandlerCalled {
                completionHandlerCalled = true
                let responseMessage = response?.message
                let confirmButtonTitle = response?.confirmButtonTitle
                self?.createAlertDialog(message: message, responseMessage: responseMessage,
                                       confirmButtonTitle: confirmButtonTitle, completionHandler: completionHandler)
            }
        }
        callback.error = { (code: String, message: String?, details: Any?) in
            if !completionHandlerCalled {
                completionHandlerCalled = true
                print(code + ", " + (message ?? ""))
                completionHandler()
            }
        }
        
        if let channelDelegate = channelDelegate {
            channelDelegate.onJsAlert(url: frame.request.url, message: message, isMainFrame: frame.isMainFrame, callback: callback)
        } else {
            callback.defaultBehaviour(nil)
        }
    }
    
    func createConfirmDialog(message: String?, responseMessage: String?, confirmButtonTitle: String?, cancelButtonTitle: String?, completionHandler: @escaping (Bool) -> Void) {
        let dialogMessage = responseMessage != nil && !responseMessage!.isEmpty ? responseMessage : message
        let okButton = confirmButtonTitle != nil && !confirmButtonTitle!.isEmpty ? confirmButtonTitle : NSLocalizedString("Ok", comment: "")
        let cancelButton = cancelButtonTitle != nil && !cancelButtonTitle!.isEmpty ? cancelButtonTitle : NSLocalizedString("Cancel", comment: "")
        
        let confirmController = UIAlertController(title: nil, message: dialogMessage, preferredStyle: .alert)
        
        confirmController.addAction(UIAlertAction(title: okButton, style: .default, handler: { (action) in
            completionHandler(true)
        }))
        
        confirmController.addAction(UIAlertAction(title: cancelButton, style: .cancel, handler: { (action) in
            completionHandler(false)
        }))
        
        guard let presentingViewController = inAppBrowserDelegate != nil ? inAppBrowserDelegate as? InAppBrowserWebViewController : window?.rootViewController else {
            completionHandler(false)
            return
        }
        presentingViewController.present(confirmController, animated: true, completion: nil)
    }
    
    public func webView(_ webView: WKWebView, runJavaScriptConfirmPanelWithMessage message: String, initiatedByFrame frame: WKFrameInfo,
                 completionHandler: @escaping (Bool) -> Void) {
        var completionHandlerCalled = false
        
        let callback = WebViewChannelDelegate.JsConfirmCallback()
        callback.nonNullSuccess = { (response: JsConfirmResponse) in
            if response.handledByClient {
                completionHandlerCalled = true
                let action = response.action ?? 1
                switch action {
                    case 0:
                        completionHandler(true)
                        break
                    case 1:
                        completionHandler(false)
                        break
                    default:
                        completionHandler(false)
                }
                return false
            }
            return true
        }
        callback.defaultBehaviour = { [weak self] (response: JsConfirmResponse?) in
            if !completionHandlerCalled {
                completionHandlerCalled = true
                let responseMessage = response?.message
                let confirmButtonTitle = response?.confirmButtonTitle
                let cancelButtonTitle = response?.cancelButtonTitle
                self?.createConfirmDialog(message: message, responseMessage: responseMessage, confirmButtonTitle: confirmButtonTitle, cancelButtonTitle: cancelButtonTitle, completionHandler: completionHandler)
            }
        }
        callback.error = { (code: String, message: String?, details: Any?) in
            if !completionHandlerCalled {
                completionHandlerCalled = true
                print(code + ", " + (message ?? ""))
                completionHandler(false)
            }
        }
        
        if let channelDelegate = channelDelegate {
            channelDelegate.onJsConfirm(url: frame.request.url, message: message, isMainFrame: frame.isMainFrame, callback: callback)
        } else {
            callback.defaultBehaviour(nil)
        }
    }

    func createPromptDialog(message: String, defaultValue: String?, responseMessage: String?, confirmButtonTitle: String?, cancelButtonTitle: String?, value: String?, completionHandler: @escaping (String?) -> Void) {
        let dialogMessage = responseMessage != nil && !responseMessage!.isEmpty ? responseMessage : message
        let okButton = confirmButtonTitle != nil && !confirmButtonTitle!.isEmpty ? confirmButtonTitle : NSLocalizedString("Ok", comment: "")
        let cancelButton = cancelButtonTitle != nil && !cancelButtonTitle!.isEmpty ? cancelButtonTitle : NSLocalizedString("Cancel", comment: "")
        
        let promptController = UIAlertController(title: nil, message: dialogMessage, preferredStyle: .alert)
        
        promptController.addTextField { (textField) in
            textField.text = defaultValue
        }
        
        promptController.addAction(UIAlertAction(title: okButton, style: .default, handler: { (action) in
            if let v = value {
                completionHandler(v)
            }
            else if let text = promptController.textFields?.first?.text {
                completionHandler(text)
            } else {
                completionHandler("")
            }
        }))
        
        promptController.addAction(UIAlertAction(title: cancelButton, style: .cancel, handler: { (action) in
            completionHandler(nil)
        }))
        
        guard let presentingViewController = inAppBrowserDelegate != nil ? inAppBrowserDelegate as? InAppBrowserWebViewController : window?.rootViewController else {
            completionHandler(nil)
            return
        }
        presentingViewController.present(promptController, animated: true, completion: nil)
    }
    
    public func webView(_ webView: WKWebView, runJavaScriptTextInputPanelWithPrompt message: String, defaultText defaultValue: String?, initiatedByFrame frame: WKFrameInfo,
                 completionHandler: @escaping (String?) -> Void) {
        
        var completionHandlerCalled = false
        
        let callback = WebViewChannelDelegate.JsPromptCallback()
        callback.nonNullSuccess = { (response: JsPromptResponse) in
            if response.handledByClient {
                completionHandlerCalled = true
                let action = response.action ?? 1
                switch action {
                    case 0:
                        completionHandler(response.value)
                        break
                    case 1:
                        completionHandler(nil)
                        break
                    default:
                        completionHandler(nil)
                }
                return false
            }
            return true
        }
        callback.defaultBehaviour = { [weak self] (response: JsPromptResponse?) in
            if !completionHandlerCalled {
                completionHandlerCalled = true
                let responseMessage = response?.message
                let confirmButtonTitle = response?.confirmButtonTitle
                let cancelButtonTitle = response?.cancelButtonTitle
                let value = response?.value
                self?.createPromptDialog(message: message, defaultValue: defaultValue, responseMessage: responseMessage, confirmButtonTitle: confirmButtonTitle,
                                        cancelButtonTitle: cancelButtonTitle, value: value, completionHandler: completionHandler)
            }
        }
        callback.error = { (code: String, message: String?, details: Any?) in
            if !completionHandlerCalled {
                completionHandlerCalled = true
                print(code + ", " + (message ?? ""))
                completionHandler(nil)
            }
        }
        
        if let channelDelegate = channelDelegate {
            channelDelegate.onJsPrompt(url: frame.request.url, message: message, defaultValue: defaultValue, isMainFrame: frame.isMainFrame, callback: callback)
        } else {
            callback.defaultBehaviour(nil)
        }
    }
    
    /// UIScrollViewDelegate is somehow bugged:
    /// if InAppWebView implements the UIScrollViewDelegate protocol and implement the scrollViewDidScroll event,
    /// then, when the user scrolls the content, the webview content is not rendered (just white space).
    /// Calling setNeedsLayout() resolves this problem, but, for some reason, the bounce effect is canceled.
    ///
    /// So, to track the same event, without implementing the scrollViewDidScroll event, we create
    /// an observer that observes the scrollView.contentOffset property.
    /// This way, we don't need to call setNeedsLayout() and all works fine.
    public func onScrollChanged(startedByUser: Bool, oldContentOffset: CGPoint?) {
        let disableVerticalScroll = settings?.disableVerticalScroll ?? false
        let disableHorizontalScroll = settings?.disableHorizontalScroll ?? false
        if startedByUser {
            if disableVerticalScroll && disableHorizontalScroll {
                scrollView.contentOffset = CGPoint(x: lastScrollX, y: lastScrollY)
            }
            else if disableVerticalScroll {
                if scrollView.contentOffset.y >= 0 || scrollView.contentOffset.y < 0 {
                    scrollView.contentOffset = CGPoint(x: scrollView.contentOffset.x, y: lastScrollY)
                }
            }
            else if disableHorizontalScroll {
                if scrollView.contentOffset.x >= 0 || scrollView.contentOffset.x < 0 {
                    scrollView.contentOffset = CGPoint(x: lastScrollX, y: scrollView.contentOffset.y)
                }
            }
        }
        if (!disableVerticalScroll && !disableHorizontalScroll) ||
            (disableVerticalScroll && scrollView.contentOffset.x != oldContentOffset?.x) ||
            (disableHorizontalScroll && scrollView.contentOffset.y != oldContentOffset?.y) {
            let x = Int(scrollView.contentOffset.x / scrollView.contentScaleFactor)
            let y = Int(scrollView.contentOffset.y / scrollView.contentScaleFactor)
            channelDelegate?.onScrollChanged(x: x, y: y)
        }
        lastScrollX = scrollView.contentOffset.x
        lastScrollY = scrollView.contentOffset.y
        
        let overScrolledHorizontally = lastScrollX < 0 || lastScrollX > (scrollView.contentSize.width - scrollView.frame.size.width)
        let overScrolledVertically = lastScrollY < 0 || lastScrollY > (scrollView.contentSize.height - scrollView.frame.size.height)
        if overScrolledHorizontally || overScrolledVertically {
            let x = Int(lastScrollX / scrollView.contentScaleFactor)
            let y = Int(lastScrollY / scrollView.contentScaleFactor)
            channelDelegate?.onOverScrolled(x: x, y: y,
                           clampedX: overScrolledHorizontally,
                           clampedY: overScrolledVertically)
        }
    }
    
    public func onContentSizeChanged(oldContentSize: CGSize) {
        channelDelegate?.onContentSizeChanged(oldContentSize: oldContentSize,
                                              newContentSize: scrollView.contentSize)
    }
    
    public func scrollViewDidZoom(_ scrollView: UIScrollView) {
        let newScale = Float(scrollView.zoomScale)
        if newScale != oldZoomScale {
            channelDelegate?.onZoomScaleChanged(newScale: newScale, oldScale: oldZoomScale)
            oldZoomScale = newScale
        }
    }
    
    public func webView(_ webView: WKWebView,
                        createWebViewWith configuration: WKWebViewConfiguration,
                  for navigationAction: WKNavigationAction,
                  windowFeatures: WKWindowFeatures) -> WKWebView? {
        
        var windowId: Int64 = 0
        let inAppWebViewManager = plugin?.inAppWebViewManager
        if let inAppWebViewManager = inAppWebViewManager {
            inAppWebViewManager.windowAutoincrementId += 1
            windowId = inAppWebViewManager.windowAutoincrementId
        }
        
        let windowWebView = InAppWebView(id: nil, plugin: nil, frame: self.bounds, configuration: configuration, contextMenu: nil)
        windowWebView.windowId = windowId

        let webViewTransport = WebViewTransport(
            webView: windowWebView,
            request: navigationAction.request
        )

        inAppWebViewManager?.windowWebViews[windowId] = webViewTransport
        
        let createWindowAction = CreateWindowAction(navigationAction: navigationAction, windowId: windowId, windowFeatures: windowFeatures, isDialog: nil)
        
        let callback = WebViewChannelDelegate.CreateWindowCallback()
        callback.nonNullSuccess = { (handledByClient: Bool) in
            return !handledByClient
        }
        callback.defaultBehaviour = { [weak self] (handledByClient: Bool?) in
            if inAppWebViewManager?.windowWebViews[windowId] != nil {
                inAppWebViewManager?.windowWebViews.removeValue(forKey: windowId)
            }
            self?.loadUrl(urlRequest: navigationAction.request, allowingReadAccessTo: nil)
        }
        callback.error = { [weak callback] (code: String, message: String?, details: Any?) in
            print(code + ", " + (message ?? ""))
            callback?.defaultBehaviour(nil)
        }
        
        if let channelDelegate = channelDelegate {
            channelDelegate.onCreateWindow(createWindowAction: createWindowAction, callback: callback)
        } else {
            callback.defaultBehaviour(nil)
        }
        
        return windowWebView
    }
    
    public func webView(_ webView: WKWebView,
                        authenticationChallenge challenge: URLAuthenticationChallenge,
                        shouldAllowDeprecatedTLS decisionHandler: @escaping (Bool) -> Void) {
        var decisionHandlerCalled = false
        let callback = WebViewChannelDelegate.ShouldAllowDeprecatedTLSCallback()
        callback.nonNullSuccess = { (action: Bool) in
            decisionHandlerCalled = true
            decisionHandler(action)
            return false
        }
        callback.defaultBehaviour = { (action: Bool?) in
            if !decisionHandlerCalled {
                decisionHandlerCalled = true
                decisionHandler(false)
            }
        }
        callback.error = { [weak callback] (code: String, message: String?, details: Any?) in
            print(code + ", " + (message ?? ""))
            callback?.defaultBehaviour(nil)
        }
        
        let runCallback = {
            if let channelDelegate = self.channelDelegate {
                channelDelegate.shouldAllowDeprecatedTLS(challenge: challenge, callback: callback)
            } else {
                callback.defaultBehaviour(nil)
            }
        }
        
        if windowId != nil, !windowCreated {
            windowBeforeCreatedCallbacks.append(runCallback)
        } else {
            runCallback()
        }
    }
    
    public func webViewDidClose(_ webView: WKWebView) {
        channelDelegate?.onCloseWindow()
    }
    
    public func webViewWebContentProcessDidTerminate(_ webView: WKWebView) {
        channelDelegate?.onWebContentProcessDidTerminate()
    }
    
    public func webView(_ webView: WKWebView,
                        didCommit navigation: WKNavigation!) {
        channelDelegate?.onPageCommitVisible(url: url?.absoluteString)
    }
    
    public func webView(_ webView: WKWebView,
                        didReceiveServerRedirectForProvisionalNavigation navigation: WKNavigation!) {
        channelDelegate?.onDidReceiveServerRedirectForProvisionalNavigation()
    }
    
//    @available(iOS 13.0, *)
//    public func webView(_ webView: WKWebView,
//                        contextMenuConfigurationForElement elementInfo: WKContextMenuElementInfo,
//                        completionHandler: @escaping (UIContextMenuConfiguration?) -> Void) {
//        print("contextMenuConfigurationForElement")
//        let actionProvider: UIContextMenuActionProvider = { _ in
//            let editMenu = UIMenu(title: "Edit...", children: [
//                UIAction(title: "Copy") { action in
//
//                },
//                UIAction(title: "Duplicate") { action in
//
//                }
//            ])
//            return UIMenu(title: "Title", children: [
//                UIAction(title: "Share") { action in
//
//                },
//                editMenu
//            ])
//        }
//        let contextMenuConfiguration = UIContextMenuConfiguration(identifier: nil, previewProvider: nil, actionProvider: actionProvider)
//        //completionHandler(contextMenuConfiguration)
//        completionHandler(nil)
////        onContextMenuConfigurationForElement(linkURL: elementInfo.linkURL?.absoluteString, result: nil/*{(result) -> Void in
////            if result is FlutterError {
////                print((result as! FlutterError).message ?? "")
////            }
////            else if (result as? NSObject) == FlutterMethodNotImplemented {
////                completionHandler(nil)
////            }
////            else {
////                var response: [String: Any]
////                if let r = result {
////                    response = r as! [String: Any]
////                    var action = response["action"] as? Int
////                    action = action != nil ? action : 0;
////                    switch action {
////                        case 0:
////                            break
////                        case 1:
////                            break
////                        default:
////                            completionHandler(nil)
////                    }
////                    return;
////                }
////                completionHandler(nil)
////            }
////        }*/)
//    }
////
//    @available(iOS 13.0, *)
//    public func webView(_ webView: WKWebView,
//                        contextMenuDidEndForElement elementInfo: WKContextMenuElementInfo) {
//        print("contextMenuDidEndForElement")
//        print(elementInfo)
//        //onContextMenuDidEndForElement(linkURL: elementInfo.linkURL?.absoluteString)
//    }
//
//    @available(iOS 13.0, *)
//    public func webView(_ webView: WKWebView,
//                        contextMenuForElement elementInfo: WKContextMenuElementInfo,
//                        willCommitWithAnimator animator: UIContextMenuInteractionCommitAnimating) {
//        print("willCommitWithAnimator")
//        print(elementInfo)
////        onWillCommitWithAnimator(linkURL: elementInfo.linkURL?.absoluteString, result: nil/*{(result) -> Void in
////            if result is FlutterError {
////                print((result as! FlutterError).message ?? "")
////            }
////            else if (result as? NSObject) == FlutterMethodNotImplemented {
////
////            }
////            else {
////                var response: [String: Any]
////                if let r = result {
////                    response = r as! [String: Any]
////                    var action = response["action"] as? Int
////                    action = action != nil ? action : 0;
//////                    switch action {
//////                        case 0:
//////                            break
//////                        case 1:
//////                            break
//////                        default:
//////
//////                    }
////                    return;
////                }
////
////            }
////        }*/)
//    }
//
//    @available(iOS 13.0, *)
//    public func webView(_ webView: WKWebView,
//                        contextMenuWillPresentForElement elementInfo: WKContextMenuElementInfo) {
//        print("contextMenuWillPresentForElement")
//        print(elementInfo.linkURL)
//        //onContextMenuWillPresentForElement(linkURL: elementInfo.linkURL?.absoluteString)
//    }
    
    
    // https://stackoverflow.com/a/42840541/4637638
    public func isVideoPlayerWindow(_ notificationObject: AnyObject?) -> Bool {
        let nonVideoClasses = ["_UIAlertControllerShimPresenterWindow",
                               "UITextEffectsWindow",
                               "UIRemoteKeyboardWindow",
                               "PGHostedWindow"]
        
        var isVideo = true
        if let obj = notificationObject {
            for nonVideoClass in nonVideoClasses {
                if let clazz = NSClassFromString(nonVideoClass) {
                    isVideo = isVideo && !(obj.isKind(of: clazz))
                }
            }
        }
        return isVideo
    }
    
    @objc func onEnterFullscreen(_ notification: Notification) {
        // TODO: Still not working on iOS 16.0!
//        if #available(iOS 16.0, *) {
//            channelDelegate?.onEnterFullscreen()
//            inFullscreen = true
//        }
//        else
        if (isVideoPlayerWindow(notification.object as AnyObject?)) {
            channelDelegate?.onEnterFullscreen()
            inFullscreen = true
        }
    }
    
    @objc func onExitFullscreen(_ notification: Notification) {
        // TODO: Still not working on iOS 16.0!
//        if #available(iOS 16.0, *) {
//            channelDelegate?.onExitFullscreen()
//            inFullscreen = false
//        }
//        else
        if (isVideoPlayerWindow(notification.object as AnyObject?)) {
            channelDelegate?.onExitFullscreen()
            inFullscreen = false
        }
    }
    
//    public func onContextMenuConfigurationForElement(linkURL: String?, result: FlutterResult?) {
//        let arguments: [String: Any?] = ["linkURL": linkURL]
//        channel?.invokeMethod("onContextMenuConfigurationForElement", arguments: arguments, result: result)
//    }
//
//    public func onContextMenuDidEndForElement(linkURL: String?) {
//        let arguments: [String: Any?] = ["linkURL": linkURL]
//        channel?.invokeMethod("onContextMenuDidEndForElement", arguments: arguments)
//    }
//
//    public func onWillCommitWithAnimator(linkURL: String?, result: FlutterResult?) {
//        let arguments: [String: Any?] = ["linkURL": linkURL]
//        channel?.invokeMethod("onWillCommitWithAnimator", arguments: arguments, result: result)
//    }
//
//    public func onContextMenuWillPresentForElement(linkURL: String?) {
//        let arguments: [String: Any?] = ["linkURL": linkURL]
//        channel?.invokeMethod("onContextMenuWillPresentForElement", arguments: arguments)
//    }
    
    public func userContentController(_ userContentController: WKUserContentController, didReceive message: WKScriptMessage) {
        guard let body = message.body as? [String: Any?] else {
            return
        }
        
        if ["consoleLog", "consoleDebug", "consoleError", "consoleInfo", "consoleWarn"].contains(message.name) {
            var messageLevel = 1
            switch (message.name) {
                case "consoleLog":
                    messageLevel = 1
                    break
                case "consoleDebug":
                    // on Android, console.debug is TIP
                    messageLevel = 0
                    break
                case "consoleError":
                    messageLevel = 3
                    break
                case "consoleInfo":
                    // on Android, console.info is LOG
                    messageLevel = 1
                    break
                case "consoleWarn":
                    messageLevel = 2
                    break
                default:
                    messageLevel = 1
                    break
            }
            let consoleMessage = body["message"] as? String ?? ""
            
            let _windowId = body["_windowId"] as? Int64
            var webView = self
            if let wId = _windowId, let webViewTransport = plugin?.inAppWebViewManager?.windowWebViews[wId] {
                webView = webViewTransport.webView
            }
            webView.channelDelegate?.onConsoleMessage(message: consoleMessage, messageLevel: messageLevel)
        } else if message.name == "callHandler", let handlerName = body["handlerName"] as? String {
            if handlerName == "onPrintRequest" {
                let settings = PrintJobSettings()
                settings.handledByClient = true
                if let printJobId = printCurrentPage(settings: settings) {
                    let callback = WebViewChannelDelegate.PrintRequestCallback()
                    callback.nonNullSuccess = { (handledByClient: Bool) in
                        return !handledByClient
                    }
                    callback.defaultBehaviour = { [weak self] (handledByClient: Bool?) in
                        if let printJob = self?.plugin?.printJobManager?.jobs[printJobId] {
                            printJob?.disposeNoDismiss()
                        }
                    }
                    callback.error = { [weak callback] (code: String, message: String?, details: Any?) in
                        print(code + ", " + (message ?? ""))
                        callback?.defaultBehaviour(nil)
                    }
                    channelDelegate?.onPrintRequest(url: url, printJobId: printJobId, callback: callback)
                }
                return
            }
            
            let _callHandlerID = body["_callHandlerID"] as? Int64 ?? 0
            let args = body["args"] as? String ?? ""
            
            let _windowId = body["_windowId"] as? Int64
            var webView = self
            if let wId = _windowId, let webViewTransport = plugin?.inAppWebViewManager?.windowWebViews[wId] {
                webView = webViewTransport.webView
            }
            
            let callback = WebViewChannelDelegate.CallJsHandlerCallback()
            callback.defaultBehaviour = { [weak self] (response: Any?) in
                var json = "null"
                if let r = response as? String {
                    json = r
                }
                
                self?.evaluateJavaScript("""
if(window.\(JAVASCRIPT_BRIDGE_NAME)[\(_callHandlerID)] != null) {
    window.\(JAVASCRIPT_BRIDGE_NAME)[\(_callHandlerID)].resolve(\(json));
    delete window.\(JAVASCRIPT_BRIDGE_NAME)[\(_callHandlerID)];
}
""", completionHandler: nil)
            }
            callback.error = { [weak self] (code: String, message: String?, details: Any?) in
                let errorMessage = code + (message != nil ? ", " + (message ?? "") : "")
                print(errorMessage)
                
                self?.evaluateJavaScript("""
if(window.\(JAVASCRIPT_BRIDGE_NAME)[\(_callHandlerID)] != null) {
    window.\(JAVASCRIPT_BRIDGE_NAME)[\(_callHandlerID)].reject(new Error('\(errorMessage.replacingOccurrences(of: "\'", with: "\\'"))'));
    delete window.\(JAVASCRIPT_BRIDGE_NAME)[\(_callHandlerID)];
}
""", completionHandler: nil)
            }
            
            if let channelDelegate = webView.channelDelegate {
                channelDelegate.onCallJsHandler(handlerName: handlerName, args: args, callback: callback)
            }
        } else if message.name == "onFindResultReceived",
                  let findResult = body["findResult"] as? [String: Any],
                  let activeMatchOrdinal = findResult["activeMatchOrdinal"] as? Int,
                  let numberOfMatches = findResult["numberOfMatches"] as? Int,
                  let isDoneCounting = findResult["isDoneCounting"] as? Bool {
            
            let _windowId = body["_windowId"] as? Int64
            var webView = self
            if let wId = _windowId, let webViewTransport = plugin?.inAppWebViewManager?.windowWebViews[wId] {
                webView = webViewTransport.webView
            }
            webView.findInteractionController?.channelDelegate?.onFindResultReceived(activeMatchOrdinal: activeMatchOrdinal, numberOfMatches: numberOfMatches, isDoneCounting: isDoneCounting)
            webView.channelDelegate?.onFindResultReceived(activeMatchOrdinal: activeMatchOrdinal, numberOfMatches: numberOfMatches, isDoneCounting: isDoneCounting)
        } else if message.name == "onCallAsyncJavaScriptResultBelowIOS14Received",
                  let resultUuid = body["resultUuid"] as? String,
                  let result = callAsyncJavaScriptBelowIOS14Results[resultUuid] {
            result([
                    "value": body["value"],
                    "error": body["error"]
            ])
            callAsyncJavaScriptBelowIOS14Results.removeValue(forKey: resultUuid)
        } else if message.name == "onWebMessagePortMessageReceived",
                  let webMessageChannelId = body["webMessageChannelId"] as? String,
                  let index = body["index"] as? Int64 {
            var webMessage: WebMessage? = nil
            if let webMessageMap = body["message"] as? [String : Any?] {
                webMessage = WebMessage.fromMap(map: webMessageMap)
            }
            
            if let webMessageChannel = webMessageChannels[webMessageChannelId] {
                webMessageChannel.channelDelegate?.onMessage(index: index, message: webMessage)
            }
        } else if message.name == "onWebMessageListenerPostMessageReceived", let jsObjectName = body["jsObjectName"] as? String {
            var webMessage: WebMessage? = nil
            if let webMessageMap = body["message"] as? [String : Any?] {
                webMessage = WebMessage.fromMap(map: webMessageMap)
            }
            
            if let webMessageListener = webMessageListeners.first(where: ({($0.jsObjectName == jsObjectName)})) {
                let isMainFrame = message.frameInfo.isMainFrame
                
                var scheme: String? = nil
                var host: String? = nil
                var port: Int? = nil
                if #available(iOS 9.0, *) {
                    let sourceOrigin = message.frameInfo.securityOrigin
                    scheme = sourceOrigin.protocol
                    host = sourceOrigin.host
                    port = sourceOrigin.port
                } else if let url = message.frameInfo.request.url {
                    scheme = url.scheme
                    host = url.host
                    port = url.port
                }
                
                if !webMessageListener.isOriginAllowed(scheme: scheme, host: host, port: port) {
                    return
                }
                
                var sourceOrigin: URL? = nil
                if let scheme = scheme, !scheme.isEmpty, let host = host, !host.isEmpty {
                    sourceOrigin = URL(string: "\(scheme)://\(host)\(port != nil && port != 0 ? ":" + String(port!) : "")")
                }
                webMessageListener.channelDelegate?.onPostMessage(message: webMessage, sourceOrigin: sourceOrigin, isMainFrame: isMainFrame)
            }
        }
    }
    
    public func scrollTo(x: Int, y: Int, animated: Bool) {
        scrollView.setContentOffset(CGPoint(x: x, y: y), animated: animated)
    }
    
    public func scrollBy(x: Int, y: Int, animated: Bool) {
        let newX = CGFloat(x) + scrollView.contentOffset.x
        let newY = CGFloat(y) + scrollView.contentOffset.y
        scrollView.setContentOffset(CGPoint(x: newX, y: newY), animated: animated)
    }
    
    
    public func pauseTimers() {
        if !isPausedTimers {
            isPausedTimers = true
            let script = "alert();";
            self.evaluateJavaScript(script, completionHandler: nil)
        }
    }
    
    public func resumeTimers() {
        if isPausedTimers {
            if let completionHandler = isPausedTimersCompletionHandler {
                self.isPausedTimersCompletionHandler = nil
                completionHandler()
            }
            isPausedTimers = false
        }
    }
    
    public func printCurrentPage(settings: PrintJobSettings? = nil,
                                 completionHandler: UIPrintInteractionController.CompletionHandler? = nil) -> String? {
        var printJobId: String? = nil
        if let settings = settings, settings.handledByClient {
            printJobId = NSUUID().uuidString
        }
        
        let printController = UIPrintInteractionController.shared
        let printFormatter = self.viewPrintFormatter()
        if let settings = settings {
            if let margins = settings.margins {
                printFormatter.perPageContentInsets = margins
            }
            if let maximumContentHeight = settings.maximumContentHeight {
                printFormatter.maximumContentHeight = maximumContentHeight
            }
            if let maximumContentWidth = settings.maximumContentWidth {
                printFormatter.maximumContentWidth = maximumContentWidth
            }
        }
        printController.printFormatter = printFormatter
        
        printController.printInfo = UIPrintInfo(dictionary: nil)
        if let printInfo = printController.printInfo {
            printInfo.jobName = settings?.jobName ?? (title ?? url?.absoluteString ?? "") + " Document"
            if let settings = settings {
                if let orientationValue = settings.orientation,
                    let orientation = UIPrintInfo.Orientation.init(rawValue: orientationValue) {
                    printInfo.orientation = orientation
                }
                if let duplexModeValue = settings.duplexMode,
                    let duplexMode = UIPrintInfo.Duplex.init(rawValue: duplexModeValue) {
                    printInfo.duplex = duplexMode
                }
                if let outputTypeValue = settings.outputType,
                    let outputType = UIPrintInfo.OutputType.init(rawValue: outputTypeValue) {
                    printInfo.outputType = outputType
                }
            }
        }
        
        // initialize print renderer and set its formatter
        let printRenderer = CustomUIPrintPageRenderer(numberOfPage: settings?.numberOfPages,
                                                      forceRenderingQuality: settings?.forceRenderingQuality)
        printRenderer.addPrintFormatter(printFormatter, startingAtPageAt: 0)
        if let settings = settings {
            if let footerHeight = settings.footerHeight {
                printRenderer.footerHeight = footerHeight
            }
            if let headerHeight = settings.headerHeight {
                printRenderer.headerHeight = headerHeight
            }
        }
        printController.printPageRenderer = printRenderer
        
        if let settings = settings {
            printController.showsNumberOfCopies = settings.showsNumberOfCopies
            printController.showsPaperSelectionForLoadedPapers = settings.showsPaperSelectionForLoadedPapers
            if #available(iOS 15.0, *) {
                printController.showsPaperOrientation = settings.showsPaperOrientation
            }
        }
        
        let animated = settings?.animated ?? true
        if let id = printJobId, let plugin = plugin {
            let printJob = PrintJobController(plugin: plugin, id: id, job: printController, settings: settings)
            plugin.printJobManager?.jobs[id] = printJob
            printJob.present(animated: animated, completionHandler: completionHandler)
        } else {
            printController.present(animated: animated, completionHandler: completionHandler)
        }
        
        return printJobId
    }
    
    public func getContentHeight() -> Int64 {
        return Int64(scrollView.contentSize.height)
    }
    
    public func getContentWidth() -> Int64 {
        return Int64(scrollView.contentSize.width)
    }
    
    public func zoomBy(zoomFactor: Float, animated: Bool) {
        let currentZoomScale = scrollView.zoomScale
        scrollView.setZoomScale(currentZoomScale * CGFloat(zoomFactor), animated: animated)
    }
    
    public func getOriginalUrl() -> URL? {
        return currentOriginalUrl
    }
    
    public func getZoomScale() -> Float {
        return Float(scrollView.zoomScale)
    }
    
    public func getSelectedText(completionHandler: @escaping (Any?, Error?) -> Void) {
        if configuration.preferences.javaScriptEnabled {
            evaluateJavaScript(PluginScriptsUtil.GET_SELECTED_TEXT_JS_SOURCE, completionHandler: completionHandler)
        } else {
            completionHandler(nil, nil)
        }
    }
    
    public func getHitTestResult(completionHandler: @escaping (HitTestResult) -> Void) {
        if configuration.preferences.javaScriptEnabled, let lastTouchLocation = lastTouchPoint {
            self.evaluateJavaScript("window.\(JAVASCRIPT_BRIDGE_NAME)._findElementsAtPoint(\(lastTouchLocation.x),\(lastTouchLocation.y))", completionHandler: {(value, error) in
                if error != nil {
                    print("getHitTestResult error: \(error?.localizedDescription ?? "")")
                    completionHandler(HitTestResult(type: .unknownType, extra: nil))
                } else if let value = value as? [String: Any?] {
                    let hitTestResult = HitTestResult.fromMap(map: value)!
                    completionHandler(hitTestResult)
                } else {
                    completionHandler(HitTestResult(type: .unknownType, extra: nil))
                }
            })
        } else {
            completionHandler(HitTestResult(type: .unknownType, extra: nil))
        }
    }
    
    public func requestFocusNodeHref(completionHandler: @escaping ([String: Any?]?, Error?) -> Void) {
        if configuration.preferences.javaScriptEnabled {
            // add some delay to make it sure _lastAnchorOrImageTouched is updated
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.15) {
                self.evaluateJavaScript("window.\(JAVASCRIPT_BRIDGE_NAME)._lastAnchorOrImageTouched", completionHandler: {(value, error) in
                    let lastAnchorOrImageTouched = value as? [String: Any?]
                    completionHandler(lastAnchorOrImageTouched, error)
                })
            }
        } else {
            completionHandler(nil, nil)
        }
    }
    
    public func requestImageRef(completionHandler: @escaping ([String: Any?]?, Error?) -> Void) {
        if configuration.preferences.javaScriptEnabled {
            // add some delay to make it sure _lastImageTouched is updated
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.15) {
                self.evaluateJavaScript("window.\(JAVASCRIPT_BRIDGE_NAME)._lastImageTouched", completionHandler: {(value, error) in
                    let lastImageTouched = value as? [String: Any?]
                    completionHandler(lastImageTouched, error)
                })
            }
        } else {
            completionHandler(nil, nil)
        }
    }
    
    public func clearFocus() {
        self.scrollView.subviews.first?.resignFirstResponder()
    }
    
    public func getCertificate() -> SslCertificate? {
        guard let scheme = url?.scheme,
              scheme == "https",
              let host = url?.host,
              let sslCertificate = InAppWebView.sslCertificatesMap[host] else {
            return nil
        }
        return sslCertificate
    }
    
    public func isSecureContext(completionHandler: @escaping (_ isSecureContext: Bool) -> Void) {
        evaluateJavascript(source: "window.isSecureContext") { (isSecureContext) in
            if let isSecureContext = isSecureContext {
                completionHandler(isSecureContext as? Bool ?? false)
                return
            }
            completionHandler(false)
        }
    }
    
    public func canScrollVertically() -> Bool {
        return scrollView.contentSize.height > self.frame.height
    }
    
    public func canScrollHorizontally() -> Bool {
        return scrollView.contentSize.width > self.frame.width
    }
    
    public func enablePullToRefresh() {
        if let pullToRefreshControl = pullToRefreshControl {
            if #available(iOS 10.0, *) {
                scrollView.refreshControl = pullToRefreshControl
            } else {
                scrollView.addSubview(pullToRefreshControl)
            }
        }
    }
    
    public func disablePullToRefresh() {
        pullToRefreshControl?.removeFromSuperview()
        if #available(iOS 10.0, *) {
            scrollView.refreshControl = nil
        }
    }
    
    public func isPullToRefreshEnabled() -> Bool {
        if #available(iOS 10.0, *) {
            return scrollView.refreshControl != nil
        } else {
            return pullToRefreshControl?.superview != nil
        }
    }
    
    public func createWebMessageChannel(completionHandler: ((WebMessageChannel?) -> Void)? = nil) -> WebMessageChannel? {
        guard let plugin = plugin else {
            completionHandler?(nil)
            return nil
        }
        let id = NSUUID().uuidString
        let webMessageChannel = WebMessageChannel(plugin: plugin, id: id)
        webMessageChannel.initJsInstance(webView: self, completionHandler: completionHandler)
        webMessageChannels[id] = webMessageChannel
        
        return webMessageChannel
    }
    
    public func postWebMessage(message: WebMessage, targetOrigin: String, completionHandler: ((Any?) -> Void)? = nil) throws {
        var portsString = "null"
        if let ports = message.ports {
            var portArrayString: [String] = []
            for port in ports {
                if port.isStarted {
                    throw NSError(domain: "Port is already started", code: 0)
                }
                if port.isClosed || port.isTransferred {
                    throw NSError(domain: "Port is already closed or transferred", code: 0)
                }
                port.isTransferred = true
                portArrayString.append("\(WEB_MESSAGE_CHANNELS_VARIABLE_NAME)['\(port.webMessageChannel!.id)'].\(port.name)")
            }
            portsString = "[" + portArrayString.joined(separator: ", ") + "]"
        }
        
        let url = URL(string: targetOrigin)?.absoluteString ?? "*"
        let source = """
        (function() {
            window.postMessage(\(message.jsData), '\(url)', \(portsString));
        })();
        """
        evaluateJavascript(source: source, completionHandler: completionHandler)
        message.dispose()
    }
    
    public func addWebMessageListener(webMessageListener: WebMessageListener) throws {
        if webMessageListeners.map({ ($0.jsObjectName) }).contains(webMessageListener.jsObjectName) {
            throw NSError(domain: "jsObjectName \(webMessageListener.jsObjectName) was already added.", code: 0)
        }
        try webMessageListener.assertOriginRulesValid()
        webMessageListener.initJsInstance(webView: self)
        webMessageListeners.append(webMessageListener)
    }
    
    public func disposeWebMessageChannels() {
        for webMessageChannel in webMessageChannels.values {
            webMessageChannel.dispose()
        }
        webMessageChannels.removeAll()
    }
    
    // https://stackoverflow.com/a/58001395/4637638
    public override var inputAccessoryView: UIView? {
        return settings?.disableInputAccessoryView ?? false ? nil : super.inputAccessoryView
    }
    
    public func runWindowBeforeCreatedCallbacks() {
        let callbacks = windowBeforeCreatedCallbacks
        callbacks.forEach { (callback) in
            callback()
        }
        windowBeforeCreatedCallbacks.removeAll()
    }
    
    public func dispose() {
        channelDelegate?.dispose()
        channelDelegate = nil
        runWindowBeforeCreatedCallbacks()
        removeObserver(self, forKeyPath: #keyPath(WKWebView.estimatedProgress))
        removeObserver(self, forKeyPath: #keyPath(WKWebView.url))
        removeObserver(self, forKeyPath: #keyPath(WKWebView.title))
        if #available(iOS 15.0, *) {
            removeObserver(self, forKeyPath: #keyPath(WKWebView.cameraCaptureState))
            removeObserver(self, forKeyPath: #keyPath(WKWebView.microphoneCaptureState))
        }
        // TODO: Still not working on iOS 16.0!
//        if #available(iOS 16.0, *) {
//            removeObserver(self, forKeyPath: #keyPath(WKWebView.fullscreenState))
//        }
        scrollView.removeObserver(self, forKeyPath: #keyPath(UIScrollView.contentOffset))
        scrollView.removeObserver(self, forKeyPath: #keyPath(UIScrollView.zoomScale))
        scrollView.removeObserver(self, forKeyPath: #keyPath(UIScrollView.contentSize))
        resumeTimers()
        stopLoading()
        disposeWebMessageChannels()
        for webMessageListener in webMessageListeners {
            webMessageListener.dispose()
        }
        webMessageListeners.removeAll()
        interceptOnlyAsyncAjaxRequestsPluginScript = nil
        if windowId == nil {
            configuration.userContentController.removeAllPluginScriptMessageHandlers()
            configuration.userContentController.removeScriptMessageHandler(forName: "onCallAsyncJavaScriptResultBelowIOS14Received")
            configuration.userContentController.removeScriptMessageHandler(forName: "onWebMessagePortMessageReceived")
            configuration.userContentController.removeScriptMessageHandler(forName: "onWebMessageListenerPostMessageReceived")
            configuration.userContentController.removeAllUserScripts()
            if #available(iOS 11.0, *) {
                configuration.userContentController.removeAllContentRuleLists()
            }
        } else if let wId = windowId, plugin?.inAppWebViewManager?.windowWebViews[wId] != nil {
            plugin?.inAppWebViewManager?.windowWebViews.removeValue(forKey: wId)
        }
        configuration.userContentController.dispose(windowId: windowId)
        NotificationCenter.default.removeObserver(self)
        for imp in customIMPs {
            imp_removeBlock(imp)
        }
        longPressRecognizer.removeTarget(self, action: #selector(longPressGestureDetected))
        longPressRecognizer.delegate = nil
        scrollView.removeGestureRecognizer(longPressRecognizer)
        recognizerForDisablingContextMenuOnLinks.removeTarget(self, action: #selector(longPressGestureDetected))
        recognizerForDisablingContextMenuOnLinks.delegate = nil
        scrollView.removeGestureRecognizer(recognizerForDisablingContextMenuOnLinks)
        panGestureRecognizer.removeTarget(self, action: #selector(endDraggingDetected))
        panGestureRecognizer.delegate = nil
        scrollView.removeGestureRecognizer(panGestureRecognizer)
        disablePullToRefresh()
        pullToRefreshControl?.dispose()
        pullToRefreshControl = nil
        findInteractionController?.dispose()
        findInteractionController = nil
        uiDelegate = nil
        navigationDelegate = nil
        scrollView.delegate = nil
        isPausedTimersCompletionHandler = nil
        SharedLastTouchPointTimestamp.removeValue(forKey: self)
        callAsyncJavaScriptBelowIOS14Results.removeAll()
        plugin = nil
    }
    
    deinit {
        debugPrint("InAppWebView - dealloc")
    }
}
