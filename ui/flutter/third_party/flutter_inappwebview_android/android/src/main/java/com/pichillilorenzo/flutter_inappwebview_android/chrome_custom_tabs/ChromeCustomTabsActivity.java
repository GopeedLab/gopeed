package com.pichillilorenzo.flutter_inappwebview_android.chrome_custom_tabs;

import android.app.Activity;
import android.app.PendingIntent;
import android.content.Intent;
import android.graphics.Bitmap;
import android.graphics.BitmapFactory;
import android.graphics.Color;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.util.Log;
import android.widget.RemoteViews;

import androidx.annotation.CallSuper;
import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.browser.customtabs.CustomTabColorSchemeParams;
import androidx.browser.customtabs.CustomTabsCallback;
import androidx.browser.customtabs.CustomTabsIntent;
import androidx.browser.customtabs.CustomTabsService;
import androidx.browser.customtabs.CustomTabsSession;
import androidx.browser.customtabs.EngagementSignalsCallback;

import com.pichillilorenzo.flutter_inappwebview_android.R;
import com.pichillilorenzo.flutter_inappwebview_android.types.AndroidResource;
import com.pichillilorenzo.flutter_inappwebview_android.types.CustomTabsActionButton;
import com.pichillilorenzo.flutter_inappwebview_android.types.CustomTabsMenuItem;
import com.pichillilorenzo.flutter_inappwebview_android.types.CustomTabsSecondaryToolbar;
import com.pichillilorenzo.flutter_inappwebview_android.types.Disposable;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import io.flutter.plugin.common.MethodChannel;

public class ChromeCustomTabsActivity extends Activity implements Disposable {
  protected static final String LOG_TAG = "CustomTabsActivity";
  public static final String METHOD_CHANNEL_NAME_PREFIX = "com.pichillilorenzo/flutter_chromesafaribrowser_";

  public String id;
  @Nullable
  public CustomTabsIntent.Builder builder;
  public ChromeCustomTabsSettings customSettings = new ChromeCustomTabsSettings();
  public CustomTabActivityHelper customTabActivityHelper = new CustomTabActivityHelper();
  @Nullable
  public CustomTabsSession customTabsSession;
  public final static int CHROME_CUSTOM_TAB_REQUEST_CODE = 100;
  public final static int NO_HISTORY_CHROME_CUSTOM_TAB_REQUEST_CODE = 101;
  protected boolean onOpened = false;
  protected boolean onCompletedInitialLoad = false;
  protected boolean isBindSuccess = false;
  @Nullable
  public ChromeSafariBrowserManager manager;
  @Nullable
  public String initialUrl;
  @Nullable
  public List<String> initialOtherLikelyURLs;
  @Nullable
  public Map<String, String> initialHeaders;
  @Nullable
  public String initialReferrer;
  public List<CustomTabsMenuItem> menuItems = new ArrayList<>();
  @Nullable
  public CustomTabsActionButton actionButton;
  @Nullable
  public CustomTabsSecondaryToolbar secondaryToolbar;
  @Nullable
  public ChromeCustomTabsChannelDelegate channelDelegate;

