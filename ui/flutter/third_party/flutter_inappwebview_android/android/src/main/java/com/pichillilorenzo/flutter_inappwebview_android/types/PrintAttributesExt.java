package com.pichillilorenzo.flutter_inappwebview_android.types;

import android.os.Build;
import android.print.PrintAttributes;

import androidx.annotation.Nullable;
import androidx.annotation.RequiresApi;

import java.util.HashMap;
import java.util.Map;

@RequiresApi(api = Build.VERSION_CODES.KITKAT)
public class PrintAttributesExt {
  private int colorMode;
  @Nullable
  private Integer duplex;
  @Nullable
  private Integer orientation;
  @Nullable
  private MediaSizeExt mediaSize;
  @Nullable
  private ResolutionExt resolution;
  @Nullable
  private MarginsExt margins;

  @Nullable
  public static PrintAttributesExt fromPrintAttributes(@Nullable PrintAttributes attributes) {
    if (attributes == null) {
      return null;
    }
    PrintAttributesExt attributesExt = new PrintAttributesExt();
    attributesExt.colorMode = attributes.getColorMode();
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
      attributesExt.duplex = attributes.getDuplexMode();
    }
    PrintAttributes.MediaSize mediaSize = attributes.getMediaSize();
    if (mediaSize != null) {
      attributesExt.mediaSize = MediaSizeExt.fromMediaSize(mediaSize);
      attributesExt.orientation = mediaSize.isPortrait() ? 0 : 1;
    }
    attributesExt.resolution = ResolutionExt.fromResolution(attributes.getResolution());
    attributesExt.margins = MarginsExt.fromMargins(attributes.getMinMargins());
    return attributesExt;
  }

  public Map<String, Object> toMap() {
    Map<String, Object> obj = new HashMap<>();
    obj.put("colorMode", colorMode);
    obj.put("duplex", duplex);
    obj.put("orientation", orientation);
    obj.put("mediaSize", mediaSize != null ? mediaSize.toMap() : null);
    obj.put("resolution", resolution != null ? resolution.toMap() : null);
    obj.put("margins", margins != null ? margins.toMap() : null);
    return obj;
  }
}
