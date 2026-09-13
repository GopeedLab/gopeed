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
    
    public func toMap () -> [String:Any?] {
        return [
            "resultCount": resultCount,
            "highlightedResultIndex": highlightedResultIndex,
            "searchResultDisplayStyle": searchResultDisplayStyle
        ]
    }
}
