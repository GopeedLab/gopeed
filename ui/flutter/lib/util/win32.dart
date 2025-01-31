import 'package:win32_registry/win32_registry.dart';

/// Check registry key
/// If the key does not exist or the value is different, return false
checkRegistry(String keyPath, String valueName, String value) {
  RegistryKey regKey;
  try {
    regKey = Registry.openPath(RegistryHive.currentUser, path: keyPath);
    return regKey.getValueAsString(valueName) == value;
  } catch (e) {
    return false;
  }
}

/// Upsert registry key
/// If the key does not exist, create it
/// If the value does not exist or is different, update it
upsertRegistry(String keyPath, String valueName, String value) {
  RegistryKey regKey;
  try {
    regKey = Registry.openPath(RegistryHive.currentUser,
        path: keyPath, desiredAccessRights: AccessRights.allAccess);
  } catch (e) {
    regKey = Registry.currentUser.createKey(keyPath);
  }

  if (regKey.getValueAsString(valueName) != value) {
    regKey
        .createValue(RegistryValue(valueName, RegistryValueType.string, value));
  }
  regKey.close();
}
