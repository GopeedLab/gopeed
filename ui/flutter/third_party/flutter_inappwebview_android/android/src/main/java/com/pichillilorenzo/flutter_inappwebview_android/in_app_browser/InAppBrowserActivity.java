package com.pichillilorenzo.flutter_inappwebview_android.in_app_browser;

import android.annotation.SuppressLint;
import android.app.Activity;
import android.content.Intent;
import android.graphics.Color;
import android.graphics.drawable.ColorDrawable;
import android.os.Build;
import android.os.Bundle;
import android.os.Message;
import android.util.Log;
import android.view.KeyEvent;
import android.view.Menu;
import android.view.MenuInflater;
import android.view.MenuItem;
import android.view.View;
import android.webkit.WebView;
import android.widget.ProgressBar;
import android.widget.RelativeLayout;
import android.widget.SearchView;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.appcompat.app.ActionBar;
import androidx.appcompat.app.AppCompatActivity;
import androidx.appcompat.view.menu.MenuBuilder;

import com.pichillilorenzo.flutter_inappwebview_android.R;
import com.pichillilorenzo.flutter_inappwebview_android.Util;
import com.pichillilorenzo.flutter_inappwebview_android.find_interaction.FindInteractionController;
import com.pichillilorenzo.flutter_inappwebview_android.pull_to_refresh.PullToRefreshChannelDelegate;
import com.pichillilorenzo.flutter_inappwebview_android.pull_to_refresh.PullToRefreshLayout;
import com.pichillilorenzo.flutter_inappwebview_android.pull_to_refresh.PullToRefreshSettings;
import com.pichillilorenzo.flutter_inappwebview_android.types.AndroidResource;
import com.pichillilorenzo.flutter_inappwebview_android.types.Disposable;
import com.pichillilorenzo.flutter_inappwebview_android.types.InAppBrowserMenuItem;
import com.pichillilorenzo.flutter_inappwebview_android.types.URLRequest;
import com.pichillilorenzo.flutter_inappwebview_android.types.UserScript;
import com.pichillilorenzo.flutter_inappwebview_android.webview.WebViewChannelDelegate;
import com.pichillilorenzo.flutter_inappwebview_android.webview.in_app_webview.InAppWebView;
import com.pichillilorenzo.flutter_inappwebview_android.webview.in_app_webview.InAppWebViewSettings;

import java.io.IOException;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import io.flutter.plugin.common.MethodChannel;

public class InAppBrowserActivity extends AppCompatActivity implements InAppBrowserDelegate, Disposable {
  protected static final String LOG_TAG = "InAppBrowserActivity";
  public static final String METHOD_CHANNEL_NAME_PREFIX = "com.pichillilorenzo/flutter_inappbrowser_";
  
  @Nullable
  public Integer windowId;
  public String id;
  @Nullable
  public InAppWebView webView;
  @Nullable
  public PullToRefreshLayout pullToRefreshLayout;
  @Nullable
  public ActionBar actionBar;
  @Nullable
  public Menu menu;
  @Nullable
  public SearchView searchView;
  public InAppBrowserSettings customSettings = new InAppBrowserSettings();
  @Nullable
  public ProgressBar progressBar;
  public boolean isHidden = false;
  @Nullable
  public String fromActivity;
  private List<ActivityResultListener> activityResultListeners = new ArrayList<>();
  @Nullable
  public InAppBrowserManager manager;
  @Nullable
  public InAppBrowserChannelDelegate channelDelegate;
  public List<InAppBrowserMenuItem> menuItems = new ArrayList<>();
  
