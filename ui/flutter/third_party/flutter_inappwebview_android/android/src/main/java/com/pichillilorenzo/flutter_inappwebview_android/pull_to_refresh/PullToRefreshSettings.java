package com.pichillilorenzo.flutter_inappwebview_android.pull_to_refresh;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import com.pichillilorenzo.flutter_inappwebview_android.ISettings;

import java.util.HashMap;
import java.util.Map;

public class PullToRefreshSettings implements ISettings<PullToRefreshLayout> {
  public static final String LOG_TAG = "PullToRefreshSettings";

  public Boolean enabled = true;
  @Nullable
  public String color;
  @Nullable
  public String backgroundColor;
  @Nullable
  public Integer distanceToTriggerSync;
  @Nullable
  public Integer slingshotDistance;
  @Nullable
  public Integer size;

  @NonNull
  @Override
  public PullToRefreshSettings parse(@NonNull Map<String, Object> settings) {
    for (Map.Entry<String, Object> pair : settings.entrySet()) {
      String key = pair.getKey();
      Object value = pair.getValue();
      if (value == null) {
        continue;
      }

      switch (key) {
        case "enabled":
          enabled = (Boolean) value;
          break;
        case "color":
          color = (String) value;
          break;
        case "backgroundColor":
          backgroundColor = (String) value;
          break;
        case "distanceToTriggerSync":
          distanceToTriggerSync = (Integer) value;
          break;
        case "slingshotDistance":
          slingshotDistance = (Integer) value;
          break;
        case "size":
          size = (Integer) value;
          break;
      }
    }

    return this;
  }

  @NonNull
  public Map<String, Object> toMap() {
    Map<String, Object> settings = new HashMap<>();
    settings.put("enabled", enabled);
    settings.put("color", color);
    settings.put("backgroundColor", backgroundColor);
    settings.put("distanceToTriggerSync", distanceToTriggerSync);
    settings.put("slingshotDistance", slingshotDistance);
    settings.put("size", size);
    return settings;
  }

  @NonNull
  @Override
  public Map<String, Object> getRealSettings(@NonNull PullToRefreshLayout pullToRefreshLayout) {
    Map<String, Object> realSettings = toMap();
    return realSettings;
  }

}