//
//  LeakAvoider.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 15/12/2019.
//

import Foundation
import FlutterMacOS

public class LeakAvoider: NSObject {
    weak var delegate : FlutterMethodCallDelegate?
    
    init(delegate: FlutterMethodCallDelegate) {
        super.init()
        self.delegate = delegate
    }
    
    public func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
        self.delegate?.handle(call, result: result)
    }
    
    deinit {
        debugPrint("LeakAvoider - dealloc")
    }
}
