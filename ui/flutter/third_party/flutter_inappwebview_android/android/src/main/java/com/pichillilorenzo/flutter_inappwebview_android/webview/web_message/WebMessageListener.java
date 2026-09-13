package com.pichillilorenzo.flutter_inappwebview_android.webview.web_message;

import android.net.Uri;
import android.text.TextUtils;
import android.webkit.WebView;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.webkit.JavaScriptReplyProxy;
import androidx.webkit.WebMessageCompat;
import androidx.webkit.WebViewCompat;
import androidx.webkit.WebViewFeature;

import com.pichillilorenzo.flutter_inappwebview_android.Util;
import com.pichillilorenzo.flutter_inappwebview_android.plugin_scripts_js.JavaScriptBridgeJS;
import com.pichillilorenzo.flutter_inappwebview_android.types.Disposable;
import com.pichillilorenzo.flutter_inappwebview_android.types.WebMessageCompatExt;
import com.pichillilorenzo.flutter_inappwebview_android.webview.InAppWebViewInterface;
import com.pichillilorenzo.flutter_inappwebview_android.types.PluginScript;
import com.pichillilorenzo.flutter_inappwebview_android.types.UserScriptInjectionTime;
import com.pichillilorenzo.flutter_inappwebview_android.webview.in_app_webview.InAppWebView;

import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

import io.flutter.plugin.common.BinaryMessenger;
import io.flutter.plugin.common.MethodChannel;

public class WebMessageListener implements Disposable {
  protected static final String LOG_TAG = "WebMessageListener";
  public static final String METHOD_CHANNEL_NAME_PREFIX = "com.pichillilorenzo/flutter_inappwebview_web_message_listener_";

  @NonNull
  public String id;
  public String jsObjectName;
  public Set<String> allowedOriginRules;
  public WebViewCompat.WebMessageListener listener;
  public JavaScriptReplyProxy replyProxy;
  @Nullable
  public InAppWebViewInterface webView;
  @Nullable
  public WebMessageListenerChannelDelegate channelDelegate;

  public WebMessageListener(@NonNull String id,
                            @NonNull InAppWebViewInterface webView, @NonNull BinaryMessenger messenger,
                            @NonNull String jsObjectName, @NonNull Set<String> allowedOriginRules) {
    this.id = id;
    this.webView = webView;
    this.jsObjectName = jsObjectName;
    this.allowedOriginRules = allowedOriginRules;
    final MethodChannel channel = new MethodChannel(messenger, METHOD_CHANNEL_NAME_PREFIX + this.id + "_" + this.jsObjectName);
    this.channelDelegate = new WebMessageListenerChannelDelegate(this, channel);

    if (this.webView instanceof InAppWebView) {
      this.listener = new WebViewCompat.WebMessageListener() {
        @Override
        public void onPostMessage(@NonNull WebView view, @NonNull WebMessageCompat message, @NonNull Uri sourceOrigin,
                                  boolean isMainFrame, @NonNull JavaScriptReplyProxy javaScriptReplyProxy) {
          replyProxy = javaScriptReplyProxy;
          if (channelDelegate != null) {
            channelDelegate.onPostMessage(WebMessageCompatExt.fromMapWebMessageCompat(message),
                    sourceOrigin.toString().equals("null") ? null : sourceOrigin.toString(),
                    isMainFrame);
          }
        }
      };
    }
  }

  public void initJsInstance() {
    if (webView != null) {
      String jsObjectNameEscaped = Util.replaceAll(jsObjectName, "\'", "\\'");
      List<String> allowedOriginRulesStringList = new ArrayList<>();
      for (String allowedOriginRule : allowedOriginRules) {
        if ("*".equals(allowedOriginRule)) {
          allowedOriginRulesStringList.add("'*'");
        } else {
          Uri rule = Uri.parse(allowedOriginRule);
          String host = rule.getHost() != null ? "'" + Util.replaceAll(rule.getHost(), "\'", "\\'") + "'" : "null";
          allowedOriginRulesStringList.add("{scheme: '" + rule.getScheme() + "', host: " + host + ", port: " + (rule.getPort() != -1 ? rule.getPort() : "null") + "}");
        }
      }
      String allowedOriginRulesString = TextUtils.join(", ", allowedOriginRulesStringList);

      String source = "(function() {" +
              "  var allowedOriginRules = [" + allowedOriginRulesString + "];" +
              "  var isPageBlank = window.location.href === 'about:blank';" +
              "  var scheme = !isPageBlank ? window.location.protocol.replace(':', '') : null;" +
              "  var host = !isPageBlank ? window.location.hostname : null;" +
              "  var port = !isPageBlank ? window.location.port : null;" +
              "  if (window." + JavaScriptBridgeJS.JAVASCRIPT_BRIDGE_NAME + "._isOriginAllowed(allowedOriginRules, scheme, host, port)) {" +
              "      window['" + jsObjectNameEscaped + "'] = new FlutterInAppWebViewWebMessageListener('" + jsObjectNameEscaped + "');" +
              "  }" +
              "})();";
      webView.getUserContentController().addPluginScript(new PluginScript(
              "WebMessageListener-" + jsObjectName,
              source,
              UserScriptInjectionTime.AT_DOCUMENT_START,
              null,
              false,
              null
      ));
    }
  }

