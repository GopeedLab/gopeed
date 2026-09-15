package com.pichillilorenzo.flutter_inappwebview_android.chrome_custom_tabs;

import android.content.Intent;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.browser.customtabs.CustomTabsIntent;
import androidx.browser.trusted.ScreenOrientation;
import androidx.browser.trusted.TrustedWebActivityDisplayMode;

import com.pichillilorenzo.flutter_inappwebview_android.ISettings;
import com.pichillilorenzo.flutter_inappwebview_android.types.AndroidResource;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class ChromeCustomTabsSettings implements ISettings<ChromeCustomTabsActivity> {

  final static String LOG_TAG = "ChromeCustomTabsSettings";

  @Deprecated
  public Boolean addDefaultShareMenuItem;
  public Integer shareState = CustomTabsIntent.SHARE_STATE_DEFAULT;
  public Boolean showTitle = true;
  @Nullable
  public String toolbarBackgroundColor;
  @Nullable
  public String navigationBarColor;
  @Nullable
  public String navigationBarDividerColor;
  @Nullable
  public String secondaryToolbarColor;
  public Boolean enableUrlBarHiding = false;
  public Boolean instantAppsEnabled = false;
  public String packageName;
  public Boolean keepAliveEnabled = false;
  public Boolean isSingleInstance = false;
  public Boolean noHistory = false;
  public Boolean isTrustedWebActivity = false;
  public List<String> additionalTrustedOrigins = new ArrayList<>();
  public TrustedWebActivityDisplayMode displayMode = null;
  public Integer screenOrientation = ScreenOrientation.DEFAULT;
  public List<AndroidResource> startAnimations = new ArrayList<>();
  public List<AndroidResource> exitAnimations = new ArrayList<>();
  public Boolean alwaysUseBrowserUI = false;

  @NonNull
  @Override
  public ChromeCustomTabsSettings parse(@NonNull Map<String, Object> options) {
    for (Map.Entry<String, Object> pair : options.entrySet()) {
      String key = pair.getKey();
      Object value = pair.getValue();
      if (value == null) {
        continue;
      }

      switch (key) {
        case "addDefaultShareMenuItem":
          addDefaultShareMenuItem = (Boolean) value;
          break;
        case "shareState":
          shareState = (Integer) value;
          break;
        case "showTitle":
          showTitle = (Boolean) value;
          break;
        case "toolbarBackgroundColor":
          toolbarBackgroundColor = (String) value;
          break;
        case "navigationBarColor":
          navigationBarColor = (String) value;
          break;
        case "navigationBarDividerColor":
          navigationBarDividerColor = (String) value;
          break;
        case "secondaryToolbarColor":
          secondaryToolbarColor = (String) value;
          break;
        case "enableUrlBarHiding":
          enableUrlBarHiding = (Boolean) value;
          break;
        case "instantAppsEnabled":
          instantAppsEnabled = (Boolean) value;
          break;
        case "packageName":
          packageName = (String) value;
          break;
        case "keepAliveEnabled":
          keepAliveEnabled = (Boolean) value;
          break;
        case "isSingleInstance":
          isSingleInstance = (Boolean) value;
          break;
        case "noHistory":
          noHistory = (Boolean) value;
          break;
        case "isTrustedWebActivity":
          isTrustedWebActivity = (Boolean) value;
          break;
        case "additionalTrustedOrigins":
          additionalTrustedOrigins = (List<String>) value;
          break;
        case "displayMode":
          Map<String, Object> displayModeMap = (Map<String, Object>) value;
          String displayModeType = (String) displayModeMap.get("type");
          if (displayModeType != null) {
            switch (displayModeType) {
              case "IMMERSIVE_MODE":
                boolean isSticky = (boolean) displayModeMap.get("isSticky");
                int layoutInDisplayCutoutMode = (int) displayModeMap.get("displayCutoutMode");
                displayMode = new TrustedWebActivityDisplayMode.ImmersiveMode(isSticky, layoutInDisplayCutoutMode);
                break;
              case "DEFAULT_MODE":
                displayMode = new TrustedWebActivityDisplayMode.DefaultMode();
                break;
            }
          }
          break;
        case "screenOrientation":
          screenOrientation = (Integer) value;
          break;
        case "startAnimations":
          List<Map<String, Object>> startAnimationsList = (List<Map<String, Object>>) value;
          for (Map<String, Object> startAnimation : startAnimationsList) {
            AndroidResource androidResource = AndroidResource.fromMap(startAnimation);
            if (androidResource != null) {
              startAnimations.add(AndroidResource.fromMap(startAnimation));
            }
          }
          break;
        case "exitAnimations":
          List<Map<String, Object>> exitAnimationsList = (List<Map<String, Object>>) value;
          for (Map<String, Object> exitAnimation : exitAnimationsList) {
            AndroidResource androidResource = AndroidResource.fromMap(exitAnimation);
            if (androidResource != null) {
              exitAnimations.add(AndroidResource.fromMap(exitAnimation));
            }
          }
          break;
        case "alwaysUseBrowserUI":
          alwaysUseBrowserUI = (Boolean) value;
          break;
      }
    }

    return this;
  }

  @NonNull
  @Override
  public Map<String, Object> toMap() {
    Map<String, Object> options = new HashMap<>();
    options.put("addDefaultShareMenuItem", addDefaultShareMenuItem);
    options.put("showTitle", showTitle);
    options.put("toolbarBackgroundColor", toolbarBackgroundColor);
    options.put("navigationBarColor", navigationBarColor);
    options.put("navigationBarDividerColor", navigationBarDividerColor);
    options.put("secondaryToolbarColor", secondaryToolbarColor);
    options.put("enableUrlBarHiding", enableUrlBarHiding);
    options.put("instantAppsEnabled", instantAppsEnabled);
    options.put("packageName", packageName);
    options.put("keepAliveEnabled", keepAliveEnabled);
    options.put("isSingleInstance", isSingleInstance);
    options.put("noHistory", noHistory);
    options.put("isTrustedWebActivity", isTrustedWebActivity);
    options.put("additionalTrustedOrigins", additionalTrustedOrigins);
    options.put("screenOrientation", screenOrientation);
    options.put("alwaysUseBrowserUI", alwaysUseBrowserUI);
    return options;
  }

  @NonNull
  @Override
  public Map<String, Object> getRealSettings(@NonNull ChromeCustomTabsActivity chromeCustomTabsActivity) {
    Map<String, Object> realOptions = toMap();
    if (chromeCustomTabsActivity != null) {
      Intent intent = chromeCustomTabsActivity.getIntent();
      if (intent != null) {
        realOptions.put("packageName", intent.getPackage());
        realOptions.put("keepAliveEnabled", intent.hasExtra(CustomTabsHelper.EXTRA_CUSTOM_TABS_KEEP_ALIVE));
      }
    }
    return realOptions;
  }
}
