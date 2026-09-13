//
//  CustomSchemeHandler.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 25/10/2019.
//

import FlutterMacOS
import Foundation
import WebKit

@available(macOS 10.13, *)
public class CustomSchemeHandler: NSObject, WKURLSchemeHandler {
    var schemeHandlers: [Int: WKURLSchemeTask] = [:]
    
    public func webView(_ webView: WKWebView, start urlSchemeTask: WKURLSchemeTask) {
        schemeHandlers[urlSchemeTask.hash] = urlSchemeTask
        let inAppWebView = webView as! InAppWebView
        let request = WebResourceRequest.init(fromURLRequest: urlSchemeTask.request)
        let callback = WebViewChannelDelegate.LoadResourceWithCustomSchemeCallback()
        callback.nonNullSuccess = { (response: CustomSchemeResponse) in
            if (self.schemeHandlers[urlSchemeTask.hash] != nil) {
                let urlResponse = URLResponse(url: request.url, mimeType: response.contentType, expectedContentLength: -1, textEncodingName: response.contentEncoding)
                urlSchemeTask.didReceive(urlResponse)
                urlSchemeTask.didReceive(response.data)
                urlSchemeTask.didFinish()
                self.schemeHandlers.removeValue(forKey: urlSchemeTask.hash)
            }
            return false
        }
        callback.error = { (code: String, message: String?, details: Any?) in
            print(code + ", " + (message ?? ""))
        }
        
        if let channelDelegate = inAppWebView.channelDelegate {
            channelDelegate.onLoadResourceWithCustomScheme(request: request, callback: callback)
        } else {
            callback.defaultBehaviour(nil)
        }
    }
    
    public func webView(_ webView: WKWebView, stop urlSchemeTask: WKURLSchemeTask) {
        schemeHandlers.removeValue(forKey: urlSchemeTask.hash)
    }
}
