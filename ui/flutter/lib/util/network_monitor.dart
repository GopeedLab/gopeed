import 'dart:async';
import 'dart:io';

import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:get/get.dart';

import '../api/api.dart';
import '../database/database.dart';
import '../util/log_util.dart';

class NetworkMonitor extends GetxService {
  static NetworkMonitor get to => Get.find();

  late final Connectivity _connectivity;
  late StreamSubscription<List<ConnectivityResult>> _connectivitySubscription;
  
  final _isWiFiConnected = false.obs;
  bool get isWiFiConnected => _isWiFiConnected.value;
  
  bool _wasWiFiConnected = false;
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
    _updateWiFiStatus(initialResult);
    _wasWiFiConnected = _isWiFiConnected.value;
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

  void _updateWiFiStatus(List<ConnectivityResult> result) {
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

    _updateWiFiStatus(result);
    final currentlyWiFi = _isWiFiConnected.value;
    
    logInfo('Network changed: wasWiFi=$_wasWiFiConnected, currentlyWiFi=$currentlyWiFi');
    
    // WiFi to Mobile Data: pause all tasks
    if (_wasWiFiConnected && !currentlyWiFi) {
      _pauseAllDownloads();
      logInfo('Switched from WiFi to mobile data - pausing all downloads');
    }
    // Mobile Data to WiFi: resume all tasks
    else if (!_wasWiFiConnected && currentlyWiFi) {
      _resumeAllDownloads();
      logInfo('Switched from mobile data to WiFi - resuming all downloads');
    }
    
    _wasWiFiConnected = currentlyWiFi;
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