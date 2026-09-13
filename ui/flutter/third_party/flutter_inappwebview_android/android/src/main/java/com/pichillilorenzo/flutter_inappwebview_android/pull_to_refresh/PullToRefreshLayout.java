package com.pichillilorenzo.flutter_inappwebview_android.pull_to_refresh;

import android.content.Context;
import android.graphics.Color;
import android.util.AttributeSet;
import android.view.View;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout;

import com.pichillilorenzo.flutter_inappwebview_android.InAppWebViewFlutterPlugin;
import com.pichillilorenzo.flutter_inappwebview_android.types.Disposable;
import com.pichillilorenzo.flutter_inappwebview_android.webview.in_app_webview.InAppWebView;

import io.flutter.plugin.common.MethodChannel;

public class PullToRefreshLayout extends SwipeRefreshLayout implements Disposable {
  static final String LOG_TAG = "PullToRefreshLayout";
  public static final String METHOD_CHANNEL_NAME_PREFIX = "com.pichillilorenzo/flutter_inappwebview_pull_to_refresh_";
  
  @Nullable
  public PullToRefreshChannelDelegate channelDelegate;
  public PullToRefreshSettings settings = new PullToRefreshSettings();

  public PullToRefreshLayout(@NonNull Context context, @NonNull InAppWebViewFlutterPlugin plugin, 
                             @NonNull Object id, @NonNull PullToRefreshSettings settings) {
    super(context);
    this.settings = settings;
    final MethodChannel channel = new MethodChannel(plugin.messenger, METHOD_CHANNEL_NAME_PREFIX + id);
    this.channelDelegate = new PullToRefreshChannelDelegate(this, channel);
  }

  public PullToRefreshLayout(@NonNull Context context) {
    super(context);
  }

  public PullToRefreshLayout(@NonNull Context context, @Nullable AttributeSet attrs) {
    super(context, attrs);
  }

  public void prepare() {
    final PullToRefreshLayout self = this;

    setEnabled(settings.enabled);
    setOnChildScrollUpCallback(new OnChildScrollUpCallback() {
      @Override
      public boolean canChildScrollUp(@NonNull SwipeRefreshLayout parent, @Nullable View child) {
        if (child instanceof InAppWebView) {
          InAppWebView inAppWebView = (InAppWebView) child;
          return (inAppWebView.canScrollVertically() && inAppWebView.getScrollY() > 0) ||
                 (!inAppWebView.canScrollVertically() && inAppWebView.getScrollY() == 0);
        }
        return true;
      }
    });
    setOnRefreshListener(new SwipeRefreshLayout.OnRefreshListener() {
      @Override
      public void onRefresh() {
        if (channelDelegate == null) {
          self.setRefreshing(false);
          return;
        }
        channelDelegate.onRefresh();
      }
    });
    if (settings.color != null)
      setColorSchemeColors(Color.parseColor(settings.color));
    if (settings.backgroundColor != null)
      setProgressBackgroundColorSchemeColor(Color.parseColor(settings.backgroundColor));
    if (settings.distanceToTriggerSync != null)
      setDistanceToTriggerSync(settings.distanceToTriggerSync);
    if (settings.slingshotDistance != null)
      setSlingshotDistance(settings.slingshotDistance);
    if (settings.size != null)
      setSize(settings.size);
  }

  public void dispose() {
    if (channelDelegate != null) {
      channelDelegate.dispose();
      channelDelegate = null;
    }
    removeAllViews();
  }
}
