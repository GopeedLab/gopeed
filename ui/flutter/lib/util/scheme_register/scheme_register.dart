import 'scheme_register_stub.dart' if (dart.library.io) 'entry/scheme_register_native.dart';

void registerUrlScheme(String scheme) => doRegisterUrlScheme(scheme);

void unregisterUrlScheme(String scheme) => doUnregisterUrlScheme(scheme);

void registerDefaultTorrentClient() => doRegisterDefaultTorrentClient();

void unregisterDefaultTorrentClient() => doUnregisterDefaultTorrentClient();
