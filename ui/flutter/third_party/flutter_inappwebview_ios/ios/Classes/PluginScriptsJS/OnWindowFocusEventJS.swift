//
//  OnWindowFocusEventJS.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 16/02/21.
//

import Foundation

let ON_WINDOW_FOCUS_EVENT_JS_PLUGIN_SCRIPT_GROUP_NAME = "IN_APP_WEBVIEW_ON_WINDOW_FOCUS_EVENT_JS_PLUGIN_SCRIPT"

let ON_WINDOW_FOCUS_EVENT_JS_PLUGIN_SCRIPT = PluginScript(
    groupName: ON_WINDOW_FOCUS_EVENT_JS_SOURCE,
    source: ON_WINDOW_FOCUS_EVENT_JS_SOURCE,
    injectionTime: .atDocumentStart,
    forMainFrameOnly: true,
    requiredInAllContentWorlds: false,
    messageHandlerNames: [])

let ON_WINDOW_FOCUS_EVENT_JS_SOURCE = """
(function(){
    window.addEventListener('focus', function(e) {
        window.\(JAVASCRIPT_BRIDGE_NAME).callHandler('onWindowFocus');
    });
})();
"""
