import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../api/model/install_extension.dart';

final pendingExtensionInstallProvider = NotifierProvider<PendingExtensionInstall, InstallExtension?>(
  PendingExtensionInstall.new,
);

class PendingExtensionInstall extends Notifier<InstallExtension?> {
  @override
  InstallExtension? build() => null;

  void set(InstallExtension? install) {
    state = install;
  }

  void clear() {
    state = null;
  }
}
