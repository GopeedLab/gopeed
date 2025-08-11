import 'dart:async';
import 'dart:io';

import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:get/get.dart';

import '../api/api.dart';
import '../database/database.dart';
import '../util/log_util.dart';

enum NetworkType { wifi, mobile, other, none }

class NetworkMonitor extends GetxService {
  static NetworkMonitor get to => Get.find();

  late final Connectivity _connectivity;
  late StreamSubscription<List<ConnectivityResult>> _connectivitySubscription;
  
  final _isWiFiConnected = false.obs;
  bool get isWiFiConnected => _isWiFiConnected.value;
  
  NetworkType _previousNetworkType = NetworkType.none;
  bool _isInitialized = false;

  @override
  Future<void> onInit() async {
    super.onInit();
    
    logInfo('NetworkMonitor.onInit() called');
    
    // Only monitor network on mobile platforms
    if (!Platform.isAndroid && !Platform.isIOS) {
      logInfo('NetworkMonitor: Not on mobile platform, skipping initialization');
      return;
    }
    
    _connectivity = Connectivity();
    
    // Check initial connectivity state
    final initialResult = await _connectivity.checkConnectivity();
    logInfo('NetworkMonitor: Initial connectivity result: $initialResult');
    _updateNetworkStatus(initialResult);
    _previousNetworkType = _getNetworkType(initialResult);
    logInfo('NetworkMonitor: Initial network type: $_previousNetworkType');
    _isInitialized = true;
    
    // Listen to connectivity changes
    _connectivitySubscription = _connectivity.onConnectivityChanged.listen(
      _onConnectivityChanged,
      onError: (error) {
        logError('Network monitoring error: $error');
      },
    );
    
    logInfo('Network monitor initialized successfully');
  }

  @override
  void onClose() {
    _connectivitySubscription.cancel();
    super.onClose();
  }

  NetworkType _getNetworkType(List<ConnectivityResult> result) {
    if (result.contains(ConnectivityResult.wifi)) {
      return NetworkType.wifi;
    } else if (result.contains(ConnectivityResult.mobile)) {
      return NetworkType.mobile;
    } else if (result.contains(ConnectivityResult.none)) {
      return NetworkType.none;
    } else {
      // Ethernet, Bluetooth, VPN, or other connection types
      return NetworkType.other;
    }
  }

  void _updateNetworkStatus(List<ConnectivityResult> result) {
    final hasWiFi = result.contains(ConnectivityResult.wifi);
    _isWiFiConnected.value = hasWiFi;
  }

  void _onConnectivityChanged(List<ConnectivityResult> result) {
    logInfo('NetworkMonitor: Connectivity changed event received: $result');
    
    if (!_isInitialized) {
      logInfo('NetworkMonitor: Not initialized yet, ignoring connectivity change');
      return;
    }
    
    // Check if network auto control is enabled
    final isNetworkAutoControlEnabled = Database.instance.getNetworkAutoControl();
    logInfo('NetworkMonitor: Network auto control enabled: $isNetworkAutoControlEnabled');
    if (!isNetworkAutoControlEnabled) {
      logInfo('NetworkMonitor: Network auto control disabled, ignoring connectivity change');
      return;
    }

    _updateNetworkStatus(result);
    final currentNetworkType = _getNetworkType(result);
    
    logInfo('Network changed: previous=$_previousNetworkType, current=$currentNetworkType');
    
    // Only handle WiFi ↔ Mobile transitions, ignore other connection types
    if (_previousNetworkType == NetworkType.wifi && currentNetworkType == NetworkType.mobile) {
      _pauseAllDownloads();
      logInfo('Switched from WiFi to mobile data - pausing all downloads');
    } else if (_previousNetworkType == NetworkType.mobile && currentNetworkType == NetworkType.wifi) {
      _resumeAllDownloads();
      logInfo('Switched from mobile data to WiFi - resuming all downloads');
    } else {
      logInfo('Network change ignored: not a WiFi ↔ Mobile transition');
    }
    
    _previousNetworkType = currentNetworkType;
  }

  Future<void> _pauseAllDownloads() async {
    try {
      await pauseAllTasks(null);
      logInfo('Successfully paused all downloads due to network change');
    } catch (e) {
      logError('Failed to pause all downloads: $e');
    }
  }

  Future<void> _resumeAllDownloads() async {
    try {
      await continueAllTasks(null);
      logInfo('Successfully resumed all downloads due to network change');
    } catch (e) {
      logError('Failed to resume all downloads: $e');
    }
  }
}