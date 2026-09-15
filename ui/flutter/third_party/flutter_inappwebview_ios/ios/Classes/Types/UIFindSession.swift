//
//  UIFindSession.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 07/10/22.
//

import Foundation

public class FindSession: NSObject {
    var resultCount: Int
    var highlightedResultIndex: Int
    var searchResultDisplayStyle: Int
    
    public init(resultCount: Int, highlightedResultIndex: Int, searchResultDisplayStyle: Int) {
        self.resultCount = resultCount
        self.highlightedResultIndex = highlightedResultIndex
        self.searchResultDisplayStyle = searchResultDisplayStyle
    }
    
    @available(iOS 16.0, *)
    public static func fromUIFindSession(uiFindSession: UIFindSession) -> FindSession {
        return FindSession(resultCount: uiFindSession.resultCount,
                           highlightedResultIndex: uiFindSession.highlightedResultIndex,
                           searchResultDisplayStyle: uiFindSession.searchResultDisplayStyle.rawValue)
    }
    
    public func toMap () -> [String:Any?] {
        return [
            "resultCount": resultCount,
            "highlightedResultIndex": highlightedResultIndex,
            "searchResultDisplayStyle": searchResultDisplayStyle
        ]
    }
}

@available(iOS 16.0, *)
extension UIFindSession {
    public func toMap () -> [String:Any?] {
        return [
            "resultCount": resultCount,
            "highlightedResultIndex": highlightedResultIndex,
            "searchResultDisplayStyle": searchResultDisplayStyle.rawValue
        ]
    }
}
