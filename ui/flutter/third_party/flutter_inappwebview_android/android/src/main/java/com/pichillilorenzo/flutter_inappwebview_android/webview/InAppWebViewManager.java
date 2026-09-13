package com.pichillilorenzo.flutter_inappwebview_android.webview;

import android.content.Context;
import android.content.pm.PackageInfo;
import android.os.Build;
import android.os.Message;
import android.view.View;
import android.view.ViewGroup;
import android.webkit.ValueCallback;
import android.webkit.WebSettings;
import android.webkit.WebView;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.webkit.WebViewCompat;
import androidx.webkit.WebViewFeature;

import com.pichillilorenzo.flutter_inappwebview_android.InAppWebViewFlutterPlugin;
import com.pichillilorenzo.flutter_inappwebview_android.types.ChannelDelegateImpl;
import com.pichillilorenzo.flutter_inappwebview_android.webview.in_app_webview.FlutterWebView;

import java.util.Collection;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

import io.flutter.plugin.common.MethodCall;
import io.flutter.plugin.common.MethodChannel;

public class InAppWebViewManager extends ChannelDelegateImpl {
  protected static final String LOG_TAG = "InAppWebViewManager";
  public static final String METHOD_CHANNEL_NAME = "com.pichillilorenzo/flutter_inappwebview_manager";
  
  @Nullable
  public InAppWebViewFlutterPlugin plugin;

  public final Map<String, FlutterWebView> keepAliveWebViews = new HashMap<>();

  public final Map<Integer, Message> windowWebViewMessages = new HashMap<>();
  public int windowAutoincrementId = 0;

  public InAppWebViewManager(final InAppWebViewFlutterPlugin plugin) {
    super(new MethodChannel(plugin.messenger, METHOD_CHANNEL_NAME));
    this.plugin = plugin;
  }

  @Override
  public void onMethodCall(@NonNull MethodCall call, @NonNull final MethodChannel.Result result) {
    switch (call.method) {
      case "getDefaultUserAgent":
        if (plugin != null) {
          result.success(WebSettings.getDefaultUserAgent(plugin.applicationContext));
        } else {
          result.success(null);
        }
        break;
      case "clearClientCertPreferences":
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
          WebView.clearClientCertPreferences(new Runnable() {
            @Override
            public void run() {
              result.success(true);
            }
          });
        } else {
          result.success(false);
        }
        break;
      case "getSafeBrowsingPrivacyPolicyUrl":
        if (WebViewFeature.isFeatureSupported(WebViewFeature.SAFE_BROWSING_PRIVACY_POLICY_URL)) {
          result.success(WebViewCompat.getSafeBrowsingPrivacyPolicyUrl().toString());
        } else
          result.success(null);
        break;
      case "setSafeBrowsingAllowlist":
        if (WebViewFeature.isFeatureSupported(WebViewFeature.SAFE_BROWSING_ALLOWLIST)) {
          Set<String> hosts = new HashSet<>((List<String>) call.argument("hosts"));
          WebViewCompat.setSafeBrowsingAllowlist(hosts, new ValueCallback<Boolean>() {
            @Override
            public void onReceiveValue(Boolean value) {
              result.success(value);
            }
          });
        }
        else if (WebViewFeature.isFeatureSupported(WebViewFeature.SAFE_BROWSING_WHITELIST)) {
          List<String> hosts = (List<String>) call.argument("hosts");
          WebViewCompat.setSafeBrowsingWhitelist(hosts, new ValueCallback<Boolean>() {
            @Override
            public void onReceiveValue(Boolean value) {
              result.success(value);
            }
          });
        } else
          result.success(false);
        break;
      case "getCurrentWebViewPackage":
        {
          Context context = null;
          if (plugin != null) {
            context = plugin.activity;
            if (context == null) {
              context = plugin.applicationContext;
            }
          }
          PackageInfo packageInfo = context != null ? WebViewCompat.getCurrentWebViewPackage(context) : null;
          result.success(packageInfo != null ? convertWebViewPackageToMap(packageInfo) : null);
        }
        break;
      case "setWebContentsDebuggingEnabled":
        {
          boolean debuggingEnabled = (boolean) call.argument("debuggingEnabled");
          if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.KITKAT) {
            WebView.setWebContentsDebuggingEnabled(debuggingEnabled);
          }
        }
        result.success(true);
        break;
      case "getVariationsHeader":
        if (WebViewFeature.isFeatureSupported(WebViewFeature.GET_VARIATIONS_HEADER)) {
          result.success(WebViewCompat.getVariationsHeader());
        }
        else {
          result.success(null);
        }
        break;
      case "isMultiProcessEnabled":
        if (WebViewFeature.isFeatureSupported(WebViewFeature.MULTI_PROCESS)) {
          result.success(WebViewCompat.isMultiProcessEnabled());
        }
        else {
          result.success(false);
        }
        break;
      case "disableWebView":
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
          WebView.disableWebView();
        }
        result.success(true);
        break;
      case "disposeKeepAlive":
        final String keepAliveId = (String) call.argument("keepAliveId");
        if (keepAliveId != null) {
          disposeKeepAlive(keepAliveId);
        }
        result.success(true);
        break;
      case "clearAllCache":
        {
          Context context = null;
          if (plugin != null) {
            context = plugin.activity;
            if (context == null) {
              context = plugin.applicationContext;
            }
            if (context != null) {
              boolean includeDiskFiles = (boolean) call.argument("includeDiskFiles");
              clearAllCache(context, includeDiskFiles);
            }
          }
        }
        result.success(true);
        break;
      default:
        result.notImplemented();
    }
  }

  @NonNull
  public Map<String, Object> convertWebViewPackageToMap(@NonNull PackageInfo webViewPackageInfo) {
    HashMap<String, Object> webViewPackageInfoMap = new HashMap<>();

    webViewPackageInfoMap.put("versionName", webViewPackageInfo.versionName);
    webViewPackageInfoMap.put("packageName", webViewPackageInfo.packageName);

    return webViewPackageInfoMap;
  }

  public void disposeKeepAlive(@NonNull String keepAliveId) {
    FlutterWebView flutterWebView = keepAliveWebViews.get(keepAliveId);
    if (flutterWebView != null) {
      flutterWebView.keepAliveId = null;
      // be sure to remove the view from the previous parent.
      View view = flutterWebView.getView();
      if (view != null) {
        ViewGroup parent = (ViewGroup) view.getParent();
        if (parent != null) {
          parent.removeView(view);
        }
      }
      flutterWebView.dispose();
    }
    if (keepAliveWebViews.containsKey(keepAliveId)) {
      keepAliveWebViews.put(keepAliveId, null);
    }
  }

  public void clearAllCache(@NonNull Context context, boolean includeDiskFiles) {
    WebView tempWebView = new WebView(context);
    tempWebView.clearCache(includeDiskFiles);
    tempWebView.destroy();
  }

  @Override
  public void dispose() {
    super.dispose();
    Collection<FlutterWebView> flutterWebViews = keepAliveWebViews.values();
    for (FlutterWebView flutterWebView : flutterWebViews) {
      String keepAliveId = flutterWebView.keepAliveId;
      if (keepAliveId != null) {
        disposeKeepAlive(flutterWebView.keepAliveId);
      }
    }
    keepAliveWebViews.clear();
    windowWebViewMessages.clear();
    plugin = null;
  }
}
