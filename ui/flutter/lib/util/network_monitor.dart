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
    
    // Only monitor network on mobile platforms
    if (!Platform.isAndroid && !Platform.isIOS) {
      return;
    }
    
    _connectivity = Connectivity();
    
    // Check initial connectivity state
    final initialResult = await _connectivity.checkConnectivity();
    _updateNetworkStatus(initialResult);
    _previousNetworkType = _getNetworkType(initialResult);
    _isInitialized = true;
    
    // Listen to connectivity changes
    _connectivitySubscription = _connectivity.onConnectivityChanged.listen(
      _onConnectivityChanged,
      onError: (error) {
        logError('Network monitoring error: $error');
      },
    );
    
    logInfo('Network monitor initialized');
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
    if (!_isInitialized) return;
    
    // Check if network auto control is enabled
    final isNetworkAutoControlEnabled = Database.instance.getNetworkAutoControl();
    if (!isNetworkAutoControlEnabled) {
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