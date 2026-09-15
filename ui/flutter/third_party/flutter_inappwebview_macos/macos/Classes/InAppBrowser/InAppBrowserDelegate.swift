//
//  InAppBrowserDelegate.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 14/02/21.
//

import Foundation

public protocol InAppBrowserDelegate {
    func didChangeTitle(title: String?)
    func didStartNavigation(url: URL?)
    func didUpdateVisitedHistory(url: URL?)
    func didFinishNavigation(url: URL?)
    func didFailNavigation(url: URL?, error: Error)
    func didChangeProgress(progress: Double)
}