  @CallSuper
  @Override
  protected void onCreate(Bundle savedInstanceState) {
    super.onCreate(savedInstanceState);

    setContentView(R.layout.chrome_custom_tabs_layout);

    Bundle b = getIntent().getExtras();
    if (b == null) return;

    id = b.getString("id");

    String managerId = b.getString("managerId");
    manager = ChromeSafariBrowserManager.shared.get(managerId);
    if (manager == null || manager.plugin == null || manager.plugin.messenger == null) return;

    manager.browsers.put(id, this);

    MethodChannel channel = new MethodChannel(manager.plugin.messenger, METHOD_CHANNEL_NAME_PREFIX + id);
    channelDelegate = new ChromeCustomTabsChannelDelegate(this, channel);

    initialUrl = b.getString("url");
    initialHeaders = (Map<String, String>) b.getSerializable("headers");
    initialReferrer = b.getString("referrer");
    initialOtherLikelyURLs = b.getStringArrayList("otherLikelyURLs");

    customSettings = new ChromeCustomTabsSettings();
    customSettings.parse((HashMap<String, Object>) b.getSerializable("settings"));
    actionButton = CustomTabsActionButton.fromMap((Map<String, Object>) b.getSerializable("actionButton"));
    secondaryToolbar = CustomTabsSecondaryToolbar.fromMap((Map<String, Object>) b.getSerializable("secondaryToolbar"));
    List<Map<String, Object>> menuItemList = (List<Map<String, Object>>) b.getSerializable("menuItemList");
    for (Map<String, Object> menuItem : menuItemList) {
      menuItems.add(CustomTabsMenuItem.fromMap(menuItem));
    }

    if (customSettings.noHistory && manager.plugin.noHistoryCustomTabsActivityCallbacks != null) {
      manager.plugin.noHistoryCustomTabsActivityCallbacks.noHistoryBrowserIDs.put(id, id);
    }

    final ChromeCustomTabsActivity chromeCustomTabsActivity = this;

    customTabActivityHelper.setConnectionCallback(new CustomTabActivityHelper.ConnectionCallback() {
      @Override
      public void onCustomTabsConnected() {
        customTabsConnected();
        if (channelDelegate != null) {
          channelDelegate.onServiceConnected();
        }
      }

      @Override
      public void onCustomTabsDisconnected() {
        chromeCustomTabsActivity.close();
        dispose();
      }
    });

    customTabActivityHelper.setCustomTabsCallback(new CustomTabsCallback() {
      @Override
      public void onNavigationEvent(int navigationEvent, Bundle extras) {
        if (navigationEvent == TAB_SHOWN && !onOpened) {
          onOpened = true;
          if (channelDelegate != null) {
            channelDelegate.onOpened();
          }
        }

        if (navigationEvent == NAVIGATION_FINISHED && !onCompletedInitialLoad) {
          onCompletedInitialLoad = true;
          if (channelDelegate != null) {
            channelDelegate.onCompletedInitialLoad();
          }
        }

        if (channelDelegate != null) {
          channelDelegate.onNavigationEvent(navigationEvent);
        }
      }

      @Override
      public void extraCallback(@NonNull String callbackName, Bundle args) {
      }

      @Override
      public void onMessageChannelReady(Bundle extras) {
        if (channelDelegate != null) {
          channelDelegate.onMessageChannelReady();
        }
      }

      @Override
      public void onPostMessage(@NonNull String message, Bundle extras) {
        if (channelDelegate != null) {
          channelDelegate.onPostMessage(message);
        }
      }

      @Override
      public void onRelationshipValidationResult(@CustomTabsService.Relation int relation,
                                                 @NonNull Uri requestedOrigin,
                                                 boolean result, Bundle extras) {
        if (channelDelegate != null) {
          channelDelegate.onRelationshipValidationResult(relation, requestedOrigin, result);
        }
      }
    });
  }

  public void launchUrl(@NonNull String url,
                        @Nullable Map<String, String> headers,
                        @Nullable String referrer,
                        @Nullable List<String> otherLikelyURLs) {
    launchUrlWithSession(customTabsSession, url, headers, referrer, otherLikelyURLs);
  }

  public void launchUrlWithSession(@Nullable CustomTabsSession session,
                                   @NonNull String url,
                                   @Nullable Map<String, String> headers,
                                   @Nullable String referrer,
                                   @Nullable List<String> otherLikelyURLs) {
    mayLaunchUrl(url, otherLikelyURLs);
    builder = new CustomTabsIntent.Builder(session);
    prepareCustomTabs();

    CustomTabsIntent customTabsIntent = builder.build();
    prepareCustomTabsIntent(customTabsIntent);

    CustomTabActivityHelper.openCustomTab(this, customTabsIntent, Uri.parse(url), headers,
            referrer != null ? Uri.parse(referrer) : null, CHROME_CUSTOM_TAB_REQUEST_CODE);
  }

  public boolean mayLaunchUrl(@Nullable String url, @Nullable List<String> otherLikelyURLs) {
    Uri uri = url != null ? Uri.parse(url) : null;

    List<Bundle> bundleOtherLikelyURLs = new ArrayList<>();
    if (otherLikelyURLs != null) {
      Bundle bundleOtherLikelyURL = new Bundle();
      for (String otherLikelyURL : otherLikelyURLs) {
        bundleOtherLikelyURL.putString(CustomTabsService.KEY_URL, otherLikelyURL);
      }
    }
    return customTabActivityHelper.mayLaunchUrl(uri, null, bundleOtherLikelyURLs);
  }

  @CallSuper
  public void customTabsConnected() {
    customTabsSession = customTabActivityHelper.getSession();

    if (customTabsSession != null) {
      try {
        Bundle bundle = new Bundle();
        if (customTabsSession.isEngagementSignalsApiAvailable(bundle)) {
          customTabsSession.setEngagementSignalsCallback(new EngagementSignalsCallback() {
            @Override
            public void onVerticalScrollEvent(boolean isDirectionUp, @NonNull Bundle extras) {
              if (channelDelegate != null) {
                channelDelegate.onVerticalScrollEvent(isDirectionUp);
              }
            }

            @Override
            public void onGreatestScrollPercentageIncreased(int scrollPercentage, @NonNull Bundle extras) {
              if (channelDelegate != null) {
                channelDelegate.onGreatestScrollPercentageIncreased(scrollPercentage);
              }
            }

            @Override
            public void onSessionEnded(boolean didUserInteract, @NonNull Bundle extras) {
              if (channelDelegate != null) {
                channelDelegate.onSessionEnded(didUserInteract);
              }
            }
          }, bundle);
        }
      } catch (Throwable e) {
        Log.d(LOG_TAG, "Custom Tabs Engagement Signals API not supported", e);
      }
    }

    // avoid webpage reopen if isBindSuccess is false: onServiceConnected->launchUrl
    if (isBindSuccess && initialUrl != null) {
      launchUrl(initialUrl, initialHeaders, initialReferrer, initialOtherLikelyURLs);
    }
  }

