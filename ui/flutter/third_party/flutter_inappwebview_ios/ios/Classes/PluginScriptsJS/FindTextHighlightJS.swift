//
//  FindTextHighlightJS.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 16/02/21.
//

import Foundation

let FIND_TEXT_HIGHLIGHT_JS_PLUGIN_SCRIPT_GROUP_NAME = "IN_APP_WEBVIEW_FIND_TEXT_HIGHLIGHT_JS_PLUGIN_SCRIPT"
let FIND_TEXT_HIGHLIGHT_SEARCH_RESULT_COUNT_JS_SOURCE = "window.\(JAVASCRIPT_BRIDGE_NAME)._searchResultCount"
let FIND_TEXT_HIGHLIGHT_CURRENT_HIGHLIGHT_JS_SOURCE = "window.\(JAVASCRIPT_BRIDGE_NAME)._currentHighlight"
let FIND_TEXT_HIGHLIGHT_IS_DONE_COUNTING_JS_SOURCE = "window.\(JAVASCRIPT_BRIDGE_NAME)._isDoneCounting"

let FIND_TEXT_HIGHLIGHT_JS_PLUGIN_SCRIPT = PluginScript(
    groupName: FIND_TEXT_HIGHLIGHT_JS_PLUGIN_SCRIPT_GROUP_NAME,
    source: FIND_TEXT_HIGHLIGHT_JS_SOURCE,
    injectionTime: .atDocumentStart,
    forMainFrameOnly: true,
    requiredInAllContentWorlds: false,
    messageHandlerNames: ["onFindResultReceived"])

