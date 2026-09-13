package com.pichillilorenzo.flutter_inappwebview_android.types;

import androidx.annotation.Nullable;

import java.util.HashMap;
import java.util.Map;

public class Size2D {
  private double width;
  private double height;

  public Size2D(double width, double height) {
    this.width = width;
    this.height = height;
  }

  @Nullable
  public static Size2D fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    Double width = (Double) map.get("width");
    Double height = (Double) map.get("height");
    assert width != null;
    assert height != null;
    return new Size2D(width, height);
  }

  public Map<String, Object> toMap() {
    Map<String, Object> sizeMap = new HashMap<>();
    sizeMap.put("width", width);
    sizeMap.put("height", height);
    return sizeMap;
  }

  public double getWidth() {
    return width;
  }

  public void setWidth(double width) {
    this.width = width;
  }

  public double getHeight() {
    return height;
  }

  public void setHeight(double height) {
    this.height = height;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    Size2D size = (Size2D) o;

    if (Double.compare(size.width, width) != 0) return false;
    return Double.compare(size.height, height) == 0;
  }

  @Override
  public int hashCode() {
    int result;
    long temp;
    temp = Double.doubleToLongBits(width);
    result = (int) (temp ^ (temp >>> 32));
    temp = Double.doubleToLongBits(height);
    result = 31 * result + (int) (temp ^ (temp >>> 32));
    return result;
  }

  @Override
  public String toString() {
    return "Size{" +
            "width=" + width +
            ", height=" + height +
            '}';
  }
}
