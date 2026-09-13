package com.pichillilorenzo.flutter_inappwebview_android.find_interaction;

import androidx.annotation.NonNull;

import com.pichillilorenzo.flutter_inappwebview_android.ISettings;

import java.util.HashMap;
import java.util.Map;

public class FindInteractionSettings implements ISettings<FindInteractionController> {
  public static final String LOG_TAG = "FindInteractionSettings";


  @NonNull
  @Override
  public FindInteractionSettings parse(@NonNull Map<String, Object> settings) {
//    for (Map.Entry<String, Object> pair : settings.entrySet()) {
//      String key = pair.getKey();
//      Object value = pair.getValue();
//      if (value == null) {
//        continue;
//      }
//
//      switch (key) {
//
//      }
//    }

    return this;
  }

  @NonNull
  public Map<String, Object> toMap() {
    Map<String, Object> settings = new HashMap<>();
    return settings;
  }

  @NonNull
  @Override
  public Map<String, Object> getRealSettings(@NonNull FindInteractionController findInteractionController) {
    Map<String, Object> realSettings = toMap();
    return realSettings;
  }

}