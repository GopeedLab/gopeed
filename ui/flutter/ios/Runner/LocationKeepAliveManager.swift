import Foundation
import CoreLocation

final class LocationKeepAliveManager: NSObject, CLLocationManagerDelegate {

    static let shared = LocationKeepAliveManager()

    private let locationManager = CLLocationManager()

    private(set) var isRunning = false

    private var permissionCompletion: ((Bool) -> Void)?

    override init() {
        super.init()

        locationManager.delegate = self
        locationManager.desiredAccuracy = kCLLocationAccuracyThreeKilometers
        locationManager.distanceFilter = kCLDistanceFilterNone
        locationManager.pausesLocationUpdatesAutomatically = false

        if #available(iOS 9.0, *) {
            locationManager.allowsBackgroundLocationUpdates = true
        }

        if #available(iOS 14.0, *) {
            locationManager.showsBackgroundLocationIndicator = true
        }
    }

    func requestPermission(completion: @escaping (Bool) -> Void) {
        let status = CLLocationManager.authorizationStatus()

        switch status {
        case .authorizedAlways:
            completion(true)

        case .authorizedWhenInUse:
            completion(false)

        case .denied, .restricted:
            completion(false)

        case .notDetermined:
            permissionCompletion = completion
            locationManager.requestAlwaysAuthorization()

        @unknown default:
            completion(false)
        }
    }


    func start() {
        guard !isRunning else {
            return
        }

        let status = CLLocationManager.authorizationStatus()

        guard status == .authorizedAlways else {
            return
        }

        if #available(iOS 9.0, *) {
            locationManager.allowsBackgroundLocationUpdates = true
        }

        locationManager.startUpdatingLocation()
        isRunning = true
    }

    func stop() {
        locationManager.stopUpdatingLocation()
        isRunning = false
    }

    private func handleAuthorizationStatus(
        _ status: CLAuthorizationStatus
    ) {
        switch status {
        case .authorizedAlways:
            permissionCompletion?(true)
            permissionCompletion = nil

        case .authorizedWhenInUse,
             .denied,
             .restricted:
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
    func locationManagerDidChangeAuthorization(
        _ manager: CLLocationManager
    ) {
        handleAuthorizationStatus(manager.authorizationStatus)
    }

    func locationManager(
        _ manager: CLLocationManager,
        didChangeAuthorization status: CLAuthorizationStatus
    ) {
        handleAuthorizationStatus(status)
    }

    func locationManager(
        _ manager: CLLocationManager,
        didUpdateLocations locations: [CLLocation]
    ) {
    }

    func locationManager(
        _ manager: CLLocationManager,
        didFailWithError error: Error
    ) {
    }
}