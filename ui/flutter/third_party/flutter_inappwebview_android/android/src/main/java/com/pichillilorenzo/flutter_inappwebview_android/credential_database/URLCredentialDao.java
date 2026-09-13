package com.pichillilorenzo.flutter_inappwebview_android.credential_database;

import android.content.ContentValues;
import android.database.Cursor;

import com.pichillilorenzo.flutter_inappwebview_android.types.URLCredential;

import java.util.ArrayList;
import java.util.List;

public class URLCredentialDao {

  CredentialDatabaseHelper credentialDatabaseHelper;
  String[] projection = {
          URLCredentialContract.FeedEntry._ID,
          URLCredentialContract.FeedEntry.COLUMN_NAME_USERNAME,
          URLCredentialContract.FeedEntry.COLUMN_NAME_PASSWORD,
          URLCredentialContract.FeedEntry.COLUMN_NAME_PROTECTION_SPACE_ID
  };

  public URLCredentialDao(CredentialDatabaseHelper credentialDatabaseHelper) {
    this.credentialDatabaseHelper = credentialDatabaseHelper;
  }

  public List<URLCredential> getAllByProtectionSpaceId(Long protectionSpaceId) {
    String selection = URLCredentialContract.FeedEntry.COLUMN_NAME_PROTECTION_SPACE_ID + " = ?";
    String[] selectionArgs = {protectionSpaceId.toString()};

    Cursor cursor = credentialDatabaseHelper.getReadableDatabase().query(
            URLCredentialContract.FeedEntry.TABLE_NAME,
            projection,
            selection,
            selectionArgs,
            null,
            null,
            null
    );

    List<URLCredential> URLCredentials = new ArrayList<>();
    while (cursor.moveToNext()) {
      Long id = cursor.getLong(cursor.getColumnIndexOrThrow(URLCredentialContract.FeedEntry._ID));
      String username = cursor.getString(cursor.getColumnIndexOrThrow(URLCredentialContract.FeedEntry.COLUMN_NAME_USERNAME));
      String password = cursor.getString(cursor.getColumnIndexOrThrow(URLCredentialContract.FeedEntry.COLUMN_NAME_PASSWORD));
      URLCredentials.add(new URLCredential(id, username, password, protectionSpaceId));
    }
    cursor.close();

    return URLCredentials;
  }

  public URLCredential find(String username, String password, Long protectionSpaceId) {
    String selection = URLCredentialContract.FeedEntry.COLUMN_NAME_USERNAME + " = ? AND " +
            URLCredentialContract.FeedEntry.COLUMN_NAME_PASSWORD + " = ? AND " +
            URLCredentialContract.FeedEntry.COLUMN_NAME_PROTECTION_SPACE_ID + " = ?";
    String[] selectionArgs = {username, password, protectionSpaceId.toString()};

    Cursor cursor = credentialDatabaseHelper.getReadableDatabase().query(
            URLCredentialContract.FeedEntry.TABLE_NAME,
            projection,
            selection,
            selectionArgs,
            null,
            null,
            null
    );

    URLCredential URLCredential = null;
    if (cursor.moveToNext()) {
      Long rowId = cursor.getLong(cursor.getColumnIndexOrThrow(URLCredentialContract.FeedEntry._ID));
      String rowUsername = cursor.getString(cursor.getColumnIndexOrThrow(URLCredentialContract.FeedEntry.COLUMN_NAME_USERNAME));
      String rowPassword = cursor.getString(cursor.getColumnIndexOrThrow(URLCredentialContract.FeedEntry.COLUMN_NAME_PASSWORD));
      URLCredential = new URLCredential(rowId, rowUsername, rowPassword, protectionSpaceId);
    }
    cursor.close();

    return URLCredential;
  }

  public long insert(URLCredential urlCredential) {
    ContentValues credentialValues = new ContentValues();
    credentialValues.put(URLCredentialContract.FeedEntry.COLUMN_NAME_USERNAME, urlCredential.getUsername());
    credentialValues.put(URLCredentialContract.FeedEntry.COLUMN_NAME_PASSWORD, urlCredential.getPassword());
    credentialValues.put(URLCredentialContract.FeedEntry.COLUMN_NAME_PROTECTION_SPACE_ID, urlCredential.getProtectionSpaceId());

    return credentialDatabaseHelper.getWritableDatabase().insert(URLCredentialContract.FeedEntry.TABLE_NAME, null, credentialValues);
  }

  public long update(URLCredential urlCredential) {
    ContentValues credentialValues = new ContentValues();
    credentialValues.put(URLCredentialContract.FeedEntry.COLUMN_NAME_USERNAME, urlCredential.getUsername());
    credentialValues.put(URLCredentialContract.FeedEntry.COLUMN_NAME_PASSWORD, urlCredential.getPassword());

    String whereClause = URLCredentialContract.FeedEntry.COLUMN_NAME_PROTECTION_SPACE_ID + " = ?";
    String[] whereArgs = {urlCredential.getProtectionSpaceId().toString()};

    return credentialDatabaseHelper.getWritableDatabase().update(URLCredentialContract.FeedEntry.TABLE_NAME, credentialValues, whereClause, whereArgs);
  }

  public long delete(URLCredential urlCredential) {
    String whereClause = URLCredentialContract.FeedEntry._ID + " = ?";
    String[] whereArgs = {urlCredential.getId().toString()};

    return credentialDatabaseHelper.getWritableDatabase().delete(URLCredentialContract.FeedEntry.TABLE_NAME, whereClause, whereArgs);
  }

}
