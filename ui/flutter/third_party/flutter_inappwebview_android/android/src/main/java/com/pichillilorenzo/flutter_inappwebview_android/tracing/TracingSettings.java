package com.pichillilorenzo.flutter_inappwebview_android.tracing;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.webkit.TracingController;

import com.pichillilorenzo.flutter_inappwebview_android.ISettings;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class TracingSettings implements ISettings<TracingController> {

  public static final String LOG_TAG = "TracingSettings";

  @NonNull
  public List<Object> categories = new ArrayList<>();
  @Nullable
  public Integer tracingMode;

  @NonNull
  @Override
  public TracingSettings parse(@NonNull Map<String, Object> settings) {
    for (Map.Entry<String, Object> pair : settings.entrySet()) {
      String key = pair.getKey();
      Object value = pair.getValue();
      if (value == null) {
        continue;
      }

      switch (key) {
        case "categories":
          categories = (List<Object>) value;
          break;
        case "tracingMode":
          tracingMode = (Integer) value;
          break;
      }
    }

    return this;
  }

  @NonNull
  @Override
  public Map<String, Object> toMap() {
    Map<String, Object> settings = new HashMap<>();
    settings.put("categories", categories);
    settings.put("tracingMode", tracingMode);
    return settings;
  }

  @NonNull
  @Override
  public Map<String, Object> getRealSettings(@NonNull TracingController tracingController) {
    Map<String, Object> realSettings = toMap();
    return realSettings;
  }
}
