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
    var resourceCustomSchemes: [String] = []
    var contentBlockers: [[String: [String : Any]]] = []
    var minimumFontSize = 0
    var useShouldInterceptAjaxRequest = false
    var interceptOnlyAsyncAjaxRequests = true
    var useShouldInterceptFetchRequest = false
    var incognito = false
    var cacheEnabled = true
    var transparentBackground = false
    var supportZoom = true
    var allowUniversalAccessFromFileURLs = false
    var allowFileAccessFromFileURLs = false

    var enableViewportScale = false
    var suppressesIncrementalRendering = false
    var allowsAirPlayForMediaPlayback = true
    var allowsBackForwardNavigationGestures = true
    var allowsLinkPreview = true
    var isFraudulentWebsiteWarningEnabled = true
    var preferredContentMode = 0
    var sharedCookiesEnabled = false
    var mediaType: String? = nil
    var pageZoom = 1.0
    var limitsNavigationsToAppBoundDomains = false
    var useOnNavigationResponse = false
    var applePayAPIEnabled = false
    var allowingReadAccessTo: String? = nil
    var underPageBackgroundColor: String?
    var isTextInteractionEnabled = true
    var isSiteSpecificQuirksModeEnabled = true
    var upgradeKnownHostsToHTTPS = true
    var isElementFullscreenEnabled = true
    var isInspectable = false
    var shouldPrintBackgrounds = false
    
    override init(){
        super.init()
    }
    
    override func parse(settings: [String: Any?]) -> InAppWebViewSettings {
        let _ = super.parse(settings: settings)
        if #available(macOS 10.15, *) {} else {
            applePayAPIEnabled = false
        }
        return self
    }
    
    override func getRealSettings(obj: InAppWebView?) -> [String: Any?] {
        var realSettings: [String: Any?] = toMap()
        if let webView = obj {
            let configuration = webView.configuration
            realSettings["userAgent"] = webView.customUserAgent
            realSettings["applicationNameForUserAgent"] = configuration.applicationNameForUserAgent
            realSettings["allowsAirPlayForMediaPlayback"] = configuration.allowsAirPlayForMediaPlayback
            realSettings["allowsLinkPreview"] = webView.allowsLinkPreview
            realSettings["javaScriptCanOpenWindowsAutomatically"] = configuration.preferences.javaScriptCanOpenWindowsAutomatically
            if #available(macOS 10.12, *) {
                realSettings["mediaPlaybackRequiresUserGesture"] = configuration.mediaTypesRequiringUserActionForPlayback == .all
            }
            realSettings["minimumFontSize"] = Int(configuration.preferences.minimumFontSize)
            realSettings["suppressesIncrementalRendering"] = configuration.suppressesIncrementalRendering
            realSettings["allowsBackForwardNavigationGestures"] = webView.allowsBackForwardNavigationGestures
            if #available(macOS 10.15, *) {
                realSettings["isFraudulentWebsiteWarningEnabled"] = configuration.preferences.isFraudulentWebsiteWarningEnabled
                realSettings["preferredContentMode"] = configuration.defaultWebpagePreferences.preferredContentMode.rawValue
            }
            realSettings["allowUniversalAccessFromFileURLs"] = configuration.value(forKey: "allowUniversalAccessFromFileURLs") as? Bool
            realSettings["allowFileAccessFromFileURLs"] = configuration.preferences.value(forKey: "allowFileAccessFromFileURLs") as? Bool
            realSettings["javaScriptEnabled"] = configuration.preferences.javaScriptEnabled
            if #available(macOS 11.0, *) {
                realSettings["mediaType"] = webView.mediaType
                realSettings["pageZoom"] = Float(webView.pageZoom)
                realSettings["limitsNavigationsToAppBoundDomains"] = configuration.limitsNavigationsToAppBoundDomains
                realSettings["javaScriptEnabled"] = configuration.defaultWebpagePreferences.allowsContentJavaScript
            }
            if #available(macOS 11.3, *) {
                realSettings["isTextInteractionEnabled"] = configuration.preferences.isTextInteractionEnabled
                realSettings["upgradeKnownHostsToHTTPS"] = configuration.upgradeKnownHostsToHTTPS
            }
            if #available(macOS 12.0, *) {
                realSettings["underPageBackgroundColor"] = webView.underPageBackgroundColor.hexString
            }
            if #available(macOS 12.3, *) {
                realSettings["isSiteSpecificQuirksModeEnabled"] = configuration.preferences.isSiteSpecificQuirksModeEnabled
                realSettings["isElementFullscreenEnabled"] = configuration.preferences.isElementFullscreenEnabled
            }
            if #available(macOS 13.3, *) {
                realSettings["isInspectable"] = webView.isInspectable
                realSettings["shouldPrintBackgrounds"] = configuration.preferences.shouldPrintBackgrounds
            }
        }
        return realSettings
    }
}
