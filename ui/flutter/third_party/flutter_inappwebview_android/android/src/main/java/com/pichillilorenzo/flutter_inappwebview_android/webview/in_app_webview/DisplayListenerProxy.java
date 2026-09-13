package com.pichillilorenzo.flutter_inappwebview_android.webview.in_app_webview;

import static android.hardware.display.DisplayManager.DisplayListener;

import android.annotation.TargetApi;
import android.hardware.display.DisplayManager;
import android.os.Build;
import android.util.Log;
import java.lang.reflect.Field;
import java.util.ArrayList;

/**
 * Works around an Android WebView bug by filtering some DisplayListener invocations.
 * https://github.com/flutter/plugins/blob/master/packages/webview_flutter/android/src/main/java/io/flutter/plugins/webviewflutter/DisplayListenerProxy.java
 */
@TargetApi(Build.VERSION_CODES.KITKAT)
public
class DisplayListenerProxy {
    private static final String TAG = "DisplayListenerProxy";

    private ArrayList<DisplayListener> listenersBeforeWebView;

    /** Should be called prior to the webview's initialization. */
    public void onPreWebViewInitialization(DisplayManager displayManager) {
        listenersBeforeWebView = yoinkDisplayListeners(displayManager);
    }

    /** Should be called after the webview's initialization. */
    public void onPostWebViewInitialization(final DisplayManager displayManager) {
        final ArrayList<DisplayListener> webViewListeners = yoinkDisplayListeners(displayManager);
        // We recorded the list of listeners prior to initializing webview, any new listeners we see
        // after initializing the webview are listeners added by the webview.
        webViewListeners.removeAll(listenersBeforeWebView);

        if (webViewListeners.isEmpty()) {
            // The Android WebView registers a single display listener per process (even if there
            // are multiple WebView instances) so this list is expected to be non-empty only the
            // first time a webview is initialized.
            // Note that in an add2app scenario if the application had instantiated a non Flutter
            // WebView prior to instantiating the Flutter WebView we are not able to get a reference
            // to the WebView's display listener and can't work around the bug.
            //
            // This means that webview resizes in add2app Flutter apps with a non Flutter WebView
            // running on a system with a webview prior to 58.0.3029.125 may crash (the Android's
            // behavior seems to be racy so it doesn't always happen).
            return;
        }

        for (DisplayListener webViewListener : webViewListeners) {
            // Note that while DisplayManager.unregisterDisplayListener throws when given an
            // unregistered listener, this isn't an issue as the WebView code never calls
            // unregisterDisplayListener.
            displayManager.unregisterDisplayListener(webViewListener);

            // We never explicitly unregister this listener as the webview's listener is never
            // unregistered (it's released when the process is terminated).
            displayManager.registerDisplayListener(
                    new DisplayListener() {
                        @Override
                        public void onDisplayAdded(int displayId) {
                            for (DisplayListener webViewListener : webViewListeners) {
                                webViewListener.onDisplayAdded(displayId);
                            }
                        }

                        @Override
                        public void onDisplayRemoved(int displayId) {
                            for (DisplayListener webViewListener : webViewListeners) {
                                webViewListener.onDisplayRemoved(displayId);
                            }
                        }

                        @Override
                        public void onDisplayChanged(int displayId) {
                            if (displayManager.getDisplay(displayId) == null) {
                                return;
                            }
                            for (DisplayListener webViewListener : webViewListeners) {
                                webViewListener.onDisplayChanged(displayId);
                            }
                        }
                    },
                    null);
        }
    }

    @SuppressWarnings({"unchecked", "PrivateApi"})
    private static ArrayList<DisplayListener> yoinkDisplayListeners(DisplayManager displayManager) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
            // We cannot use reflection on Android P, but it shouldn't matter as it shipped
            // with WebView 66.0.3359.158 and the WebView version the bug this code is working around was
            // fixed in 61.0.3116.0.
            return new ArrayList<>();
        }
        try {
            Field displayManagerGlobalField = DisplayManager.class.getDeclaredField("mGlobal");
            displayManagerGlobalField.setAccessible(true);
            Object displayManagerGlobal = displayManagerGlobalField.get(displayManager);
            Field displayListenersField =
                    displayManagerGlobal.getClass().getDeclaredField("mDisplayListeners");
            displayListenersField.setAccessible(true);
            ArrayList<Object> delegates =
                    (ArrayList<Object>) displayListenersField.get(displayManagerGlobal);

            Field listenerField = null;
            ArrayList<DisplayManager.DisplayListener> listeners = new ArrayList<>();
            for (Object delegate : delegates) {
                if (listenerField == null) {
                    listenerField = delegate.getClass().getField("mListener");
                    listenerField.setAccessible(true);
                }
                DisplayManager.DisplayListener listener =
                        (DisplayManager.DisplayListener) listenerField.get(delegate);
                listeners.add(listener);
            }
            return listeners;
        } catch (NoSuchFieldException | IllegalAccessException e) {
            Log.w(TAG, "Could not extract WebView's display listeners. " + e);
            return new ArrayList<>();
        }
    }
}