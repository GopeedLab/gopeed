package com.pichillilorenzo.flutter_inappwebview_android.types;

import android.content.Context;
import android.os.Build;
import android.util.Log;
import android.webkit.WebResourceResponse;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.webkit.WebViewAssetLoader;

import com.pichillilorenzo.flutter_inappwebview_android.InAppWebViewFlutterPlugin;
import com.pichillilorenzo.flutter_inappwebview_android.Util;

import java.io.ByteArrayInputStream;
import java.io.File;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import io.flutter.plugin.common.MethodChannel;

public class WebViewAssetLoaderExt implements Disposable {
  @Nullable
  public WebViewAssetLoader loader;
  @NonNull
  public List<PathHandlerExt> customPathHandlers;

  public WebViewAssetLoaderExt(@Nullable WebViewAssetLoader loader, @NonNull List<PathHandlerExt> customPathHandlers) {
    this.loader = loader;
    this.customPathHandlers = customPathHandlers;
  }

  @Nullable
  public static WebViewAssetLoaderExt fromMap(@Nullable Map<String, Object> map, @NonNull InAppWebViewFlutterPlugin plugin, @NonNull Context context) {
    if (map == null) {
      return null;
    }
    WebViewAssetLoader.Builder builder = new WebViewAssetLoader.Builder();
    String domain = (String) map.get("domain");
    Boolean httpAllowed = (Boolean) map.get("httpAllowed");
    List<Map<String, Object>> pathHandlers = (List<Map<String, Object>>) map.get("pathHandlers");
    List<PathHandlerExt> customPathHandlers = new ArrayList<>();
    if (domain != null && !domain.isEmpty()) {
      builder.setDomain(domain);
    }
    if (httpAllowed != null) {
      builder.setHttpAllowed(httpAllowed);
    }
    if (pathHandlers != null) {
      for (Map<String, Object> pathHandler : pathHandlers) {
        String type = (String) pathHandler.get("type");
        String path = (String) pathHandler.get("path");
        if (type == null || path == null) {
          continue;
        }
        switch (type) {
          case "AssetsPathHandler":
            WebViewAssetLoader.AssetsPathHandler assetsPathHandler =
                    new WebViewAssetLoader.AssetsPathHandler(context);
            builder.addPathHandler(path, assetsPathHandler);
            break;
          case "InternalStoragePathHandler":
            String directory = (String) pathHandler.get("directory");
            if (directory == null) {
              continue;
            }
            File dir = new File(directory);
            WebViewAssetLoader.InternalStoragePathHandler internalStoragePathHandler =
                    new WebViewAssetLoader.InternalStoragePathHandler(context, dir);
            builder.addPathHandler(path, internalStoragePathHandler);
            break;
          case "ResourcesPathHandler":
            WebViewAssetLoader.ResourcesPathHandler resourcesPathHandler = new WebViewAssetLoader.ResourcesPathHandler(context);
            builder.addPathHandler(path, resourcesPathHandler);
            break;
          default:
            String id = (String) pathHandler.get("id");
            if (id == null) {
              continue;
            }
            PathHandlerExt customPathHandler = new PathHandlerExt(id, plugin);
            builder.addPathHandler(path, customPathHandler);
            customPathHandlers.add(customPathHandler);
            break;
        }
      }
    }
    return new WebViewAssetLoaderExt(builder.build(), customPathHandlers);
  }

  @Override
  public void dispose() {
    for (PathHandlerExt pathHandler : customPathHandlers) {
      pathHandler.dispose();
    }
    customPathHandlers.clear();
  }

  public static class PathHandlerExt implements WebViewAssetLoader.PathHandler, Disposable {

    protected static final String LOG_TAG = "PathHandlerExt";
    public static final String METHOD_CHANNEL_NAME_PREFIX = "com.pichillilorenzo/flutter_inappwebview_custompathhandler_";

    @NonNull
    public String id;
    @Nullable
    public PathHandlerExtChannelDelegate channelDelegate;

    public PathHandlerExt(@NonNull String id, @NonNull InAppWebViewFlutterPlugin plugin) {
      this.id = id;
      final MethodChannel channel = new MethodChannel(plugin.messenger, METHOD_CHANNEL_NAME_PREFIX + id);
      this.channelDelegate = new PathHandlerExtChannelDelegate(this, channel);
    }

    @Nullable
    @Override
    public WebResourceResponse handle(@NonNull String path) {
      if (channelDelegate != null) {
        WebResourceResponseExt response = null;

        try {
          response = channelDelegate.handle(path);
        } catch (InterruptedException e) {
          Log.e(LOG_TAG, "", e);
          return null;
        }

        if (response != null) {
          String contentType = response.getContentType();
          String contentEncoding = response.getContentEncoding();
          byte[] data = response.getData();
          Map<String, String> responseHeaders = response.getHeaders();
          Integer statusCode = response.getStatusCode();
          String reasonPhrase = response.getReasonPhrase();

          ByteArrayInputStream inputStream = (data != null) ? new ByteArrayInputStream(data) : null;

          if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP && statusCode != null && reasonPhrase != null) {
            return new WebResourceResponse(contentType, contentEncoding, statusCode, reasonPhrase, responseHeaders, inputStream);
          } else {
            return new WebResourceResponse(contentType, contentEncoding, inputStream);
          }
        }
      }
      return null;
    }

    @Override
    public void dispose() {
      if (channelDelegate != null) {
        channelDelegate.dispose();
        channelDelegate = null;
      }
    }
  }

  public static class PathHandlerExtChannelDelegate extends ChannelDelegateImpl {

    @Nullable
    private PathHandlerExt pathHandler;

    public PathHandlerExtChannelDelegate(@NonNull PathHandlerExt pathHandler, @NonNull MethodChannel channel) {
      super(channel);
      this.pathHandler = pathHandler;
    }

    public static class HandleCallback extends BaseCallbackResultImpl<WebResourceResponseExt> {
      @Nullable
      @Override
      public WebResourceResponseExt decodeResult(@Nullable Object obj) {
        return WebResourceResponseExt.fromMap((Map<String, Object>) obj);
      }
    }

    public void handle(String path, @NonNull HandleCallback callback) {
      MethodChannel channel = getChannel();
      if (channel == null) return;
      Map<String, Object> obj = new HashMap<>();
      obj.put("path", path);
      channel.invokeMethod("handle", obj, callback);
    }

    public static class SyncHandleCallback extends SyncBaseCallbackResultImpl<WebResourceResponseExt> {
      @Nullable
      @Override
      public WebResourceResponseExt decodeResult(@Nullable Object obj) {
        return (new HandleCallback()).decodeResult(obj);
      }
    }

    @Nullable
    public WebResourceResponseExt handle(String path) throws InterruptedException {
      MethodChannel channel = getChannel();
      if (channel == null) return null;
      final SyncHandleCallback callback = new SyncHandleCallback();
      Map<String, Object> obj = new HashMap<>();
      obj.put("path", path);
      return Util.invokeMethodAndWaitResult(channel, "handle", obj, callback);
    }

    @Override
    public void dispose() {
      super.dispose();
      pathHandler = null;
    }
  }
}
