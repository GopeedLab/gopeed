/** Public extension API additions. info.version remains the extension version. */
export interface ExtensionInfo {
  identity: string;
  name: string;
  author: string;
  title: string;
  version: string;
  hostVersion: string;
}

/** Functions are serialized; they cannot capture extension-side variables. */
export type WebviewExecutable<T = unknown> =
  | string
  | ((...args: unknown[]) => T | Promise<T>);

export type WebviewUnsubscribe = () => void;

/** Navigation and load notifications are for the main document only. */
export interface WebviewEventMap {
  "url-changed": { url: string; sameDocument: boolean };
  /** window.load, including reloads, excluding hash/History API navigation. */
  "load": { url: string };
  "load-error": { url: string; message: string };
  "closed": { reason: "user" | "api" };
}

export interface WebviewGotoOptions {
  timeoutMs?: number;
  waitUntil?: "load" | "domcontentloaded";
}

/** These members extend the existing WebviewPage API. */
export interface WebviewPage {
  /**
   * Resolves when listening is ready. Does not replay current state.
   * Register before goto() to observe navigation. Unsubscribe is idempotent.
   * Closed is delivered once, then listeners are released automatically.
   * Callbacks cannot cancel navigation; returned promises are not awaited.
   * Callers handle errors in asynchronous callbacks.
   */
  on<K extends keyof WebviewEventMap>(
    event: K,
    handler: (data: WebviewEventMap[K]) => void,
  ): Promise<WebviewUnsubscribe>;
  /** Runs at document creation on future navigations, never immediately. */
  addInitScript(script: string): Promise<void>;
  goto(url: string, opts?: WebviewGotoOptions): Promise<void>;
}
