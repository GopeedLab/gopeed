package com.pichillilorenzo.flutter_inappwebview_android.print_job;

import android.os.Build;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.annotation.RequiresApi;

import com.pichillilorenzo.flutter_inappwebview_android.ISettings;
import com.pichillilorenzo.flutter_inappwebview_android.types.MediaSizeExt;
import com.pichillilorenzo.flutter_inappwebview_android.types.ResolutionExt;

import java.util.HashMap;
import java.util.Map;

@RequiresApi(api = Build.VERSION_CODES.KITKAT)
public class PrintJobSettings implements ISettings<PrintJobController> {

  public static final String LOG_TAG = "PrintJobSettings";

  public Boolean handledByClient = false;
  @Nullable
  public String jobName;
  @Nullable
  public Integer orientation;
//  @Nullable
//  public MarginsExt margins;
  @Nullable
  public MediaSizeExt mediaSize;
  @Nullable
  public Integer colorMode;
  @Nullable
  public Integer duplexMode;
  @Nullable
  public ResolutionExt resolution;

  @NonNull
  @Override
  public PrintJobSettings parse(@NonNull Map<String, Object> settings) {
    for (Map.Entry<String, Object> pair : settings.entrySet()) {
      String key = pair.getKey();
      Object value = pair.getValue();
      if (value == null) {
        continue;
      }

      switch (key) {
        case "handledByClient":
          handledByClient = (Boolean) value;
          break;
        case "jobName":
          jobName = (String) value;
          break;
        case "orientation":
          orientation = (Integer) value;
          break;
//        case "margins":
//          margins = MarginsExt.fromMap((Map<String, Object>) value);
//          break;
        case "mediaSize":
          mediaSize = MediaSizeExt.fromMap((Map<String, Object>) value);
          break;
        case "colorMode":
          colorMode = (Integer) value;
          break;
        case "duplexMode":
          duplexMode = (Integer) value;
          break;
        case "resolution":
          resolution = ResolutionExt.fromMap((Map<String, Object>) value);
          break;
      }
    }

    return this;
  }

  @NonNull
  @Override
  public Map<String, Object> toMap() {
    Map<String, Object> settings = new HashMap<>();
    settings.put("handledByClient", handledByClient);
    settings.put("jobName", jobName);
    settings.put("orientation", orientation);
//    settings.put("margins", margins != null ? margins.toMap() : null);
    settings.put("mediaSize", mediaSize != null ? mediaSize.toMap() : null);
    settings.put("colorMode", colorMode);
    settings.put("duplexMode", duplexMode);
    settings.put("resolution", resolution != null ? resolution.toMap() : null);
    return settings;
  }

  @NonNull
  @Override
  public Map<String, Object> getRealSettings(@NonNull PrintJobController printJobController) {
    Map<String, Object> realSettings = toMap();
    return realSettings;
  }
}
