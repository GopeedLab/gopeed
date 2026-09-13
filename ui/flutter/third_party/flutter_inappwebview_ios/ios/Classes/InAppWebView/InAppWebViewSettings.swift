//
//  InAppWebViewSettings.swift
//  flutter_inappwebview
//
//  Created by Lorenzo on 21/10/18.
//

import Foundation
import WebKit

@objcMembers
public class InAppWebViewSettings: ISettings<InAppWebView> {
    
    var gopeedProfileId = ""
    var useShouldOverrideUrlLoading = false
    var useOnLoadResource = false
    var useOnDownloadStart = false
    @available(*, deprecated, message: "Use InAppWebViewManager.clearAllCache instead.")
    var clearCache = false
    var userAgent = ""
    var applicationNameForUserAgent = ""
    var javaScriptEnabled = true
    var javaScriptCanOpenWindowsAutomatically = false
    var mediaPlaybackRequiresUserGesture = true
    var verticalScrollBarEnabled = true
    var horizontalScrollBarEnabled = true
    var resourceCustomSchemes: [String] = []
    var contentBlockers: [[String: [String : Any]]] = []
    var minimumFontSize = 0
    var useShouldInterceptAjaxRequest = false
    var interceptOnlyAsyncAjaxRequests = true
    var useShouldInterceptFetchRequest = false
    var incognito = false
    var cacheEnabled = true
    var transparentBackground = false
    var disableVerticalScroll = false
    var disableHorizontalScroll = false
    var disableContextMenu = false
    var supportZoom = true
    var allowUniversalAccessFromFileURLs = false
    var allowFileAccessFromFileURLs = false

    var disallowOverScroll = false
    var enableViewportScale = false
    var suppressesIncrementalRendering = false
    var allowsAirPlayForMediaPlayback = true
    var allowsBackForwardNavigationGestures = true
    var allowsLinkPreview = true
    var ignoresViewportScaleLimits = false
    var allowsInlineMediaPlayback = false
    var allowsPictureInPictureMediaPlayback = true
    var isFraudulentWebsiteWarningEnabled = true
    var selectionGranularity = 0
    var dataDetectorTypes: [String] = ["NONE"] // WKDataDetectorTypeNone
    var preferredContentMode = 0
    var sharedCookiesEnabled = false
    var automaticallyAdjustsScrollIndicatorInsets = false
    var accessibilityIgnoresInvertColors = false
    var decelerationRate = "NORMAL" // UIScrollView.DecelerationRate.normal
    var alwaysBounceVertical = false
    var alwaysBounceHorizontal = false
    var scrollsToTop = true
    var isPagingEnabled = false
    var maximumZoomScale = 1.0
    var minimumZoomScale = 1.0
    var contentInsetAdjustmentBehavior = 2 // UIScrollView.ContentInsetAdjustmentBehavior.never
    var isDirectionalLockEnabled = false
    var mediaType: String? = nil
    var pageZoom = 1.0
    var limitsNavigationsToAppBoundDomains = false
    var useOnNavigationResponse = false
    var applePayAPIEnabled = false
    var allowingReadAccessTo: String? = nil
    var disableLongPressContextMenuOnLinks = false
    var disableInputAccessoryView = false
    var underPageBackgroundColor: String?
    var isTextInteractionEnabled = true
    var isSiteSpecificQuirksModeEnabled = true
    var upgradeKnownHostsToHTTPS = true
    var isElementFullscreenEnabled = true
    var isFindInteractionEnabled = false
    var minimumViewportInset: UIEdgeInsets? = nil
    var maximumViewportInset: UIEdgeInsets? = nil
    var isInspectable = false
    var shouldPrintBackgrounds = false
    
    override init(){
        super.init()
    }
    
