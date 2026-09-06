import 'package:win32_registry/win32_registry.dart';

/// Check registry key
/// If the key does not exist or the value is different, return false
bool checkRegistry(String keyPath, String valueName, String value) {
  RegistryKey? regKey;
  try {
    regKey = CURRENT_USER.open(keyPath);
    return regKey.getString(valueName) == value;
  } catch (e) {
    return false;
  } finally {
    regKey?.close();
  }
}

/// Upsert registry key
/// If the key does not exist, create it
/// If the value does not exist or is different, update it
void upsertRegistry(String keyPath, String valueName, String value) {
  final regKey = CURRENT_USER.create(
    keyPath,
    config: const RegistryOpenConfig(access: RegistryAccess.all, create: true),
  );
  try {
    if (regKey.getString(valueName) != value) {
      regKey.setValue(valueName, RegistryValue.string(value));
    }
  } finally {
    regKey.close();
  }
}
