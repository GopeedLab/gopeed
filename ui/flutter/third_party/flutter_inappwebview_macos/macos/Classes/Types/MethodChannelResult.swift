//
//  MethodChannelResult.swift
//  shared-apple
//
//  Created by Lorenzo Pichilli on 17/10/22.
//

import Foundation

public protocol MethodChannelResult {
    var success: (_ obj: Any?) -> Void { get set }
    var error: (_ code: String, _ message: String?, _ details: Any?) -> Void { get set }
    var notImplemented: () -> Void { get set }
}
