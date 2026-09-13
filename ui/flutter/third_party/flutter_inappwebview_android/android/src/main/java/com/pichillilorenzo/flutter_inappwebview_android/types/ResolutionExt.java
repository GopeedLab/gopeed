package com.pichillilorenzo.flutter_inappwebview_android.types;

import android.os.Build;
import android.print.PrintAttributes;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.annotation.RequiresApi;

import java.util.HashMap;
import java.util.Map;

@RequiresApi(api = Build.VERSION_CODES.KITKAT)
public class ResolutionExt {
  @NonNull
  private String id;
  @NonNull
  private String label;
  private int verticalDpi;
  private int horizontalDpi;

  public ResolutionExt(@NonNull String id, @NonNull String label, int verticalDpi, int horizontalDpi) {
    this.id = id;
    this.label = label;
    this.verticalDpi = verticalDpi;
    this.horizontalDpi = horizontalDpi;
  }

  @Nullable
  public static ResolutionExt fromResolution(@Nullable PrintAttributes.Resolution resolution) {
    if (resolution == null) {
      return null;
    }
    return new ResolutionExt(
            resolution.getId(),
            resolution.getLabel(),
            resolution.getVerticalDpi(),
            resolution.getHorizontalDpi()
    );
  }

  @Nullable
  public static ResolutionExt fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    String id = (String) map.get("id");
    String label = (String) map.get("label");
    int verticalDpi = (int) map.get("verticalDpi");
    int horizontalDpi = (int) map.get("horizontalDpi");
    return new ResolutionExt(id, label, verticalDpi, horizontalDpi);
  }

  public PrintAttributes.Resolution toResolution() {
    return new PrintAttributes.Resolution(
            id, label, horizontalDpi, verticalDpi
    );
  }

  public Map<String, Object> toMap() {
    Map<String, Object> obj = new HashMap<>();
    obj.put("id", id);
    obj.put("label", label);
    obj.put("verticalDpi", verticalDpi);
    obj.put("horizontalDpi", horizontalDpi);
    return obj;
  }

  @NonNull
  public String getId() {
    return id;
  }

  public void setId(@NonNull String id) {
    this.id = id;
  }

  @NonNull
  public String getLabel() {
    return label;
  }

  public void setLabel(@NonNull String label) {
    this.label = label;
  }

  public int getVerticalDpi() {
    return verticalDpi;
  }

  public void setVerticalDpi(int verticalDpi) {
    this.verticalDpi = verticalDpi;
  }

  public int getHorizontalDpi() {
    return horizontalDpi;
  }

  public void setHorizontalDpi(int horizontalDpi) {
    this.horizontalDpi = horizontalDpi;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    ResolutionExt that = (ResolutionExt) o;

    if (verticalDpi != that.verticalDpi) return false;
    if (horizontalDpi != that.horizontalDpi) return false;
    if (!id.equals(that.id)) return false;
    return label.equals(that.label);
  }

  @Override
  public int hashCode() {
    int result = id.hashCode();
    result = 31 * result + label.hashCode();
    result = 31 * result + verticalDpi;
    result = 31 * result + horizontalDpi;
    return result;
  }

  @Override
  public String toString() {
    return "ResolutionExt{" +
            "id='" + id + '\'' +
            ", label='" + label + '\'' +
            ", verticalDpi=" + verticalDpi +
            ", horizontalDpi=" + horizontalDpi +
            '}';
  }
}
