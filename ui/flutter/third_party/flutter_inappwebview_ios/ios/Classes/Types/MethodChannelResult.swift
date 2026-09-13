//
//  MethodChannelResult.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 06/05/22.
//

import Foundation

public protocol MethodChannelResult {
    var success: (_ obj: Any?) -> Void { get set }
    var error: (_ code: String, _ message: String?, _ details: Any?) -> Void { get set }
    var notImplemented: () -> Void { get set }
}
