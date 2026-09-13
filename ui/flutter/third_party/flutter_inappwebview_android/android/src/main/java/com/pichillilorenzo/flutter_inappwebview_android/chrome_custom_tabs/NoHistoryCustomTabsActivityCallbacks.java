package com.pichillilorenzo.flutter_inappwebview_android.chrome_custom_tabs;

import android.app.Activity;
import android.app.Application;
import android.os.Bundle;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import com.pichillilorenzo.flutter_inappwebview_android.InAppWebViewFlutterPlugin;
import com.pichillilorenzo.flutter_inappwebview_android.types.Disposable;

import java.util.Collection;
import java.util.HashMap;
import java.util.Map;

import io.flutter.embedding.android.FlutterActivity;

public class NoHistoryCustomTabsActivityCallbacks implements Disposable {
  @Nullable
  public InAppWebViewFlutterPlugin plugin;

  public final Map<String, String> noHistoryBrowserIDs = new HashMap<>();

  public NoHistoryCustomTabsActivityCallbacks(@NonNull final InAppWebViewFlutterPlugin plugin) {
    this.plugin = plugin;
  }

  public Application.ActivityLifecycleCallbacks activityLifecycleCallbacks = new Application.ActivityLifecycleCallbacks() {
    @Override
    public void onActivityCreated(@NonNull Activity activity, @Nullable Bundle savedInstanceState) {
    }

    @Override
    public void onActivityStarted(@NonNull Activity activity) {
    }

    @Override
    public void onActivityResumed(@NonNull Activity activity) {
      if (activity instanceof FlutterActivity && plugin != null && plugin.chromeSafariBrowserManager != null) {
        Collection<String> browserIds = noHistoryBrowserIDs.values();
        for (String browserId : browserIds) {
          if (browserId != null) {
            noHistoryBrowserIDs.put(browserId, null);
            ChromeCustomTabsActivity browser = plugin.chromeSafariBrowserManager.browsers.get(browserId);
            if (browser != null) {
              browser.close();
              browser.dispose();
            }
          }
        }
      }
    }

    @Override
    public void onActivityPaused(@NonNull Activity activity) {
    }

    @Override
    public void onActivityStopped(@NonNull Activity activity) {
    }

    @Override
    public void onActivitySaveInstanceState(@NonNull Activity activity, @NonNull Bundle outState) {
    }

    @Override
    public void onActivityDestroyed(@NonNull Activity activity) {
    }
  };

  @Override
  public void dispose() {
    noHistoryBrowserIDs.clear();
    plugin = null;
  }
}
