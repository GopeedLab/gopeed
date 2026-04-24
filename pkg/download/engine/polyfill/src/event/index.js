import { Event, EventTarget } from "event-target-shim";

if (typeof globalThis.Event !== "function") {
  globalThis.Event = Event;
}

if (typeof globalThis.EventTarget !== "function") {
  globalThis.EventTarget = EventTarget;
}
