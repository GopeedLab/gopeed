//
//  InAppBrowserNavigationController.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 14/02/21.
//

import Foundation

struct ToolbarIdentifiers {
    static let searchBar = NSToolbarItem.Identifier(rawValue: "SearchBar")
    static let backButton = NSToolbarItem.Identifier(rawValue: "BackButton")
    static let forwardButton = NSToolbarItem.Identifier(rawValue: "ForwardButton")
    static let reloadButton = NSToolbarItem.Identifier(rawValue: "ReloadButton")
    static let menuButton = NSToolbarItem.Identifier(rawValue: "MenuButton")
}

public class InAppBrowserWindow: NSWindow, NSWindowDelegate, NSToolbarDelegate, NSSearchFieldDelegate {
    var searchItem: NSToolbarItem?
    var backItem: NSToolbarItem?
    var forwardItem: NSToolbarItem?
    var reloadItem: NSToolbarItem?
    var menuItem: NSToolbarItem?
    var actionItems: [NSToolbarItem] = []
    
    var reloadButton: NSButton? {
        get {
            return reloadItem?.view as? NSButton
        }
    }
    var backButton: NSButton? {
        get {
            return backItem?.view as? NSButton
        }
    }
    var forwardButton: NSButton? {
        get {
            return forwardItem?.view as? NSButton
        }
    }
    var searchBar: NSSearchField? {
        get {
            if #available(macOS 11.0, *), let searchItem = searchItem as? NSSearchToolbarItem {
                return searchItem.searchField
            } else {
                return searchItem?.view as? NSSearchField
            }
        }
    }
    var menuButton: NSPopUpButton? {
        get {
            return menuItem?.view as? NSPopUpButton
        }
    }
    
    var browserSettings: InAppBrowserSettings?
    var menuItems: [InAppBrowserMenuItem] = []
    
    public func prepare() {
        title = ""
        collectionBehavior = .fullScreenPrimary
        delegate = self
        
        NotificationCenter.default.addObserver(self,
                                               selector: #selector(onMainWindowWillClose(_:)),
                                               name: NSWindow.willCloseNotification,
                                               object: NSApplication.shared.mainWindow)
        
        if #available(macOS 10.13, *) {
            let windowToolbar = NSToolbar()
            windowToolbar.delegate = self
            if #available(macOS 11.0, *) {
                searchItem = NSSearchToolbarItem(itemIdentifier: ToolbarIdentifiers.searchBar)
                (searchItem as! NSSearchToolbarItem).searchField.delegate = self
                toolbarStyle = .expanded
            } else {
                searchItem = NSToolbarItem(itemIdentifier: ToolbarIdentifiers.searchBar)
                let textField = NSSearchField()
                textField.usesSingleLineMode = true
                textField.delegate = self
                searchItem?.view = textField
            }
            searchItem?.label = ""
            windowToolbar.displayMode = .default
            
            backItem = NSToolbarItem(itemIdentifier: ToolbarIdentifiers.backButton)
            backItem?.label = ""
            if let webViewController = contentViewController as? InAppBrowserWebViewController {
                if #available(macOS 11.0, *) {
                    backItem?.view = NSButton(image: NSImage(systemSymbolName: "chevron.left",
                                                                  accessibilityDescription: "Go Back")!,
                                                   target: webViewController,
                                                   action: #selector(InAppBrowserWebViewController.goBack))
                } else {
                    backItem?.view = NSButton(title: "\u{2039}",
                                                target: webViewController,
                                                action: #selector(InAppBrowserWebViewController.goBack))
                }
            }
            
            forwardItem = NSToolbarItem(itemIdentifier: ToolbarIdentifiers.forwardButton)
            forwardItem?.label = ""
            if let webViewController = contentViewController as? InAppBrowserWebViewController {
                if #available(macOS 11.0, *) {
                    forwardItem?.view = NSButton(image: NSImage(systemSymbolName: "chevron.right",
                                                                  accessibilityDescription: "Go Forward")!,
                                                   target: webViewController,
                                                   action: #selector(InAppBrowserWebViewController.goForward))
                } else {
                    forwardItem?.view = NSButton(title: "\u{203A}",
                                                   target: webViewController,
                                                   action: #selector(InAppBrowserWebViewController.goForward))
                }
            }
            
            reloadItem = NSToolbarItem(itemIdentifier: ToolbarIdentifiers.reloadButton)
            reloadItem?.label = ""
            if let webViewController = contentViewController as? InAppBrowserWebViewController {
                if #available(macOS 11.0, *) {
                    reloadItem?.view = NSButton(image: NSImage(systemSymbolName: "arrow.counterclockwise",
                                                                  accessibilityDescription: "Reload")!,
                                                   target: webViewController,
                                                   action: #selector(InAppBrowserWebViewController.reload))
                } else {
                    reloadItem?.view = NSButton(title: "Reload",
                                                   target: webViewController,
                                                   action: #selector(InAppBrowserWebViewController.reload))
                }
            }
            
            if #available(macOS 10.15, *), !menuItems.isEmpty {
                menuItem = NSMenuToolbarItem(itemIdentifier: ToolbarIdentifiers.menuButton)
                if let menuItem = menuItem as? NSMenuToolbarItem {
                    menuItem.label = ""
                    if #available(macOS 11.0, *) {
                        menuItem.image = NSImage(systemSymbolName: "ellipsis.circle",
                                                 accessibilityDescription: "Options")
                        menuItem.showsIndicator = true
                        menuItem.isBordered = true
                    } else {
                        menuItem.title = "Options"
                    }
                    let menu = NSMenu()
                    menuItems = menuItems.sorted(by: {$0.order ?? 0 < $1.order ?? 0})
                    for item in menuItems {
                        if !item.showAsAction {
                            let nsItem = NSMenuItem(title: item.title, action: #selector(InAppBrowserWebViewController.onMenuItemClicked), keyEquivalent: "")
                            nsItem.identifier = NSUserInterfaceItemIdentifier.init(String(item.id))
                            nsItem.image = item.icon
                            menu.addItem(nsItem)
                        } else {
                            let actionItem = NSMenuToolbarItem(itemIdentifier: NSToolbarItem.Identifier(rawValue: String(item.id)))
                            actionItem.label = ""
                            if let webViewController = contentViewController as? InAppBrowserWebViewController {
                                let actionButton = NSButton(title: item.title,
                                                           target: webViewController,
                                                           action: #selector(InAppBrowserWebViewController.onMenuItemClicked))
                                actionButton.identifier = NSUserInterfaceItemIdentifier.init(String(item.id))
                                actionButton.image = item.icon
                                actionItem.view = actionButton
                            }
                            actionItems.append(actionItem)
                        }
                    }
                    menuItem.menu = menu
                }
            }
            
            
            if #available(macOS 10.14, *) {
                windowToolbar.centeredItemIdentifier = ToolbarIdentifiers.searchBar
            }
            toolbar = windowToolbar
        }
        
        forwardButton?.isEnabled = false
        backButton?.isEnabled = false
        
        if let browserSettings = browserSettings {
            if let toolbarTopFixedTitle = browserSettings.toolbarTopFixedTitle {
                title = toolbarTopFixedTitle
            }
            if !browserSettings.hideToolbarTop {
                toolbar?.isVisible = true
                if browserSettings.hideUrlBar {
                    if #available(macOS 11.0, *) {
                        (searchItem as! NSSearchToolbarItem).searchField.isHidden = true
                    } else {
                        searchItem?.view?.isHidden = true
                    }
                }
                if let bgColor = browserSettings.toolbarTopBackgroundColor, !bgColor.isEmpty {
                    backgroundColor = NSColor(hexString: bgColor)
                }
            }
            else {
                toolbar?.isVisible = false
            }
            if #available(macOS 11.0, *), let windowTitlebarSeparatorStyle = browserSettings.windowTitlebarSeparatorStyle {
                titlebarSeparatorStyle = windowTitlebarSeparatorStyle
            }
            alphaValue = browserSettings.windowAlphaValue
            if let windowStyleMask = browserSettings.windowStyleMask {
                styleMask = windowStyleMask
            }
            if let windowFrame = browserSettings.windowFrame {
                setFrame(windowFrame, display: true)
            }
            reloadItem?.view?.isHidden = browserSettings.hideDefaultMenuItems
            backItem?.view?.isHidden = browserSettings.hideDefaultMenuItems
            forwardItem?.view?.isHidden = browserSettings.hideDefaultMenuItems
        }
    }
    
    public func toolbarAllowedItemIdentifiers(_ toolbar: NSToolbar) -> [NSToolbarItem.Identifier] {
        return [[ ToolbarIdentifiers.menuButton,
                 ToolbarIdentifiers.searchBar,
                 ToolbarIdentifiers.backButton,
                 ToolbarIdentifiers.forwardButton,
                 ToolbarIdentifiers.reloadButton,
                 .flexibleSpace ], actionItems.compactMap({ item in
                     return item.itemIdentifier
                 })].flatMap { $0 }
    }
    
    public func toolbarDefaultItemIdentifiers(_ toolbar: NSToolbar) -> [NSToolbarItem.Identifier] {
        return [[.flexibleSpace,
                ToolbarIdentifiers.searchBar,
                .flexibleSpace,
                ToolbarIdentifiers.reloadButton,
                ToolbarIdentifiers.backButton,
                ToolbarIdentifiers.forwardButton],
                actionItems.compactMap({ item in
                    return item.itemIdentifier
                }),
                [ToolbarIdentifiers.menuButton]].flatMap { $0 }
    }
    
    public func toolbar(_ toolbar: NSToolbar, itemForItemIdentifier itemIdentifier: NSToolbarItem.Identifier, willBeInsertedIntoToolbar flag: Bool) -> NSToolbarItem? {
        switch(itemIdentifier) {
        case ToolbarIdentifiers.searchBar:
            return searchItem
        case ToolbarIdentifiers.backButton:
            return backItem
        case ToolbarIdentifiers.forwardButton:
            return forwardItem
        case ToolbarIdentifiers.reloadButton:
            return reloadItem
        case ToolbarIdentifiers.menuButton:
            return menuItem
        default:
            let actionItem = actionItems.first { item in
                return item.itemIdentifier == itemIdentifier
            }
            return actionItem
        }
    }
    
    public func control(_ control: NSControl, textView: NSTextView, doCommandBy commandSelector: Selector) -> Bool {
        if (commandSelector == #selector(NSResponder.insertNewline(_:))) {
            // ENTER key
            var searchField: NSSearchField? = nil
            if #available(macOS 11.0, *), let searchBar = searchItem as? NSSearchToolbarItem {
                searchField = searchBar.searchField
            } else if let searchBar = searchItem {
                searchField = searchBar.view as? NSSearchField
            }
            
            guard let searchField,
                  let urlEncoded = searchField.stringValue.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed),
                  let url = URL(string: urlEncoded) else {
                return false
            }
            
            let request = URLRequest(url: url)
            (contentViewController as? InAppBrowserWebViewController)?.webView?.load(request)
            
            return true
        }

        return false
    }
    
    public func hide() {
        orderOut(self)
    }
    
    public func show() {
        let mainWindow = parent ?? NSApplication.shared.mainWindow
        if #available(macOS 10.12, *),
           !(mainWindow?.tabbedWindows?.contains(self) ?? false),
           browserSettings?.windowType == .tabbed {
            mainWindow?.addTabbedWindow(self, ordered: .above)
        } else if !(mainWindow?.childWindows?.contains(self) ?? false) {
            mainWindow?.addChildWindow(self, ordered: .above)
        }
        makeKeyAndOrderFront(self)
        NSApplication.shared.activate(ignoringOtherApps: true)
    }
    
    public func setSettings(newSettings: InAppBrowserSettings, newSettingsMap: [String: Any]) {
        if newSettingsMap["hidden"] != nil, browserSettings?.hidden != newSettings.hidden {
            if newSettings.hidden {
                hide()
            }
            else {
                show()
            }
        }

        if newSettingsMap["hideUrlBar"] != nil, browserSettings?.hideUrlBar != newSettings.hideUrlBar {
            searchBar?.isHidden = newSettings.hideUrlBar
        }
        
        if newSettingsMap["hideToolbarTop"] != nil, browserSettings?.hideToolbarTop != newSettings.hideToolbarTop {
            toolbar?.isVisible = !newSettings.hideToolbarTop
        }

        if newSettingsMap["toolbarTopBackgroundColor"] != nil, browserSettings?.toolbarTopBackgroundColor != newSettings.toolbarTopBackgroundColor {
            if let bgColor = newSettings.toolbarTopBackgroundColor, !bgColor.isEmpty {
                backgroundColor = NSColor(hexString: bgColor)
            } else {
                backgroundColor = nil
            }
        }
        if #available(macOS 11.0, *), newSettingsMap["windowTitlebarSeparatorStyle"] != nil,
           browserSettings?.windowTitlebarSeparatorStyle != newSettings.windowTitlebarSeparatorStyle {
            titlebarSeparatorStyle = newSettings.windowTitlebarSeparatorStyle!
        }
        if newSettingsMap["windowAlphaValue"] != nil, browserSettings?.windowAlphaValue != newSettings.windowAlphaValue {
            alphaValue = newSettings.windowAlphaValue
        }
        if newSettingsMap["windowStyleMask"] != nil, browserSettings?.windowStyleMask != newSettings.windowStyleMask {
            styleMask = newSettings.windowStyleMask!
        }
        if newSettingsMap["windowFrame"] != nil, browserSettings?.windowFrame != newSettings.windowFrame {
            setFrame(newSettings.windowFrame!, display: true)
        }
        if newSettingsMap["hideDefaultMenuItems"] != nil, browserSettings?.hideDefaultMenuItems != newSettings.hideDefaultMenuItems {
            reloadItem?.view?.isHidden = newSettings.hideDefaultMenuItems
            backItem?.view?.isHidden = newSettings.hideDefaultMenuItems
            forwardItem?.view?.isHidden = newSettings.hideDefaultMenuItems
        }
        browserSettings = newSettings
    }
    
    public func windowWillClose(_ notification: Notification) {
        dispose()
    }
    
    @objc func onMainWindowWillClose(_ notification: Notification) {
        if let webViewController = contentViewController as? InAppBrowserWebViewController {
            webViewController.channelDelegate?.onMainWindowWillClose()
        }
    }
    
    public func dispose() {
        delegate = nil
        NotificationCenter.default.removeObserver(self,
                                                  name: NSWindow.willCloseNotification,
                                                  object: NSApplication.shared.mainWindow)
        if let webViewController = contentViewController as? InAppBrowserWebViewController {
            webViewController.dispose()
        }
        if #available(macOS 11.0, *) {
            (searchItem as? NSSearchToolbarItem)?.searchField.delegate = nil
        } else {
            (searchItem?.view as? NSTextField)?.delegate = nil
            searchItem?.view = nil
        }
        searchItem = nil
        (backItem?.view as? NSButton)?.target = nil
        backItem?.view = nil
        backItem = nil
        (forwardItem?.view as? NSButton)?.target = nil
        forwardItem?.view = nil
        forwardItem = nil
        (reloadItem?.view as? NSButton)?.target = nil
        reloadItem?.view = nil
        reloadItem = nil
    }
    
    deinit {
        debugPrint("InAppBrowserWindow - dealloc")
        dispose()
    }
}