let FIND_TEXT_HIGHLIGHT_JS_SOURCE = """
\(FIND_TEXT_HIGHLIGHT_SEARCH_RESULT_COUNT_JS_SOURCE) = 0;
\(FIND_TEXT_HIGHLIGHT_CURRENT_HIGHLIGHT_JS_SOURCE) = 0;
\(FIND_TEXT_HIGHLIGHT_IS_DONE_COUNTING_JS_SOURCE) = false;
window.\(JAVASCRIPT_BRIDGE_NAME)._findAllAsyncForElement = function(element, keyword) {
  if (element) {
    if (element.nodeType == 3) {
      // Text node

      var elementTmp = element;
      while (true) {
        var value = elementTmp.nodeValue; // Search for keyword in text node
        var idx = value.toLowerCase().indexOf(keyword);

        if (idx < 0) break;

        var span = document.createElement("span");
        var text = document.createTextNode(value.substr(idx, keyword.length));
        span.appendChild(text);

        span.setAttribute(
          "id",
          "\(JAVASCRIPT_BRIDGE_NAME)_SEARCH_WORD_" + \(FIND_TEXT_HIGHLIGHT_SEARCH_RESULT_COUNT_JS_SOURCE)
        );
        span.setAttribute("class", "\(JAVASCRIPT_BRIDGE_NAME)_Highlight");
        var backgroundColor = \(FIND_TEXT_HIGHLIGHT_SEARCH_RESULT_COUNT_JS_SOURCE) == 0 ? "#FF9732" : "#FFFF00";
        span.setAttribute("style", "color: #000 !important; background: " + backgroundColor + " !important; padding: 0px !important; margin: 0px !important; border: 0px !important;");

        text = document.createTextNode(value.substr(idx + keyword.length));
        element.deleteData(idx, value.length - idx);

        var next = element.nextSibling;
        element.parentNode.insertBefore(span, next);
        element.parentNode.insertBefore(text, next);
        element = text;

        \(FIND_TEXT_HIGHLIGHT_SEARCH_RESULT_COUNT_JS_SOURCE)++;
        elementTmp = document.createTextNode(
          value.substr(idx + keyword.length)
        );

        var _windowId = \(WINDOW_ID_VARIABLE_JS_SOURCE);

        window.webkit.messageHandlers["onFindResultReceived"].postMessage(
            {
                'findResult': {
                    'activeMatchOrdinal': \(FIND_TEXT_HIGHLIGHT_CURRENT_HIGHLIGHT_JS_SOURCE),
                    'numberOfMatches': \(FIND_TEXT_HIGHLIGHT_SEARCH_RESULT_COUNT_JS_SOURCE),
                    'isDoneCounting': \(FIND_TEXT_HIGHLIGHT_IS_DONE_COUNTING_JS_SOURCE)
                },
                '_windowId': _windowId
            }
        );
      }
    } else if (element.nodeType == 1) {
      // Element node
      if (
        element.style.display != "none" &&
        element.nodeName.toLowerCase() != "select"
      ) {
        for (var i = element.childNodes.length - 1; i >= 0; i--) {
          window.\(JAVASCRIPT_BRIDGE_NAME)._findAllAsyncForElement(
            element.childNodes[element.childNodes.length - 1 - i],
            keyword
          );
        }
      }
    }
  }
}

// the main entry point to start the search
window.\(JAVASCRIPT_BRIDGE_NAME)._findAllAsync = function(keyword) {
  window.\(JAVASCRIPT_BRIDGE_NAME)._clearMatches();
  window.\(JAVASCRIPT_BRIDGE_NAME)._findAllAsyncForElement(document.body, keyword.toLowerCase());
  \(FIND_TEXT_HIGHLIGHT_IS_DONE_COUNTING_JS_SOURCE) = true;

  var _windowId = \(WINDOW_ID_VARIABLE_JS_SOURCE);

  window.webkit.messageHandlers["onFindResultReceived"].postMessage(
      {
          'findResult': {
              'activeMatchOrdinal': \(FIND_TEXT_HIGHLIGHT_CURRENT_HIGHLIGHT_JS_SOURCE),
              'numberOfMatches': \(FIND_TEXT_HIGHLIGHT_SEARCH_RESULT_COUNT_JS_SOURCE),
              'isDoneCounting': \(FIND_TEXT_HIGHLIGHT_IS_DONE_COUNTING_JS_SOURCE)
          },
          '_windowId': _windowId
      }
  );
}

// helper function, recursively removes the highlights in elements and their children
window.\(JAVASCRIPT_BRIDGE_NAME)._clearMatchesForElement = function(element) {
  if (element) {
    if (element.nodeType == 1) {
      if (element.getAttribute("class") == "\(JAVASCRIPT_BRIDGE_NAME)_Highlight") {
        var text = element.removeChild(element.firstChild);
        element.parentNode.insertBefore(text, element);
        element.parentNode.removeChild(element);
        return true;
      } else {
        var normalize = false;
        for (var i = element.childNodes.length - 1; i >= 0; i--) {
          if (window.\(JAVASCRIPT_BRIDGE_NAME)._clearMatchesForElement(element.childNodes[i])) {
            normalize = true;
          }
        }
        if (normalize) {
          element.normalize();
        }
      }
    }
  }
  return false;
}

// the main entry point to remove the highlights
window.\(JAVASCRIPT_BRIDGE_NAME)._clearMatches = function() {
  \(FIND_TEXT_HIGHLIGHT_SEARCH_RESULT_COUNT_JS_SOURCE) = 0;
  \(FIND_TEXT_HIGHLIGHT_CURRENT_HIGHLIGHT_JS_SOURCE) = 0;
  \(FIND_TEXT_HIGHLIGHT_IS_DONE_COUNTING_JS_SOURCE) = false;
  window.\(JAVASCRIPT_BRIDGE_NAME)._clearMatchesForElement(document.body);
}

window.\(JAVASCRIPT_BRIDGE_NAME)._findNext = function(forward) {
  if (\(FIND_TEXT_HIGHLIGHT_SEARCH_RESULT_COUNT_JS_SOURCE) <= 0) return;

  var idx = \(FIND_TEXT_HIGHLIGHT_CURRENT_HIGHLIGHT_JS_SOURCE) + (forward ? +1 : -1);
  idx =
    idx < 0
      ? \(FIND_TEXT_HIGHLIGHT_SEARCH_RESULT_COUNT_JS_SOURCE) - 1
      : idx >= \(FIND_TEXT_HIGHLIGHT_SEARCH_RESULT_COUNT_JS_SOURCE)
      ? 0
      : idx;
  \(FIND_TEXT_HIGHLIGHT_CURRENT_HIGHLIGHT_JS_SOURCE) = idx;

  var scrollTo = document.getElementById("\(JAVASCRIPT_BRIDGE_NAME)_SEARCH_WORD_" + idx);
  if (scrollTo) {
    var highlights = document.getElementsByClassName("\(JAVASCRIPT_BRIDGE_NAME)_Highlight");
    for (var i = 0; i < highlights.length; i++) {
      var span = highlights[i];
      span.style.backgroundColor = "#FFFF00";
    }
    scrollTo.style.backgroundColor = "#FF9732";

    scrollTo.scrollIntoView({
      behavior: "auto",
      block: "center"
    });

    var _windowId = \(WINDOW_ID_VARIABLE_JS_SOURCE);

    window.webkit.messageHandlers["onFindResultReceived"].postMessage(
        {
            'findResult': {
                'activeMatchOrdinal': \(FIND_TEXT_HIGHLIGHT_CURRENT_HIGHLIGHT_JS_SOURCE),
                'numberOfMatches': \(FIND_TEXT_HIGHLIGHT_SEARCH_RESULT_COUNT_JS_SOURCE),
                'isDoneCounting': \(FIND_TEXT_HIGHLIGHT_IS_DONE_COUNTING_JS_SOURCE)
            },
            '_windowId': _windowId
        }
    );
  }
}
"""