  @Nullable
  public static WebMessageListener fromMap(@NonNull InAppWebViewInterface webView, @NonNull BinaryMessenger messenger, @Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    String id = (String) map.get("id");
    String jsObjectName = (String) map.get("jsObjectName");
    assert jsObjectName != null;
    List<String> allowedOriginRuleList = (List<String>) map.get("allowedOriginRules");
    assert allowedOriginRuleList != null;
    Set<String> allowedOriginRules = new HashSet<>(allowedOriginRuleList);
    return new WebMessageListener(id, webView, messenger, jsObjectName, allowedOriginRules);
  }

  public void assertOriginRulesValid() throws Exception {
    int index = 0;
    for (String originRule : allowedOriginRules) {
      if (originRule == null) {
        throw new Exception("allowedOriginRules[" + index + "] is null");
      }
      if (originRule.isEmpty()) {
        throw new Exception("allowedOriginRules[" + index + "] is empty");
      }
      if ("*".equals(originRule)) {
        continue;
      }
      Uri url = Uri.parse(originRule);
      String scheme = url.getScheme();
      String host = url.getHost();
      String path = url.getPath();
      int port = url.getPort();
      if (scheme == null) {
        throw new Exception("allowedOriginRules " + originRule + " is invalid");
      }
      if (("http".equals(scheme) || "https".equals(scheme)) && (host == null || host.isEmpty())) {
        throw new Exception("allowedOriginRules " + originRule + " is invalid");
      }
      if (!"http".equals(scheme) && !"https".equals(scheme) && (host != null || port != -1)) {
        throw new Exception("allowedOriginRules " + originRule + " is invalid");
      }
      if ((host == null || host.isEmpty()) && port != -1) {
        throw new Exception("allowedOriginRules " + originRule + " is invalid");
      }
      if (!path.isEmpty()) {
        throw new Exception("allowedOriginRules " + originRule + " is invalid");
      }
      if (host != null) {
        int distance = host.indexOf("*");
        if (distance != 0 || (distance == 0 && !host.startsWith("*."))) {
          throw new Exception("allowedOriginRules " + originRule + " is invalid");
        }
        if (host.startsWith("[")) {
          if (!host.endsWith("]")) {
            throw new Exception("allowedOriginRules " + originRule + " is invalid");
          }
          String ipv6 = host.substring(1, host.length() - 1);
          if (!Util.isIPv6(ipv6)) {
            throw new Exception("allowedOriginRules " + originRule + " is invalid");
          }
        }
      }
      index++;
    }
  }

  public void postMessageForInAppWebView(WebMessageCompatExt message, @NonNull MethodChannel.Result result) {
    if (replyProxy != null && WebViewFeature.isFeatureSupported(WebViewFeature.WEB_MESSAGE_LISTENER)) {
      Object data = message.getData();
      if (data != null) {
        if (WebViewFeature.isFeatureSupported(WebViewFeature.WEB_MESSAGE_ARRAY_BUFFER) && message.getType() == WebMessageCompat.TYPE_ARRAY_BUFFER) {
          replyProxy.postMessage((byte[]) data);
        } else {
          replyProxy.postMessage(data.toString());
        }
      }
    }
    result.success(true);
  }

  public boolean isOriginAllowed(String scheme, String host, int port) {
    for (String allowedOriginRule : allowedOriginRules) {
      if ("*".equals(allowedOriginRule)) {
        return true;
      }
      if (scheme == null || scheme.isEmpty()) {
        continue;
      }
      if ((scheme == null || scheme.isEmpty()) && (host == null || host.isEmpty()) && (port == 0 || port == -1)) {
        continue;
      }
      Uri rule = Uri.parse(allowedOriginRule);
      int rulePort = rule.getPort() == -1 || rule.getPort() == 0 ? ("https".equals(rule.getScheme()) ? 443 : 80) : rule.getPort();
      int currentPort = port == 0 || port == -1 ? ("https".equals(scheme) ? 443 : 80) : port;
      String IPv6 = null;
      if (rule.getHost() != null && rule.getHost().startsWith("[")) {
        try {
          IPv6 = Util.normalizeIPv6(rule.getHost().substring(1, rule.getHost().length() - 1));
        } catch (Exception ignored) {
        }
      }
      String hostIPv6 = null;
      try {
        hostIPv6 = Util.normalizeIPv6(host);
      } catch (Exception ignored) {
      }

      boolean schemeAllowed = rule.getScheme().equals(scheme);

      boolean hostAllowed = rule.getHost() == null ||
              rule.getHost().isEmpty() ||
              rule.getHost().equals(host) ||
              (rule.getHost().startsWith("*") && host != null && host.contains(rule.getHost().split("\\*")[1])) ||
              (hostIPv6 != null && IPv6 != null && hostIPv6.equals(IPv6));

      boolean portAllowed = rulePort == currentPort;

      if (schemeAllowed && hostAllowed && portAllowed) {
        return true;
      }
    }
    return false;
  }

  public void dispose() {
    if (channelDelegate != null) {
      channelDelegate.dispose();
      channelDelegate = null;
    }
    listener = null;
    replyProxy = null;
    webView = null;
  }
}
