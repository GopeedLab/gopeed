package com.pichillilorenzo.flutter_inappwebview_android.chrome_custom_tabs;

import android.content.Intent;
import android.graphics.Color;
import android.net.Uri;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.browser.customtabs.CustomTabColorSchemeParams;
import androidx.browser.customtabs.CustomTabsIntent;
import androidx.browser.trusted.TrustedWebActivityIntent;
import androidx.browser.trusted.TrustedWebActivityIntentBuilder;

import java.util.List;
import java.util.Map;

public class TrustedWebActivity extends ChromeCustomTabsActivity {

  protected static final String LOG_TAG = "TrustedWebActivity";

  public TrustedWebActivityIntentBuilder builder;

  @Override
  public void launchUrl(@NonNull String url,
                        @Nullable Map<String, String> headers,
                        @Nullable String referrer,
                        @Nullable List<String> otherLikelyURLs) {
    if (customTabsSession == null) {
      return;
    }
    Uri uri = Uri.parse(url);

    mayLaunchUrl(url, otherLikelyURLs);
    builder = new TrustedWebActivityIntentBuilder(uri);
    prepareCustomTabs();

    TrustedWebActivityIntent trustedWebActivityIntent = builder.build(customTabsSession);
    prepareCustomTabsIntent(trustedWebActivityIntent);

    CustomTabActivityHelper.openTrustedWebActivity(this, trustedWebActivityIntent, uri, headers,
            referrer != null ? Uri.parse(referrer) : null, CHROME_CUSTOM_TAB_REQUEST_CODE);
  }

  private void prepareCustomTabs() {
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

    if (customSettings.additionalTrustedOrigins != null && !customSettings.additionalTrustedOrigins.isEmpty()) {
      builder.setAdditionalTrustedOrigins(customSettings.additionalTrustedOrigins);
    }

    if (customSettings.displayMode != null) {
      builder.setDisplayMode(customSettings.displayMode);
    }

    builder.setScreenOrientation(customSettings.screenOrientation);
  }

  private void prepareCustomTabsIntent(TrustedWebActivityIntent trustedWebActivityIntent) {
    Intent intent = trustedWebActivityIntent.getIntent();
    if (customSettings.packageName != null)
      intent.setPackage(customSettings.packageName);
    else
      intent.setPackage(CustomTabsHelper.getPackageNameToUse(this));

    if (customSettings.keepAliveEnabled)
      CustomTabsHelper.addKeepAliveExtra(this, intent);

    if (customSettings.alwaysUseBrowserUI)
      CustomTabsIntent.setAlwaysUseBrowserUI(intent);
  }
}
