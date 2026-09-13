package com.pichillilorenzo.flutter_inappwebview_android.webview;

import android.content.Context;
import android.view.View;
import android.view.ViewGroup;

import com.pichillilorenzo.flutter_inappwebview_android.InAppWebViewFlutterPlugin;
import com.pichillilorenzo.flutter_inappwebview_android.headless_in_app_webview.HeadlessInAppWebView;
import com.pichillilorenzo.flutter_inappwebview_android.headless_in_app_webview.HeadlessInAppWebViewManager;
import com.pichillilorenzo.flutter_inappwebview_android.webview.in_app_webview.FlutterWebView;

import java.util.HashMap;

import io.flutter.plugin.common.StandardMessageCodec;
import io.flutter.plugin.platform.PlatformView;
import io.flutter.plugin.platform.PlatformViewFactory;

public class FlutterWebViewFactory extends PlatformViewFactory {
  public static final String VIEW_TYPE_ID = "com.pichillilorenzo/flutter_inappwebview";
  private final InAppWebViewFlutterPlugin plugin;

  public FlutterWebViewFactory(final InAppWebViewFlutterPlugin plugin) {
    super(StandardMessageCodec.INSTANCE);
    this.plugin = plugin;
  }

  @Override
  public PlatformView create(Context context, int id, Object args) {
    HashMap<String, Object> params = (HashMap<String, Object>) args;
    FlutterWebView flutterWebView = null;
    Object viewId = id;

    String keepAliveId = (String) params.get("keepAliveId");
    String headlessWebViewId = (String) params.get("headlessWebViewId");

    HeadlessInAppWebViewManager headlessInAppWebViewManager = plugin.headlessInAppWebViewManager;
    if (headlessWebViewId != null && headlessInAppWebViewManager != null) {
      HeadlessInAppWebView headlessInAppWebView = headlessInAppWebViewManager.webViews.get(headlessWebViewId);
      if (headlessInAppWebView != null) {
        flutterWebView = headlessInAppWebView.disposeAndGetFlutterWebView();
        if (flutterWebView != null) {
          flutterWebView.keepAliveId = keepAliveId;
        }
      }
    }

    InAppWebViewManager inAppWebViewManager = plugin.inAppWebViewManager;
    if (keepAliveId != null && flutterWebView == null && inAppWebViewManager != null) {
      flutterWebView = inAppWebViewManager.keepAliveWebViews.get(keepAliveId);
      if (flutterWebView != null) {
        // be sure to remove the view from the previous parent.
        View view = flutterWebView.getView();
        if (view != null) {
          ViewGroup parent = (ViewGroup) view.getParent();
          if (parent != null) {
            parent.removeView(view);
          }
        }
      }
    }

    boolean shouldMakeInitialLoad = flutterWebView == null;
    if (flutterWebView == null) {
      if (keepAliveId != null) {
        viewId = keepAliveId;
      }
      flutterWebView = new FlutterWebView(plugin, context, viewId, params);
    }

    if (keepAliveId != null && inAppWebViewManager != null) {
      inAppWebViewManager.keepAliveWebViews.put(keepAliveId, flutterWebView);
    }

    if (shouldMakeInitialLoad) {
      flutterWebView.makeInitialLoad(params);
    }
    
    return flutterWebView;
  }
}

