//
//  CustomUIPrintPageRenderer.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 10/05/22.
//

import Foundation
//
//public class CustomUIPrintPageRenderer: UIPrintPageRenderer {
//    private var _numberOfPages: Int?
//    private var forceRenderingQuality: Int?
//
//    public init(numberOfPage: Int? = nil, forceRenderingQuality: Int? = nil) {
//        super.init()
//        self._numberOfPages = numberOfPage
//        self.forceRenderingQuality = forceRenderingQuality
//    }
//
//    open override var numberOfPages: Int {
//        get {
//            return _numberOfPages ?? super.numberOfPages
//        }
//    }
//    
//    @available(iOS 14.5, *)
//    open override func currentRenderingQuality(forRequested requestedRenderingQuality: UIPrintRenderingQuality) -> UIPrintRenderingQuality {
//        if let forceRenderingQuality = forceRenderingQuality,
//           let quality = UIPrintRenderingQuality.init(rawValue: forceRenderingQuality) {
//            return quality
//        }
//        return super.currentRenderingQuality(forRequested: requestedRenderingQuality)
//    }
//}