  @Override
  protected void onCreate(Bundle savedInstanceState) {
    super.onCreate(savedInstanceState);

    Bundle b = getIntent().getExtras();
    if (b == null) return;
    
    id = b.getString("id");

    String managerId = b.getString("managerId");
    manager = InAppBrowserManager.shared.get(managerId);
    if (manager == null || manager.plugin == null|| manager.plugin.messenger == null) return;

    Map<String, Object> settingsMap = (Map<String, Object>) b.getSerializable("settings");
    customSettings.parse(settingsMap);

    windowId = b.getInt("windowId");

    setContentView(R.layout.activity_web_view);

    Map<String, Object> pullToRefreshInitialSettings = (Map<String, Object>) b.getSerializable("pullToRefreshInitialSettings");
    MethodChannel pullToRefreshLayoutChannel = new MethodChannel(manager.plugin.messenger, PullToRefreshLayout.METHOD_CHANNEL_NAME_PREFIX + id);
    PullToRefreshSettings pullToRefreshSettings = new PullToRefreshSettings();
    pullToRefreshSettings.parse(pullToRefreshInitialSettings);
    pullToRefreshLayout = findViewById(R.id.pullToRefresh);
    pullToRefreshLayout.channelDelegate = new PullToRefreshChannelDelegate(pullToRefreshLayout, pullToRefreshLayoutChannel);
    pullToRefreshLayout.settings = pullToRefreshSettings;
    pullToRefreshLayout.prepare();
    
    webView = findViewById(R.id.webView);
    webView.id = id;
    webView.windowId = windowId;
    webView.inAppBrowserDelegate = this;
    webView.plugin = manager.plugin;

    FindInteractionController findInteractionController = new FindInteractionController(webView, manager.plugin, id, null);
    webView.findInteractionController = findInteractionController;
    findInteractionController.prepare();

    final MethodChannel channel = new MethodChannel(manager.plugin.messenger, METHOD_CHANNEL_NAME_PREFIX + id);
    channelDelegate = new InAppBrowserChannelDelegate(channel);
    webView.channelDelegate = new WebViewChannelDelegate(webView, channel);

    fromActivity = b.getString("fromActivity");

    Map<String, Object> contextMenu = (Map<String, Object>) b.getSerializable("contextMenu");
    List<Map<String, Object>> initialUserScripts = (List<Map<String, Object>>) b.getSerializable("initialUserScripts");
    List<Map<String, Object>> menuItemList = (List<Map<String, Object>>) b.getSerializable("menuItems");
    for (Map<String, Object> menuItem : menuItemList) {
      menuItems.add(InAppBrowserMenuItem.fromMap(menuItem));
    }

    InAppWebViewSettings webViewSettings = new InAppWebViewSettings();
    webViewSettings.parse(settingsMap);
    webView.customSettings = webViewSettings;
    webView.contextMenu = contextMenu;

    List<UserScript> userScripts = new ArrayList<>();
    if (initialUserScripts != null) {
      for (Map<String, Object> initialUserScript : initialUserScripts) {
        userScripts.add(UserScript.fromMap(initialUserScript));
      }
    }
    webView.userContentController.addUserOnlyScripts(userScripts);

    actionBar = getSupportActionBar();

    prepareView();

    if (windowId != -1) {
      if (webView.plugin != null && webView.plugin.inAppWebViewManager != null) {
        Message resultMsg = webView.plugin.inAppWebViewManager.windowWebViewMessages.get(windowId);
        if (resultMsg != null) {
          ((WebView.WebViewTransport) resultMsg.obj).setWebView(webView);
          resultMsg.sendToTarget();
        }
      }
    } else {
      String initialFile = b.getString("initialFile");
      Map<String, Object> initialUrlRequest = (Map<String, Object>) b.getSerializable("initialUrlRequest");
      String initialData = b.getString("initialData");
      if (initialFile != null) {
        try {
          webView.loadFile(initialFile);
        } catch (IOException e) {
          Log.e(LOG_TAG, initialFile + " asset file cannot be found!", e);
          return;
        }
      }
      else if (initialData != null) {
        String mimeType = b.getString("initialMimeType");
        String encoding = b.getString("initialEncoding");
        String baseUrl = b.getString("initialBaseUrl");
        String historyUrl = b.getString("initialHistoryUrl");
        webView.loadDataWithBaseURL(baseUrl, initialData, mimeType, encoding, historyUrl);
      }
      else if (initialUrlRequest != null) {
        URLRequest urlRequest = URLRequest.fromMap(initialUrlRequest);
        if (urlRequest != null) {
          webView.loadUrl(urlRequest);
        }
      }
    }

    if (channelDelegate != null) {
      channelDelegate.onBrowserCreated();
    }
  }

  private void prepareView() {

    if (webView != null) {
      webView.prepare();
    }

    if (customSettings.hidden)
      hide();
    else
      show();

    progressBar = findViewById(R.id.progressBar);

    if (progressBar != null) {
      if (customSettings.hideProgressBar)
        progressBar.setMax(0);
      else
        progressBar.setMax(100);
    }

    if (actionBar != null) {
      actionBar.setDisplayShowTitleEnabled(!customSettings.hideTitleBar);

      if (customSettings.hideToolbarTop)
        actionBar.hide();

      if (customSettings.toolbarTopBackgroundColor != null && !customSettings.toolbarTopBackgroundColor.isEmpty())
        actionBar.setBackgroundDrawable(new ColorDrawable(Color.parseColor(customSettings.toolbarTopBackgroundColor)));

      if (customSettings.toolbarTopFixedTitle != null && !customSettings.toolbarTopFixedTitle.isEmpty())
        actionBar.setTitle(customSettings.toolbarTopFixedTitle);
    }
  }

