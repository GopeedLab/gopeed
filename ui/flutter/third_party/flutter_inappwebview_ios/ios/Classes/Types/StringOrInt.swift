//
//  StringOrInt.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 21/04/22.
//

import Foundation

public protocol StringOrInt { }

extension Int: StringOrInt { }
extension String: StringOrInt { }
