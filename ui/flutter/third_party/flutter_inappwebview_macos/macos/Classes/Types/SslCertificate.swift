//
//  SslCertificate.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 15/02/21.
//

import Foundation

public class SslCertificate: NSObject {
    var x509Certificate: Data
    var issuedBy: Any?
    var issuedTo: Any?
    var validNotAfterDate: Any?
    var validNotBeforeDate: Any?
    
    public init(x509Certificate: Data) {
        self.x509Certificate = x509Certificate
    }
    
    public func toMap () -> [String:Any?] {
        return [
            "x509Certificate": x509Certificate,
            "issuedBy": issuedBy,
            "issuedTo": issuedTo,
            "validNotAfterDate": validNotAfterDate,
            "validNotBeforeDate": validNotBeforeDate
        ]
    }
}
