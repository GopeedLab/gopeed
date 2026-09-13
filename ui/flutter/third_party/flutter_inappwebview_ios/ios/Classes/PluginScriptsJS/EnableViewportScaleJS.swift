//
//  EnableViewportScaleJS.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 16/02/21.
//

import Foundation

let ENABLE_VIEWPORT_SCALE_JS_PLUGIN_SCRIPT_GROUP_NAME = "IN_APP_WEBVIEW_ENABLE_VIEWPORT_SCALE_JS_PLUGIN_SCRIPT"

let ENABLE_VIEWPORT_SCALE_JS_PLUGIN_SCRIPT = PluginScript(
    groupName: ENABLE_VIEWPORT_SCALE_JS_PLUGIN_SCRIPT_GROUP_NAME,
    source: ENABLE_VIEWPORT_SCALE_JS_SOURCE,
    injectionTime: .atDocumentEnd,
    forMainFrameOnly: true,
    requiredInAllContentWorlds: false,
    messageHandlerNames: [])

let ENABLE_VIEWPORT_SCALE_JS_SOURCE = """
(function() {
    var meta = document.createElement('meta');
    meta.setAttribute('name', 'viewport');
    meta.setAttribute('content', 'width=device-width');
    document.getElementsByTagName('head')[0].appendChild(meta);
})()
"""

let NOT_ENABLE_VIEWPORT_SCALE_JS_SOURCE = """
(function() {
    var meta = document.createElement('meta');
    meta.setAttribute('name', 'viewport');
    meta.setAttribute('content', window.\(JAVASCRIPT_BRIDGE_NAME)._originalViewPortMetaTagContent);
    document.getElementsByTagName('head')[0].appendChild(meta);
})()
"""
