package com.pichillilorenzo.flutter_inappwebview_android.types;

public enum UserScriptInjectionTime {
  AT_DOCUMENT_START (0),
  AT_DOCUMENT_END (1);

  private final int value;

  private UserScriptInjectionTime(int value) {
    this.value = value;
  }

  public boolean equalsValue(int otherValue) {
    return value == otherValue;
  }

  public static UserScriptInjectionTime fromValue(int value) {
    for( UserScriptInjectionTime type : UserScriptInjectionTime.values()) {
      if(value == type.toValue())
        return type;
    }
    throw new IllegalArgumentException("No enum constant: " + value);
  }

  public int toValue() {
    return this.value;
  }
}
