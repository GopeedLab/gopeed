package com.pichillilorenzo.flutter_inappwebview_android.chrome_custom_tabs;

import android.app.Activity;
import android.content.Intent;
import android.net.Uri;
import android.os.Bundle;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.browser.customtabs.CustomTabsService;

import com.pichillilorenzo.flutter_inappwebview_android.types.ChannelDelegateImpl;
import com.pichillilorenzo.flutter_inappwebview_android.types.CustomTabsSecondaryToolbar;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

import io.flutter.plugin.common.MethodCall;
import io.flutter.plugin.common.MethodChannel;

public class ChromeCustomTabsChannelDelegate extends ChannelDelegateImpl {
  @Nullable
  private ChromeCustomTabsActivity chromeCustomTabsActivity;

  public ChromeCustomTabsChannelDelegate(@NonNull ChromeCustomTabsActivity chromeCustomTabsActivity, @NonNull MethodChannel channel) {
    super(channel);
    this.chromeCustomTabsActivity = chromeCustomTabsActivity;
  }

  @Override
  public void onMethodCall(@NonNull final MethodCall call, @NonNull final MethodChannel.Result result) {
    switch (call.method) {
      case "launchUrl":
        if (chromeCustomTabsActivity != null) {
          String url = (String) call.argument("url");
          if (url != null) {
            Map<String, String> headers = (Map<String, String>) call.argument("headers");
            String referrer = (String) call.argument("referrer");
            List<String> otherLikelyURLs = (List<String>) call.argument("otherLikelyURLs");
            chromeCustomTabsActivity.launchUrl(url, headers, referrer, otherLikelyURLs);
            result.success(true);
          } else {
            result.success(false);
          }
        } else {
          result.success(false);
        }
        break;
      case "mayLaunchUrl":
        if (chromeCustomTabsActivity != null) {
          String url = (String) call.argument("url");
          List<String> otherLikelyURLs = (List<String>) call.argument("otherLikelyURLs");
          result.success(chromeCustomTabsActivity.mayLaunchUrl(url, otherLikelyURLs));
        } else {
          result.success(false);
        }
        break;
      case "updateActionButton":
        if (chromeCustomTabsActivity != null) {
          byte[] icon = (byte[]) call.argument("icon");
          String description = (String) call.argument("description");
          chromeCustomTabsActivity.updateActionButton(icon, description);
          result.success(true);
        } else {
          result.success(false);
        }
        break;
      case "validateRelationship":
        if (chromeCustomTabsActivity != null && chromeCustomTabsActivity.customTabsSession != null) {
          Integer relation = (Integer) call.argument("relation");
          String origin = (String) call.argument("origin");
          result.success(chromeCustomTabsActivity.customTabsSession.validateRelationship(relation, Uri.parse(origin), null));
        } else {
          result.success(false);
        }
        break;
      case "updateSecondaryToolbar":
        if (chromeCustomTabsActivity != null) {
          CustomTabsSecondaryToolbar secondaryToolbar = CustomTabsSecondaryToolbar.fromMap((Map<String, Object>) call.argument("secondaryToolbar"));
          chromeCustomTabsActivity.updateSecondaryToolbar(secondaryToolbar);
          result.success(true);
        } else {
          result.success(false);
        }
        break;
      case "requestPostMessageChannel":
        if (chromeCustomTabsActivity != null && chromeCustomTabsActivity.customTabsSession != null) {
          String sourceOrigin = (String) call.argument("sourceOrigin");
          String targetOrigin = (String) call.argument("targetOrigin");
          result.success(chromeCustomTabsActivity.customTabsSession.requestPostMessageChannel(Uri.parse(sourceOrigin),
                  targetOrigin != null ? Uri.parse(targetOrigin) : null, new Bundle()));
        } else {
          result.success(false);
        }
        break;
      case "postMessage":
        if (chromeCustomTabsActivity != null && chromeCustomTabsActivity.customTabsSession != null) {
          String message = (String) call.argument("message");
          result.success(chromeCustomTabsActivity.customTabsSession.postMessage(message, new Bundle()));
        } else {
          result.success(CustomTabsService.RESULT_FAILURE_MESSAGING_ERROR);
        }
        break;
      case "isEngagementSignalsApiAvailable":
        if (chromeCustomTabsActivity != null && chromeCustomTabsActivity.customTabsSession != null) {
          try {
            result.success(chromeCustomTabsActivity.customTabsSession.isEngagementSignalsApiAvailable(new Bundle()));
          } catch (Throwable e) {
            result.success(false);
          }
        } else {
          result.success(false);
        }
        break;
      case "close":
        if (chromeCustomTabsActivity != null) {
          chromeCustomTabsActivity.onStop();
          chromeCustomTabsActivity.onDestroy();
          chromeCustomTabsActivity.close();

          if (chromeCustomTabsActivity.manager != null && chromeCustomTabsActivity.manager.plugin != null && 
                  chromeCustomTabsActivity.manager.plugin.activity != null) {
            Activity activity = chromeCustomTabsActivity.manager.plugin.activity;
            // https://stackoverflow.com/a/41596629/4637638
            Intent myIntent = new Intent(activity, activity.getClass());
            myIntent.addFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP);
            myIntent.addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP);
            activity.startActivity(myIntent);
          }
          chromeCustomTabsActivity.dispose();
          result.success(true);
        } else {
          result.success(false);
        }
        break;
      default:
        result.notImplemented();
    }
  }

  public void onServiceConnected() {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    channel.invokeMethod("onServiceConnected", obj);
  }

  public void onOpened() {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    channel.invokeMethod("onOpened", obj);
  }

  public void onCompletedInitialLoad() {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    channel.invokeMethod("onCompletedInitialLoad", obj);
  }

  public void onNavigationEvent(int navigationEvent) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("navigationEvent", navigationEvent);
    channel.invokeMethod("onNavigationEvent", obj);
  }

  public void onClosed() {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    channel.invokeMethod("onClosed", obj);
  }

  public void onItemActionPerform(int id, String url, String title) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("id", id);
    obj.put("url", url);
    obj.put("title", title);
    channel.invokeMethod("onItemActionPerform", obj);
  }

  public void onSecondaryItemActionPerform(String name, String url) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("name", name);
    obj.put("url", url);
    channel.invokeMethod("onSecondaryItemActionPerform", obj);
  }

  public void onRelationshipValidationResult(int relation, @NonNull Uri requestedOrigin, boolean result) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("relation", relation);
    obj.put("requestedOrigin", requestedOrigin.toString());
    obj.put("result", result);
    channel.invokeMethod("onRelationshipValidationResult", obj);
  }

  public void onMessageChannelReady() {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    channel.invokeMethod("onMessageChannelReady", obj);
  }

  public void onPostMessage(@NonNull String message) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("message", message);
    channel.invokeMethod("onPostMessage", obj);
  }

  public void onVerticalScrollEvent(boolean isDirectionUp) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("isDirectionUp", isDirectionUp);
    channel.invokeMethod("onVerticalScrollEvent", obj);
  }

  public void onGreatestScrollPercentageIncreased(int scrollPercentage) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("scrollPercentage", scrollPercentage);
    channel.invokeMethod("onGreatestScrollPercentageIncreased", obj);
  }

  public void onSessionEnded(boolean didUserInteract) {
    MethodChannel channel = getChannel();
    if (channel == null) return;
    Map<String, Object> obj = new HashMap<>();
    obj.put("didUserInteract", didUserInteract);
    channel.invokeMethod("onSessionEnded", obj);
  }

  @Override
  public void dispose() {
    super.dispose();
    chromeCustomTabsActivity = null; 
  }
}
