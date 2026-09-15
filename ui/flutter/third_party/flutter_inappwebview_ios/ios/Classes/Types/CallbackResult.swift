//
//  CallbackResult.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 06/05/22.
//

import Foundation

public class CallbackResult<T>: MethodChannelResult {
    public var notImplemented: () -> Void = {}
    public var success: (Any?) -> Void = {_ in }
    public var error: (String, String?, Any?) -> Void = {_,_,_ in }
    public var nonNullSuccess: (T) -> Bool = {_ in true}
    public var nullSuccess: () -> Bool = {true}
    public var defaultBehaviour: (T?) -> Void = {_ in }
    public var decodeResult: (Any?) -> T? = {_ in nil}
}
