//
//  WebViewChannelDelegate.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 06/05/22.
//

import Foundation
import WebKit

public class WebViewChannelDelegate: ChannelDelegate {
    private weak var webView: InAppWebView?
    
    public init(webView: InAppWebView, channel: FlutterMethodChannel) {
        super.init(channel: channel)
        self.webView = webView
    }
    
    public override func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
        let arguments = call.arguments as? NSDictionary
        
        guard let method = WebViewChannelDelegateMethods.init(rawValue: call.method) else {
            result(FlutterMethodNotImplemented)
            return
        }
        
        switch method {
        case .getUrl:
            result(webView?.url?.absoluteString)
            break
        case .getTitle:
            result(webView?.title)
            break
        case .getProgress:
            result( (webView != nil) ? Int(webView!.estimatedProgress * 100) : nil )
            break
        case .loadUrl:
            let urlRequest = arguments!["urlRequest"] as! [String:Any?]
            let allowingReadAccessTo = arguments!["allowingReadAccessTo"] as? String
            var allowingReadAccessToURL: URL? = nil
            if let allowingReadAccessTo = allowingReadAccessTo {
                allowingReadAccessToURL = URL(string: allowingReadAccessTo)
            }
            webView?.loadUrl(urlRequest: URLRequest.init(fromPluginMap: urlRequest), allowingReadAccessTo: allowingReadAccessToURL)
            result(true)
            break
        case .postUrl:
            if let webView = webView {
                let url = arguments!["url"] as! String
                let postData = arguments!["postData"] as! FlutterStandardTypedData
                webView.postUrl(url: URL(string: url)!, postData: postData.data)
            }
            result(true)
            break
        case .loadData:
            let data = arguments!["data"] as! String
            let mimeType = arguments!["mimeType"] as! String
            let encoding = arguments!["encoding"] as! String
            let baseUrl = URL(string: arguments!["baseUrl"] as! String)!
            let allowingReadAccessTo = arguments!["allowingReadAccessTo"] as? String
            var allowingReadAccessToURL: URL? = nil
            if let allowingReadAccessTo = allowingReadAccessTo {
                allowingReadAccessToURL = URL(string: allowingReadAccessTo)
            }
            webView?.loadData(data: data, mimeType: mimeType, encoding: encoding, baseUrl: baseUrl, allowingReadAccessTo: allowingReadAccessToURL)
            result(true)
            break
        case .loadFile:
            let assetFilePath = arguments!["assetFilePath"] as! String
            
            do {
                try webView?.loadFile(assetFilePath: assetFilePath)
            }
            catch let error as NSError {
                result(FlutterError(code: "WebViewChannelDelegate", message: error.domain, details: nil))
                return
            }
            result(true)
            break
        case .evaluateJavascript:
            if let webView = webView {
                let source = arguments!["source"] as! String
                let contentWorldMap = arguments!["contentWorld"] as? [String:Any?]
                if #available(iOS 14.0, *), let contentWorldMap = contentWorldMap {
                    let contentWorld = WKContentWorld.fromMap(map: contentWorldMap, windowId: webView.windowId)!
                    webView.evaluateJavascript(source: source, contentWorld: contentWorld) { (value) in
                        result(value)
                    }
                } else {
                    webView.evaluateJavascript(source: source) { (value) in
                        result(value)
                    }
                }
            }
            else {
                result(nil)
            }
            break
        case .injectJavascriptFileFromUrl:
            let urlFile = arguments!["urlFile"] as! String
            let scriptHtmlTagAttributes = arguments!["scriptHtmlTagAttributes"] as? [String:Any?]
            webView?.injectJavascriptFileFromUrl(urlFile: urlFile, scriptHtmlTagAttributes: scriptHtmlTagAttributes)
            result(true)
            break
        case .injectCSSCode:
            let source = arguments!["source"] as! String
            webView?.injectCSSCode(source: source)
            result(true)
            break
        case .injectCSSFileFromUrl:
            let urlFile = arguments!["urlFile"] as! String
            let cssLinkHtmlTagAttributes = arguments!["cssLinkHtmlTagAttributes"] as? [String:Any?]
            webView?.injectCSSFileFromUrl(urlFile: urlFile, cssLinkHtmlTagAttributes: cssLinkHtmlTagAttributes)
            result(true)
            break
        case .reload:
            webView?.reload()
            result(true)
            break
        case .goBack:
            webView?.goBack()
            result(true)
            break
        case .canGoBack:
            result(webView?.canGoBack ?? false)
            break
        case .goForward:
            webView?.goForward()
            result(true)
            break
        case .canGoForward:
            result(webView?.canGoForward ?? false)
            break
        case .goBackOrForward:
            let steps = arguments!["steps"] as! Int
            webView?.goBackOrForward(steps: steps)
            result(true)
            break
        case .canGoBackOrForward:
            let steps = arguments!["steps"] as! Int
            result(webView?.canGoBackOrForward(steps: steps) ?? false)
            break
        case .stopLoading:
            webView?.stopLoading()
            result(true)
            break
        case .isLoading:
            result(webView?.isLoading ?? false)
            break
        case .takeScreenshot:
            if let webView = webView, #available(iOS 11.0, *) {
                let screenshotConfiguration = arguments!["screenshotConfiguration"] as? [String: Any?]
                webView.takeScreenshot(with: screenshotConfiguration, completionHandler: { (screenshot) -> Void in
                    result(screenshot)
                })
            }
            else {
                result(nil)
            }
            break
        case .setSettings:
            if let iabController = webView?.inAppBrowserDelegate as? InAppBrowserWebViewController {
                let inAppBrowserSettings = InAppBrowserSettings()
                let inAppBrowserSettingsMap = arguments!["settings"] as! [String: Any]
                let _ = inAppBrowserSettings.parse(settings: inAppBrowserSettingsMap)
                iabController.setSettings(newSettings: inAppBrowserSettings, newSettingsMap: inAppBrowserSettingsMap)
            } else {
                let inAppWebViewSettings = InAppWebViewSettings()
                let inAppWebViewSettingsMap = arguments!["settings"] as! [String: Any]
                let _ = inAppWebViewSettings.parse(settings: inAppWebViewSettingsMap)
                webView?.setSettings(newSettings: inAppWebViewSettings, newSettingsMap: inAppWebViewSettingsMap)
            }
            result(true)
            break
        case .getSettings:
            if let iabController = webView?.inAppBrowserDelegate as? InAppBrowserWebViewController {
                result(iabController.getSettings())
            } else {
                result(webView?.getSettings())
            }
            break
        case .close:
            if let iabController = webView?.inAppBrowserDelegate as? InAppBrowserWebViewController {
                iabController.close {
                    result(true)
                }
            } else {
                result(FlutterMethodNotImplemented)
            }
            break
        case .show:
            if let iabController = webView?.inAppBrowserDelegate as? InAppBrowserWebViewController {
                iabController.show {
                    result(true)
                }
            } else {
                result(FlutterMethodNotImplemented)
            }
            break
        case .hide:
            if let iabController = webView?.inAppBrowserDelegate as? InAppBrowserWebViewController {
                iabController.hide {
                    result(true)
                }
            } else {
                result(FlutterMethodNotImplemented)
            }
            break
        case .isHidden:
            if let iabController = webView?.inAppBrowserDelegate as? InAppBrowserWebViewController {
                result(iabController.isHidden)
            } else {
                result(FlutterMethodNotImplemented)
            }
            break
        case .getCopyBackForwardList:
            result(webView?.getCopyBackForwardList())
            break
        case .findAll:
            if let webView = webView, let findInteractionController = webView.findInteractionController {
                let find = arguments!["find"] as! String
                findInteractionController.findAll(find: find, completionHandler: {(value, error) in
                    if error != nil {
                        result(FlutterError(code: "WebViewChannelDelegate", message: error?.localizedDescription, details: nil))
                        return
                    }
                    result(true)
                })
            } else {
                result(false)
            }
            break
        case .findNext:
            if let webView = webView, let findInteractionController = webView.findInteractionController {
                let forward = arguments!["forward"] as! Bool
                findInteractionController.findNext(forward: forward, completionHandler: {(value, error) in
                    if error != nil {
                        result(FlutterError(code: "WebViewChannelDelegate", message: error?.localizedDescription, details: nil))
                        return
                    }
                    result(true)
                })
            } else {
                result(false)
            }
            break
        case .clearMatches:
            if let webView = webView, let findInteractionController = webView.findInteractionController {
                findInteractionController.clearMatches(completionHandler: {(value, error) in
                    if error != nil {
                        result(FlutterError(code: "WebViewChannelDelegate", message: error?.localizedDescription, details: nil))
                        return
                    }
                    result(true)
                })
            } else {
                result(false)
            }
            break
        case .clearCache:
            webView?.clearCache()
            result(true)
            break
        case .scrollTo:
            let x = arguments!["x"] as! Int
            let y = arguments!["y"] as! Int
            let animated = arguments!["animated"] as! Bool
            webView?.scrollTo(x: x, y: y, animated: animated)
            result(true)
            break
        case .scrollBy:
            let x = arguments!["x"] as! Int
            let y = arguments!["y"] as! Int
            let animated = arguments!["animated"] as! Bool
            webView?.scrollBy(x: x, y: y, animated: animated)
            result(true)
            break
        case .pauseTimers:
            webView?.pauseTimers()
            result(true)
            break
        case .resumeTimers:
            webView?.resumeTimers()
            result(true)
            break
        case .printCurrentPage:
            if let webView = webView {
                let settings = PrintJobSettings()
                if let settingsMap = arguments!["settings"] as? [String: Any?] {
                    let _ = settings.parse(settings: settingsMap)
                }
                result(webView.printCurrentPage(settings: settings))
            } else {
                result(nil)
            }
            break
        case .getContentHeight:
            result(webView?.getContentHeight())
            break
        case .getContentWidth:
            result(webView?.getContentWidth())
            break
        case .zoomBy:
            let zoomFactor = (arguments!["zoomFactor"] as! NSNumber).floatValue
            let animated = arguments!["animated"] as! Bool
            webView?.zoomBy(zoomFactor: zoomFactor, animated: animated)
            result(true)
            break
        case .reloadFromOrigin:
            webView?.reloadFromOrigin()
            result(true)
            break
        case .getOriginalUrl:
            result(webView?.getOriginalUrl()?.absoluteString)
            break
        case .getZoomScale:
            result(webView?.getZoomScale())
            break
        case .hasOnlySecureContent:
            result(webView?.hasOnlySecureContent ?? false)
            break
        case .getSelectedText:
            if let webView = webView {
                webView.getSelectedText { (value, error) in
                    if let err = error {
                        print(err.localizedDescription)
                        result("")
                        return
                    }
                    result(value)
                }
            }
            else {
                result(nil)
            }
            break
        case .getHitTestResult:
            if let webView = webView {
                webView.getHitTestResult { (hitTestResult) in
                    result(hitTestResult.toMap())
                }
            }
            else {
                result(nil)
            }
            break
        case .clearFocus:
            webView?.clearFocus()
            result(true)
            break
        case .setContextMenu:
            if let webView = webView {
                let contextMenu = arguments!["contextMenu"] as? [String: Any]
                webView.contextMenu = contextMenu
                result(true)
            } else {
                result(false)
            }
            break
        case .requestFocusNodeHref:
            if let webView = webView {
                webView.requestFocusNodeHref { (value, error) in
                    if let err = error {
                        print(err.localizedDescription)
                        result(nil)
                        return
                    }
                    result(value)
                }
            } else {
                result(nil)
            }
            break
        case .requestImageRef:
            if let webView = webView {
                webView.requestImageRef { (value, error) in
                    if let err = error {
                        print(err.localizedDescription)
                        result(nil)
                        return
                    }
                    result(value)
                }
            } else {
                result(nil)
            }
            break
        case .getScrollX:
            if let webView = webView {
                result(Int(webView.scrollView.contentOffset.x))
            } else {
                result(nil)
            }
            break
        case .getScrollY:
            if let webView = webView {
                result(Int(webView.scrollView.contentOffset.y))
            } else {
                result(nil)
            }
            break
        case .getCertificate:
            result(webView?.getCertificate()?.toMap())
            break
        case .addUserScript:
            if let webView = webView {
                let userScriptMap = arguments!["userScript"] as! [String: Any?]
                let userScript = UserScript.fromMap(map: userScriptMap, windowId: webView.windowId)!
                webView.configuration.userContentController.addUserOnlyScript(userScript)
                webView.configuration.userContentController.sync(scriptMessageHandler: webView)
            }
            result(true)
            break
        case .removeUserScript:
            let index = arguments!["index"] as! Int
            let userScriptMap = arguments!["userScript"] as! [String: Any?]
            let userScript = UserScript.fromMap(map: userScriptMap, windowId: webView?.windowId)!
            webView?.configuration.userContentController.removeUserOnlyScript(at: index, injectionTime: userScript.injectionTime)
            result(true)
            break
        case .removeUserScriptsByGroupName:
            let groupName = arguments!["groupName"] as! String
            webView?.configuration.userContentController.removeUserOnlyScripts(with: groupName)
            result(true)
            break
        case .removeAllUserScripts:
            webView?.configuration.userContentController.removeAllUserOnlyScripts()
            result(true)
            break
        case .callAsyncJavaScript:
            if let webView = webView, #available(iOS 10.3, *) {
                if #available(iOS 14.3, *) { // on iOS 14.0, for some reason, it crashes
                    let functionBody = arguments!["functionBody"] as! String
                    let functionArguments = arguments!["arguments"] as! [String:Any]
                    var contentWorld = WKContentWorld.page
                    if let contentWorldMap = arguments!["contentWorld"] as? [String:Any?] {
                        contentWorld = WKContentWorld.fromMap(map: contentWorldMap, windowId: webView.windowId)!
                    }
                    webView.callAsyncJavaScript(functionBody: functionBody, arguments: functionArguments, contentWorld: contentWorld) { (value) in
                        result(value)
                    }
                } else {
                    let functionBody = arguments!["functionBody"] as! String
                    let functionArguments = arguments!["arguments"] as! [String:Any]
                    webView.callAsyncJavaScript(functionBody: functionBody, arguments: functionArguments) { (value) in
                        result(value)
                    }
                }
            }
            else {
                result(nil)
            }
            break
        case .createPdf:
            if let webView = webView, #available(iOS 14.0, *) {
                let configuration = arguments!["pdfConfiguration"] as? [String: Any?]
                webView.createPdf(configuration: configuration, completionHandler: { (pdf) -> Void in
                    result(pdf)
                })
            }
            else {
                result(nil)
            }
            break
        case .createWebArchiveData:
            if let webView = webView, #available(iOS 14.0, *) {
                webView.createWebArchiveData(dataCompletionHandler: { (webArchiveData) -> Void in
                    result(webArchiveData)
                })
            }
            else {
                result(nil)
            }
            break
        case .saveWebArchive:
            if let webView = webView, #available(iOS 14.0, *) {
                let filePath = arguments!["filePath"] as! String
                let autoname = arguments!["autoname"] as! Bool
                webView.saveWebArchive(filePath: filePath, autoname: autoname, completionHandler: { (path) -> Void in
                    result(path)
                })
            }
            else {
                result(nil)
            }
            break
        case .isSecureContext:
            if let webView = webView {
                webView.isSecureContext(completionHandler: { (isSecureContext) in
                    result(isSecureContext)
                })
            }
            else {
                result(false)
            }
            break
        case .createWebMessageChannel:
            if let webView = webView {
                let _ = webView.createWebMessageChannel { (webMessageChannel) in
                    guard let webMessageChannel = webMessageChannel else {
                        result(nil)
                        return
                    }
                    result(webMessageChannel.toMap())
                }
            } else {
                result(nil)
            }
            break
        case .postWebMessage:
            if let webView = webView {
                let message = WebMessage.fromMap(map: arguments!["message"] as! [String: Any?])
                let targetOrigin = arguments!["targetOrigin"] as! String
                
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
                    try webView.postWebMessage(message: message, targetOrigin: targetOrigin) { (_) in
                        result(true)
                    }
                } catch let error as NSError {
                    result(FlutterError(code: "WebViewChannelDelegate", message: error.domain, details: nil))
                }
            } else {
                result(false)
            }
            break
        case .addWebMessageListener:
            if let webView = webView, let plugin = webView.plugin {
                let webMessageListenerMap = arguments!["webMessageListener"] as! [String: Any?]
                let webMessageListener = WebMessageListener.fromMap(plugin: plugin, map: webMessageListenerMap)!
                do {
                    try webView.addWebMessageListener(webMessageListener: webMessageListener)
                    result(false)
                } catch let error as NSError {
                    result(FlutterError(code: "WebViewChannelDelegate", message: error.domain, details: nil))
                }
            } else {
                result(false)
            }
            break
        case .canScrollVertically:
            if let webView = webView {
                result(webView.canScrollVertically())
            } else {
                result(false)
            }
            break
        case .canScrollHorizontally:
            if let webView = webView {
                result(webView.canScrollHorizontally())
            } else {
                result(false)
            }
            break
        case .pauseAllMediaPlayback:
            if let webView = webView, #available(iOS 15.0, *) {
                webView.pauseAllMediaPlayback(completionHandler: { () -> Void in
                    result(true)
                })
            } else {
                result(false)
            }
            break
        case .setAllMediaPlaybackSuspended:
            if let webView = webView, #available(iOS 15.0, *) {
                let suspended = arguments!["suspended"] as! Bool
                webView.setAllMediaPlaybackSuspended(suspended, completionHandler: { () -> Void in
                    result(true)
                })
            } else {
                result(false)
            }
            break
        case .closeAllMediaPresentations:
            if let webView = self.webView, #available(iOS 14.5, *) {
                // closeAllMediaPresentations with completionHandler v15.0 makes the app crash
                // with error EXC_BAD_ACCESS, so use closeAllMediaPresentations v14.5
                if #available(iOS 16.0, *) {
                    webView.closeAllMediaPresentations {
                        result(true)
                    }
                } else {
                    webView.closeAllMediaPresentations()
                    result(true)
                }
            } else {
                result(false)
            }
            break
        case .requestMediaPlaybackState:
            if let webView = webView, #available(iOS 15.0, *) {
                webView.requestMediaPlaybackState(completionHandler: { (state) -> Void in
                    result(state.rawValue)
                })
            } else {
                result(nil)
            }
            break
        case .getMetaThemeColor:
            if let webView = webView, #available(iOS 15.0, *) {
                result(webView.themeColor?.hexString)
            } else {
                result(nil)
            }
            break
        case .isInFullscreen:
            if let webView = webView {
                if #available(iOS 16.0, *) {
                    result(webView.fullscreenState == .inFullscreen)
                } else {
                    result(webView.inFullscreen)
                }
            }
            else {
                result(false)
            }
            break
        case .getCameraCaptureState:
            if let webView = webView, #available(iOS 15.0, *) {
                result(webView.cameraCaptureState.rawValue)
            } else {
                result(nil)
            }
            break
        case .setCameraCaptureState:
            if let webView = webView, #available(iOS 15.0, *) {
                let state = WKMediaCaptureState.init(rawValue: arguments!["state"] as! Int) ?? WKMediaCaptureState.none
                webView.setCameraCaptureState(state) {
                    result(true)
                }
            } else {
                result(false)
            }
            break
        case .getMicrophoneCaptureState:
            if let webView = webView, #available(iOS 15.0, *) {
                result(webView.microphoneCaptureState.rawValue)
            } else {
                result(nil)
            }
            break
        case .setMicrophoneCaptureState:
            if let webView = webView, #available(iOS 15.0, *) {
                let state = WKMediaCaptureState.init(rawValue: arguments!["state"] as! Int) ?? WKMediaCaptureState.none
                webView.setMicrophoneCaptureState(state) {
                    result(true)
                }
            } else {
                result(false)
            }
            break
        case .loadSimulatedRequest:
            if let webView = webView, #available(iOS 15.0, *) {
                let request = URLRequest.init(fromPluginMap: arguments!["urlRequest"] as! [String:Any?])
                let data = arguments!["data"] as! FlutterStandardTypedData
                var response: URLResponse? = nil
                if let urlResponse = arguments!["urlResponse"] as? [String:Any?] {
                    response = URLResponse.init(fromPluginMap: urlResponse)
                }
                if let response = response {
                    webView.loadSimulatedRequest(request, response: response, responseData: data.data)
                } else {
                    webView.loadSimulatedRequest(request, responseHTML: String(decoding: data.data, as: UTF8.self))
                }
                result(true)
            } else {
                result(false)
            }
        }
    }
    
    @available(*, deprecated, message: "Use FindInteractionChannelDelegate.onFindResultReceived instead.")
    public func onFindResultReceived(activeMatchOrdinal: Int, numberOfMatches: Int, isDoneCounting: Bool) {
        let arguments: [String : Any?] = [
            "activeMatchOrdinal": activeMatchOrdinal,
            "numberOfMatches": numberOfMatches,
            "isDoneCounting": isDoneCounting
        ]
        channel?.invokeMethod("onFindResultReceived", arguments: arguments)
    }
    
    public func onLongPressHitTestResult(hitTestResult: HitTestResult) {
        channel?.invokeMethod("onLongPressHitTestResult", arguments: hitTestResult.toMap())
    }
    
    public func onScrollChanged(x: Int, y: Int) {
        let arguments: [String: Any?] = ["x": x, "y": y]
        channel?.invokeMethod("onScrollChanged", arguments: arguments)
    }
    
    public func onContentSizeChanged(oldContentSize: CGSize, newContentSize: CGSize) {
        let arguments: [String: Any?] = [
            "oldContentSize": oldContentSize.toMap(),
            "newContentSize": newContentSize.toMap()
        ]
        channel?.invokeMethod("onContentSizeChanged", arguments: arguments)
    }
    
    public func onDownloadStartRequest(request: DownloadStartRequest) {
        channel?.invokeMethod("onDownloadStartRequest", arguments: request.toMap())
    }
    
    public func onCreateContextMenu(hitTestResult: HitTestResult) {
        channel?.invokeMethod("onCreateContextMenu", arguments: hitTestResult.toMap())
    }
    
    public func onOverScrolled(x: Int, y: Int, clampedX: Bool, clampedY: Bool) {
        let arguments: [String: Any?] = ["x": x, "y": y, "clampedX": clampedX, "clampedY": clampedY]
        channel?.invokeMethod("onOverScrolled", arguments: arguments)
    }
    
    public func onContextMenuActionItemClicked(id: Any, title: String) {
        let arguments: [String: Any?] = [
            "id": id,
            "iosId": id is Int64 ? String(id as! Int64) : id as! String,
            "androidId": nil,
            "title": title
        ]
        channel?.invokeMethod("onContextMenuActionItemClicked", arguments: arguments)
    }
    
    public func onHideContextMenu() {
        let arguments: [String: Any?] = [:]
        channel?.invokeMethod("onHideContextMenu", arguments: arguments)
    }
    
    public func onEnterFullscreen() {
        let arguments: [String: Any?] = [:]
        channel?.invokeMethod("onEnterFullscreen", arguments: arguments)
    }
    
    public func onExitFullscreen() {
        let arguments: [String: Any?] = [:]
        channel?.invokeMethod("onExitFullscreen", arguments: arguments)
    }
    
    public class JsAlertCallback: BaseCallbackResult<JsAlertResponse> {
        override init() {
            super.init()
            self.decodeResult = { (obj: Any?) in
                return JsAlertResponse.fromMap(map: obj as? [String:Any?])
            }
        }
        
        deinit {
            self.defaultBehaviour(nil)
        }
    }
    
    public func onJsAlert(url: URL?, message: String, isMainFrame: Bool, callback: JsAlertCallback) {
        if channel == nil {
            callback.defaultBehaviour(nil)
            return
        }
        let arguments: [String: Any?] = [
            "url": url?.absoluteString,
            "message": message,
            "isMainFrame": isMainFrame
        ]
        channel?.invokeMethod("onJsAlert", arguments: arguments, callback: callback)
    }
    
    public class JsConfirmCallback: BaseCallbackResult<JsConfirmResponse> {
        override init() {
            super.init()
            self.decodeResult = { (obj: Any?) in
                return JsConfirmResponse.fromMap(map: obj as? [String:Any?])
            }
        }
        
        deinit {
            self.defaultBehaviour(nil)
        }
    }
    
    public func onJsConfirm(url: URL?, message: String, isMainFrame: Bool, callback: JsConfirmCallback) {
        if channel == nil {
            callback.defaultBehaviour(nil)
            return
        }
        let arguments: [String: Any?] = [
            "url": url?.absoluteString,
            "message": message,
            "isMainFrame": isMainFrame
        ]
        channel?.invokeMethod("onJsConfirm", arguments: arguments, callback: callback)
    }
    
    public class JsPromptCallback: BaseCallbackResult<JsPromptResponse> {
        override init() {
            super.init()
            self.decodeResult = { (obj: Any?) in
                return JsPromptResponse.fromMap(map: obj as? [String:Any?])
            }
        }
        
        deinit {
            self.defaultBehaviour(nil)
        }
    }
    
    public func onJsPrompt(url: URL?, message: String, defaultValue: String?, isMainFrame: Bool, callback: JsPromptCallback) {
        if channel == nil {
            callback.defaultBehaviour(nil)
            return
        }
        let arguments: [String: Any?] = [
            "url": url?.absoluteString,
            "message": message,
            "defaultValue": defaultValue,
            "isMainFrame": isMainFrame
        ]
        channel?.invokeMethod("onJsPrompt", arguments: arguments, callback: callback)
    }
    
    public class CreateWindowCallback: BaseCallbackResult<Bool> {
        override init() {
            super.init()
            self.decodeResult = { (obj: Any?) in
                return obj is Bool && obj as! Bool
            }
        }
    }
    
    public func onCreateWindow(createWindowAction: CreateWindowAction, callback: CreateWindowCallback) {
        if channel == nil {
            callback.defaultBehaviour(nil)
            return
        }
        channel?.invokeMethod("onCreateWindow", arguments: createWindowAction.toMap(), callback: callback)
    }
    
    public func onCloseWindow() {
        let arguments: [String: Any?] = [:]
        channel?.invokeMethod("onCloseWindow", arguments: arguments)
    }
    
    public func onConsoleMessage(message: String, messageLevel: Int) {
        let arguments: [String: Any?] = [
            "message": message,
            "messageLevel": messageLevel
        ]
        channel?.invokeMethod("onConsoleMessage", arguments: arguments)
    }
    
    public func onProgressChanged(progress: Int) {
        let arguments: [String: Any?] = [
            "progress": progress
        ]
        channel?.invokeMethod("onProgressChanged", arguments: arguments)
    }
    
    public func onTitleChanged(title: String?) {
        let arguments: [String: Any?] = [
            "title": title
        ]
        channel?.invokeMethod("onTitleChanged", arguments: arguments)
    }
    
    public class PermissionRequestCallback: BaseCallbackResult<PermissionResponse> {
        override init() {
            super.init()
            self.decodeResult = { (obj: Any?) in
                return PermissionResponse.fromMap(map: obj as? [String:Any?])
            }
        }
        
        deinit {
            self.defaultBehaviour(nil)
        }
    }
    
    public func onPermissionRequest(request: PermissionRequest, callback: PermissionRequestCallback) {
        if channel == nil {
            callback.defaultBehaviour(nil)
            return
        }
        channel?.invokeMethod("onPermissionRequest", arguments: request.toMap(), callback: callback)
    }
    
    public class ShouldOverrideUrlLoadingCallback: BaseCallbackResult<WKNavigationActionPolicy> {
        override init() {
            super.init()
            self.decodeResult = { (obj: Any?) in
                if let action = obj as? Int {
                    return WKNavigationActionPolicy.init(rawValue: action) ?? WKNavigationActionPolicy.cancel
                }
                return WKNavigationActionPolicy.cancel
            }
        }
        
        deinit {
            self.defaultBehaviour(nil)
        }
    }
    
    public func shouldOverrideUrlLoading(navigationAction: WKNavigationAction, callback: ShouldOverrideUrlLoadingCallback) {
        if channel == nil {
            callback.defaultBehaviour(nil)
            return
        }
        channel?.invokeMethod("shouldOverrideUrlLoading", arguments: navigationAction.toMap(), callback: callback)
    }
    
    public func onLoadStart(url: String?) {
        let arguments: [String: Any?] = ["url": url]
        channel?.invokeMethod("onLoadStart", arguments: arguments)
    }
    
    public func onLoadStop(url: String?) {
        let arguments: [String: Any?] = ["url": url]
        channel?.invokeMethod("onLoadStop", arguments: arguments)
    }
    
    public func onUpdateVisitedHistory(url: String?, isReload: Bool?) {
        let arguments: [String: Any?] = [
            "url": url,
            "isReload": nil
        ]
        channel?.invokeMethod("onUpdateVisitedHistory", arguments: arguments)
    }
    
    public func onReceivedError(request: WebResourceRequest, error: WebResourceError) {
        let arguments: [String: Any?] = [
            "request": request.toMap(),
            "error": error.toMap()
        ]
        channel?.invokeMethod("onReceivedError", arguments: arguments)
    }
    
    public func onReceivedHttpError(request: WebResourceRequest, errorResponse: WebResourceResponse) {
        let arguments: [String: Any?] = [
            "request": request.toMap(),
            "errorResponse": errorResponse.toMap()
        ]
        channel?.invokeMethod("onReceivedHttpError", arguments: arguments)
    }
    
    public class ReceivedHttpAuthRequestCallback: BaseCallbackResult<HttpAuthResponse> {
        override init() {
            super.init()
            self.decodeResult = { (obj: Any?) in
                return HttpAuthResponse.fromMap(map: obj as? [String:Any?])
            }
        }
        
        deinit {
            self.defaultBehaviour(nil)
        }
    }
    
    public func onReceivedHttpAuthRequest(challenge: HttpAuthenticationChallenge, callback: ReceivedHttpAuthRequestCallback) {
        if channel == nil {
            callback.defaultBehaviour(nil)
            return
        }
        // workaround for ProtectionSpace.toMap() SSL Certificate
        // https://github.com/pichillilorenzo/flutter_inappwebview/issues/1678
        DispatchQueue.global(qos: .background).async {
            let arguments = challenge.toMap()
            DispatchQueue.main.async { [weak self] in
                if self?.channel == nil {
                    callback.defaultBehaviour(nil)
                    return
                }
                self?.channel?.invokeMethod("onReceivedHttpAuthRequest", arguments: arguments, callback: callback)
            }
        }
    }
    
    public class ReceivedServerTrustAuthRequestCallback: BaseCallbackResult<ServerTrustAuthResponse> {
        override init() {
            super.init()
            self.decodeResult = { (obj: Any?) in
                return ServerTrustAuthResponse.fromMap(map: obj as? [String:Any?])
            }
        }
        
        deinit {
            self.defaultBehaviour(nil)
        }
    }
    
    public func onReceivedServerTrustAuthRequest(challenge: ServerTrustChallenge, callback: ReceivedServerTrustAuthRequestCallback) {
        if channel == nil {
            callback.defaultBehaviour(nil)
            return
        }
        // workaround for ProtectionSpace.toMap() SSL Certificate
        // https://github.com/pichillilorenzo/flutter_inappwebview/issues/1678
        DispatchQueue.global(qos: .background).async {
            let arguments = challenge.toMap()
            DispatchQueue.main.async { [weak self] in
                if self?.channel == nil {
                    callback.defaultBehaviour(nil)
                    return
                }
                self?.channel?.invokeMethod("onReceivedServerTrustAuthRequest", arguments: arguments, callback: callback)
            }
        }
    }
    
    public class ReceivedClientCertRequestCallback: BaseCallbackResult<ClientCertResponse> {
        override init() {
            super.init()
            self.decodeResult = { (obj: Any?) in
                return ClientCertResponse.fromMap(map: obj as? [String:Any?])
            }
        }
        
        deinit {
            self.defaultBehaviour(nil)
        }
    }
    
    public func onReceivedClientCertRequest(challenge: ClientCertChallenge, callback: ReceivedClientCertRequestCallback) {
        if channel == nil {
            callback.defaultBehaviour(nil)
            return
        }
        // workaround for ProtectionSpace.toMap() SSL Certificate
        // https://github.com/pichillilorenzo/flutter_inappwebview/issues/1678
        DispatchQueue.global(qos: .background).async {
            let arguments = challenge.toMap()
            DispatchQueue.main.async { [weak self] in
                if self?.channel == nil {
                    callback.defaultBehaviour(nil)
                    return
                }
                self?.channel?.invokeMethod("onReceivedClientCertRequest", arguments: arguments, callback: callback)
            }
        }
    }
    
    public func onZoomScaleChanged(newScale: Float, oldScale: Float) {
        let arguments: [String: Any?] = [
            "newScale": newScale,
            "oldScale": oldScale
        ]
        channel?.invokeMethod("onZoomScaleChanged", arguments: arguments)
    }
    
    public func onPageCommitVisible(url: String?) {
        let arguments: [String: Any?] = [
            "url": url
        ]
        channel?.invokeMethod("onPageCommitVisible", arguments: arguments)
    }
    
    public class LoadResourceWithCustomSchemeCallback: BaseCallbackResult<CustomSchemeResponse> {
        override init() {
            super.init()
            self.decodeResult = { (obj: Any?) in
                return CustomSchemeResponse.fromMap(map: obj as? [String:Any?])
            }
        }
    }
    
    public func onLoadResourceWithCustomScheme(request: WebResourceRequest, callback: LoadResourceWithCustomSchemeCallback) {
        if channel == nil {
            callback.defaultBehaviour(nil)
            return
        }
        let arguments: [String: Any?] = ["request": request.toMap()]
        channel?.invokeMethod("onLoadResourceWithCustomScheme", arguments: arguments, callback: callback)
    }
    
    public class CallJsHandlerCallback: BaseCallbackResult<Any> {
        override init() {
            super.init()
            self.decodeResult = { (obj: Any?) in
                return obj
            }
        }
    }
    
    public func onCallJsHandler(handlerName: String, args: String, callback: CallJsHandlerCallback) {
        if channel == nil {
            callback.defaultBehaviour(nil)
            return
        }
        let arguments: [String: Any?] = [
            "handlerName": handlerName,
            "args": args
        ]
        channel?.invokeMethod("onCallJsHandler", arguments: arguments, callback: callback)
    }
    
    public class NavigationResponseCallback: BaseCallbackResult<WKNavigationResponsePolicy> {
        override init() {
            super.init()
            self.decodeResult = { (obj: Any?) in
                if let action = obj as? Int {
                    return WKNavigationResponsePolicy.init(rawValue: action) ?? WKNavigationResponsePolicy.cancel
                }
                return WKNavigationResponsePolicy.cancel
            }
        }
        
        deinit {
            self.defaultBehaviour(nil)
        }
    }
    
    public func onNavigationResponse(navigationResponse: WKNavigationResponse, callback: NavigationResponseCallback) {
        if channel == nil {
            callback.defaultBehaviour(nil)
            return
        }
        channel?.invokeMethod("onNavigationResponse", arguments: navigationResponse.toMap(), callback: callback)
    }
    
    public class ShouldAllowDeprecatedTLSCallback: BaseCallbackResult<Bool> {
        override init() {
            super.init()
            self.decodeResult = { (obj: Any?) in
                if let action = obj as? Int {
                    return action == 1
                }
                return false
            }
        }
        
        deinit {
            self.defaultBehaviour(nil)
        }
    }
    
    public func shouldAllowDeprecatedTLS(challenge: URLAuthenticationChallenge, callback: ShouldAllowDeprecatedTLSCallback) {
        if channel == nil {
            callback.defaultBehaviour(nil)
            return
        }
        // workaround for ProtectionSpace.toMap() SSL Certificate
        // https://github.com/pichillilorenzo/flutter_inappwebview/issues/1678
        DispatchQueue.global(qos: .background).async {
            let arguments = challenge.toMap()
            DispatchQueue.main.async { [weak self] in
                if self?.channel == nil {
                    callback.defaultBehaviour(nil)
                    return
                }
                self?.channel?.invokeMethod("shouldAllowDeprecatedTLS", arguments: arguments, callback: callback)
            }
        }
    }
    
    public func onWebContentProcessDidTerminate() {
        let arguments: [String: Any?] = [:]
        channel?.invokeMethod("onWebContentProcessDidTerminate", arguments: arguments)
    }
    
    public func onDidReceiveServerRedirectForProvisionalNavigation() {
        let arguments: [String: Any?] = [:]
        channel?.invokeMethod("onDidReceiveServerRedirectForProvisionalNavigation", arguments: arguments)
    }
    
    @available(iOS 15.0, *)
    public func onCameraCaptureStateChanged(oldState: WKMediaCaptureState?, newState: WKMediaCaptureState?) {
        let arguments = [
            "oldState": oldState?.rawValue,
            "newState": newState?.rawValue
        ]
        channel?.invokeMethod("onCameraCaptureStateChanged", arguments: arguments)
    }
    
    @available(iOS 15.0, *)
    public func onMicrophoneCaptureStateChanged(oldState: WKMediaCaptureState?, newState: WKMediaCaptureState?) {
        let arguments = [
            "oldState": oldState?.rawValue,
            "newState": newState?.rawValue
        ]
        channel?.invokeMethod("onMicrophoneCaptureStateChanged", arguments: arguments)
    }
    
    public class PrintRequestCallback: BaseCallbackResult<Bool> {
        override init() {
            super.init()
            self.decodeResult = { (obj: Any?) in
                return obj is Bool && obj as! Bool
            }
        }
    }
    
    public func onPrintRequest(url: URL?, printJobId: String?, callback: PrintRequestCallback) {
        if channel == nil {
            callback.defaultBehaviour(nil)
            return
        }
        let arguments = [
            "url": url?.absoluteString,
            "printJobId": printJobId,
        ]
        channel?.invokeMethod("onPrintRequest", arguments: arguments, callback: callback)
    }
    
    public override func dispose() {
        super.dispose()
        webView = nil
    }
    
    deinit {
        debugPrint("WebViewChannelDelegate - dealloc")
        dispose()
    }
}
