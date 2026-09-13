package com.pichillilorenzo.flutter_inappwebview_android.service_worker;

import android.os.Build;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.annotation.RequiresApi;
import androidx.webkit.ServiceWorkerControllerCompat;
import androidx.webkit.ServiceWorkerWebSettingsCompat;
import androidx.webkit.WebViewFeature;

import com.pichillilorenzo.flutter_inappwebview_android.Util;
import com.pichillilorenzo.flutter_inappwebview_android.types.BaseCallbackResultImpl;
import com.pichillilorenzo.flutter_inappwebview_android.types.ChannelDelegateImpl;
import com.pichillilorenzo.flutter_inappwebview_android.types.SyncBaseCallbackResultImpl;
import com.pichillilorenzo.flutter_inappwebview_android.types.WebResourceRequestExt;
import com.pichillilorenzo.flutter_inappwebview_android.types.WebResourceResponseExt;

import java.util.Map;

import io.flutter.plugin.common.MethodCall;
import io.flutter.plugin.common.MethodChannel;

@RequiresApi(api = Build.VERSION_CODES.N)
public class ServiceWorkerChannelDelegate extends ChannelDelegateImpl {
  @Nullable
  private ServiceWorkerManager serviceWorkerManager;

  public ServiceWorkerChannelDelegate(@NonNull ServiceWorkerManager serviceWorkerManager, @NonNull MethodChannel channel) {
    super(channel);
    this.serviceWorkerManager = serviceWorkerManager;
  }

  @Override
  public void onMethodCall(@NonNull MethodCall call, @NonNull MethodChannel.Result result) {
    ServiceWorkerManager.init();
    ServiceWorkerControllerCompat serviceWorkerController = ServiceWorkerManager.serviceWorkerController;
    ServiceWorkerWebSettingsCompat serviceWorkerWebSettings = (serviceWorkerController != null) ? 
            serviceWorkerController.getServiceWorkerWebSettings() : null;

    switch (call.method) {
      case "setServiceWorkerClient":
        if (serviceWorkerManager != null) {
          Boolean isNull = (Boolean) call.argument("isNull");
          serviceWorkerManager.setServiceWorkerClient(isNull);
          result.success(true);
        } else {
          result.success(false);
        }
        break;
      case "getAllowContentAccess":
        if (serviceWorkerWebSettings != null && WebViewFeature.isFeatureSupported(WebViewFeature.SERVICE_WORKER_CONTENT_ACCESS)) {
          result.success(serviceWorkerWebSettings.getAllowContentAccess());
        } else {
          result.success(false);
        }
        break;
      case "getAllowFileAccess":
        if (serviceWorkerWebSettings != null && WebViewFeature.isFeatureSupported(WebViewFeature.SERVICE_WORKER_FILE_ACCESS)) {
          result.success(serviceWorkerWebSettings.getAllowFileAccess());
        } else {
          result.success(false);
        }
        break;
      case "getBlockNetworkLoads":
        if (serviceWorkerWebSettings != null && WebViewFeature.isFeatureSupported(WebViewFeature.SERVICE_WORKER_BLOCK_NETWORK_LOADS)) {
          result.success(serviceWorkerWebSettings.getBlockNetworkLoads());
        } else {
          result.success(false);
        }
        break;
      case "getCacheMode":
        if (serviceWorkerWebSettings != null && WebViewFeature.isFeatureSupported(WebViewFeature.SERVICE_WORKER_CACHE_MODE)) {
          result.success(serviceWorkerWebSettings.getCacheMode());
        } else {
          result.success(null);
        }
        break;
      case "setAllowContentAccess":
        if (serviceWorkerWebSettings != null && WebViewFeature.isFeatureSupported(WebViewFeature.SERVICE_WORKER_CONTENT_ACCESS)) {
          Boolean allow = (Boolean) call.argument("allow");
          serviceWorkerWebSettings.setAllowContentAccess(allow);
        }
        result.success(true);
        break;
      case "setAllowFileAccess":
        if (serviceWorkerWebSettings != null && WebViewFeature.isFeatureSupported(WebViewFeature.SERVICE_WORKER_FILE_ACCESS)) {
          Boolean allow = (Boolean) call.argument("allow");
          serviceWorkerWebSettings.setAllowFileAccess(allow);
        }
        result.success(true);
        break;
      case "setBlockNetworkLoads":
        if (serviceWorkerWebSettings != null && WebViewFeature.isFeatureSupported(WebViewFeature.SERVICE_WORKER_BLOCK_NETWORK_LOADS)) {
          Boolean flag = (Boolean) call.argument("flag");
          serviceWorkerWebSettings.setBlockNetworkLoads(flag);
        }
        result.success(true);
        break;
      case "setCacheMode":
        if (serviceWorkerWebSettings != null && WebViewFeature.isFeatureSupported(WebViewFeature.SERVICE_WORKER_CACHE_MODE)) {
          Integer mode = (Integer) call.argument("mode");
          serviceWorkerWebSettings.setCacheMode(mode);
        }
        result.success(true);
        break;
      default:
        result.notImplemented();
    }
  }

  public static class ShouldInterceptRequestCallback extends BaseCallbackResultImpl<WebResourceResponseExt> {
    @Nullable
    @Override
    public WebResourceResponseExt decodeResult(@Nullable Object obj) {
      return WebResourceResponseExt.fromMap((Map<String, Object>) obj);
    }
  }

  public void shouldInterceptRequest(WebResourceRequestExt request, @NonNull ShouldInterceptRequestCallback callback) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    channel.invokeMethod("shouldInterceptRequest", request.toMap(), callback);
  }

  public static class SyncShouldInterceptRequestCallback extends SyncBaseCallbackResultImpl<WebResourceResponseExt> {
    @Nullable
    @Override
    public WebResourceResponseExt decodeResult(@Nullable Object obj) {
      return (new ShouldInterceptRequestCallback()).decodeResult(obj);
    }
  }

  @Nullable
  public WebResourceResponseExt shouldInterceptRequest(WebResourceRequestExt request) throws InterruptedException {
    MethodChannel channel = getChannel();
    if (channel == null) return null;
    final SyncShouldInterceptRequestCallback callback = new SyncShouldInterceptRequestCallback();
    return Util.invokeMethodAndWaitResult(channel, "shouldInterceptRequest", request.toMap(), callback);
  }

  @Override
  public void dispose() {
    super.dispose();
    serviceWorkerManager = null;
  }
}
