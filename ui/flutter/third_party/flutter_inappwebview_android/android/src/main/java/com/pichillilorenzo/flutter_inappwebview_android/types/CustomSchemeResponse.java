package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import java.util.Arrays;
import java.util.Map;

public class CustomSchemeResponse {
  @NonNull
  private byte[] data;
  @NonNull
  private String contentType;
  @NonNull
  private String contentEncoding;

  public CustomSchemeResponse(@NonNull byte[] data, @NonNull String contentType, @NonNull String contentEncoding) {
    this.data = data;
    this.contentType = contentType;
    this.contentEncoding = contentEncoding;
  }

  @Nullable
  public static CustomSchemeResponse fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    byte[] data = (byte[]) map.get("data");
    String contentType = (String) map.get("contentType");
    String contentEncoding = (String) map.get("contentEncoding");
    return new CustomSchemeResponse(data, contentType, contentEncoding);
  }

  @NonNull
  public byte[] getData() {
    return data;
  }

  public void setData(@NonNull byte[] data) {
    this.data = data;
  }

  @NonNull
  public String getContentType() {
    return contentType;
  }

  public void setContentType(@NonNull String contentType) {
    this.contentType = contentType;
  }

  @NonNull
  public String getContentEncoding() {
    return contentEncoding;
  }

  public void setContentEncoding(@NonNull String contentEncoding) {
    this.contentEncoding = contentEncoding;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    CustomSchemeResponse that = (CustomSchemeResponse) o;

    if (!Arrays.equals(data, that.data)) return false;
    if (!contentType.equals(that.contentType)) return false;
    return contentEncoding.equals(that.contentEncoding);
  }

  @Override
  public int hashCode() {
    int result = Arrays.hashCode(data);
    result = 31 * result + contentType.hashCode();
    result = 31 * result + contentEncoding.hashCode();
    return result;
  }

  @Override
  public String toString() {
    return "CustomSchemeResponse{" +
            "data=" + Arrays.toString(data) +
            ", contentType='" + contentType + '\'' +
            ", contentEncoding='" + contentEncoding + '\'' +
            '}';
  }
}
