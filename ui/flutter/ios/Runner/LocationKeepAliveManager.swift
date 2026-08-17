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

        case .denied, .restricted:
            completion(false)

        case .notDetermined, .authorizedWhenInUse:
            permissionCompletion = completion
            locationManager.requestAlwaysAuthorization()

        @unknown default:
            completion(false)
        }
    }

    func start() {
        guard !isRunning else { return }

        guard CLLocationManager.authorizationStatus() == .authorizedAlways else {
            return
        }

        locationManager.startUpdatingLocation()
        isRunning = true
    }

    func stop() {
        guard isRunning else { return }

        locationManager.stopUpdatingLocation()
        isRunning = false
    }

    private func handleAuthorizationStatus(_ status: CLAuthorizationStatus) {
        switch status {
        case .authorizedAlways:
            permissionCompletion?(true)
            permissionCompletion = nil

        case .denied, .restricted:
            permissionCompletion?(false)
            permissionCompletion = nil

        default:
            break
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
}