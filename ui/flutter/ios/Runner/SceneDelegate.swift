import Flutter
import UIKit
import app_links

class SceneDelegate: FlutterSceneDelegate {
  override func scene(
    _ scene: UIScene,
    willConnectTo session: UISceneSession,
    options connectionOptions: UIScene.ConnectionOptions
  ) {
    super.scene(scene, willConnectTo: session, options: connectionOptions)
    forwardGopeedLinks(connectionOptions.urlContexts)
  }

  override func scene(_ scene: UIScene, openURLContexts URLContexts: Set<UIOpenURLContext>) {
    super.scene(scene, openURLContexts: URLContexts)
    forwardGopeedLinks(URLContexts)
  }

  private func forwardGopeedLinks(_ contexts: Set<UIOpenURLContext>) {
    // app_links 6.x only registers AppDelegate callbacks. UIScene delivers
    // both cold-start and running-app scheme links here instead.
    for context in contexts where context.url.scheme?.lowercased() == "gopeed" {
      AppLinks.shared.handleLink(url: context.url)
    }
  }
}
