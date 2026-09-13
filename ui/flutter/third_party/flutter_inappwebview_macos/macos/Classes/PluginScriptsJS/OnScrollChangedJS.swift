//
//  OnScrollEvent.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 16/10/22.
//

import Foundation

let ON_SCROLL_CHANGED_EVENT_JS_PLUGIN_SCRIPT_GROUP_NAME = "IN_APP_WEBVIEW_ON_SCROLL_CHANGED_EVENT_JS_PLUGIN_SCRIPT"

let ON_SCROLL_CHANGED_EVENT_JS_PLUGIN_SCRIPT = PluginScript(
    groupName: ON_SCROLL_CHANGED_EVENT_JS_PLUGIN_SCRIPT_GROUP_NAME,
    source: ON_SCROLL_CHANGED_EVENT_JS_SOURCE,
    injectionTime: .atDocumentStart,
    forMainFrameOnly: true,
    requiredInAllContentWorlds: false,
    messageHandlerNames: ["onScrollChanged"])

let ON_SCROLL_CHANGED_EVENT_JS_SOURCE = """
(function(){
    document.addEventListener('scroll', function(e) {
        var _windowId = \(WINDOW_ID_VARIABLE_JS_SOURCE);
        window.webkit.messageHandlers["onScrollChanged"].postMessage(
            {
                x: window.scrollX,
                y: window.scrollY,
                _windowId: _windowId
            }
        );
    });
})();
"""
