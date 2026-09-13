package com.pichillilorenzo.flutter_inappwebview_android.types;

public enum NavigationActionPolicy {
  CANCEL(0),
  ALLOW(1);

  private final int value;

  private NavigationActionPolicy(int value) {
    this.value = value;
  }

  public boolean equalsValue(int otherValue) {
    return value == otherValue;
  }

  public static NavigationActionPolicy fromValue(int value) {
    for(NavigationActionPolicy type : NavigationActionPolicy.values()) {
      if(value == type.value)
        return type;
    }
    throw new IllegalArgumentException("No enum constant: " + value);
  }

  public int rawValue() {
    return this.value;
  }
  
  @Override
  public String toString() {
    return String.valueOf(this.value);
  }
}
