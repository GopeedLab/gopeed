import 'dart:math';

final _androidClientIdPattern = RegExp(r'^android-v1-[0-9a-f]{32}$');

/// Reuses installation IDs stored in the existing analyticsClientId preference.
/// The prefix distinguishes them from Android Build.ID values saved by older
/// versions, which can be shared by many devices running the same OS build.
String resolveAndroidAnalyticsClientId(String storedClientId) {
  if (_androidClientIdPattern.hasMatch(storedClientId)) return storedClientId;

  final random = Random.secure();
  final id = List.generate(16, (_) => random.nextInt(256).toRadixString(16).padLeft(2, '0')).join();
  return 'android-v1-$id';
}
