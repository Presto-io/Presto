/**
 * Stub for @wailsio/runtime in static builds (showcase)
 * Provides empty implementations to avoid build errors
 */

export const Events = {
	On: () => {},
	Off: () => {},
	Emit: () => {},
};

export const Window = {
	Maximise: () => {},
	Unmaximise: () => {},
	Minimise: () => {},
	Unminimise: () => {},
	Close: () => {},
};

export const Browser = {
	OpenURL: () => {},
};

export default {
	Events,
	Window,
	Browser,
};
