//
//  SupportZoomJS.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 16/02/21.
//

import Foundation

let NOT_SUPPORT_ZOOM_JS_PLUGIN_SCRIPT_GROUP_NAME = "IN_APP_WEBVIEW_NOT_SUPPORT_ZOOM_JS_PLUGIN_SCRIPT"

let NOT_SUPPORT_ZOOM_JS_PLUGIN_SCRIPT = PluginScript(
    groupName: NOT_SUPPORT_ZOOM_JS_PLUGIN_SCRIPT_GROUP_NAME,
    source: NOT_SUPPORT_ZOOM_JS_SOURCE,
    injectionTime: .atDocumentEnd,
    forMainFrameOnly: true,
    requiredInAllContentWorlds: false,
    messageHandlerNames: [])

let NOT_SUPPORT_ZOOM_JS_SOURCE = """
(function() {
    var meta = document.createElement('meta');
    meta.setAttribute('name', 'viewport');
    meta.setAttribute('content', 'width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no');
    document.getElementsByTagName('head')[0].appendChild(meta);
})()
"""

let SUPPORT_ZOOM_JS_SOURCE = """
(function() {
    var meta = document.createElement('meta');
    meta.setAttribute('name', 'viewport');
    meta.setAttribute('content', window.\(JAVASCRIPT_BRIDGE_NAME)._originalViewPortMetaTagContent);
    document.getElementsByTagName('head')[0].appendChild(meta);
})()
"""

