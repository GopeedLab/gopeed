//
//  OriginalViewPortMetaTagContentJS.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 16/02/21.
//

import Foundation

let ORIGINAL_VIEWPORT_METATAG_CONTENT_JS_PLUGIN_SCRIPT_GROUP_NAME = "IN_APP_WEBVIEW_ORIGINAL_VIEWPORT_METATAG_CONTENT_JS_PLUGIN_SCRIPT"

let ORIGINAL_VIEWPORT_METATAG_CONTENT_JS_PLUGIN_SCRIPT = PluginScript(
    groupName: ORIGINAL_VIEWPORT_METATAG_CONTENT_JS_PLUGIN_SCRIPT_GROUP_NAME,
    source: ORIGINAL_VIEWPORT_METATAG_CONTENT_JS_SOURCE,
    injectionTime: .atDocumentEnd,
    forMainFrameOnly: true,
    requiredInAllContentWorlds: false,
    messageHandlerNames: [])

let ORIGINAL_VIEWPORT_METATAG_CONTENT_JS_SOURCE = """
window.\(JAVASCRIPT_BRIDGE_NAME)._originalViewPortMetaTagContent = "";
(function() {
    var metaTagNodes = document.head.getElementsByTagName('meta');
    for (var i = 0; i < metaTagNodes.length; i++) {
        var metaTagNode = metaTagNodes[i];
        if (metaTagNode.name === "viewport") {
            window.\(JAVASCRIPT_BRIDGE_NAME)._originalViewPortMetaTagContent = metaTagNode.content;
        }
    }
})();
"""
