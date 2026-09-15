//
//  FlutterMethodChannel.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 06/05/22.
//

import Foundation
import Flutter

extension FlutterMethodChannel {
    public func invokeMethod(_ method: String, arguments: Any, callback: MethodChannelResult) {
        invokeMethod(method, arguments: arguments) {(result) -> Void in
            if result is FlutterError {
                let error = result as! FlutterError
                callback.error(error.code, error.message, error.details)
            }
            else if (result as? NSObject) == FlutterMethodNotImplemented {
                callback.notImplemented()
            }
            else {
                callback.success(result)
            }
        }
    }
}
