package com.pichillilorenzo.flutter_inappwebview_android;

import android.webkit.ValueCallback;
import android.webkit.WebStorage;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import com.pichillilorenzo.flutter_inappwebview_android.types.ChannelDelegateImpl;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import io.flutter.plugin.common.MethodCall;
import io.flutter.plugin.common.MethodChannel;

public class MyWebStorage extends ChannelDelegateImpl {
  protected static final String LOG_TAG = "MyWebStorage";
  public static final String METHOD_CHANNEL_NAME = "com.pichillilorenzo/flutter_inappwebview_webstoragemanager";

  @Nullable
  public static WebStorage webStorageManager;
  @Nullable
  public InAppWebViewFlutterPlugin plugin;

  public MyWebStorage(@NonNull final InAppWebViewFlutterPlugin plugin) {
    super(new MethodChannel(plugin.messenger, METHOD_CHANNEL_NAME));
    this.plugin = plugin;
  }

  public static void init() {
    if (webStorageManager == null) {
      webStorageManager = WebStorage.getInstance();
    }
  }

  @Override
  public void onMethodCall(@NonNull MethodCall call, @NonNull MethodChannel.Result result) {
    init();

    switch (call.method) {
      case "getOrigins":
        getOrigins(result);
        break;
      case "deleteAllData":
        if (webStorageManager != null) {
          webStorageManager.deleteAllData();
          result.success(true);
        } else {
          result.success(false);
        }
        break;
      case "deleteOrigin":
        {
          if (webStorageManager != null) {
            String origin = (String) call.argument("origin");
            webStorageManager.deleteOrigin(origin);
            result.success(true);
          } else {
            result.success(false);
          }
        }
        break;
      case "getQuotaForOrigin":
        {
          String origin = (String) call.argument("origin");
          getQuotaForOrigin(origin, result);
        }
        break;
      case "getUsageForOrigin":
       {
          String origin = (String) call.argument("origin");
          getUsageForOrigin(origin, result);
       }
       break;
      default:
        result.notImplemented();
    }
  }

  public void getOrigins(final MethodChannel.Result result) {
    if (webStorageManager == null) {
      result.success(new ArrayList<>());
      return;
    }
    webStorageManager.getOrigins(new ValueCallback<Map>() {
      @Override
      public void onReceiveValue(Map value) {
        List<Map<String, Object>> origins = new ArrayList<>();
        for(Object key : value.keySet()) {
          WebStorage.Origin originObj = (WebStorage.Origin) value.get(key);

          Map<String, Object> originInfo = new HashMap<>();
          originInfo.put("origin", originObj.getOrigin());
          originInfo.put("quota", originObj.getQuota());
          originInfo.put("usage", originObj.getUsage());

          origins.add(originInfo);
        }
        result.success(origins);
      }
    });
  }

  public void getQuotaForOrigin(String origin, final MethodChannel.Result result) {
    if (webStorageManager == null) {
      result.success(0);
      return;
    }
    webStorageManager.getQuotaForOrigin(origin, new ValueCallback<Long>() {
      @Override
      public void onReceiveValue(Long value) {
        result.success(value);
      }
    });
  }

  public void getUsageForOrigin(String origin, final MethodChannel.Result result) {
    if (webStorageManager == null) {
      result.success(0);
      return;
    }
    webStorageManager.getUsageForOrigin(origin, new ValueCallback<Long>() {
      @Override
      public void onReceiveValue(Long value) {
        result.success(value);
      }
    });
  }

  @Override
  public void dispose() {
    super.dispose();
    plugin = null;
  }
}
