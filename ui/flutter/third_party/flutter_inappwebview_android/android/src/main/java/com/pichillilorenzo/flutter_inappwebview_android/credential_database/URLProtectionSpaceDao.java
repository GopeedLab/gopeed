package com.pichillilorenzo.flutter_inappwebview_android.credential_database;

import android.content.ContentValues;
import android.database.Cursor;
import android.database.sqlite.SQLiteDatabase;

import com.pichillilorenzo.flutter_inappwebview_android.types.URLProtectionSpace;

import java.util.ArrayList;
import java.util.List;

public class URLProtectionSpaceDao {
  CredentialDatabaseHelper credentialDatabaseHelper;
  String[] projection = {
          URLProtectionSpaceContract.FeedEntry._ID,
          URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_HOST,
          URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_PROTOCOL,
          URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_REALM,
          URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_PORT
  };

  public URLProtectionSpaceDao(CredentialDatabaseHelper credentialDatabaseHelper) {
    this.credentialDatabaseHelper = credentialDatabaseHelper;
  }

  public List<URLProtectionSpace> getAll() {
    SQLiteDatabase readableDatabase = credentialDatabaseHelper.getReadableDatabase();

    Cursor cursor = readableDatabase.query(
            URLProtectionSpaceContract.FeedEntry.TABLE_NAME,
            projection,
            null,
            null,
            null,
            null,
            null
    );

    List<URLProtectionSpace> URLProtectionSpaces = new ArrayList<>();
    while (cursor.moveToNext()) {
      Long rowId = cursor.getLong(cursor.getColumnIndexOrThrow(URLProtectionSpaceContract.FeedEntry._ID));
      String rowHost = cursor.getString(cursor.getColumnIndexOrThrow(URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_HOST));
      String rowProtocol = cursor.getString(cursor.getColumnIndexOrThrow(URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_PROTOCOL));
      String rowRealm = cursor.getString(cursor.getColumnIndexOrThrow(URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_REALM));
      Integer rowPort = cursor.getInt(cursor.getColumnIndexOrThrow(URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_PORT));
      URLProtectionSpaces.add(new URLProtectionSpace(rowId, rowHost, rowProtocol, rowRealm, rowPort));
    }
    cursor.close();

    return URLProtectionSpaces;
  }

  public URLProtectionSpace find(String host, String protocol, String realm, Integer port) {
    SQLiteDatabase readableDatabase = credentialDatabaseHelper.getReadableDatabase();

    String selection = URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_HOST + " = ? AND " + URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_PROTOCOL + " = ? AND " +
            URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_REALM + " = ? AND " + URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_PORT + " = ?";
    String[] selectionArgs = {host, protocol, realm, port.toString()};

    Cursor cursor = readableDatabase.query(
            URLProtectionSpaceContract.FeedEntry.TABLE_NAME,
            projection,
            selection,
            selectionArgs,
            null,
            null,
            null
    );

    URLProtectionSpace URLProtectionSpace = null;
    if (cursor.moveToNext()) {
      Long rowId = cursor.getLong(cursor.getColumnIndexOrThrow(URLProtectionSpaceContract.FeedEntry._ID));
      String rowHost = cursor.getString(cursor.getColumnIndexOrThrow(URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_HOST));
      String rowProtocol = cursor.getString(cursor.getColumnIndexOrThrow(URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_PROTOCOL));
      String rowRealm = cursor.getString(cursor.getColumnIndexOrThrow(URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_REALM));
      Integer rowPort = cursor.getInt(cursor.getColumnIndexOrThrow(URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_PORT));
      URLProtectionSpace = new URLProtectionSpace(rowId, rowHost, rowProtocol, rowRealm, rowPort);
    }
    cursor.close();

    return URLProtectionSpace;
  }

  public long insert(URLProtectionSpace URLProtectionSpace) {
    ContentValues protectionSpaceValues = new ContentValues();
    protectionSpaceValues.put(URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_HOST, URLProtectionSpace.getHost());
    protectionSpaceValues.put(URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_PROTOCOL, URLProtectionSpace.getProtocol());
    protectionSpaceValues.put(URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_REALM, URLProtectionSpace.getRealm());
    protectionSpaceValues.put(URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_PORT, URLProtectionSpace.getPort());

    return credentialDatabaseHelper.getWritableDatabase().insert(URLProtectionSpaceContract.FeedEntry.TABLE_NAME, null, protectionSpaceValues);
  };

  public long delete(URLProtectionSpace URLProtectionSpace) {
    String whereClause = URLProtectionSpaceContract.FeedEntry._ID + " = ?";
    String[] whereArgs = {URLProtectionSpace.getId().toString()};

    return credentialDatabaseHelper.getWritableDatabase().delete(URLProtectionSpaceContract.FeedEntry.TABLE_NAME, whereClause, whereArgs);
  }
}