  private void prepareCustomTabs() {
    if (builder == null) {
      return;
    }

    if (customSettings.addDefaultShareMenuItem != null) {
      builder.setShareState(customSettings.addDefaultShareMenuItem ?
              CustomTabsIntent.SHARE_STATE_ON : CustomTabsIntent.SHARE_STATE_OFF);
    } else {
      builder.setShareState(customSettings.shareState);
    }

    CustomTabColorSchemeParams.Builder defaultColorSchemeBuilder = new CustomTabColorSchemeParams.Builder();
    if (customSettings.toolbarBackgroundColor != null && !customSettings.toolbarBackgroundColor.isEmpty()) {
      defaultColorSchemeBuilder.setToolbarColor(Color.parseColor(customSettings.toolbarBackgroundColor));
    }
    if (customSettings.navigationBarColor != null && !customSettings.navigationBarColor.isEmpty()) {
      defaultColorSchemeBuilder.setNavigationBarColor(Color.parseColor(customSettings.navigationBarColor));
    }
    if (customSettings.navigationBarDividerColor != null && !customSettings.navigationBarDividerColor.isEmpty()) {
      defaultColorSchemeBuilder.setNavigationBarDividerColor(Color.parseColor(customSettings.navigationBarDividerColor));
    }
    if (customSettings.secondaryToolbarColor != null && !customSettings.secondaryToolbarColor.isEmpty()) {
      defaultColorSchemeBuilder.setSecondaryToolbarColor(Color.parseColor(customSettings.secondaryToolbarColor));
    }
    builder.setDefaultColorSchemeParams(defaultColorSchemeBuilder.build());

    builder.setShowTitle(customSettings.showTitle);
    builder.setUrlBarHidingEnabled(customSettings.enableUrlBarHiding);
    builder.setInstantAppsEnabled(customSettings.instantAppsEnabled);
    if (customSettings.startAnimations.size() == 2) {
      builder.setStartAnimations(this,
              customSettings.startAnimations.get(0).getIdentifier(this),
              customSettings.startAnimations.get(1).getIdentifier(this));
    }
    if (customSettings.exitAnimations.size() == 2) {
      builder.setExitAnimations(this,
              customSettings.exitAnimations.get(0).getIdentifier(this),
              customSettings.exitAnimations.get(1).getIdentifier(this));
    }

    for (CustomTabsMenuItem menuItem : menuItems) {
      builder.addMenuItem(menuItem.getLabel(),
              createPendingIntent(menuItem.getId()));
    }

    if (actionButton != null) {
      byte[] data = actionButton.getIcon();
      BitmapFactory.Options bitmapOptions = new BitmapFactory.Options();
      bitmapOptions.inMutable = true;
      Bitmap bmp = BitmapFactory.decodeByteArray(
              data, 0, data.length, bitmapOptions
      );
      builder.setActionButton(bmp, actionButton.getDescription(),
              createPendingIntent(actionButton.getId()),
              actionButton.isShouldTint());
    }

    if (secondaryToolbar != null) {
      AndroidResource layout = secondaryToolbar.getLayout();
      RemoteViews remoteViews = new RemoteViews(layout.getDefPackage(), layout.getIdentifier(this));
      int[] clickableIDs = new int[secondaryToolbar.getClickableIDs().size()];
      for (int i = 0, length = secondaryToolbar.getClickableIDs().size(); i < length; i++) {
        AndroidResource clickableID = secondaryToolbar.getClickableIDs().get(i);
        clickableIDs[i] = clickableID.getIdentifier(this);
      }
      builder.setSecondaryToolbarViews(remoteViews, clickableIDs, getSecondaryToolbarOnClickPendingIntent());
    }
  }

