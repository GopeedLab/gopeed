package com.pichillilorenzo.flutter_inappwebview_android.credential_database;

import android.content.Context;
import android.database.sqlite.SQLiteDatabase;
import android.database.sqlite.SQLiteOpenHelper;

public class CredentialDatabaseHelper extends SQLiteOpenHelper {

  private static final String SQL_CREATE_PROTECTION_SPACE_TABLE =
          "CREATE TABLE " + URLProtectionSpaceContract.FeedEntry.TABLE_NAME + " (" +
                  URLProtectionSpaceContract.FeedEntry._ID + " INTEGER PRIMARY KEY," +
                  URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_HOST + " TEXT NOT NULL," +
                  URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_PROTOCOL + " TEXT," +
                  URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_REALM + " TEXT," +
                  URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_PORT + " INTEGER," +
                  "UNIQUE(" + URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_HOST + ", " + URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_PROTOCOL + ", " +
                  URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_REALM + ", " + URLProtectionSpaceContract.FeedEntry.COLUMN_NAME_PORT +
                  ")" +
          ");";

  private static final String SQL_CREATE_CREDENTIAL_TABLE =
          "CREATE TABLE " + URLCredentialContract.FeedEntry.TABLE_NAME + " (" +
                  URLCredentialContract.FeedEntry._ID + " INTEGER PRIMARY KEY," +
                  URLCredentialContract.FeedEntry.COLUMN_NAME_USERNAME + " TEXT NOT NULL," +
                  URLCredentialContract.FeedEntry.COLUMN_NAME_PASSWORD + " TEXT NOT NULL," +
                  URLCredentialContract.FeedEntry.COLUMN_NAME_PROTECTION_SPACE_ID + " INTEGER NOT NULL," +
                  "UNIQUE(" + URLCredentialContract.FeedEntry.COLUMN_NAME_USERNAME + ", " + URLCredentialContract.FeedEntry.COLUMN_NAME_PASSWORD + ", " +
                  URLCredentialContract.FeedEntry.COLUMN_NAME_PROTECTION_SPACE_ID +
                  ")," +
                  "FOREIGN KEY (" + URLCredentialContract.FeedEntry.COLUMN_NAME_PROTECTION_SPACE_ID + ") REFERENCES " +
                  URLProtectionSpaceContract.FeedEntry.TABLE_NAME + " (" + URLProtectionSpaceContract.FeedEntry._ID + ") ON DELETE CASCADE" +
          ");";

  private static final String SQL_DELETE_PROTECTION_SPACE_TABLE =
          "DROP TABLE IF EXISTS " + URLProtectionSpaceContract.FeedEntry.TABLE_NAME;

  private static final String SQL_DELETE_CREDENTIAL_TABLE =
          "DROP TABLE IF EXISTS " + URLCredentialContract.FeedEntry.TABLE_NAME;

  public CredentialDatabaseHelper(Context context) {
    super(context, CredentialDatabase.DATABASE_NAME, null, CredentialDatabase.DATABASE_VERSION);
  }

  public void onCreate(SQLiteDatabase db) {
    db.execSQL(SQL_CREATE_PROTECTION_SPACE_TABLE);
    db.execSQL(SQL_CREATE_CREDENTIAL_TABLE);
  }

  public void onUpgrade(SQLiteDatabase db, int oldVersion, int newVersion) {
    // This database is only a cache for online data, so its upgrade policy is
    // to simply to discard the data and start over
    db.execSQL(SQL_DELETE_PROTECTION_SPACE_TABLE);
    db.execSQL(SQL_DELETE_CREDENTIAL_TABLE);
    onCreate(db);
  }

  public void onDowngrade(SQLiteDatabase db, int oldVersion, int newVersion) {
    onUpgrade(db, oldVersion, newVersion);
  }

  public void clearAllTables(SQLiteDatabase db) {
    db.execSQL(SQL_DELETE_PROTECTION_SPACE_TABLE);
    db.execSQL(SQL_DELETE_CREDENTIAL_TABLE);
    onCreate(db);
  }
}
