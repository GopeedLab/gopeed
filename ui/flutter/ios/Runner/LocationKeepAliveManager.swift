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
        let status = locationManager.authorizationStatus

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

        guard locationManager.authorizationStatus == .authorizedAlways else {
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

    func locationManagerDidChangeAuthorization(
        _ manager: CLLocationManager
    ) {
        let status = manager.authorizationStatus

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

    func locationManager(
        _ manager: CLLocationManager,
        didUpdateLocations locations: [CLLocation]
    ) {
    }
}