package com.pichillilorenzo.flutter_inappwebview_android.types;

import android.content.Context;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import java.util.HashMap;
import java.util.Map;

public class AndroidResource {
  @NonNull
  private String name;
  @Nullable
  private String defType;
  @Nullable
  private String defPackage;

  public AndroidResource(@NonNull String name, @Nullable String defType, @Nullable String defPackage) {
    this.name = name;
    this.defType = defType;
    this.defPackage = defPackage;
  }

  @Nullable
  public static AndroidResource fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    String name = (String) map.get("name");
    String defType = (String) map.get("defType");
    String defPackage = (String) map.get("defPackage");
    return new AndroidResource(name, defType, defPackage);
  }

  public Map<String, Object> toMap() {
    Map<String, Object> urlRequestMap = new HashMap<>();
    urlRequestMap.put("name", name);
    urlRequestMap.put("defType", defType);
    urlRequestMap.put("defPackage", defPackage);
    return urlRequestMap;
  }

  @NonNull
  public String getName() {
    return name;
  }

  public void setName(@NonNull String name) {
    this.name = name;
  }

  @Nullable
  public String getDefType() {
    return defType;
  }

  public void setDefType(@Nullable String defType) {
    this.defType = defType;
  }

  @Nullable
  public String getDefPackage() {
    return defPackage;
  }

  public void setDefPackage(@Nullable String defPackage) {
    this.defPackage = defPackage;
  }

  public int getIdentifier(@NonNull Context ctx) {
    return ctx.getResources().getIdentifier(name, defType, defPackage);
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    AndroidResource that = (AndroidResource) o;

    if (!name.equals(that.name)) return false;
    if (defType != null ? !defType.equals(that.defType) : that.defType != null) return false;
    return defPackage != null ? defPackage.equals(that.defPackage) : that.defPackage == null;
  }

  @Override
  public int hashCode() {
    int result = name.hashCode();
    result = 31 * result + (defType != null ? defType.hashCode() : 0);
    result = 31 * result + (defPackage != null ? defPackage.hashCode() : 0);
    return result;
  }

  @Override
  public String toString() {
    return "AndroidResource{" +
            "name='" + name + '\'' +
            ", type='" + defType + '\'' +
            ", defPackage='" + defPackage + '\'' +
            '}';
  }
}
