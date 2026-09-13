package com.pichillilorenzo.flutter_inappwebview_android.types;

import android.os.Build;
import android.print.PrintAttributes;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.annotation.RequiresApi;

import java.util.HashMap;
import java.util.Map;

@RequiresApi(api = Build.VERSION_CODES.KITKAT)
public class MediaSizeExt {
  @NonNull
  private String id;
  @Nullable
  private String label;
  private int widthMils;
  private int heightMils;

  public MediaSizeExt(@NonNull String id, @Nullable String label, int widthMils, int heightMils) {
    this.id = id;
    this.label = label;
    this.widthMils = widthMils;
    this.heightMils = heightMils;
  }

  @Nullable
  public static MediaSizeExt fromMediaSize(@Nullable PrintAttributes.MediaSize mediaSize) {
    if (mediaSize == null) {
      return null;
    }
    return new MediaSizeExt(
            mediaSize.getId(),
            null,
            mediaSize.getHeightMils(),
            mediaSize.getWidthMils()
    );
  }

  @Nullable
  public static MediaSizeExt fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    String id = (String) map.get("id");
    String label = (String) map.get("label");
    int widthMils = (int) map.get("widthMils");
    int heightMils = (int) map.get("heightMils");
    return new MediaSizeExt(id, label, widthMils, heightMils);
  }

  public PrintAttributes.MediaSize toMediaSize() {
    return new PrintAttributes.MediaSize(
            id, "Custom", widthMils, heightMils
    );
  }

  public Map<String, Object> toMap() {
    Map<String, Object> obj = new HashMap<>();
    obj.put("id", id);
    obj.put("label", label);
    obj.put("heightMils", heightMils);
    obj.put("widthMils", widthMils);
    return obj;
  }

  @NonNull
  public String getId() {
    return id;
  }

  public void setId(@NonNull String id) {
    this.id = id;
  }

  @Nullable
  public String getLabel() {
    return label;
  }

  public void setLabel(@Nullable String label) {
    this.label = label;
  }

  public int getWidthMils() {
    return widthMils;
  }

  public void setWidthMils(int widthMils) {
    this.widthMils = widthMils;
  }

  public int getHeightMils() {
    return heightMils;
  }

  public void setHeightMils(int heightMils) {
    this.heightMils = heightMils;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    MediaSizeExt that = (MediaSizeExt) o;

    if (widthMils != that.widthMils) return false;
    if (heightMils != that.heightMils) return false;
    if (!id.equals(that.id)) return false;
    return label != null ? label.equals(that.label) : that.label == null;
  }

  @Override
  public int hashCode() {
    int result = id.hashCode();
    result = 31 * result + (label != null ? label.hashCode() : 0);
    result = 31 * result + widthMils;
    result = 31 * result + heightMils;
    return result;
  }

  @Override
  public String toString() {
    return "MediaSizeExt{" +
            "id='" + id + '\'' +
            ", label='" + label + '\'' +
            ", widthMils=" + widthMils +
            ", heightMils=" + heightMils +
            '}';
  }
}
