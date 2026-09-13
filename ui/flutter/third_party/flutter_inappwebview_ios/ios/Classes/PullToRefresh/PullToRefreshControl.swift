//
//  File.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 03/03/21.
//

import Foundation
import Flutter

public class PullToRefreshControl: UIRefreshControl, Disposable {
    static let METHOD_CHANNEL_NAME_PREFIX = "com.pichillilorenzo/flutter_inappwebview_pull_to_refresh_"

    var plugin: SwiftFlutterPlugin?
    var channelDelegate: PullToRefreshChannelDelegate?
    var settings: PullToRefreshSettings?
    var shouldCallOnRefresh = false
    var delegate: PullToRefreshDelegate?
    
    public init(plugin: SwiftFlutterPlugin, id: Any, settings: PullToRefreshSettings?) {
        super.init()
        self.plugin = plugin
        self.settings = settings
        if let registrar = plugin.registrar {
            let channel = FlutterMethodChannel(name: PullToRefreshControl.METHOD_CHANNEL_NAME_PREFIX + String(describing: id),
                                               binaryMessenger: registrar.messenger())
            self.channelDelegate = PullToRefreshChannelDelegate(pullToRefreshControl: self, channel: channel)
        }
    }
    
    required init?(coder: NSCoder) {
        super.init(coder: coder)
    }
    
    public func prepare() {
        if let settings = settings {
            if settings.enabled {
                delegate?.enablePullToRefresh()
            }
            if let color = settings.color, !color.isEmpty {
                tintColor = UIColor(hexString: color)
            }
            if let backgroundTintColor = settings.backgroundColor, !backgroundTintColor.isEmpty {
                backgroundColor = UIColor(hexString: backgroundTintColor)
            }
            if let attributedTitleMap = settings.attributedTitle {
                attributedTitle = NSAttributedString.fromMap(map: attributedTitleMap)
            }
        }
        addTarget(self, action: #selector(updateShouldCallOnRefresh), for: .valueChanged)
    }
    
    public func onRefresh() {
        shouldCallOnRefresh = false
        channelDelegate?.onRefresh()
    }
    
    @objc public func updateShouldCallOnRefresh() {
        shouldCallOnRefresh = true
    }
    
    public func dispose() {
        channelDelegate?.dispose()
        channelDelegate = nil
        removeTarget(self, action: #selector(updateShouldCallOnRefresh), for: .valueChanged)
        delegate = nil
        plugin = nil
    }
    
    deinit {
        debugPrint("PullToRefreshControl - dealloc")
        dispose()
    }
}
