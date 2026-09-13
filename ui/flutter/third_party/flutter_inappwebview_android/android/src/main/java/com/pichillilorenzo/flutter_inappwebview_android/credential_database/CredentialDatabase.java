package com.pichillilorenzo.flutter_inappwebview_android.credential_database;

import android.content.Context;

import com.pichillilorenzo.flutter_inappwebview_android.types.URLCredential;
import com.pichillilorenzo.flutter_inappwebview_android.types.URLProtectionSpace;

import java.util.ArrayList;
import java.util.List;

public class CredentialDatabase {

  private static CredentialDatabase instance;
  static final String LOG_TAG = "CredentialDatabase";

  // If you change the database schema, you must increment the database version.
  public static final int DATABASE_VERSION = 2;
  public static final String DATABASE_NAME = "CredentialDatabase.db";

  public URLProtectionSpaceDao protectionSpaceDao;
  public URLCredentialDao credentialDao;
  public CredentialDatabaseHelper db;

  private CredentialDatabase() {}

  private CredentialDatabase(CredentialDatabaseHelper db, URLProtectionSpaceDao protectionSpaceDao, URLCredentialDao credentialDao) {
    this.db = db;
    this.protectionSpaceDao = protectionSpaceDao;
    this.credentialDao = credentialDao;
  }

  public static CredentialDatabase getInstance(Context context) {
    if (instance != null)
      return instance;
    CredentialDatabaseHelper db = new CredentialDatabaseHelper(context);
    instance = new CredentialDatabase(db, new URLProtectionSpaceDao(db), new URLCredentialDao(db));
    return instance;
  }

  public List<URLCredential> getHttpAuthCredentials(String host, String protocol, String realm, Integer port) {
    List<URLCredential> credentials = new ArrayList<>();
    URLProtectionSpace protectionSpace = protectionSpaceDao.find(host, protocol, realm, port);
    if (protectionSpace != null) {
      credentials = credentialDao.getAllByProtectionSpaceId(protectionSpace.getId());
    }
    return credentials;
  }

  public void clearAllAuthCredentials() {
    db.clearAllTables(db.getWritableDatabase());
  }

  public void removeHttpAuthCredentials(String host, String protocol, String realm, Integer port) {
    URLProtectionSpace URLProtectionSpace = protectionSpaceDao.find(host, protocol, realm, port);
    if (URLProtectionSpace != null) {
      protectionSpaceDao.delete(URLProtectionSpace);
    }
  }

  public void removeHttpAuthCredential(String host, String protocol, String realm, Integer port, String username, String password) {
    URLProtectionSpace protectionSpace = protectionSpaceDao.find(host, protocol, realm, port);
    if (protectionSpace != null) {
      URLCredential credential = credentialDao.find(username, password, protectionSpace.getId());
      credentialDao.delete(credential);
    }
  }

  public void setHttpAuthCredential(String host, String protocol, String realm, Integer port, String username, String password) {
    URLProtectionSpace protectionSpace = protectionSpaceDao.find(host, protocol, realm, port);
    Long protectionSpaceId;
    if (protectionSpace == null) {
      protectionSpaceId = protectionSpaceDao.insert(new URLProtectionSpace(null, host, protocol, realm, port));
    } else {
      protectionSpaceId = protectionSpace.getId();
    }

    URLCredential credential = credentialDao.find(username, password, protectionSpaceId);
    if (credential != null) {
      boolean needUpdate = false;
      if (!credential.getUsername().equals(username)) {
        credential.setUsername(username);
        needUpdate = true;
      }
      if (!credential.getPassword().equals(password)) {
        credential.setPassword(password);
        needUpdate = true;
      }
      if (needUpdate)
        credentialDao.update(credential);
    } else {
      credential = new URLCredential(null, username, password, protectionSpaceId);
      credential.setId(credentialDao.insert(credential));
    }
  }
}