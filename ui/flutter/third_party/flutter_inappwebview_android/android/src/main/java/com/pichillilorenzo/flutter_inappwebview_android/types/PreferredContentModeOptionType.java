package com.pichillilorenzo.flutter_inappwebview_android.types;

public enum PreferredContentModeOptionType {
  RECOMMENDED (0),
  MOBILE (1),
  DESKTOP (2);

  private final int value;

  private PreferredContentModeOptionType(int value) {
    this.value = value;
  }

  public boolean equalsValue(int otherValue) {
    return value == otherValue;
  }

  public static PreferredContentModeOptionType fromValue(int value) {
    for( PreferredContentModeOptionType type : PreferredContentModeOptionType.values()) {
      if(value == type.toValue())
        return type;
    }
    throw new IllegalArgumentException("No enum constant: " + value);
  }

  public int toValue() {
    return this.value;
  }
}
