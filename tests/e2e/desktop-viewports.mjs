// CHNHCM-012 §6.2. V01 is the Staffing identifier for the standard desktop.
/** @type {ReadonlyArray<readonly [string, number, number]>} */
export const desktopViewportCases = [
  ['V01', 1440, 900],
];

/** @type {ReadonlyArray<readonly [number, number]>} */
export const desktopViewports = desktopViewportCases.map(([, width, height]) => [width, height]);

// Fixed starting window for the personnel forms' 200% keyboard and scrolling checks.
export const desktopFormViewports = [
  ...desktopViewports.map(([width, height]) => [width, height, 1]),
  [1280, 900, 2],
];

// Native Chromium shortcut steps and the resulting zoom factor.
/** @type {ReadonlyArray<readonly [number, number]>} */
export const nativeZoomCases = [[5, 2]];

// Standalone viewport runs also drive the browser's native zoom shortcuts.
// Historical phone captures can be reproduced by selecting their names explicitly.
export function scenarioViewports(selection = process.env.CHNHCM_UI_CONDITIONS) {
  const [[width, height]] = desktopViewports;
  /** @type {Array<[string, number, number, number]>} */
  const options = [
    ['desktop', width, height, 0],
    ['narrow', 390, 844, 0],
    ['width320', 320, 900, 0],
    ['stress', 320, 256, 0],
    ['zoom200', 1280, 900, 5],
  ];
  const names = selection === undefined ? ['desktop', 'zoom200'] : selection.split(',').map(name => name.trim());
  for (const name of names) {
    if (!options.some(([label]) => label === name)) throw new Error(`Unknown viewport condition: ${name}`);
  }
  return options.filter(([label]) => names.includes(label));
}
