/*
 *
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 *
 */

package com.pichillilorenzo.flutter_inappwebview_android.in_app_browser;

import android.app.Activity;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.content.pm.ResolveInfo;
import android.net.Uri;
import android.os.Bundle;
import android.os.Parcelable;
import android.provider.Browser;
import android.util.Log;
import android.webkit.MimeTypeMap;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import com.pichillilorenzo.flutter_inappwebview_android.InAppWebViewFlutterPlugin;
import com.pichillilorenzo.flutter_inappwebview_android.types.ChannelDelegateImpl;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;

import io.flutter.plugin.common.MethodCall;
import io.flutter.plugin.common.MethodChannel;
import io.flutter.plugin.common.MethodChannel.Result;

/**
 * InAppBrowserManager
 */
public class InAppBrowserManager extends ChannelDelegateImpl {
  protected static final String LOG_TAG = "InAppBrowserManager";
  public static final String METHOD_CHANNEL_NAME = "com.pichillilorenzo/flutter_inappbrowser";
  
  @Nullable
  public InAppWebViewFlutterPlugin plugin;
  public String id;
  public static final Map<String, InAppBrowserManager> shared = new HashMap<>();

  public InAppBrowserManager(final InAppWebViewFlutterPlugin plugin) {
    super(new MethodChannel(plugin.messenger, METHOD_CHANNEL_NAME));
    this.id = UUID.randomUUID().toString();
    this.plugin = plugin;
    shared.put(this.id, this);
  }

  @Override
  public void onMethodCall(@NonNull MethodCall call, @NonNull MethodChannel.Result result) {
    switch (call.method) {
      case "open":
        if (plugin != null && plugin.activity != null) {
          open(plugin.activity, (Map<String, Object>) call.arguments());
          result.success(true);
        } else {
          result.success(false);
        }
        break;
      case "openWithSystemBrowser":
        if (plugin != null && plugin.activity != null) {
          String url = (String) call.argument("url");
          openWithSystemBrowser(plugin.activity, url, result);
        } else {
          result.success(false);
        }
        break;
      default:
        result.notImplemented();
    }
  }

  public static String getMimeType(String url) {
    String type = null;
    String extension = MimeTypeMap.getFileExtensionFromUrl(url);
    if (extension != null) {
      type = MimeTypeMap.getSingleton().getMimeTypeFromExtension(extension);
    }
    return type;
  }

  /**
   * Display a new browser with the specified URL.
   *
   * @param url the url to load.
   * @param result
   * @return "" if ok, or error message.
   */
  public void openWithSystemBrowser(Activity activity, String url, Result result) {
    try {
      Intent intent;
      intent = new Intent(Intent.ACTION_VIEW);
      // Omitting the MIME type for file: URLs causes "No Activity found to handle Intent".
      // Adding the MIME type to http: URLs causes them to not be handled by the downloader.
      Uri uri = Uri.parse(url);
      if ("file".equals(uri.getScheme())) {
        intent.setDataAndType(uri, getMimeType(url));
      } else {
        intent.setData(uri);
      }
      intent.putExtra(Browser.EXTRA_APPLICATION_ID, activity.getPackageName());
      // CB-10795: Avoid circular loops by preventing it from opening in the current app
      this.openExternalExcludeCurrentApp(activity, intent);
      result.success(true);
      // not catching FileUriExposedException explicitly because buildtools<24 doesn't know about it
    } catch (java.lang.RuntimeException e) {
      Log.d(LOG_TAG, url + " cannot be opened: " + e.toString());
      result.error(LOG_TAG, url + " cannot be opened!", null);
    }
  }

  /**
   * Opens the intent, providing a chooser that excludes the current app to avoid
   * circular loops.
   */
  public void openExternalExcludeCurrentApp(Activity activity, Intent intent) {
    String currentPackage = activity.getPackageName();
    boolean hasCurrentPackage = false;
    PackageManager pm = activity.getPackageManager();
    List<ResolveInfo> activities = pm.queryIntentActivities(intent, 0);
    ArrayList<Intent> targetIntents = new ArrayList<Intent>();
    for (ResolveInfo ri : activities) {
      if (!currentPackage.equals(ri.activityInfo.packageName)) {
        Intent targetIntent = (Intent) intent.clone();
        targetIntent.setPackage(ri.activityInfo.packageName);
        targetIntents.add(targetIntent);
      } else {
        hasCurrentPackage = true;
      }
    }
    // If the current app package isn't a target for this URL, then use
    // the normal launch behavior
    if (!hasCurrentPackage || targetIntents.size() == 0) {
      activity.startActivity(intent);
    }
    // If there's only one possible intent, launch it directly
    else if (targetIntents.size() == 1) {
      activity.startActivity(targetIntents.get(0));
    }
    // Otherwise, show a custom chooser without the current app listed
    else if (targetIntents.size() > 0) {
      Intent chooser = Intent.createChooser(targetIntents.remove(targetIntents.size() - 1), null);
      chooser.putExtra(Intent.EXTRA_INITIAL_INTENTS, targetIntents.toArray(new Parcelable[]{}));
      activity.startActivity(chooser);
    }
  }

  public void open(Activity activity, Map<String, Object> arguments) {
    String id = (String) arguments.get("id");
    Map<String, Object> urlRequest = (Map<String, Object>) arguments.get("urlRequest");
    String assetFilePath = (String) arguments.get("assetFilePath");
    String data = (String) arguments.get("data");
    String mimeType = (String) arguments.get("mimeType");
    String encoding = (String) arguments.get("encoding");
    String baseUrl = (String) arguments.get("baseUrl");
    String historyUrl = (String) arguments.get("historyUrl");
    Map<String, Object> settings = (Map<String, Object>) arguments.get("settings");
    Map<String, Object> contextMenu = (Map<String, Object>) arguments.get("contextMenu");
    Integer windowId = (Integer) arguments.get("windowId");
    List<Map<String, Object>> initialUserScripts = (List<Map<String, Object>>) arguments.get("initialUserScripts");
    Map<String, Object> pullToRefreshInitialSettings = (Map<String, Object>) arguments.get("pullToRefreshSettings");
    List<Map<String, Object>> menuItems = (List<Map<String, Object>>) arguments.get("menuItems");

    Bundle extras = new Bundle();
    extras.putString("fromActivity", activity.getClass().getName());
    extras.putSerializable("initialUrlRequest", (Serializable) urlRequest);
    extras.putString("initialFile", assetFilePath);
    extras.putString("initialData", data);
    extras.putString("initialMimeType", mimeType);
    extras.putString("initialEncoding", encoding);
    extras.putString("initialBaseUrl", baseUrl);
    extras.putString("initialHistoryUrl", historyUrl);
    extras.putString("id", id);
    extras.putString("managerId", this.id);
    extras.putSerializable("settings", (Serializable) settings);
    extras.putSerializable("contextMenu", (Serializable) contextMenu);
    extras.putInt("windowId", windowId != null ? windowId : -1);
    extras.putSerializable("initialUserScripts", (Serializable) initialUserScripts);
    extras.putSerializable("pullToRefreshInitialSettings", (Serializable) pullToRefreshInitialSettings);
    extras.putSerializable("menuItems", (Serializable) menuItems);
    startInAppBrowserActivity(activity, extras);
  }

  public void startInAppBrowserActivity(Activity activity, Bundle extras) {
    Intent intent = new Intent(activity, InAppBrowserActivity.class);
    if (extras != null)
      intent.putExtras(extras);
    activity.startActivity(intent);
  }

  @Override
  public void dispose() {
    super.dispose();
    shared.remove(this.id);
    plugin = null;
  }
}
