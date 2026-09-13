//
//  WebAuthenticationSession.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 08/05/22.
//

import Foundation
import AuthenticationServices
import SafariServices

public class WebAuthenticationSession: NSObject, ASWebAuthenticationPresentationContextProviding, Disposable {
    static let METHOD_CHANNEL_NAME_PREFIX = "com.pichillilorenzo/flutter_webauthenticationsession_"
    var id: String
    var plugin: SwiftFlutterPlugin?
    var url: URL
    var callbackURLScheme: String?
    var settings: WebAuthenticationSessionSettings
    var session: Any?
    var channelDelegate: WebAuthenticationSessionChannelDelegate?
    private var _canStart = true
    
    public init(plugin: SwiftFlutterPlugin, id: String, url: URL, callbackURLScheme: String?, settings: WebAuthenticationSessionSettings) {
        self.id = id
        self.plugin = plugin
        self.url = url
        self.settings = settings
        super.init()
        self.callbackURLScheme = callbackURLScheme
        if #available(iOS 12.0, *) {
            let session = ASWebAuthenticationSession(url: self.url, callbackURLScheme: self.callbackURLScheme, completionHandler: self.completionHandler)
            if #available(iOS 13.0, *) {
                session.presentationContextProvider = self
            }
            self.session = session
        } else if #available(iOS 11.0, *) {
            self.session = SFAuthenticationSession(url: self.url, callbackURLScheme: self.callbackURLScheme, completionHandler: self.completionHandler)
        }
        let channel = FlutterMethodChannel(name: WebAuthenticationSession.METHOD_CHANNEL_NAME_PREFIX + id,
                                           binaryMessenger: plugin.registrar!.messenger())
        self.channelDelegate = WebAuthenticationSessionChannelDelegate(webAuthenticationSession: self, channel: channel)
    }
    
    public func prepare() {
        if #available(iOS 13.0, *), let session = session as? ASWebAuthenticationSession {
            session.prefersEphemeralWebBrowserSession = settings.prefersEphemeralWebBrowserSession
        }
    }
    
    public func completionHandler(url: URL?, error: Error?) -> Void {
        channelDelegate?.onComplete(url: url, errorCode: error?._code)
    }
    
    public func canStart() -> Bool {
        guard let session = session else {
            return false
        }
        if #available(iOS 13.4, *), let session = session as? ASWebAuthenticationSession {
            return session.canStart
        }
        return _canStart
    }
    
    public func start() -> Bool {
        guard let session = session else {
            return false
        }
        var started = false
        if #available(iOS 12.0, *), let session = session as? ASWebAuthenticationSession {
            started = session.start()
        } else if #available(iOS 11.0, *), let session = session as? SFAuthenticationSession {
            started = session.start()
        }
        if started {
            _canStart = false
        }
        return started
    }
    
    public func cancel() {
        guard let session = session else {
            return
        }
        if #available(iOS 12.0, *), let session = session as? ASWebAuthenticationSession {
            session.cancel()
        } else if #available(iOS 11.0, *), let session = session as? SFAuthenticationSession {
            session.cancel()
        }
    }
    
    @available(iOS 12.0, *)
    public func presentationAnchor(for session: ASWebAuthenticationSession) -> ASPresentationAnchor {
        return UIApplication.shared.windows.first { $0.isKeyWindow } ?? ASPresentationAnchor()
    }
    
    public func dispose() {
        cancel()
        channelDelegate?.dispose()
        channelDelegate = nil
        session = nil
        plugin?.webAuthenticationSessionManager?.sessions[id] = nil
        plugin = nil
    }
    
    deinit {
        debugPrint("WebAuthenticationSession - dealloc")
        dispose()
    }
}