  @SuppressLint("RestrictedApi")
  @Override
  public boolean onCreateOptionsMenu(Menu m) {
    menu = m;

    if (actionBar != null && (customSettings.toolbarTopFixedTitle == null || customSettings.toolbarTopFixedTitle.isEmpty()))
      actionBar.setTitle(webView != null ? webView.getTitle() : "");

    if (menu == null)
      return super.onCreateOptionsMenu(m);

    if (menu instanceof MenuBuilder) {
      ((MenuBuilder) menu).setOptionalIconsVisible(true);
    }

    MenuInflater inflater = getMenuInflater();
    // Inflate menu to add items to action bar if it is present.
    inflater.inflate(R.menu.menu_main, menu);

    MenuItem menuSearchItem = menu.findItem(R.id.menu_search);
    if (menuSearchItem != null) {
      if (customSettings.hideUrlBar)
        menuSearchItem.setVisible(false);

      searchView = (SearchView) menuSearchItem.getActionView();
      if (searchView != null) {
        searchView.setFocusable(true);

        searchView.setQuery(webView != null ? webView.getUrl() : "", false);

        searchView.setOnQueryTextListener(new SearchView.OnQueryTextListener() {
          @Override
          public boolean onQueryTextSubmit(String query) {
            if (!query.isEmpty()) {
              if (webView != null)
                webView.loadUrl(query);
              if (searchView != null) {
                searchView.setQuery("", false);
                searchView.setIconified(true);
              }
              return true;
            }
            return false;
          }

          @Override
          public boolean onQueryTextChange(String newText) {
            return false;
          }

        });

        searchView.setOnCloseListener(new SearchView.OnCloseListener() {
          @Override
          public boolean onClose() {
            if (searchView != null && searchView.getQuery().toString().isEmpty())
              searchView.setQuery(webView != null ? webView.getUrl() : "", false);
            return false;
          }
        });

        searchView.setOnQueryTextFocusChangeListener(new View.OnFocusChangeListener() {
          @Override
          public void onFocusChange(View view, boolean b) {
            if (!b && searchView != null) {
              searchView.setQuery("", false);
              searchView.setIconified(true);
            }
          }
        });
      }
    }

    if (customSettings.hideDefaultMenuItems) {
      MenuItem actionClose = menu.findItem(R.id.action_close);
      if (actionClose != null) {
        actionClose.setVisible(false);
      }
      MenuItem actionGoBack = menu.findItem(R.id.action_go_back);
      if (actionGoBack != null) {
        actionGoBack.setVisible(false);
      }
      MenuItem actionReload = menu.findItem(R.id.action_reload);
      if (actionReload != null) {
        actionReload.setVisible(false);
      }
      MenuItem actionGoForward = menu.findItem(R.id.action_go_forward);
      if (actionGoForward != null) {
        actionGoForward.setVisible(false);
      }
      MenuItem actionShare = menu.findItem(R.id.action_share);
      if (actionShare != null) {
        actionShare.setVisible(false);
      }
    }

    for (final InAppBrowserMenuItem menuItem : menuItems) {
      int order = menuItem.getOrder() != null ? menuItem.getOrder() : Menu.NONE;
      MenuItem item = menu.add(Menu.NONE, menuItem.getId(), order, menuItem.getTitle());
      if (menuItem.isShowAsAction()) {
        item.setShowAsAction(MenuItem.SHOW_AS_ACTION_ALWAYS);
      }
      Object icon = menuItem.getIcon();
      if (icon != null) {
        if (icon instanceof AndroidResource) {
          item.setIcon(((AndroidResource) icon).getIdentifier(this));
        } else {
          item.setIcon(Util.drawableFromBytes(this, (byte[]) icon));
        }
        String iconColor = menuItem.getIconColor();
        if (iconColor != null && !iconColor.isEmpty() && Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
          item.getIcon().setTint(Color.parseColor(iconColor));
        }
      }
      item.setOnMenuItemClickListener(new MenuItem.OnMenuItemClickListener() {
        @Override
        public boolean onMenuItemClick(@NonNull MenuItem item) {
          if (channelDelegate != null) {
            channelDelegate.onMenuItemClicked(menuItem);
          }
          return true;
        }
      });
    }

    return true;
  }

