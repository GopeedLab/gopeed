class ApiServerState {
  const ApiServerState({
    required this.enabled,
    required this.running,
    required this.network,
    required this.address,
    required this.runningPort,
    required this.pendingApply,
    required this.lastError,
  });

  final bool enabled;
  final bool running;
  final String network;
  final String address;
  final int runningPort;
  final bool pendingApply;
  final String lastError;

  factory ApiServerState.fromJson(Map<String, dynamic> json) {
    return ApiServerState(
      enabled: json['enabled'] as bool? ?? false,
      running: json['running'] as bool? ?? false,
      network: json['network'] as String? ?? '',
      address: json['address'] as String? ?? '',
      runningPort: json['runningPort'] as int? ?? 0,
      pendingApply: json['pendingApply'] as bool? ?? false,
      lastError: json['lastError'] as String? ?? '',
    );
  }

  String runningAddress() {
    if (!running) return '';
    if (network == 'unix') return address;
    final separator = address.lastIndexOf(':');
    final host = separator < 0 ? address : address.substring(0, separator);
    return '$host:$runningPort';
  }
}

class ApiServerOperationResult {
  const ApiServerOperationResult({required this.state, required this.error});

  final ApiServerState state;
  final String error;

  factory ApiServerOperationResult.fromJson(Map<String, dynamic> json) {
    final rawState = json['state'] as Map<String, dynamic>?;
    return ApiServerOperationResult(
      state: rawState == null
          ? const ApiServerState(
              enabled: false,
              running: false,
              network: '',
              address: '',
              runningPort: 0,
              pendingApply: true,
              lastError: '',
            )
          : ApiServerState.fromJson(rawState),
      error: json['error'] as String? ?? '',
    );
  }
}
