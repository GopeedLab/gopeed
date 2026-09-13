//
//  HeadlessInAppWebView.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 26/03/21.
//

import Foundation
import FlutterMacOS

public class HeadlessInAppWebView: Disposable {
    static let METHOD_CHANNEL_NAME_PREFIX = "com.pichillilorenzo/flutter_headless_inappwebview_"
    var id: String
    var channelDelegate: HeadlessWebViewChannelDelegate?
    var flutterWebView: FlutterWebViewController?
    var plugin: InAppWebViewFlutterPlugin?
    
    public init(plugin: InAppWebViewFlutterPlugin, id: String, flutterWebView: FlutterWebViewController) {
        self.id = id
        self.flutterWebView = flutterWebView
        self.plugin = plugin
        let channel = FlutterMethodChannel(name: HeadlessInAppWebView.METHOD_CHANNEL_NAME_PREFIX + id,
                                       binaryMessenger: plugin.registrar!.messenger)
        self.channelDelegate = HeadlessWebViewChannelDelegate(headlessWebView: self, channel: channel)
    }
    
    public func onWebViewCreated() {
        channelDelegate?.onWebViewCreated();
    }
    
    public func prepare(params: NSDictionary) {
        if let view = flutterWebView?.view() {
            view.alphaValue = 0.01
            let initialSize = params["initialSize"] as? [String: Any?]
            if let size = Size2D.fromMap(map: initialSize) {
                setSize(size: size)
            } else {
                view.frame = CGRect(x: 0.0, y: 0.0, width: NSApplication.shared.mainWindow?.contentView?.bounds.width ?? 0.0,
                                    height: NSApplication.shared.mainWindow?.contentView?.bounds.height ?? 0.0)
            }
            /// Note: The WKWebView behaves very unreliable when rendering offscreen
            /// on a device. This is especially true with JavaScript, which simply
            /// won't be executed sometimes.
            /// So, add the headless WKWebView to the view hierarchy.
            /// This way is also possible to take screenshots.
            let wrapperView = NSView() // wrapper view with frame zero
            wrapperView.addSubview(view, positioned: .below, relativeTo: nil)
            NSApplication.shared.mainWindow?.contentView?.addSubview(wrapperView, positioned: .below, relativeTo: nil)
        }
    }
    
    public func setSize(size: Size2D) {
        if let view = flutterWebView?.view() {
            let width = size.width == -1.0 ? NSApplication.shared.mainWindow?.contentView?.bounds.width ?? 0.0 : CGFloat(size.width)
            let height = size.height == -1.0 ? NSApplication.shared.mainWindow?.contentView?.bounds.height ?? 0.0 : CGFloat(size.height)
            view.frame = CGRect(x: 0.0, y: 0.0, width: width, height: height)
        }
    }
    
    public func getSize() -> Size2D? {
        if let view = flutterWebView?.view() {
            return Size2D(width: Double(view.frame.width), height: Double(view.frame.height))
        }
        return nil
    }
    
    public func disposeAndGetFlutterWebView(withFrame frame: CGRect) -> FlutterWebViewController? {
        let newFlutterWebView = flutterWebView
        if let view = flutterWebView?.view() {
            // restore WebView frame and alpha
            view.frame = frame
            view.alphaValue = 1.0
            // remove from parent
            view.removeFromSuperview()
            dispose(disposeWebView: false)
        }
        return newFlutterWebView
    }
    
    
    public func dispose(disposeWebView: Bool) {
        channelDelegate?.dispose()
        channelDelegate = nil
        plugin?.headlessInAppWebViewManager?.webViews[id] = nil
        if disposeWebView {
            flutterWebView?.dispose(removeFromSuperview: true)
        }
        flutterWebView = nil
        plugin = nil
    }
    
    public func dispose() {
        dispose(disposeWebView: true)
    }
    
    deinit {
        debugPrint("HeadlessInAppWebView - dealloc")
        dispose()
    }
}
