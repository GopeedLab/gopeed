package com.pichillilorenzo.flutter_inappwebview_android.types;

import android.os.Build;
import android.print.PrintAttributes;

import androidx.annotation.Nullable;
import androidx.annotation.RequiresApi;

import java.util.HashMap;
import java.util.Map;

@RequiresApi(api = Build.VERSION_CODES.KITKAT)
public class MarginsExt {
  private double top;
  private double right;
  private double bottom;
  private double left;

  public MarginsExt() {}

  public MarginsExt(double top, double right, double bottom, double left) {
    this.top = top;
    this.right = right;
    this.bottom = bottom;
    this.left = left;
  }

  @Nullable
  public static MarginsExt fromMargins(@Nullable PrintAttributes.Margins margins) {
    if (margins == null) {
      return null;
    }
    MarginsExt marginsExt = new MarginsExt();
    marginsExt.top = milsToPixels(margins.getTopMils());
    marginsExt.right = milsToPixels(margins.getRightMils());
    marginsExt.bottom = milsToPixels(margins.getBottomMils());
    marginsExt.left = milsToPixels(margins.getLeftMils());
    return marginsExt;
  }

  @Nullable
  public static MarginsExt fromMap(@Nullable Map<String, Object> map) {
    if (map == null) {
      return null;
    }
    return new MarginsExt(
            (double) map.get("top"),
            (double) map.get("right"),
            (double) map.get("bottom"),
            (double) map.get("left"));
  }

  public PrintAttributes.Margins toMargins() {
    return new PrintAttributes.Margins(
            pixelsToMils(left),
            pixelsToMils(top),
            pixelsToMils(right),
            pixelsToMils(bottom)
    );
  }

  // from mils to pixels
  private static double milsToPixels(int mils) {
    return mils * 0.09600001209449;
  }

  // from pixels to mils
  private static int pixelsToMils(double pixels) {
    return (int) Math.round(pixels * 10.416665354331);
  }

  public Map<String, Object> toMap() {
    Map<String, Object> obj = new HashMap<>();
    obj.put("top", top);
    obj.put("right", right);
    obj.put("bottom", bottom);
    obj.put("left", left);
    return obj;
  }

  public double getTop() {
    return top;
  }

  public void setTop(double top) {
    this.top = top;
  }

  public double getRight() {
    return right;
  }

  public void setRight(double right) {
    this.right = right;
  }

  public double getBottom() {
    return bottom;
  }

  public void setBottom(double bottom) {
    this.bottom = bottom;
  }

  public double getLeft() {
    return left;
  }

  public void setLeft(double left) {
    this.left = left;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    MarginsExt that = (MarginsExt) o;

    if (Double.compare(that.top, top) != 0) return false;
    if (Double.compare(that.right, right) != 0) return false;
    if (Double.compare(that.bottom, bottom) != 0) return false;
    return Double.compare(that.left, left) == 0;
  }

  @Override
  public int hashCode() {
    int result;
    long temp;
    temp = Double.doubleToLongBits(top);
    result = (int) (temp ^ (temp >>> 32));
    temp = Double.doubleToLongBits(right);
    result = 31 * result + (int) (temp ^ (temp >>> 32));
    temp = Double.doubleToLongBits(bottom);
    result = 31 * result + (int) (temp ^ (temp >>> 32));
    temp = Double.doubleToLongBits(left);
    result = 31 * result + (int) (temp ^ (temp >>> 32));
    return result;
  }

  @Override
  public String toString() {
    return "MarginsExt{" +
            "top=" + top +
            ", right=" + right +
            ", bottom=" + bottom +
            ", left=" + left +
            '}';
  }
}