    override func parse(settings: [String: Any?]) -> InAppWebViewSettings {
        var settings = settings // re-assing to be able to use removeValue
        if let minimumViewportInsetMap = settings["minimumViewportInset"] as? [String : Double] {
            minimumViewportInset = UIEdgeInsets.fromMap(map: minimumViewportInsetMap)
            settings.removeValue(forKey: "minimumViewportInset")
        }
        if let maximumViewportInsetMap = settings["maximumViewportInset"] as? [String : Double] {
            maximumViewportInset = UIEdgeInsets.fromMap(map: maximumViewportInsetMap)
            settings.removeValue(forKey: "maximumViewportInset")
        }
        let _ = super.parse(settings: settings)
        if #available(iOS 13.0, *) {} else {
            applePayAPIEnabled = false
        }
        return self
    }
    
    override func getRealSettings(obj: InAppWebView?) -> [String: Any?] {
        var realSettings: [String: Any?] = toMap()
        if let webView = obj {
            let configuration = webView.configuration
            if #available(iOS 9.0, *) {
                realSettings["userAgent"] = webView.customUserAgent
                realSettings["applicationNameForUserAgent"] = configuration.applicationNameForUserAgent
                realSettings["allowsAirPlayForMediaPlayback"] = configuration.allowsAirPlayForMediaPlayback
                realSettings["allowsLinkPreview"] = webView.allowsLinkPreview
                realSettings["allowsPictureInPictureMediaPlayback"] = configuration.allowsPictureInPictureMediaPlayback
            }
            realSettings["javaScriptCanOpenWindowsAutomatically"] = configuration.preferences.javaScriptCanOpenWindowsAutomatically
            if #available(iOS 10.0, *) {
                realSettings["mediaPlaybackRequiresUserGesture"] = configuration.mediaTypesRequiringUserActionForPlayback == .all
                realSettings["ignoresViewportScaleLimits"] = configuration.ignoresViewportScaleLimits
                realSettings["dataDetectorTypes"] = Util.getDataDetectorTypeString(type: configuration.dataDetectorTypes)
            } else {
                realSettings["mediaPlaybackRequiresUserGesture"] = configuration.mediaPlaybackRequiresUserAction
            }
            realSettings["minimumFontSize"] = Int(configuration.preferences.minimumFontSize)
            realSettings["suppressesIncrementalRendering"] = configuration.suppressesIncrementalRendering
            realSettings["allowsBackForwardNavigationGestures"] = webView.allowsBackForwardNavigationGestures
            realSettings["allowsInlineMediaPlayback"] = configuration.allowsInlineMediaPlayback
            if #available(iOS 13.0, *) {
                realSettings["isFraudulentWebsiteWarningEnabled"] = configuration.preferences.isFraudulentWebsiteWarningEnabled
                realSettings["preferredContentMode"] = configuration.defaultWebpagePreferences.preferredContentMode.rawValue
                realSettings["automaticallyAdjustsScrollIndicatorInsets"] = webView.scrollView.automaticallyAdjustsScrollIndicatorInsets
            }
            realSettings["selectionGranularity"] = configuration.selectionGranularity.rawValue
            if #available(iOS 11.0, *) {
                realSettings["accessibilityIgnoresInvertColors"] = webView.accessibilityIgnoresInvertColors
                realSettings["contentInsetAdjustmentBehavior"] = webView.scrollView.contentInsetAdjustmentBehavior.rawValue
            }
            realSettings["decelerationRate"] = Util.getDecelerationRateString(type: webView.scrollView.decelerationRate)
            realSettings["alwaysBounceVertical"] = webView.scrollView.alwaysBounceVertical
            realSettings["alwaysBounceHorizontal"] = webView.scrollView.alwaysBounceHorizontal
            realSettings["scrollsToTop"] = webView.scrollView.scrollsToTop
            realSettings["isPagingEnabled"] = webView.scrollView.isPagingEnabled
            realSettings["maximumZoomScale"] = webView.scrollView.maximumZoomScale
            realSettings["minimumZoomScale"] = webView.scrollView.minimumZoomScale
            realSettings["allowUniversalAccessFromFileURLs"] = configuration.value(forKey: "allowUniversalAccessFromFileURLs")
            realSettings["allowFileAccessFromFileURLs"] = configuration.preferences.value(forKey: "allowFileAccessFromFileURLs")
            realSettings["isDirectionalLockEnabled"] = webView.scrollView.isDirectionalLockEnabled
            realSettings["javaScriptEnabled"] = configuration.preferences.javaScriptEnabled
            if #available(iOS 14.0, *) {
                realSettings["mediaType"] = webView.mediaType
                realSettings["pageZoom"] = Float(webView.pageZoom)
                realSettings["limitsNavigationsToAppBoundDomains"] = configuration.limitsNavigationsToAppBoundDomains
                realSettings["javaScriptEnabled"] = configuration.defaultWebpagePreferences.allowsContentJavaScript
            }
            if #available(iOS 15.0, *) {
                realSettings["isTextInteractionEnabled"] = configuration.preferences.isTextInteractionEnabled
                realSettings["upgradeKnownHostsToHTTPS"] = configuration.upgradeKnownHostsToHTTPS
                realSettings["underPageBackgroundColor"] = webView.underPageBackgroundColor.hexString
            }
            if #available(iOS 15.4, *) {
                realSettings["isSiteSpecificQuirksModeEnabled"] = configuration.preferences.isSiteSpecificQuirksModeEnabled
                realSettings["isElementFullscreenEnabled"] = configuration.preferences.isElementFullscreenEnabled
            }
            if #available(iOS 15.5, *) {
                realSettings["minimumViewportInset"] = webView.minimumViewportInset.toMap()
                realSettings["maximumViewportInset"] = webView.maximumViewportInset.toMap()
            }
            if #available(iOS 16.0, *) {
                realSettings["isFindInteractionEnabled"] = webView.isFindInteractionEnabled
            }
            if #available(iOS 16.4, *) {
                realSettings["isInspectable"] = webView.isInspectable
                realSettings["shouldPrintBackgrounds"] = configuration.preferences.shouldPrintBackgrounds
            }
        }
        return realSettings
    }
}
