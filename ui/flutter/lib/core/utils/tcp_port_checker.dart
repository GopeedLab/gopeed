import 'tcp_port_checker_stub.dart' if (dart.library.io) 'tcp_port_checker_native.dart' as implementation;

/// Returns whether a TCP connection can be established to [host]:[port].
///
/// The settings page uses this before saving a changed API listen address,
/// matching the legacy UI's occupied-port check. Unsupported platforms return
/// false so validation never blocks configuration that cannot be probed locally.
Future<bool> isTcpPortInUse(String host, int port) => implementation.isTcpPortInUse(host, port);
