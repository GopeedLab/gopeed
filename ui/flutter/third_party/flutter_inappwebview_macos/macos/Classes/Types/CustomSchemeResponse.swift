//
//  CustomSchemeResponse.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 07/05/22.
//

import Foundation
import FlutterMacOS

public class CustomSchemeResponse: NSObject {
    var data: Data
    var contentType: String
    var contentEncoding: String
    
    public init(data: Data, contentType: String, contentEncoding: String) {
        self.data = data
        self.contentType = contentType
        self.contentEncoding = contentEncoding
    }
    
    public static func fromMap(map: [String:Any?]?) -> CustomSchemeResponse? {
        guard let map = map else {
            return nil
        }
        let data = map["data"] as! FlutterStandardTypedData
        let contentType = map["contentType"] as! String
        let contentEncoding = map["contentEncoding"] as! String
        return CustomSchemeResponse(data: data.data, contentType: contentType, contentEncoding: contentEncoding)
    }
}
