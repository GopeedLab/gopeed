//
//  URLRequest.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 19/02/21.
//

import Foundation
import FlutterMacOS

extension URLRequest {
    public init(fromPluginMap: [String:Any?]) {
        if let urlString = fromPluginMap["url"] as? String, let url = URL(string: urlString) {
            self.init(url: url)
        } else {
            self.init(url: URL(string: "about:blank")!)
        }
        
        if let method = fromPluginMap["method"] as? String {
            httpMethod = method
        }
        if let body = fromPluginMap["body"] as? FlutterStandardTypedData {
            httpBody = body.data
        }
        if let headers = fromPluginMap["headers"] as? [String:String] {
            for (key, value) in headers {
                setValue(value, forHTTPHeaderField: key)
            }
        }
        if let _allowsCellularAccess = fromPluginMap["allowsCellularAccess"] as? Bool {
            allowsCellularAccess = _allowsCellularAccess
        }
        if #available(macOS 10.15, *), let _allowsConstrainedNetworkAccess = fromPluginMap["allowsConstrainedNetworkAccess"] as? Bool {
            allowsConstrainedNetworkAccess = _allowsConstrainedNetworkAccess
        }
        if #available(macOS 10.15, *), let _allowsExpensiveNetworkAccess = fromPluginMap["allowsExpensiveNetworkAccess"] as? Bool {
            allowsExpensiveNetworkAccess = _allowsExpensiveNetworkAccess
        }
        if let _cachePolicy = fromPluginMap["cachePolicy"] as? Int {
            cachePolicy = CachePolicy.init(rawValue: UInt(_cachePolicy)) ?? .useProtocolCachePolicy
        }
        if let _httpShouldHandleCookies = fromPluginMap["httpShouldHandleCookies"] as? Bool {
            httpShouldHandleCookies = _httpShouldHandleCookies
        }
        if let _httpShouldUsePipelining = fromPluginMap["httpShouldUsePipelining"] as? Bool {
            httpShouldUsePipelining = _httpShouldUsePipelining
        }
        if let _networkServiceType = fromPluginMap["networkServiceType"] as? Int {
            networkServiceType = NetworkServiceType.init(rawValue: UInt(_networkServiceType)) ?? .default
        }
        if let _timeoutInterval = fromPluginMap["timeoutInterval"] as? Double {
            timeoutInterval = _timeoutInterval
        }
        if let _mainDocumentURL = fromPluginMap["mainDocumentURL"] as? String {
            mainDocumentURL = URL(string: _mainDocumentURL)!
        }
        if #available(macOS 11.3, *), let _assumesHTTP3Capable = fromPluginMap["assumesHTTP3Capable"] as? Bool {
            assumesHTTP3Capable = _assumesHTTP3Capable
        }
        if #available(macOS 12.0, *), let attributionRawValue = fromPluginMap["attribution"] as? UInt,
            let _attribution = URLRequest.Attribution(rawValue: attributionRawValue) {
            attribution = _attribution
        }
    }
    
    public func toMap () -> [String:Any?] {
        var _allowsConstrainedNetworkAccess: Bool? = nil
        var _allowsExpensiveNetworkAccess: Bool? = nil
        if #available(macOS 10.15, *) {
            _allowsConstrainedNetworkAccess = allowsConstrainedNetworkAccess
            _allowsExpensiveNetworkAccess = allowsExpensiveNetworkAccess
        }
        var _assumesHTTP3Capable: Bool? = nil
        if #available(macOS 11.3, *) {
            _assumesHTTP3Capable = assumesHTTP3Capable
        }
        var _attribution: UInt? = nil
        if #available(macOS 12.0, *) {
            _attribution = attribution.rawValue
        }
        return [
            "url": url?.absoluteString,
            "method": httpMethod,
            "headers": allHTTPHeaderFields,
            "body": httpBody.map(FlutterStandardTypedData.init(bytes:)),
            "allowsCellularAccess": allowsCellularAccess,
            "allowsConstrainedNetworkAccess": _allowsConstrainedNetworkAccess,
            "allowsExpensiveNetworkAccess": _allowsExpensiveNetworkAccess,
            "cachePolicy": cachePolicy.rawValue,
            "httpShouldHandleCookies": httpShouldHandleCookies,
            "httpShouldUsePipelining": httpShouldUsePipelining,
            "networkServiceType": networkServiceType.rawValue,
            "timeoutInterval": timeoutInterval,
            "mainDocumentURL": mainDocumentURL?.absoluteString,
            "assumesHTTP3Capable": _assumesHTTP3Capable,
            "attribution": _attribution
        ]
    }
}
