package com.pichillilorenzo.flutter_inappwebview_android.headless_in_app_webview;

import android.app.Activity;
import android.view.View;
import android.view.ViewGroup;
import android.widget.FrameLayout;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import com.pichillilorenzo.flutter_inappwebview_android.InAppWebViewFlutterPlugin;
import com.pichillilorenzo.flutter_inappwebview_android.Util;
import com.pichillilorenzo.flutter_inappwebview_android.types.Disposable;
import com.pichillilorenzo.flutter_inappwebview_android.types.Size2D;
import com.pichillilorenzo.flutter_inappwebview_android.webview.in_app_webview.FlutterWebView;

import java.util.Map;

import io.flutter.plugin.common.MethodChannel;

public class HeadlessInAppWebView implements Disposable {
  protected static final String LOG_TAG = "HeadlessInAppWebView";
  public static final String METHOD_CHANNEL_NAME_PREFIX = "com.pichillilorenzo/flutter_headless_inappwebview_";
  
  @NonNull
  public final String id;
  @Nullable
  public HeadlessWebViewChannelDelegate channelDelegate;
  @Nullable
  public FlutterWebView flutterWebView;
  @Nullable
  public InAppWebViewFlutterPlugin plugin;

  public HeadlessInAppWebView(@NonNull final InAppWebViewFlutterPlugin plugin, @NonNull String id, @NonNull FlutterWebView flutterWebView) {
    this.id = id;
    this.plugin = plugin;
    this.flutterWebView = flutterWebView;
    final MethodChannel channel = new MethodChannel(plugin.messenger, METHOD_CHANNEL_NAME_PREFIX + id);
    this.channelDelegate = new HeadlessWebViewChannelDelegate(this, channel);
  }

  public void onWebViewCreated() {
    if (channelDelegate != null) {
      channelDelegate.onWebViewCreated();
    }
  }

  public void prepare(Map<String, Object> params) {
    if (flutterWebView != null) {
      View view = flutterWebView.getView();
      if (view != null) {
        final Map<String, Object> initialSize = (Map<String, Object>) params.get("initialSize");
        Size2D size = Size2D.fromMap(initialSize);
        if (size == null) {
          size = new Size2D(-1, -1);
        }
        setSize(size);
        view.setVisibility(View.INVISIBLE);
      }
    }
    if (plugin != null && plugin.activity != null) {
      // Add the headless WebView to the view hierarchy.
      // This way is also possible to take screenshots.
      ViewGroup contentView = (ViewGroup) plugin.activity.findViewById(android.R.id.content);
      if (contentView != null) {
        ViewGroup mainView = (ViewGroup) (contentView).getChildAt(0);
        if (mainView != null && flutterWebView != null) {
          View view = flutterWebView.getView();
          if (view != null) {
            mainView.addView(view, 0);
          }
        } 
      }
    }
  }
  
  public void setSize(@NonNull Size2D size) {
    if (flutterWebView != null && flutterWebView.webView != null) {
      View view = flutterWebView.getView();
      if (view != null) {
        float scale = Util.getPixelDensity(view.getContext());
        Size2D fullscreenSize = Util.getFullscreenSize(view.getContext());
        int width = (int) (size.getWidth() == -1 ? fullscreenSize.getWidth() : (size.getWidth() * scale));
        int height = (int) (size.getWidth() == -1 ? fullscreenSize.getHeight() : (size.getHeight() * scale));
        view.setLayoutParams(new FrameLayout.LayoutParams(width, height));
      }
    }
  }

  @Nullable
  public Size2D getSize() {
    if (flutterWebView != null && flutterWebView.webView != null) {
      View view = flutterWebView.getView();
      if (view != null) {
        float scale = Util.getPixelDensity(view.getContext());
        Size2D fullscreenSize = Util.getFullscreenSize(view.getContext());
        ViewGroup.LayoutParams layoutParams = view.getLayoutParams();
        return new Size2D(
                fullscreenSize.getWidth() == layoutParams.width ? layoutParams.width : (layoutParams.width / scale),
                fullscreenSize.getHeight() == layoutParams.height ? layoutParams.height : (layoutParams.height / scale)
        );
      }
    }
    return null;
  }

  @Nullable
  public FlutterWebView disposeAndGetFlutterWebView() {
    FlutterWebView newFlutterWebView = flutterWebView;
    if (flutterWebView != null) {
      View view = flutterWebView.getView();
      if (view != null) {
        // restore WebView layout params and visibility
        view.setLayoutParams(new FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT));
        view.setVisibility(View.VISIBLE);
        // remove from parent
        ViewGroup parent = (ViewGroup) view.getParent();
        if (parent != null) {
          parent.removeView(view);
        }
      }
      // set to null to avoid to be disposed before calling "dispose()"
      flutterWebView = null;
      dispose();
    }
    return newFlutterWebView;
  }

  public void dispose() {
    if (channelDelegate != null) {
      channelDelegate.dispose();
      channelDelegate = null;
    }
    if (plugin != null) {
      HeadlessInAppWebViewManager headlessInAppWebViewManager = plugin.headlessInAppWebViewManager;
      if (headlessInAppWebViewManager != null && headlessInAppWebViewManager.webViews.containsKey(id)) {
        headlessInAppWebViewManager.webViews.put(id, null);
      }
      Activity activity =  plugin.activity;
      if (activity != null) {
        ViewGroup contentView = plugin.activity.findViewById(android.R.id.content);
        if (contentView != null) {
          ViewGroup mainView = (ViewGroup) (contentView).getChildAt(0);
          if (mainView != null && flutterWebView != null) {
            View view = flutterWebView.getView();
            if (view != null) {
              mainView.removeView(flutterWebView.getView());
            }
          }
        }
      }
    }
    if (flutterWebView != null) {
      flutterWebView.dispose();
    }
    flutterWebView = null;
    plugin = null;
  }
}
