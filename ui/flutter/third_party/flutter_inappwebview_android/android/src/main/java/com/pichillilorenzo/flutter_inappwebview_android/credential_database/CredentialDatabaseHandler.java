package com.pichillilorenzo.flutter_inappwebview_android.credential_database;

import android.os.Build;
import android.webkit.WebViewDatabase;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.annotation.RequiresApi;

import com.pichillilorenzo.flutter_inappwebview_android.InAppWebViewFlutterPlugin;
import com.pichillilorenzo.flutter_inappwebview_android.types.ChannelDelegateImpl;
import com.pichillilorenzo.flutter_inappwebview_android.types.URLCredential;
import com.pichillilorenzo.flutter_inappwebview_android.types.URLProtectionSpace;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import io.flutter.plugin.common.MethodCall;
import io.flutter.plugin.common.MethodChannel;

@RequiresApi(api = Build.VERSION_CODES.O)
public class CredentialDatabaseHandler extends ChannelDelegateImpl {
  protected static final String LOG_TAG = "CredentialDatabaseHandler";
  public static final String METHOD_CHANNEL_NAME = "com.pichillilorenzo/flutter_inappwebview_credential_database";

  @Nullable
  public static CredentialDatabase credentialDatabase;
  @Nullable
  public InAppWebViewFlutterPlugin plugin;

  public CredentialDatabaseHandler(@NonNull final InAppWebViewFlutterPlugin plugin) {
    super(new MethodChannel(plugin.messenger, METHOD_CHANNEL_NAME));
    this.plugin = plugin;
  }

  public static void init(@NonNull InAppWebViewFlutterPlugin plugin) {
    if (credentialDatabase == null) {
      credentialDatabase = CredentialDatabase.getInstance(plugin.applicationContext);
    }
  }

  @Override
  public void onMethodCall(@NonNull MethodCall call, @NonNull MethodChannel.Result result) {
    if (plugin != null) {
      init(plugin);
    }

    switch (call.method) {
      case "getAllAuthCredentials":
        {
          List<Map<String, Object>> allCredentials = new ArrayList<>();
          if (credentialDatabase != null) {
            List<URLProtectionSpace> protectionSpaces = credentialDatabase.protectionSpaceDao.getAll();
            for (URLProtectionSpace protectionSpace : protectionSpaces) {
              List<Map<String, Object>> credentials = new ArrayList<>();
              for (URLCredential credential : credentialDatabase.credentialDao.getAllByProtectionSpaceId(protectionSpace.getId())) {
                credentials.add(credential.toMap());
              }
              Map<String, Object> obj = new HashMap<>();
              obj.put("protectionSpace", protectionSpace.toMap());
              obj.put("credentials", credentials);
              allCredentials.add(obj);
            }
          }
          result.success(allCredentials);
        }
        break;
      case "getHttpAuthCredentials":
        {
          List<Map<String, Object>> credentials = new ArrayList<>();
          if (credentialDatabase != null) {
            String host = (String) call.argument("host");
            String protocol = (String) call.argument("protocol");
            String realm = (String) call.argument("realm");
            Integer port = (Integer) call.argument("port");

            for (URLCredential credential : credentialDatabase.getHttpAuthCredentials(host, protocol, realm, port)) {
              credentials.add(credential.toMap());
            }
          }
          result.success(credentials);
        }
        break;
      case "setHttpAuthCredential":
        {
          if (credentialDatabase != null) {
            String host = (String) call.argument("host");
            String protocol = (String) call.argument("protocol");
            String realm = (String) call.argument("realm");
            Integer port = (Integer) call.argument("port");
            String username = (String) call.argument("username");
            String password = (String) call.argument("password");

            credentialDatabase.setHttpAuthCredential(host, protocol, realm, port, username, password);
            result.success(true);
          } else {
            result.success(false);
          }
        }
        break;
      case "removeHttpAuthCredential":
        {
          if (credentialDatabase != null) {
            String host = (String) call.argument("host");
            String protocol = (String) call.argument("protocol");
            String realm = (String) call.argument("realm");
            Integer port = (Integer) call.argument("port");
            String username = (String) call.argument("username");
            String password = (String) call.argument("password");

            credentialDatabase.removeHttpAuthCredential(host, protocol, realm, port, username, password);
            result.success(true);
          } else {
            result.success(false);
          }
        }
        break;
      case "removeHttpAuthCredentials":
        {
          if (credentialDatabase != null) {
            String host = (String) call.argument("host");
            String protocol = (String) call.argument("protocol");
            String realm = (String) call.argument("realm");
            Integer port = (Integer) call.argument("port");

            credentialDatabase.removeHttpAuthCredentials(host, protocol, realm, port);
            result.success(true);
          } else {
            result.success(false);
          }
        }
        break;
      case "clearAllAuthCredentials":
        if (credentialDatabase != null) {
          credentialDatabase.clearAllAuthCredentials();
          if (plugin != null && plugin.applicationContext != null) {
            WebViewDatabase.getInstance(plugin.applicationContext).clearHttpAuthUsernamePassword();
          }
          result.success(true);
        } else {
          result.success(false);
        }
        break;
      default:
        result.notImplemented();
    }
  }

  @Override
  public void dispose() {
    super.dispose();
    plugin = null;
    credentialDatabase = null;
  }
}
