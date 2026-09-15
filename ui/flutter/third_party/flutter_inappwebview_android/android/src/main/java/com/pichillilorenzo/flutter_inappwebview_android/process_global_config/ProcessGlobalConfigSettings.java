package com.pichillilorenzo.flutter_inappwebview_android.process_global_config;

import android.content.Context;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.webkit.ProcessGlobalConfig;
import androidx.webkit.WebViewFeature;

import com.pichillilorenzo.flutter_inappwebview_android.ISettings;

import java.io.File;
import java.util.HashMap;
import java.util.Map;

public class ProcessGlobalConfigSettings implements ISettings<ProcessGlobalConfig> {
  public static final String LOG_TAG = "ProcessGlobalConfigSettings";

  @Nullable
  public String dataDirectorySuffix;
  @Nullable
  public DirectoryBasePaths directoryBasePaths;

  @NonNull
  @Override
  public ProcessGlobalConfigSettings parse(@NonNull Map<String, Object> settings) {
    for (Map.Entry<String, Object> pair : settings.entrySet()) {
      String key = pair.getKey();
      Object value = pair.getValue();
      if (value == null) {
        continue;
      }

      switch (key) {
        case "dataDirectorySuffix":
          dataDirectorySuffix = (String) value;
          break;
        case "directoryBasePaths":
          directoryBasePaths = (new DirectoryBasePaths()).parse((Map<String, Object>) value);
          break;
      }
    }

    return this;
  }

  public ProcessGlobalConfig toProcessGlobalConfig(@NonNull Context context) {
    ProcessGlobalConfig config = new ProcessGlobalConfig();
    if (dataDirectorySuffix != null &&
            WebViewFeature.isStartupFeatureSupported(context, WebViewFeature.STARTUP_FEATURE_SET_DATA_DIRECTORY_SUFFIX)) {
      config.setDataDirectorySuffix(context, dataDirectorySuffix);
    }
    if (directoryBasePaths != null &&
            WebViewFeature.isStartupFeatureSupported(context, WebViewFeature.STARTUP_FEATURE_SET_DIRECTORY_BASE_PATHS)) {
      config.setDirectoryBasePaths(context,
              new File(directoryBasePaths.dataDirectoryBasePath),
              new File(directoryBasePaths.cacheDirectoryBasePath));
    }
    return config;
  }
  @NonNull
  public Map<String, Object> toMap() {
    Map<String, Object> settings = new HashMap<>();
    settings.put("dataDirectorySuffix", dataDirectorySuffix);
    return settings;
  }

  @NonNull
  @Override
  public Map<String, Object> getRealSettings(@NonNull ProcessGlobalConfig processGlobalConfig) {
    Map<String, Object> realSettings = toMap();
    return realSettings;
  }

  static class DirectoryBasePaths implements ISettings<Object> {
    public static final String LOG_TAG = "ProcessGlobalConfigSettings";

    public String cacheDirectoryBasePath;
    public String dataDirectoryBasePath;

    @NonNull
    @Override
    public DirectoryBasePaths parse(@NonNull Map<String, Object> settings) {
      for (Map.Entry<String, Object> pair : settings.entrySet()) {
        String key = pair.getKey();
        Object value = pair.getValue();
        if (value == null) {
          continue;
        }

        switch (key) {
          case "cacheDirectoryBasePath":
            cacheDirectoryBasePath = (String) value;
            break;
          case "dataDirectoryBasePath":
            dataDirectoryBasePath = (String) value;
            break;
        }
      }

      return this;
    }

    @NonNull
    public Map<String, Object> toMap() {
      Map<String, Object> settings = new HashMap<>();
      settings.put("cacheDirectoryBasePath", cacheDirectoryBasePath);
      settings.put("dataDirectoryBasePath", dataDirectoryBasePath);
      return settings;
    }

    @NonNull
    @Override
    public Map<String, Object> getRealSettings(@NonNull Object obj) {
      Map<String, Object> realSettings = toMap();
      return realSettings;
    }
  }
}