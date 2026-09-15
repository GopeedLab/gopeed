//
//  SslError.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 15/02/21.
//

import Foundation

public class SslError: NSObject {
    var errorType: SecTrustResultType?
    var message: String?
    
    public init(errorType: SecTrustResultType?) {
        self.errorType = errorType
        
        var sslErrorMessage: String? = nil
        switch errorType {
            case .deny:
                sslErrorMessage = "Indicates a user-configured deny; do not proceed."
                break
            case .fatalTrustFailure:
                sslErrorMessage = "Indicates a trust failure which cannot be overridden by the user."
                break
            case .invalid:
                sslErrorMessage = "Indicates an invalid setting or result."
                break
            case .otherError:
                sslErrorMessage = "Indicates a failure other than that of trust evaluation."
                break
            case .recoverableTrustFailure:
                sslErrorMessage = "Indicates a trust policy failure which can be overridden by the user."
                break
            case .unspecified:
                sslErrorMessage = "Indicates the evaluation succeeded and the certificate is implicitly trusted, but user intent was not explicitly specified."
                break
            default:
                sslErrorMessage = nil
        }
        
        self.message = sslErrorMessage
    }
    
    public func toMap () -> [String:Any?] {
        return [
            "code": errorType?.rawValue,
            "message": message
        ]
    }
}
