//
//  PullToRefreshChannelDelegate.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 05/05/22.
//

import Foundation

public class PullToRefreshChannelDelegate: ChannelDelegate {
    private weak var pullToRefreshControl: PullToRefreshControl?
    
    public init(pullToRefreshControl: PullToRefreshControl, channel: FlutterMethodChannel) {
        super.init(channel: channel)
        self.pullToRefreshControl = pullToRefreshControl
    }
    
    public override func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
        let arguments = call.arguments as? NSDictionary
        
        switch call.method {
        case "setEnabled":
            if let pullToRefreshView = pullToRefreshControl {
                let enabled = arguments!["enabled"] as! Bool
                if enabled {
                    pullToRefreshView.delegate?.enablePullToRefresh()
                } else {
                    pullToRefreshView.delegate?.disablePullToRefresh()
                }
                result(true)
            } else {
                result(false)
            }
            break
        case "isEnabled":
            if let pullToRefreshView = pullToRefreshControl {
                result(pullToRefreshView.delegate?.isPullToRefreshEnabled())
            } else {
                result(false)
            }
            break
        case "setRefreshing":
            if let pullToRefreshView = pullToRefreshControl {
                let refreshing = arguments!["refreshing"] as! Bool
                if refreshing {
                    pullToRefreshView.beginRefreshing()
                } else {
                    pullToRefreshView.endRefreshing()
                }
                result(true)
            } else {
                result(false)
            }
            break
        case "isRefreshing":
            if let pullToRefreshView = pullToRefreshControl {
                result(pullToRefreshView.isRefreshing)
            } else {
                result(false)
            }
            break
        case "setColor":
            if let pullToRefreshView = pullToRefreshControl {
                let color = arguments!["color"] as! String
                pullToRefreshView.tintColor = UIColor(hexString: color)
                result(true)
            } else {
                result(false)
            }
            break
        case "setBackgroundColor":
            if let pullToRefreshView = pullToRefreshControl {
                let color = arguments!["color"] as! String
                pullToRefreshView.backgroundColor = UIColor(hexString: color)
                result(true)
            } else {
                result(false)
            }
            break
        case "setStyledTitle":
            if let pullToRefreshView = pullToRefreshControl {
                let attributedTitleMap = arguments!["attributedTitle"] as! [String: Any?]
                pullToRefreshView.attributedTitle = NSAttributedString.fromMap(map: attributedTitleMap)
                result(true)
            } else {
                result(false)
            }
            break
        default:
            result(FlutterMethodNotImplemented)
            break
        }
    }
    
    public func onRefresh() {
        let arguments: [String: Any?] = [:]
        channel?.invokeMethod("onRefresh", arguments: arguments)
    }
    
    public override func dispose() {
        super.dispose()
        pullToRefreshControl = nil
    }
    
    deinit {
        dispose()
    }
}
