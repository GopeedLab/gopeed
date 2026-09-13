package com.pichillilorenzo.flutter_inappwebview_android;

import androidx.annotation.NonNull;

import java.util.Map;

public interface ISettings<T> {
  @NonNull ISettings<T> parse(@NonNull Map<String, Object> settings);
  @NonNull Map<String, Object> toMap();
  @NonNull Map<String, Object> getRealSettings(@NonNull T obj);
}
