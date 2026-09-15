package com.pichillilorenzo.flutter_inappwebview_android.content_blocker;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import java.util.Map;

public class ContentBlockerAction {
    @NonNull
    private ContentBlockerActionType type;

    @Nullable
    private String selector;

    ContentBlockerAction(@NonNull ContentBlockerActionType type, @Nullable String selector) {
        this.type = type;
        if (this.type.equals(ContentBlockerActionType.CSS_DISPLAY_NONE)) {
            assert(selector != null);
        }
        this.selector = selector;
    }

    public static ContentBlockerAction fromMap(Map<String, Object> map) {
        ContentBlockerActionType type = ContentBlockerActionType.fromValue((String) map.get("type"));
        String selector = (String) map.get("selector");
        return new ContentBlockerAction(type, selector);
    }

    @NonNull
    public ContentBlockerActionType getType() {
        return type;
    }

    public void setType(@NonNull ContentBlockerActionType type) {
        this.type = type;
    }

    public String getSelector() {
        return selector;
    }

    public void setSelector(String selector) {
        this.selector = selector;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;

        ContentBlockerAction that = (ContentBlockerAction) o;

        if (type != that.type) return false;
        return selector != null ? selector.equals(that.selector) : that.selector == null;
    }

    @Override
    public int hashCode() {
        int result = type.hashCode();
        result = 31 * result + (selector != null ? selector.hashCode() : 0);
        return result;
    }

    @Override
    public String toString() {
        return "ContentBlockerAction{" +
                "type=" + type +
                ", selector='" + selector + '\'' +
                '}';
    }
}
