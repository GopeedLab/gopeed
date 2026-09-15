package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import java.util.Map;

public class GeolocationPermissionShowPromptResponse {
  @NonNull
  private String origin;
  boolean allow;
  boolean retain;

  public GeolocationPermissionShowPromptResponse(@NonNull String origin, boolean allow, boolean retain) {
    this.origin = origin;
    this.allow = allow;
    this.retain = retain;
  }

  @Nullable
  public static GeolocationPermissionShowPromptResponse fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    String origin = (String) map.get("origin");
    boolean allow = (boolean) map.get("allow");
    boolean retain = (boolean) map.get("retain");
    return new GeolocationPermissionShowPromptResponse(origin, allow, retain);
  }

  @NonNull
  public String getOrigin() {
    return origin;
  }

  public void setOrigin(@NonNull String origin) {
    this.origin = origin;
  }

  public boolean isAllow() {
    return allow;
  }

  public void setAllow(boolean allow) {
    this.allow = allow;
  }

  public boolean isRetain() {
    return retain;
  }

  public void setRetain(boolean retain) {
    this.retain = retain;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    GeolocationPermissionShowPromptResponse that = (GeolocationPermissionShowPromptResponse) o;

    if (allow != that.allow) return false;
    if (retain != that.retain) return false;
    return origin.equals(that.origin);
  }

  @Override
  public int hashCode() {
    int result = origin.hashCode();
    result = 31 * result + (allow ? 1 : 0);
    result = 31 * result + (retain ? 1 : 0);
    return result;
  }

  @Override
  public String toString() {
    return "GeolocationPermissionShowPromptResponse{" +
            "origin='" + origin + '\'' +
            ", allow=" + allow +
            ", retain=" + retain +
            '}';
  }
}
