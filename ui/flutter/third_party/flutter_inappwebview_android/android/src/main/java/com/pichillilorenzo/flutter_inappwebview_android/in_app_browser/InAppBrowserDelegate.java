package com.pichillilorenzo.flutter_inappwebview_android.in_app_browser;

import android.app.Activity;

import java.util.List;

public interface InAppBrowserDelegate {
  Activity getActivity();
  List<ActivityResultListener> getActivityResultListeners();
  void didChangeTitle(String title);
  void didStartNavigation(String url);
  void didUpdateVisitedHistory(String url);
  void didFinishNavigation(String url);
  void didFailNavigation(String url, int errorCode, String description);
  void didChangeProgress(int progress);
}