  public PendingIntent getSecondaryToolbarOnClickPendingIntent() {
    Intent broadcastIntent = new Intent(this, ActionBroadcastReceiver.class);

    Bundle extras = new Bundle();
    extras.putString(ActionBroadcastReceiver.KEY_ACTION_VIEW_ID, id);
    extras.putString(ActionBroadcastReceiver.KEY_ACTION_MANAGER_ID, manager != null ? manager.id : null);
    broadcastIntent.putExtras(extras);

    if (android.os.Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
      return PendingIntent.getBroadcast(
              this, 0, broadcastIntent, PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_MUTABLE);
    } else {
      return PendingIntent.getBroadcast(
              this, 0, broadcastIntent, PendingIntent.FLAG_UPDATE_CURRENT);
    }
  }

  private void prepareCustomTabsIntent(CustomTabsIntent customTabsIntent) {
    if (customSettings.packageName != null)
      customTabsIntent.intent.setPackage(customSettings.packageName);
    else
      customTabsIntent.intent.setPackage(CustomTabsHelper.getPackageNameToUse(this));

    if (customSettings.keepAliveEnabled)
      CustomTabsHelper.addKeepAliveExtra(this, customTabsIntent.intent);

    if (customSettings.alwaysUseBrowserUI)
      CustomTabsIntent.setAlwaysUseBrowserUI(customTabsIntent.intent);
  }

  public void updateActionButton(@NonNull byte[] icon, @NonNull String description) {
    if (customTabsSession == null || actionButton == null) {
      return;
    }
    BitmapFactory.Options bitmapOptions = new BitmapFactory.Options();
    bitmapOptions.inMutable = true;
    Bitmap bmp = BitmapFactory.decodeByteArray(
            icon, 0, icon.length, bitmapOptions
    );
    customTabsSession.setActionButton(bmp, description);
    actionButton.setIcon(icon);
    actionButton.setDescription(description);
  }

  public void updateSecondaryToolbar(CustomTabsSecondaryToolbar secondaryToolbar) {
    if (customTabsSession == null) {
      return;
    }
    AndroidResource layout = secondaryToolbar.getLayout();
    RemoteViews remoteViews = new RemoteViews(layout.getDefPackage(), layout.getIdentifier(this));
    int[] clickableIDs = new int[secondaryToolbar.getClickableIDs().size()];
    for (int i = 0, length = secondaryToolbar.getClickableIDs().size(); i < length; i++) {
      AndroidResource clickableID = secondaryToolbar.getClickableIDs().get(i);
      clickableIDs[i] = clickableID.getIdentifier(this);
    }
    customTabsSession.setSecondaryToolbarViews(remoteViews, clickableIDs, getSecondaryToolbarOnClickPendingIntent());
    this.secondaryToolbar = secondaryToolbar;
  }

  @Override
  protected void onStart() {
    super.onStart();
    isBindSuccess = customTabActivityHelper.bindCustomTabsService(this);

    if (!isBindSuccess && initialUrl != null) {
      // chrome process not running, start tab directly
      launchUrlWithSession(null, initialUrl, initialHeaders, initialReferrer, initialOtherLikelyURLs);
    }
  }

  @Override
  protected void onStop() {
    super.onStop();
    customTabActivityHelper.unbindCustomTabsService(this);
    isBindSuccess = false;
  }

  @Override
  public void onDestroy() {
    super.onDestroy();
  }

  @Override
  protected void onActivityResult(int requestCode, int resultCode, Intent data) {
    if (requestCode == CHROME_CUSTOM_TAB_REQUEST_CODE) {
      close();
      dispose();
    }
  }

  private PendingIntent createPendingIntent(int actionSourceId) {
    Intent actionIntent = new Intent(this, ActionBroadcastReceiver.class);

    Bundle extras = new Bundle();
    extras.putInt(ActionBroadcastReceiver.KEY_ACTION_ID, actionSourceId);
    extras.putString(ActionBroadcastReceiver.KEY_ACTION_VIEW_ID, id);
    extras.putString(ActionBroadcastReceiver.KEY_ACTION_MANAGER_ID, manager != null ? manager.id : null);
    actionIntent.putExtras(extras);

    if (android.os.Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
      return PendingIntent.getBroadcast(
              this, actionSourceId, actionIntent, PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_MUTABLE);
    } else {
      return PendingIntent.getBroadcast(
              this, actionSourceId, actionIntent, PendingIntent.FLAG_UPDATE_CURRENT);
    }
  }

  @Override
  public void dispose() {
    onStop();
    onDestroy();
    if (channelDelegate != null) {
      channelDelegate.dispose();
      channelDelegate = null;
    }
    if (manager != null) {
      if (manager.browsers.containsKey(id)) {
        manager.browsers.put(id, null);
      }
    }
    manager = null;
  }

  public void close() {
    onStop();
    onDestroy();
    customTabsSession = null;
    finish();
    if (channelDelegate != null) {
      channelDelegate.onClosed();
    }
  }
}
