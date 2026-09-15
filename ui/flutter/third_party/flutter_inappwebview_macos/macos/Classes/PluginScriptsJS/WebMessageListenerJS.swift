//
//  WebMessageListenerJS.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 10/03/21.
//

import Foundation

let WEB_MESSAGE_LISTENER_JS_SOURCE = """
function FlutterInAppWebViewWebMessageListener(jsObjectName) {
    this.jsObjectName = jsObjectName;
    this.listeners = [];
    this.onmessage = null;
}
FlutterInAppWebViewWebMessageListener.prototype.postMessage = function(data) {
    var message = {
        "data": window.ArrayBuffer != null && data instanceof ArrayBuffer ? Array.from(new Uint8Array(data)) : (data != null ? data.toString() : null),
        "type": window.ArrayBuffer != null && data instanceof ArrayBuffer ? 1 : 0
    };
    window.webkit.messageHandlers['onWebMessageListenerPostMessageReceived'].postMessage({jsObjectName: this.jsObjectName, message: message});
};
FlutterInAppWebViewWebMessageListener.prototype.addEventListener = function(type, listener) {
    if (listener == null) {
        return;
    }
    this.listeners.push(listener);
};
FlutterInAppWebViewWebMessageListener.prototype.removeEventListener = function(type, listener) {
    if (listener == null) {
        return;
    }
    var index = this.listeners.indexOf(listener);
    if (index >= 0) {
        this.listeners.splice(index, 1);
    }
};

window.\(JAVASCRIPT_BRIDGE_NAME)._normalizeIPv6 = function(ip_string) {
    // replace ipv4 address if any
    var ipv4 = ip_string.match(/(.*:)([0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+$)/);
    if (ipv4) {
        ip_string = ipv4[1];
        ipv4 = ipv4[2].match(/[0-9]+/g);
        for (var i = 0;i < 4;i ++) {
            var byte = parseInt(ipv4[i],10);
            ipv4[i] = ("0" + byte.toString(16)).substr(-2);
        }
        ip_string += ipv4[0] + ipv4[1] + ':' + ipv4[2] + ipv4[3];
    }

    // take care of leading and trailing ::
    ip_string = ip_string.replace(/^:|:$/g, '');

    var ipv6 = ip_string.split(':');

    for (var i = 0; i < ipv6.length; i ++) {
        var hex = ipv6[i];
        if (hex != "") {
            // normalize leading zeros
            ipv6[i] = ("0000" + hex).substr(-4);
        }
        else {
            // normalize grouped zeros ::
            hex = [];
            for (var j = ipv6.length; j <= 8; j ++) {
                hex.push('0000');
            }
            ipv6[i] = hex.join(':');
        }
    }

    return ipv6.join(':');
}

window.\(JAVASCRIPT_BRIDGE_NAME)._isOriginAllowed = function(allowedOriginRules, scheme, host, port) {
    for (var rule of allowedOriginRules) {
        if (rule === "*") {
            return true;
        }
        if (scheme == null || scheme === "") {
            continue;
        }
        if ((scheme == null || scheme === "") && (host == null || host === "") && (port === 0 || port === "" || port == null)) {
            continue;
        }
        var rulePort = rule.port == null || rule.port === 0 ? (rule.scheme == "https" ? 443 : 80) : rule.port;
        var currentPort = port === 0 || port === "" || port == null ? (scheme == "https" ? 443 : 80) : port;
        var IPv6 = null;
        if (rule.host != null && rule.host[0] === "[") {
            try {
                IPv6 = window.\(JAVASCRIPT_BRIDGE_NAME)._normalizeIPv6(rule.host.substring(1, rule.host.length - 1));
            } catch {}
        }
        var hostIPv6 = null;
        try {
            hostIPv6 = window.\(JAVASCRIPT_BRIDGE_NAME)._normalizeIPv6(host);
        } catch {}

        var schemeAllowed = scheme == rule.scheme;
        
        var hostAllowed = rule.host == null ||
            rule.host === "" ||
            host === rule.host ||
            (rule.host[0] === "*" && host != null && host.indexOf(rule.host.split("*")[1]) >= 0) ||
            (hostIPv6 != null && IPv6 != null && hostIPv6 === IPv6);

        var portAllowed = rulePort === currentPort;

        if (schemeAllowed && hostAllowed && portAllowed) {
            return true;
        }
    }
    return false;
}
"""
