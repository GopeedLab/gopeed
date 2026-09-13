package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import com.pichillilorenzo.flutter_inappwebview_android.Util;

import java.util.Map;
import java.util.Objects;

public class InAppBrowserMenuItem {
  private int id;

  @NonNull
  private String title;

  @Nullable
  private Integer order;

  @Nullable
  private Object icon;

  @Nullable
  private String iconColor;

  private boolean showAsAction;

  public InAppBrowserMenuItem(int id, @NonNull String title, @Nullable Integer order, @Nullable Object icon,
                              @Nullable String iconColor, boolean showAsAction) {
    this.id = id;
    this.title = title;
    this.order = order;
    this.icon = icon;
    this.iconColor = iconColor;
    this.showAsAction = showAsAction;
  }

  @Nullable
  public static InAppBrowserMenuItem fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    int id = (int) map.get("id");
    String title = (String) map.get("title");
    Integer order = (Integer) map.get("order");
    Object icon = map.get("icon");
    if (icon instanceof Map) {
      icon = AndroidResource.fromMap((Map<String, Object>) map.get("icon"));
    } else if (!(icon instanceof byte[])) {
      icon = null;
    }
    String iconColor = (String) map.get("iconColor");
    boolean showAsAction = Util.getOrDefault( map, "showAsAction", false);
    return new InAppBrowserMenuItem(id, title, order, icon, iconColor, showAsAction);
  }

  public int getId() {
    return id;
  }

  public void setId(int id) {
    this.id = id;
  }

  @NonNull
  public String getTitle() {
    return title;
  }

  public void setTitle(@NonNull String title) {
    this.title = title;
  }

  @Nullable
  public Integer getOrder() {
    return order;
  }

  public void setOrder(@Nullable Integer order) {
    this.order = order;
  }

  @Nullable
  public Object getIcon() {
    return icon;
  }

  public void setIcon(@Nullable Object icon) {
    this.icon = icon;
  }

  @Nullable
  public String getIconColor() {
    return iconColor;
  }

  public void setIconColor(@Nullable String iconColor) {
    this.iconColor = iconColor;
  }

  public boolean isShowAsAction() {
    return showAsAction;
  }

  public void setShowAsAction(boolean showAsAction) {
    this.showAsAction = showAsAction;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    InAppBrowserMenuItem that = (InAppBrowserMenuItem) o;

    if (id != that.id) return false;
    if (showAsAction != that.showAsAction) return false;
    if (!title.equals(that.title)) return false;
    if (!Objects.equals(order, that.order)) return false;
    if (!Objects.equals(icon, that.icon)) return false;
    return Objects.equals(iconColor, that.iconColor);
  }

  @Override
  public int hashCode() {
    int result = id;
    result = 31 * result + title.hashCode();
    result = 31 * result + (order != null ? order.hashCode() : 0);
    result = 31 * result + (icon != null ? icon.hashCode() : 0);
    result = 31 * result + (iconColor != null ? iconColor.hashCode() : 0);
    result = 31 * result + (showAsAction ? 1 : 0);
    return result;
  }

  @Override
  public String toString() {
    return "InAppBrowserMenuItem{" +
            "id=" + id +
            ", title='" + title + '\'' +
            ", order=" + order +
            ", icon=" + icon +
            ", iconColor='" + iconColor + '\'' +
            ", showAsAction=" + showAsAction +
            '}';
  }
}
