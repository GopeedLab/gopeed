//
//  FindElementsAtPointJS.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 16/02/21.
//

import Foundation

let FIND_ELEMENTS_AT_POINT_JS_PLUGIN_SCRIPT_GROUP_NAME = "IN_APP_WEBVIEW_FIND_ELEMENTS_AT_POINT_JS_PLUGIN_SCRIPT"

let FIND_ELEMENTS_AT_POINT_JS_PLUGIN_SCRIPT = PluginScript(
    groupName: FIND_ELEMENTS_AT_POINT_JS_PLUGIN_SCRIPT_GROUP_NAME,
    source: FIND_ELEMENTS_AT_POINT_JS_SOURCE,
    injectionTime: .atDocumentStart,
    forMainFrameOnly: true,
    requiredInAllContentWorlds: false,
    messageHandlerNames: [])

/**
 https://developer.android.com/reference/android/webkit/WebView.HitTestResult
 */
let FIND_ELEMENTS_AT_POINT_JS_SOURCE = """
window.\(JAVASCRIPT_BRIDGE_NAME)._findElementsAtPoint = function(x, y) {
    var hitTestResultType = {
        UNKNOWN_TYPE: 0,
        PHONE_TYPE: 2,
        GEO_TYPE: 3,
        EMAIL_TYPE: 4,
        IMAGE_TYPE: 5,
        SRC_ANCHOR_TYPE: 7,
        SRC_IMAGE_ANCHOR_TYPE: 8,
        EDIT_TEXT_TYPE: 9
    };
    var element = document.elementFromPoint(x, y);
    var data = {
        type: 0,
        extra: null
    };
    while (element) {
        if (element.tagName === 'IMG' && element.src) {
            if (element.parentNode && element.parentNode.tagName === 'A' && element.parentNode.href) {
                data.type = hitTestResultType.SRC_IMAGE_ANCHOR_TYPE;
            } else {
                data.type = hitTestResultType.IMAGE_TYPE;
            }
            data.extra = element.src;
            break;
        } else if (element.tagName === 'A' && element.href) {
            if (element.href.indexOf('mailto:') === 0) {
                data.type = hitTestResultType.EMAIL_TYPE;
                data.extra = element.href.replace('mailto:', '');
            } else if (element.href.indexOf('tel:') === 0) {
                data.type = hitTestResultType.PHONE_TYPE;
                data.extra = element.href.replace('tel:', '');
            } else if (element.href.indexOf('geo:') === 0) {
                data.type = hitTestResultType.GEO_TYPE;
                data.extra = element.href.replace('geo:', '');
            } else {
                data.type = hitTestResultType.SRC_ANCHOR_TYPE;
                data.extra = element.href;
            }
            break;
        } else if (
            (element.tagName === 'INPUT' && ['text', 'email', 'password', 'number', 'search', 'tel', 'url'].indexOf(element.type) >= 0) ||
            element.tagName === 'TEXTAREA') {
            data.type = hitTestResultType.EDIT_TEXT_TYPE
        }
        element = element.parentNode;
    }
    return data;
}
"""
