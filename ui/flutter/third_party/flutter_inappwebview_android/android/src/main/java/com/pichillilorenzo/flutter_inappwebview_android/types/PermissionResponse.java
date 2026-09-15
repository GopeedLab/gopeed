package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import java.util.List;
import java.util.Map;

public class PermissionResponse {
  @NonNull
  private List<String> resources;
  @Nullable
  private Integer action;

  public PermissionResponse(@NonNull List<String> resources, @Nullable Integer action) {
    this.resources = resources;
    this.action = action;
  }

  @Nullable
  public static PermissionResponse fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    List<String> resources = (List<String>) map.get("resources");
    Integer action = (Integer) map.get("action");
    return new PermissionResponse(resources, action);
  }

  @NonNull
  public List<String> getResources() {
    return resources;
  }

  public void setResources(@NonNull List<String> resources) {
    this.resources = resources;
  }

  @Nullable
  public Integer getAction() {
    return action;
  }

  public void setAction(@Nullable Integer action) {
    this.action = action;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    PermissionResponse that = (PermissionResponse) o;

    if (!resources.equals(that.resources)) return false;
    return action != null ? action.equals(that.action) : that.action == null;
  }

  @Override
  public int hashCode() {
    int result = resources.hashCode();
    result = 31 * result + (action != null ? action.hashCode() : 0);
    return result;
  }

  @Override
  public String toString() {
    return "PermissionResponse{" +
            "resources=" + resources +
            ", action=" + action +
            '}';
  }
}
