import CoreLocation
import Foundation

final class LocationKeepAliveManager: NSObject, CLLocationManagerDelegate {
  static let shared = LocationKeepAliveManager()

  private let locationManager = CLLocationManager()
  private var permissionCompletion: ((Bool) -> Void)?
  private(set) var isRunning = false

  private var authorizationStatus: CLAuthorizationStatus {
    if #available(iOS 14.0, *) {
      return locationManager.authorizationStatus
    }
    return CLLocationManager.authorizationStatus()
  }

  override init() {
    super.init()
    locationManager.delegate = self
    locationManager.desiredAccuracy = kCLLocationAccuracyThreeKilometers
    locationManager.distanceFilter = kCLDistanceFilterNone
    locationManager.pausesLocationUpdatesAutomatically = false
    locationManager.allowsBackgroundLocationUpdates = true
    if #available(iOS 14.0, *) {
      locationManager.showsBackgroundLocationIndicator = true
    }
  }

  func requestPermission(completion: @escaping (Bool) -> Void) {
    switch authorizationStatus {
    case .authorizedAlways:
      completion(true)
    case .authorizedWhenInUse, .denied, .restricted:
      completion(false)
    case .notDetermined:
      permissionCompletion = completion
      locationManager.requestAlwaysAuthorization()
    @unknown default:
      completion(false)
    }
  }

  func start() {
    guard !isRunning, authorizationStatus == .authorizedAlways else { return }
    locationManager.startUpdatingLocation()
    isRunning = true
  }

  func stop() {
    locationManager.stopUpdatingLocation()
    isRunning = false
  }

  private func handleAuthorization(_ status: CLAuthorizationStatus) {
    switch status {
    case .authorizedAlways:
      permissionCompletion?(true)
      permissionCompletion = nil
    case .authorizedWhenInUse, .denied, .restricted:
      permissionCompletion?(false)
      permissionCompletion = nil
    case .notDetermined:
      break
    @unknown default:
      permissionCompletion?(false)
      permissionCompletion = nil
    }
  }

  @available(iOS 14.0, *)
  func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
    handleAuthorization(manager.authorizationStatus)
  }

  func locationManager(_ manager: CLLocationManager, didChangeAuthorization status: CLAuthorizationStatus) {
    handleAuthorization(status)
  }

  func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {}

  func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {}
}
