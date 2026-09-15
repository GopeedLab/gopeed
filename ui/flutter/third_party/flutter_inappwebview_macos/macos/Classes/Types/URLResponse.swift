//
//  URLResponse.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 19/02/21.
//

import Foundation

extension URLResponse {
    public convenience init?(fromPluginMap: [String:Any?]) {
        let url = URL(string: fromPluginMap["url"] as? String ?? "about:blank")!
        let mimeType = fromPluginMap["mimeType"] as? String
        let expectedContentLength = fromPluginMap["expectedContentLength"] as? Int64 ?? 0
        let textEncodingName = fromPluginMap["textEncodingName"] as? String
        self.init(url: url, mimeType: mimeType, expectedContentLength: Int(expectedContentLength), textEncodingName: textEncodingName)
    }
    
    public func toMap () -> [String:Any?] {
        let httpResponse: HTTPURLResponse? = self as? HTTPURLResponse
        return [
            "expectedContentLength": expectedContentLength,
            "mimeType": mimeType,
            "suggestedFilename": suggestedFilename,
            "textEncodingName": textEncodingName,
            "url": url?.absoluteString,
            "headers": httpResponse?.allHeaderFields,
            "statusCode": httpResponse?.statusCode
        ]
    }
}
