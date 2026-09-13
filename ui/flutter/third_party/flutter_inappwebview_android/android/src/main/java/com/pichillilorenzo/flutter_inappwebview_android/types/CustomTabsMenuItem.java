package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import java.util.Map;

public class CustomTabsMenuItem {
  private int id;

  @NonNull
  private String label;

  public CustomTabsMenuItem(int id, @NonNull String label) {
    this.id = id;
    this.label = label;
  }

  @Nullable
  public static CustomTabsMenuItem fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    int id = (int) map.get("id");
    String label = (String) map.get("label");
    return new CustomTabsMenuItem(id, label);
  }

  public int getId() {
    return id;
  }

  public void setId(int id) {
    this.id = id;
  }

  @NonNull
  public String getLabel() {
    return label;
  }

  public void setLabel(@NonNull String label) {
    this.label = label;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    CustomTabsMenuItem that = (CustomTabsMenuItem) o;

    if (id != that.id) return false;
    return label.equals(that.label);
  }

  @Override
  public int hashCode() {
    int result = id;
    result = 31 * result + label.hashCode();
    return result;
  }

  @Override
  public String toString() {
    return "CustomTabsMenuItem{" +
            "id=" + id +
            ", label='" + label + '\'' +
            '}';
  }
}
