package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import java.util.Map;

public class ContentWorld {
  @NonNull
  private String name;

  public static final ContentWorld PAGE = new ContentWorld("page");
  public static final ContentWorld DEFAULT_CLIENT = new ContentWorld("defaultClient");

  private ContentWorld(@NonNull String name) {
    this.name = name;
  }

  public static ContentWorld world(@NonNull String name) {
    return new ContentWorld(name);
  }

  @Nullable
  public static ContentWorld fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    String name = (String) map.get("name");
    assert name != null;
    return new ContentWorld(name);
  }

  @NonNull
  public String getName() {
    return name;
  }

  public void setName(@NonNull String name) {
    this.name = name;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    ContentWorld that = (ContentWorld) o;

    return name.equals(that.name);
  }

  @Override
  public int hashCode() {
    return name.hashCode();
  }

  @Override
  public String toString() {
    return "ContentWorld{" +
            "name='" + name + '\'' +
            '}';
  }
}
