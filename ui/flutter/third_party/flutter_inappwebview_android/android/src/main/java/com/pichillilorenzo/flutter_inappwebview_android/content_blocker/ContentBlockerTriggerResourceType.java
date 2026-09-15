package com.pichillilorenzo.flutter_inappwebview_android.content_blocker;

public enum ContentBlockerTriggerResourceType {
    DOCUMENT ("document"),
    IMAGE ("image"),
    STYLE_SHEET ("style-sheet"),
    SCRIPT ("script"),
    FONT ("font"),
    SVG_DOCUMENT ("svg-document"),
    MEDIA ("media"),
    POPUP ("popup"),
    RAW ("raw");

    private final String value;

    private ContentBlockerTriggerResourceType(String value) {
        this.value = value;
    }

    public boolean equalsValue(String otherValue) {
        return value.equals(otherValue);
    }

    public static ContentBlockerTriggerResourceType fromValue(String value) {
        for( ContentBlockerTriggerResourceType type : ContentBlockerTriggerResourceType.values()) {
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
