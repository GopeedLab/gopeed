package com.pichillilorenzo.flutter_inappwebview_android.content_blocker;

import androidx.annotation.NonNull;

public class ContentBlocker {
    @NonNull
    private ContentBlockerTrigger trigger;
    @NonNull
    private ContentBlockerAction action;

    public ContentBlocker (@NonNull ContentBlockerTrigger trigger, @NonNull ContentBlockerAction action) {
        this.trigger = trigger;
        this.action = action;
    }

    @NonNull
    public ContentBlockerTrigger getTrigger() {
        return trigger;
    }

    public void setTrigger(@NonNull ContentBlockerTrigger trigger) {
        this.trigger = trigger;
    }

    @NonNull
    public ContentBlockerAction getAction() {
        return action;
    }

    public void setAction(@NonNull ContentBlockerAction action) {
        this.action = action;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;

        ContentBlocker that = (ContentBlocker) o;

        if (!trigger.equals(that.trigger)) return false;
        return action.equals(that.action);
    }

    @Override
    public int hashCode() {
        int result = trigger.hashCode();
        result = 31 * result + action.hashCode();
        return result;
    }

    @Override
    public String toString() {
        return "ContentBlocker{" +
                "trigger=" + trigger +
                ", action=" + action +
                '}';
    }
}
