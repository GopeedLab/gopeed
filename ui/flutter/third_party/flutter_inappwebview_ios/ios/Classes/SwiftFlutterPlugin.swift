/*
 Licensed to the Apache Software Foundation (ASF) under one
 or more contributor license agreements.  See the NOTICE file
 distributed with this work for additional information
 regarding copyright ownership.  The ASF licenses this file
 to you under the Apache License, Version 2.0 (the
 "License"); you may not use this file except in compliance
 with the License.  You may obtain a copy of the License at
 http://www.apache.org/licenses/LICENSE-2.0
 Unless required by applicable law or agreed to in writing,
 software distributed under the License is distributed on an
 "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 KIND, either express or implied.  See the License for the
 specific language governing permissions and limitations
 under the License.
 */

import Flutter
import UIKit
import WebKit
import Foundation
import AVFoundation
import SafariServices

public class SwiftFlutterPlugin: NSObject, FlutterPlugin {
    
    var registrar: FlutterPluginRegistrar?
    var platformUtil: PlatformUtil?
    var inAppWebViewManager: InAppWebViewManager?
    var myCookieManager: Any?
    var myWebStorageManager: Any?
    var credentialDatabase: CredentialDatabase?
    var inAppBrowserManager: InAppBrowserManager?
    var headlessInAppWebViewManager: HeadlessInAppWebViewManager?
    var chromeSafariBrowserManager: ChromeSafariBrowserManager?
    var webAuthenticationSessionManager: WebAuthenticationSessionManager?
    var printJobManager: PrintJobManager?
    
    var webViewControllers: [String: InAppBrowserWebViewController?] = [:]
    var safariViewControllers: [String: Any?] = [:]
    
    public init(with registrar: FlutterPluginRegistrar) {
        super.init()
        
        self.registrar = registrar
        registrar.register(FlutterWebViewFactory(plugin: self) as FlutterPlatformViewFactory, withId: FlutterWebViewFactory.VIEW_TYPE_ID)
        
        platformUtil = PlatformUtil(plugin: self)
        inAppBrowserManager = InAppBrowserManager(plugin: self)
        headlessInAppWebViewManager = HeadlessInAppWebViewManager(plugin: self)
        chromeSafariBrowserManager = ChromeSafariBrowserManager(plugin: self)
        inAppWebViewManager = InAppWebViewManager(plugin: self)
        credentialDatabase = CredentialDatabase(plugin: self)
        if #available(iOS 11.0, *) {
            myCookieManager = MyCookieManager(plugin: self)
        }
        if #available(iOS 9.0, *) {
            myWebStorageManager = MyWebStorageManager(plugin: self)
        }
        webAuthenticationSessionManager = WebAuthenticationSessionManager(plugin: self)
        printJobManager = PrintJobManager(plugin: self)
    }
    
    public static func register(with registrar: FlutterPluginRegistrar) {
        let _ = SwiftFlutterPlugin(with: registrar)
    }
    
    public func detachFromEngine(for registrar: FlutterPluginRegistrar) {
        platformUtil?.dispose()
        platformUtil = nil
        inAppBrowserManager?.dispose()
        inAppBrowserManager = nil
        headlessInAppWebViewManager?.dispose()
        headlessInAppWebViewManager = nil
        chromeSafariBrowserManager?.dispose()
        chromeSafariBrowserManager = nil
        inAppWebViewManager?.dispose()
        inAppWebViewManager = nil
        credentialDatabase?.dispose()
        credentialDatabase = nil
        if #available(iOS 11.0, *) {
            (myCookieManager as! MyCookieManager?)?.dispose()
            myCookieManager = nil
        }
        if #available(iOS 9.0, *) {
            (myWebStorageManager as! MyWebStorageManager?)?.dispose()
            myWebStorageManager = nil
        }
        webAuthenticationSessionManager?.dispose()
        webAuthenticationSessionManager = nil
        printJobManager?.dispose()
        printJobManager = nil
    }
}
