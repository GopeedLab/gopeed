package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import java.util.Arrays;
import java.util.Map;

public class CustomTabsActionButton {
  private int id;

  @NonNull
  private byte[] icon;

  @NonNull
  private String description;
  
  private boolean shouldTint;

  public CustomTabsActionButton(int id, @NonNull byte[] icon, @NonNull String description, boolean shouldTint) {
    this.id = id;
    this.icon = icon;
    this.description = description;
    this.shouldTint = shouldTint;
  }

  @Nullable
  public static CustomTabsActionButton fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    int id = (int) map.get("id");
    byte[] icon = (byte[]) map.get("icon");
    String description = (String) map.get("description");
    boolean shouldTint = (boolean) map.get("shouldTint");
    return new CustomTabsActionButton(id, icon, description, shouldTint);
  }

  public int getId() {
    return id;
  }

  public void setId(int id) {
    this.id = id;
  }

  @NonNull
  public byte[] getIcon() {
    return icon;
  }

  public void setIcon(@NonNull byte[] icon) {
    this.icon = icon;
  }

  @NonNull
  public String getDescription() {
    return description;
  }

  public void setDescription(@NonNull String description) {
    this.description = description;
  }

  public boolean isShouldTint() {
    return shouldTint;
  }

  public void setShouldTint(boolean shouldTint) {
    this.shouldTint = shouldTint;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    CustomTabsActionButton that = (CustomTabsActionButton) o;

    if (id != that.id) return false;
    if (shouldTint != that.shouldTint) return false;
    if (!Arrays.equals(icon, that.icon)) return false;
    return description.equals(that.description);
  }

  @Override
  public int hashCode() {
    int result = id;
    result = 31 * result + Arrays.hashCode(icon);
    result = 31 * result + description.hashCode();
    result = 31 * result + (shouldTint ? 1 : 0);
    return result;
  }

  @Override
  public String toString() {
    return "CustomTabsActionButton{" +
            "id=" + id +
            ", icon=" + Arrays.toString(icon) +
            ", description='" + description + '\'' +
            ", shouldTint=" + shouldTint +
            '}';
  }
}