  public boolean onKeyDown(int keyCode, KeyEvent event) {
    if (keyCode == KeyEvent.KEYCODE_BACK) {
      if (customSettings.shouldCloseOnBackButtonPressed) {
        close(null);
        return true;
      }
      if (customSettings.allowGoBackWithBackButton) {
        if (canGoBack())
          goBack();
        else if (customSettings.closeOnCannotGoBack)
          close(null);
        return true;
      }
      if (!customSettings.shouldCloseOnBackButtonPressed) {
        return true;
      }
    }
    return super.onKeyDown(keyCode, event);
  }

  public void close(final MethodChannel.Result result) {
    if (channelDelegate != null) {
      channelDelegate.onExit();
    }

    dispose();

    if (result != null) {
      result.success(true);
    }
  }

  public void reload() {
    if (webView != null)
      webView.reload();
  }

  public void goBack() {
    if (webView != null && canGoBack())
      webView.goBack();
  }

  public boolean canGoBack() {
    if (webView != null)
      return webView.canGoBack();
    return false;
  }

  public void goForward() {
    if (webView != null && canGoForward())
      webView.goForward();
  }

  public boolean canGoForward() {
    if (webView != null)
      return webView.canGoForward();
    return false;
  }

  public void hide() {
    if (fromActivity != null) {
      try {
        isHidden = true;
        Intent openActivity = new Intent(this, Class.forName(fromActivity));
        openActivity.setFlags(Intent.FLAG_ACTIVITY_REORDER_TO_FRONT);
        startActivityIfNeeded(openActivity, 0);
      } catch (ClassNotFoundException e) {
        Log.d(LOG_TAG, "", e);
      }
    }
  }

  public void show() {
    isHidden = false;
    Intent openActivity = new Intent(this, InAppBrowserActivity.class);
    openActivity.setFlags(Intent.FLAG_ACTIVITY_REORDER_TO_FRONT);
    startActivityIfNeeded(openActivity, 0);
  }

  public void goBackButtonClicked(MenuItem item) {
    goBack();
  }

  public void goForwardButtonClicked(MenuItem item) {
    goForward();
  }

  public void shareButtonClicked(MenuItem item) {
    Intent share = new Intent(Intent.ACTION_SEND);
    share.setType("text/plain");
    share.putExtra(Intent.EXTRA_TEXT, webView != null ? webView.getUrl() : "");
    startActivity(Intent.createChooser(share, "Share"));
  }

  public void reloadButtonClicked(MenuItem item) {
    reload();
  }

  public void closeButtonClicked(MenuItem item) {
    close(null);
  }

  public void setSettings(InAppBrowserSettings newSettings, HashMap<String, Object> newSettingsMap) {

    InAppWebViewSettings newInAppWebViewSettings = new InAppWebViewSettings();
    newInAppWebViewSettings.parse(newSettingsMap);
    if (webView != null) {
      webView.setSettings(newInAppWebViewSettings, newSettingsMap);
    }

    if (newSettingsMap.get("hidden") != null && customSettings.hidden != newSettings.hidden) {
      if (newSettings.hidden)
        hide();
      else
        show();
    }

    if (newSettingsMap.get("hideProgressBar") != null && customSettings.hideProgressBar != newSettings.hideProgressBar && progressBar != null) {
      if (newSettings.hideProgressBar)
        progressBar.setMax(0);
      else
        progressBar.setMax(100);
    }

    if (actionBar != null && newSettingsMap.get("hideTitleBar") != null && customSettings.hideTitleBar != newSettings.hideTitleBar)
      actionBar.setDisplayShowTitleEnabled(!newSettings.hideTitleBar);

    if (actionBar != null && newSettingsMap.get("hideToolbarTop") != null && customSettings.hideToolbarTop != newSettings.hideToolbarTop) {
      if (newSettings.hideToolbarTop)
        actionBar.hide();
      else
        actionBar.show();
    }

    if (actionBar != null && newSettingsMap.get("toolbarTopBackgroundColor") != null &&
            !Util.objEquals(customSettings.toolbarTopBackgroundColor, newSettings.toolbarTopBackgroundColor) &&
            newSettings.toolbarTopBackgroundColor != null && !newSettings.toolbarTopBackgroundColor.isEmpty())
      actionBar.setBackgroundDrawable(new ColorDrawable(Color.parseColor(newSettings.toolbarTopBackgroundColor)));

    if (actionBar != null && newSettingsMap.get("toolbarTopFixedTitle") != null &&
            !Util.objEquals(customSettings.toolbarTopFixedTitle, newSettings.toolbarTopFixedTitle) &&
            newSettings.toolbarTopFixedTitle != null && !newSettings.toolbarTopFixedTitle.isEmpty())
      actionBar.setTitle(newSettings.toolbarTopFixedTitle);

    if (menu != null && newSettingsMap.get("hideUrlBar") != null && customSettings.hideUrlBar != newSettings.hideUrlBar) {
      MenuItem menuSearchItem = menu.findItem(R.id.menu_search);
      if (menuSearchItem != null) {
        menuSearchItem.setVisible(!newSettings.hideUrlBar);
      }
    }

    if (menu != null && newSettingsMap.get("hideDefaultMenuItems") != null && customSettings.hideDefaultMenuItems != newSettings.hideDefaultMenuItems) {
      MenuItem actionClose = menu.findItem(R.id.action_close);
      if (actionClose != null) {
        actionClose.setVisible(!newSettings.hideDefaultMenuItems);
      }
      MenuItem actionGoBack = menu.findItem(R.id.action_go_back);
      if (actionGoBack != null) {
        actionGoBack.setVisible(!newSettings.hideDefaultMenuItems);
      }
      MenuItem actionReload = menu.findItem(R.id.action_reload);
      if (actionReload != null) {
        actionReload.setVisible(!newSettings.hideDefaultMenuItems);
      }
      MenuItem actionGoForward = menu.findItem(R.id.action_go_forward);
      if (actionGoForward != null) {
        actionGoForward.setVisible(!newSettings.hideDefaultMenuItems);
      }
      MenuItem actionShare = menu.findItem(R.id.action_share);
      if (actionShare != null) {
        actionShare.setVisible(!newSettings.hideDefaultMenuItems);
      }
    }

    customSettings = newSettings;
  }

