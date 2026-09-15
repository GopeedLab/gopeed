//
//  BaseCallbackResult.swift
//  shared-apple
//
//  Created by Lorenzo Pichilli on 17/10/22.
//

import Foundation

public class BaseCallbackResult<T>: CallbackResult<T> {
    
    override init() {
        super.init()
        
        self.success = { [weak self] (obj: Any?) in
            let result: T? = self?.decodeResult(obj)
            var shouldRunDefaultBehaviour = false
            if let result = result {
                shouldRunDefaultBehaviour = self?.nonNullSuccess(result) ?? shouldRunDefaultBehaviour
            } else {
                shouldRunDefaultBehaviour = self?.nullSuccess() ?? shouldRunDefaultBehaviour
            }
            if shouldRunDefaultBehaviour {
                self?.defaultBehaviour(result)
            }
        }
        
        self.notImplemented = { [weak self] in
            self?.defaultBehaviour(nil)
        }
    }
}
