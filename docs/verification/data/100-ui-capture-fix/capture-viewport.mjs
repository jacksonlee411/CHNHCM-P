import { writeFileSync } from 'node:fs';
import { randomUUID } from 'node:crypto';

// Chromium's native page zoom changes CSS metrics. Omitting CDP's clip captures
// the complete visible surface without treating CSS dimensions as pixel bounds.
// This helper neither changes zoom nor emulates a different viewport.
export async function captureViewport(page, filePath, mask = []) {
  const dimensions = await page.evaluate(() => ({
    width: innerWidth, height: innerHeight, dpr: devicePixelRatio,
    outerWidth, outerHeight, visualScale: visualViewport.scale,
  }));
  if (dimensions.visualScale !== 1) throw new Error('Pinch zoom is not supported by this capture lane');
  const rectangles = [];
  for (const locator of mask) {
    rectangles.push(...await locator.evaluateAll(elements => elements.flatMap(element => {
      const r = element.getBoundingClientRect();
      const left = Math.max(0, r.left), top = Math.max(0, r.top);
      const right = Math.min(innerWidth, r.right), bottom = Math.min(innerHeight, r.bottom);
      return right > left && bottom > top ? [{ left, top, width: right - left, height: bottom - top }] : [];
    })));
  }
  const id = 'evidence-mask-' + randomUUID();
  let session;
  try {
    await page.evaluate(({ id, rectangles }) => {
      const host = document.createElement('div');
      host.id = id;
      host.style.cssText = 'position:fixed!important;inset:0!important;pointer-events:none!important;z-index:2147483647!important;';
      for (const r of rectangles) {
        const block = document.createElement('div');
        block.style.cssText = `position:absolute!important;left:${r.left}px!important;top:${r.top}px!important;width:${r.width}px!important;height:${r.height}px!important;background:#ff00ff!important;border:0!important;margin:0!important;padding:0!important;`;
        host.appendChild(block);
      }
      document.documentElement.appendChild(host);
      return new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));
    }, { id, rectangles });
    session = await page.context().newCDPSession(page);
    const { data } = await session.send('Page.captureScreenshot', {
      format: 'png', fromSurface: true, captureBeyondViewport: false,
    });
    const png = Buffer.from(data, 'base64');
    const pixels = { width: png.readUInt32BE(16), height: png.readUInt32BE(20) };
    if (pixels.width !== Math.round(dimensions.width * dimensions.dpr) ||
        pixels.height !== Math.round(dimensions.height * dimensions.dpr)) {
      throw new Error('Visible surface dimensions do not match the native zoom viewport');
    }
    // Check the actual captured pixels before persisting any image with identities.
    const masks = await page.evaluate(async ({ data, rectangles, dpr }) => {
      const image = new Image(); image.src = 'data:image/png;base64,' + data; await image.decode();
      const canvas = document.createElement('canvas'); canvas.width = image.width; canvas.height = image.height;
      const context = canvas.getContext('2d'); context.drawImage(image, 0, 0);
      return rectangles.map(r => {
        const x = Math.floor((r.left + r.width / 2) * dpr), y = Math.floor((r.top + r.height / 2) * dpr);
        const pixel = [...context.getImageData(x, y, 1, 1).data];
        return { x, y, pixel, covered: pixel.join(',') === '255,0,255,255' };
      });
    }, { data, rectangles, dpr: dimensions.dpr });
    if (masks.some(m => !m.covered)) throw new Error('Identity mask pixel check failed');
    writeFileSync(filePath, png, { flag: 'wx' });
    return { dimensions, pixels, maskCount: rectangles.length, masks, method: 'CDP visible surface without clip' };
  } finally {
    try { await page.evaluate(id => document.getElementById(id)?.remove(), id); }
    finally { if (session) await session.detach(); }
  }
}
