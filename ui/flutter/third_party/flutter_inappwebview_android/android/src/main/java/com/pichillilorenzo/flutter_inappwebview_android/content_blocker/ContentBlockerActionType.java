package com.pichillilorenzo.flutter_inappwebview_android.content_blocker;

public enum ContentBlockerActionType {
    BLOCK ("block"),
    CSS_DISPLAY_NONE ("css-display-none"),
    MAKE_HTTPS ("make-https");

    private final String value;

    private ContentBlockerActionType(String value) {
        this.value = value;
    }

    public boolean equalsValue(String otherValue) {
        return value.equals(otherValue);
    }

    public static ContentBlockerActionType fromValue(String value) {
        for( ContentBlockerActionType type : ContentBlockerActionType.values()) {
            if(value.equals(type.value))
                return type;
        }
        throw new IllegalArgumentException("No enum constant: " + value);
    }

    @Override
    public String toString() {
        return this.value;
    }
}
