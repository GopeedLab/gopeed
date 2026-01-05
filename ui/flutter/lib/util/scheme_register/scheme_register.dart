import 'scheme_register_stub.dart'
    if (dart.library.io) 'entry/scheme_register_native.dart';

registerUrlScheme(String scheme) => doRegisterUrlScheme(scheme);

unregisterUrlScheme(String scheme) => doUnregisterUrlScheme(scheme);

registerDefaultTorrentClient() => doRegisterDefaultTorrentClient();

unregisterDefaultTorrentClient() => doUnregisterDefaultTorrentClient();