  public Map<String, Object> getCustomSettings() {
    Map<String, Object> webViewSettingsMap = webView != null ? webView.getCustomSettings() : null;
    if (customSettings == null || webViewSettingsMap == null)
      return null;

    Map<String, Object> settingsMap = customSettings.getRealSettings(this);
    settingsMap.putAll(webViewSettingsMap);
    return settingsMap;
  }

  @Override
  public Activity getActivity() {
    return this;
  }

  @Override
  public void didChangeTitle(String title) {
    if (actionBar != null && (customSettings.toolbarTopFixedTitle == null || customSettings.toolbarTopFixedTitle.isEmpty())) {
      actionBar.setTitle(title);
    }
  }

  @Override
  public void didStartNavigation(String url) {
    if (progressBar != null) {
      progressBar.setProgress(0);
    }
    if (searchView != null) {
      searchView.setQuery(url, false);
    }
  }

  @Override
  public void didUpdateVisitedHistory(String url) {
    if (searchView != null) {
      searchView.setQuery(url, false);
    }
  }

  @Override
  public void didFinishNavigation(String url) {
    if (searchView != null) {
      searchView.setQuery(url, false);
    }
    if (progressBar != null) {
      progressBar.setProgress(0);
    }
  }

  @Override
  public void didFailNavigation(String url, int errorCode, String description) {
    if (progressBar != null) {
      progressBar.setProgress(0);
    }
  }

  @Override
  public void didChangeProgress(int progress) {
    if (progressBar != null) {
      progressBar.setVisibility(View.VISIBLE);
      if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.N) {
        progressBar.setProgress(progress, true);
      } else {
        progressBar.setProgress(progress);
      }
      if (progress == 100) {
        progressBar.setVisibility(View.GONE);
      }
    }
  }

  public List<ActivityResultListener> getActivityResultListeners() {
    return activityResultListeners;
  }

  @Override
  protected void onActivityResult (int requestCode,
                                   int resultCode,
                                   Intent data) {
    for (ActivityResultListener listener : activityResultListeners) {
      if (listener.onActivityResult(requestCode, resultCode, data)) {
        return;
      }
    }
    super.onActivityResult(requestCode, resultCode, data);
  }

  @Override
  public void dispose() {
    if (channelDelegate != null) {
      channelDelegate.dispose();
      channelDelegate = null;
    }
    activityResultListeners.clear();
    if (webView != null) {
      if (manager != null && manager.plugin != null &&
              manager.plugin.activityPluginBinding != null && webView.inAppWebViewChromeClient != null) {
        manager.plugin.activityPluginBinding.removeActivityResultListener(webView.inAppWebViewChromeClient);
      }
      RelativeLayout containerView = (RelativeLayout) findViewById(R.id.container);
      if (containerView != null) {
        containerView.removeAllViews();
      }
      webView.dispose();
      webView = null;
      finish();
    }
  }

  @Override
  public void onDestroy() {
    dispose();
    super.onDestroy();
  }
}
