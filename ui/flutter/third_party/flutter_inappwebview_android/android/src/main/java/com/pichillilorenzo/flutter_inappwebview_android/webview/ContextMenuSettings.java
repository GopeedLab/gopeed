package com.pichillilorenzo.flutter_inappwebview_android.webview;

import androidx.annotation.NonNull;

import com.pichillilorenzo.flutter_inappwebview_android.ISettings;

import java.util.HashMap;
import java.util.Map;

public class ContextMenuSettings implements ISettings<Object> {
  public static final String LOG_TAG = "ContextMenuOptions";

  public Boolean hideDefaultSystemContextMenuItems = false;

  @NonNull
  @Override
  public ContextMenuSettings parse(@NonNull Map<String, Object> options) {
    for (Map.Entry<String, Object> pair : options.entrySet()) {
      String key = pair.getKey();
      Object value = pair.getValue();
      if (value == null) {
        continue;
      }

      switch (key) {
        case "hideDefaultSystemContextMenuItems":
          hideDefaultSystemContextMenuItems = (Boolean) value;
          break;
      }
    }

    return this;
  }

  @NonNull
  public Map<String, Object> toMap() {
    Map<String, Object> options = new HashMap<>();
    options.put("hideDefaultSystemContextMenuItems", hideDefaultSystemContextMenuItems);
    return options;
  }

  @NonNull
  @Override
  public Map<String, Object> getRealSettings(@NonNull Object obj) {
    Map<String, Object> realOptions = toMap();
    return realOptions;
  }

}
