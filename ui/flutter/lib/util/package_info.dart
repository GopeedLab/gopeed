import 'package:package_info_plus/package_info_plus.dart';

late PackageInfo packageInfo;

String get appVersion => normalizeAppVersion(packageInfo.version);

String normalizeAppVersion(String platformVersion) {
  final version = platformVersion.trim();
  final parts = version.split('.');
  if (parts.length != 4 || parts.any((part) => int.tryParse(part) == null)) return version;
  return '${parts[0]}.${parts[1]}.${parts[2]}-beta.${parts[3]}';
}

Future<void> initPackageInfo() async {
  packageInfo = await PackageInfo.fromPlatform();
}
