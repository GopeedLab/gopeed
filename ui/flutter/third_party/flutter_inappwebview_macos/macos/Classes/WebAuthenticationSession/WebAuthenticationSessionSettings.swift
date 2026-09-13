//
//  WebAuthenticationSessionSettings.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 08/05/22.
//

import Foundation
import AuthenticationServices
import SafariServices

@objcMembers
public class WebAuthenticationSessionSettings: ISettings<WebAuthenticationSession> {
    
    var prefersEphemeralWebBrowserSession = false
    
    override init(){
        super.init()
    }
    
    override func getRealSettings(obj: WebAuthenticationSession?) -> [String: Any?] {
        var realOptions: [String: Any?] = toMap()
        if #available(macOS 10.15, *), let session = obj?.session as? ASWebAuthenticationSession {
            realOptions["prefersEphemeralWebBrowserSession"] = session.prefersEphemeralWebBrowserSession
        }
        return realOptions
    }
}
